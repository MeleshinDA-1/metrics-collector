package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/MeleshinDA-1/metrics-collector/internal/model"
	"github.com/MeleshinDA-1/metrics-collector/internal/retry"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	setGaugeQuery = `INSERT INTO gauges (metric_name, value) VALUES ($1, $2)
		 ON CONFLICT (metric_name) DO UPDATE
		 SET value = EXCLUDED.value, updated_at_utc = now()`

	addCounterQuery = `INSERT INTO counters (metric_name, value) VALUES ($1, $2)
		 ON CONFLICT (metric_name) DO UPDATE
		 SET value = counters.value + EXCLUDED.value, updated_at_utc = now()`

	addCounterReturningQuery = addCounterQuery + ` RETURNING value`
)

type DbMetricStorage struct {
	pool        *pgxpool.Pool
	retryPolicy retry.Policy
}

func NewDbMetricStorage(pool *pgxpool.Pool, retryPolicy retry.Policy) *DbMetricStorage {
	return &DbMetricStorage{
		pool:        pool,
		retryPolicy: retryPolicy,
	}
}

func (storage *DbMetricStorage) SetGauge(ctx context.Context, name string, value float64) error {
	err := storage.retryPolicy.Do(ctx, func(ctx context.Context) error {
		_, err := storage.pool.Exec(ctx, setGaugeQuery, name, value)
		return err
	}, isRetriablePgError)
	if err != nil {
		return fmt.Errorf("set gauge %q: %w", name, err)
	}

	return nil
}

func (storage *DbMetricStorage) AddCounter(ctx context.Context, name string, delta int64) (int64, error) {
	var value int64

	err := storage.retryPolicy.Do(ctx, func(ctx context.Context) error {
		return storage.pool.QueryRow(ctx, addCounterReturningQuery, name, delta).Scan(&value)
	}, isRetriablePgError)
	if err != nil {
		return 0, fmt.Errorf("add counter %q: %w", name, err)
	}

	return value, nil
}

func (storage *DbMetricStorage) UpdateBatch(ctx context.Context, metrics []model.Metrics) error {
	if len(metrics) == 0 {
		return nil
	}

	err := storage.retryPolicy.Do(ctx, func(ctx context.Context) error {
		return storage.updateBatch(ctx, metrics)
	}, isRetriablePgError)
	if err != nil {
		return fmt.Errorf("update batch: %w", err)
	}

	return nil
}

func (storage *DbMetricStorage) updateBatch(ctx context.Context, metrics []model.Metrics) error {
	batch := &pgx.Batch{}
	for _, metric := range metrics {
		switch metric.MType {
		case model.Gauge:
			batch.Queue(setGaugeQuery, metric.ID, *metric.Value)
		case model.Counter:
			batch.Queue(addCounterQuery, metric.ID, *metric.Delta)
		}
	}

	tx, err := storage.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	results := tx.SendBatch(ctx, batch)
	for range metrics {
		if _, err := results.Exec(); err != nil {
			_ = results.Close()
			return err
		}
	}
	if err := results.Close(); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (storage *DbMetricStorage) GetGauge(ctx context.Context, name string) (float64, bool, error) {
	var value float64

	err := storage.retryPolicy.Do(ctx, func(ctx context.Context) error {
		return storage.pool.QueryRow(ctx,
			`SELECT value FROM gauges WHERE metric_name = $1`, name).Scan(&value)
	}, isRetriablePgError)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("get gauge %q: %w", name, err)
	}

	return value, true, nil
}

func (storage *DbMetricStorage) GetCounter(ctx context.Context, name string) (int64, bool, error) {
	var value int64

	err := storage.retryPolicy.Do(ctx, func(ctx context.Context) error {
		return storage.pool.QueryRow(ctx,
			`SELECT value FROM counters WHERE metric_name = $1`, name).Scan(&value)
	}, isRetriablePgError)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("get counter %q: %w", name, err)
	}

	return value, true, nil
}

func (storage *DbMetricStorage) Snapshot(ctx context.Context) (model.MetricsSnapshot, error) {
	var snapshot model.MetricsSnapshot

	err := storage.retryPolicy.Do(ctx, func(ctx context.Context) error {
		var err error
		snapshot, err = storage.snapshot(ctx)
		return err
	}, isRetriablePgError)
	if err != nil {
		return model.MetricsSnapshot{}, fmt.Errorf("snapshot: %w", err)
	}

	return snapshot, nil
}

func (storage *DbMetricStorage) snapshot(ctx context.Context) (model.MetricsSnapshot, error) {
	snapshot := model.MetricsSnapshot{
		Gauges:   make(map[string]float64),
		Counters: make(map[string]int64),
	}

	gaugeRows, err := storage.pool.Query(ctx, `SELECT metric_name, value FROM gauges`)
	if err != nil {
		return model.MetricsSnapshot{}, err
	}
	defer gaugeRows.Close()

	for gaugeRows.Next() {
		var name string
		var value float64

		if err := gaugeRows.Scan(&name, &value); err != nil {
			return model.MetricsSnapshot{}, err
		}
		snapshot.Gauges[name] = value
	}
	if err := gaugeRows.Err(); err != nil {
		return model.MetricsSnapshot{}, err
	}

	counterRows, err := storage.pool.Query(ctx, `SELECT metric_name, value FROM counters`)
	if err != nil {
		return model.MetricsSnapshot{}, err
	}
	defer counterRows.Close()

	for counterRows.Next() {
		var name string
		var value int64

		if err := counterRows.Scan(&name, &value); err != nil {
			return model.MetricsSnapshot{}, err
		}
		snapshot.Counters[name] = value
	}
	if err := counterRows.Err(); err != nil {
		return model.MetricsSnapshot{}, err
	}

	return snapshot, nil
}
