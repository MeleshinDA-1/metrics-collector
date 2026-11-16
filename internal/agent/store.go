package agent

import "sync"

type metricsStore struct {
	mu      sync.Mutex
	current metricsSnapshot
}

type metricsSnapshot struct {
	gauges   map[string]float64
	counters map[string]int64
}

func newMetricsStore() *metricsStore {
	return &metricsStore{
		current: metricsSnapshot{
			gauges:   make(map[string]float64),
			counters: make(map[string]int64),
		},
	}
}

func (store *metricsStore) update(gauges map[string]float64, counters map[string]int64) {
	store.mu.Lock()
	defer store.mu.Unlock()

	for metricName, metricValue := range gauges {
		store.current.gauges[metricName] = metricValue
	}
	for metricName, metricValue := range counters {
		store.current.counters[metricName] += metricValue
	}
}

func (store *metricsStore) drain() metricsSnapshot {
	store.mu.Lock()
	defer store.mu.Unlock()

	counters := store.current.counters
	store.current.counters = make(map[string]int64, len(counters))

	return metricsSnapshot{
		gauges:   copyGauges(store.current.gauges),
		counters: counters,
	}
}

func (store *metricsStore) restoreCounters(counters map[string]int64) {
	store.mu.Lock()
	defer store.mu.Unlock()

	for metricName, metricValue := range counters {
		store.current.counters[metricName] += metricValue
	}
}

func copyGauges(gauges map[string]float64) map[string]float64 {
	copied := make(map[string]float64, len(gauges))
	for metricName, metricValue := range gauges {
		copied[metricName] = metricValue
	}

	return copied
}
