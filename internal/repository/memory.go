package repository

import (
	"context"
	"sync"

	"github.com/MeleshinDA-1/metrics-collector/internal/model"
)

type MemStorage struct {
	mutex    sync.Mutex
	gauges   map[string]float64
	counters map[string]int64
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
	}
}

func (storage *MemStorage) SetGauge(_ context.Context, name string, value float64) error {
	storage.mutex.Lock()
	defer storage.mutex.Unlock()

	storage.gauges[name] = value

	return nil
}

func (storage *MemStorage) AddCounter(_ context.Context, name string, delta int64) (int64, error) {
	storage.mutex.Lock()
	defer storage.mutex.Unlock()

	storage.counters[name] += delta

	return storage.counters[name], nil
}

func (storage *MemStorage) UpdateBatch(_ context.Context, metrics []model.Metrics) error {
	storage.mutex.Lock()
	defer storage.mutex.Unlock()

	for _, metric := range metrics {
		switch metric.MType {
		case model.Gauge:
			storage.gauges[metric.ID] = *metric.Value
		case model.Counter:
			storage.counters[metric.ID] += *metric.Delta
		}
	}

	return nil
}

func (storage *MemStorage) GetGauge(_ context.Context, name string) (float64, bool, error) {
	storage.mutex.Lock()
	defer storage.mutex.Unlock()

	value, ok := storage.gauges[name]
	return value, ok, nil
}

func (storage *MemStorage) GetCounter(_ context.Context, name string) (int64, bool, error) {
	storage.mutex.Lock()
	defer storage.mutex.Unlock()

	value, ok := storage.counters[name]
	return value, ok, nil
}

func (storage *MemStorage) Snapshot(_ context.Context) (model.MetricsSnapshot, error) {
	storage.mutex.Lock()
	defer storage.mutex.Unlock()

	gauges := make(map[string]float64, len(storage.gauges))
	for name, value := range storage.gauges {
		gauges[name] = value
	}

	counters := make(map[string]int64, len(storage.counters))
	for name, value := range storage.counters {
		counters[name] = value
	}

	return model.MetricsSnapshot{
		Gauges:   gauges,
		Counters: counters,
	}, nil
}
