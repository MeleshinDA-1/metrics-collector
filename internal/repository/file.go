package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/MeleshinDA-1/metrics-collector/internal/model"
)

type FileMetricsRepository struct {
	FilePath string
	mutex    sync.Mutex
}

func (repo *FileMetricsRepository) Flush(ctx context.Context, storage MetricsSnapshotProvider) error {
	repo.mutex.Lock()
	defer repo.mutex.Unlock()

	file, err := os.Create(repo.FilePath)
	if err != nil {
		return err
	}
	defer file.Close()

	snapshot, err := storage.Snapshot(ctx)
	if err != nil {
		return err
	}

	metrics := make([]model.Metrics, 0, len(snapshot.Gauges)+len(snapshot.Counters))

	for name, value := range snapshot.Gauges {
		metricValue := value
		metrics = append(metrics, model.Metrics{
			ID:    name,
			MType: model.Gauge,
			Value: &metricValue,
		})
	}

	for name, delta := range snapshot.Counters {
		metricDelta := delta
		metrics = append(metrics, model.Metrics{
			ID:    name,
			MType: model.Counter,
			Delta: &metricDelta,
		})
	}

	err = json.NewEncoder(file).Encode(metrics)
	if err != nil {
		return err
	}

	return nil
}

func (repo *FileMetricsRepository) Restore(ctx context.Context, storage MetricsWriter) error {
	file, err := os.Open(repo.FilePath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	defer file.Close()

	var metrics []model.Metrics
	if err := json.NewDecoder(file).Decode(&metrics); errors.Is(err, io.EOF) {
		return nil
	} else if err != nil {
		return err
	}

	for _, metric := range metrics {
		switch metric.MType {
		case model.Gauge:
			if metric.Value == nil {
				return fmt.Errorf("gauge %q has no value", metric.ID)
			}
			if err := storage.SetGauge(ctx, metric.ID, *metric.Value); err != nil {
				return err
			}
		case model.Counter:
			if metric.Delta == nil {
				return fmt.Errorf("counter %q has no delta", metric.ID)
			}
			if _, err := storage.AddCounter(ctx, metric.ID, *metric.Delta); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unknown metric type %q", metric.MType)
		}
	}

	return nil
}
