package repository

import (
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

func (repo *FileMetricsRepository) Flush(storage MetricsSnapshotProvider) error {
	repo.mutex.Lock()
	defer repo.mutex.Unlock()

	file, err := os.Create(repo.FilePath)
	if err != nil {
		return err
	}
	defer file.Close()

	snapshot := storage.Snapshot()
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

func (repo *FileMetricsRepository) Restore(memStorage *MemStorage) error {
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
			memStorage.SetGauge(metric.ID, *metric.Value)
		case model.Counter:
			if metric.Delta == nil {
				return fmt.Errorf("counter %q has no delta", metric.ID)
			}
			memStorage.AddCounter(metric.ID, *metric.Delta)
		default:
			return fmt.Errorf("unknown metric type %q", metric.MType)
		}
	}

	return nil
}
