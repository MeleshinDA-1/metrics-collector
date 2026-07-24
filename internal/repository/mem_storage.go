package repository

import (
	"sync"

	models "github.com/MeleshinDA-1/metrics-collector/internal/model"
)

type MemStorage struct {
	mutex    sync.Mutex
	gauges   map[string]float64
	counters map[string]int64

	MetricsRepository
}

type MetricsRepository interface {
	Flush(*MemStorage) error
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
	}
}

func (storage *MemStorage) Save() error {
	if storage.MetricsRepository == nil {
		return nil
	}

	return storage.MetricsRepository.Flush(storage)
}

func (storage *MemStorage) SetGauge(name string, value float64) {
	storage.mutex.Lock()
	defer storage.mutex.Unlock()

	storage.gauges[name] = value
}

func (storage *MemStorage) AddCounter(name string, delta int64) {
	storage.mutex.Lock()
	defer storage.mutex.Unlock()

	storage.counters[name] += delta
}

func (storage *MemStorage) GetGauge(name string) (float64, bool) {
	storage.mutex.Lock()
	defer storage.mutex.Unlock()

	value, ok := storage.gauges[name]
	return value, ok
}

func (storage *MemStorage) GetCounter(name string) (int64, bool) {
	storage.mutex.Lock()
	defer storage.mutex.Unlock()

	value, ok := storage.counters[name]
	return value, ok
}

func (storage *MemStorage) Snapshot() models.MetricsSnapshot {
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

	return models.MetricsSnapshot{
		Gauges:   gauges,
		Counters: counters,
	}
}
