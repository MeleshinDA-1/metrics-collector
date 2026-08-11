package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/MeleshinDA-1/metrics-collector/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DbMetricStorage struct {
	pool *pgxpool.Pool
}

func NewDbMetricStorage(pool *pgxpool.Pool) *DbMetricStorage {
	return &DbMetricStorage{
		pool: pool,
	}
}

func (storage *DbMetricStorage) SetGauge(name string, value float64) error {
	_, err := storage.pool.Exec(context.TODO(),
		`INSERT INTO gauges (metric_name, value) VALUES ($1, $2)
		 ON CONFLICT (metric_name) DO UPDATE
		 SET value = EXCLUDED.value, updated_at_utc = now()`,
		name, value)
	if err != nil {
		return fmt.Errorf("set gauge %q: %w", name, err)
	}

	return nil
}

func (storage *DbMetricStorage) AddCounter(name string, delta int64) (int64, error) {
	var value int64

	err := storage.pool.QueryRow(context.TODO(),
		`INSERT INTO counters (metric_name, value) VALUES ($1, $2)
		 ON CONFLICT (metric_name) DO UPDATE
		 SET value = counters.value + EXCLUDED.value, updated_at_utc = now()
		 RETURNING value`,
		name, delta).Scan(&value)
	if err != nil {
		return 0, fmt.Errorf("add counter %q: %w", name, err)
	}

	return value, nil
}

func (storage *DbMetricStorage) UpdateBatch(metrics []model.Metrics) error {
	if err := validateBatch(metrics); err != nil {
		return err
	}
	if len(metrics) == 0 {
		return nil
	}

	ctx := context.TODO()

	batch := &pgx.Batch{}
	for _, metric := range metrics {
		switch metric.MType {
		case model.Gauge:
			batch.Queue(
				`INSERT INTO gauges (metric_name, value) VALUES ($1, $2)
				 ON CONFLICT (metric_name) DO UPDATE
				 SET value = EXCLUDED.value, updated_at_utc = now()`,
				metric.ID, *metric.Value)
		case model.Counter:
			batch.Queue(
				`INSERT INTO counters (metric_name, value) VALUES ($1, $2)
				 ON CONFLICT (metric_name) DO UPDATE
				 SET value = counters.value + EXCLUDED.value, updated_at_utc = now()`,
				metric.ID, *metric.Delta)
		}
	}

	tx, err := storage.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	results := tx.SendBatch(ctx, batch)
	for range metrics {
		if _, err := results.Exec(); err != nil {
			_ = results.Close()
			return fmt.Errorf("update batch: %w", err)
		}
	}
	if err := results.Close(); err != nil {
		return fmt.Errorf("close batch: %w", err)
	}

	return tx.Commit(ctx)
}

func (storage *DbMetricStorage) GetGauge(name string) (float64, bool, error) {
	var value float64

	err := storage.pool.QueryRow(context.TODO(),
		`SELECT value FROM gauges WHERE metric_name = $1`, name).Scan(&value)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("get gauge %q: %w", name, err)
	}

	return value, true, nil
}

func (storage *DbMetricStorage) GetCounter(name string) (int64, bool, error) {
	var value int64

	err := storage.pool.QueryRow(context.TODO(),
		`SELECT value FROM counters WHERE metric_name = $1`, name).Scan(&value)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("get counter %q: %w", name, err)
	}

	return value, true, nil
}

func (storage *DbMetricStorage) Snapshot() (model.MetricsSnapshot, error) {
	snapshot := model.MetricsSnapshot{
		Gauges:   make(map[string]float64),
		Counters: make(map[string]int64),
	}

	gaugeRows, err := storage.pool.Query(context.TODO(),
		`SELECT metric_name, value FROM gauges`)
	if err != nil {
		return model.MetricsSnapshot{}, fmt.Errorf("select gauges: %w", err)
	}
	defer gaugeRows.Close()

	for gaugeRows.Next() {
		var name string
		var value float64

		if err := gaugeRows.Scan(&name, &value); err != nil {
			return model.MetricsSnapshot{}, fmt.Errorf("scan gauge: %w", err)
		}
		snapshot.Gauges[name] = value
	}
	if err := gaugeRows.Err(); err != nil {
		return model.MetricsSnapshot{}, fmt.Errorf("read gauges: %w", err)
	}

	counterRows, err := storage.pool.Query(context.TODO(),
		`SELECT metric_name, value FROM counters`)
	if err != nil {
		return model.MetricsSnapshot{}, fmt.Errorf("select counters: %w", err)
	}
	defer counterRows.Close()

	for counterRows.Next() {
		var name string
		var value int64

		if err := counterRows.Scan(&name, &value); err != nil {
			return model.MetricsSnapshot{}, fmt.Errorf("scan counter: %w", err)
		}
		snapshot.Counters[name] = value
	}
	if err := counterRows.Err(); err != nil {
		return model.MetricsSnapshot{}, fmt.Errorf("read counters: %w", err)
	}

	return snapshot, nil
}
