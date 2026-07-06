package repository

import "sync"

type MetricsSnapshot struct {
	Gauges   map[string]float64
	Counters map[string]int64
}

type MemStorage struct {
	mutex    sync.RWMutex
	gauges   map[string]float64
	counters map[string]int64
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
	}
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
	storage.mutex.RLock()
	defer storage.mutex.RUnlock()

	value, ok := storage.gauges[name]
	return value, ok
}

func (storage *MemStorage) GetCounter(name string) (int64, bool) {
	storage.mutex.RLock()
	defer storage.mutex.RUnlock()

	value, ok := storage.counters[name]
	return value, ok
}

func (storage *MemStorage) Snapshot() MetricsSnapshot {
	storage.mutex.RLock()
	defer storage.mutex.RUnlock()

	gauges := make(map[string]float64, len(storage.gauges))
	for name, value := range storage.gauges {
		gauges[name] = value
	}

	counters := make(map[string]int64, len(storage.counters))
	for name, value := range storage.counters {
		counters[name] = value
	}

	return MetricsSnapshot{
		Gauges:   gauges,
		Counters: counters,
	}
}
