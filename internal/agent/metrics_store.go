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

	store.current.gauges = gauges
	for metricName, metricValue := range counters {
		store.current.counters[metricName] += metricValue
	}
}

func (store *metricsStore) snapshot() metricsSnapshot {
	store.mu.Lock()
	defer store.mu.Unlock()

	gauges := make(map[string]float64, len(store.current.gauges))
	for metricName, metricValue := range store.current.gauges {
		gauges[metricName] = metricValue
	}

	counters := make(map[string]int64, len(store.current.counters))
	for metricName, metricValue := range store.current.counters {
		counters[metricName] = metricValue
	}

	return metricsSnapshot{
		gauges:   gauges,
		counters: counters,
	}
}

func (store *metricsStore) acknowledgeCounter(counterName string, sentValue int64) {
	store.mu.Lock()
	defer store.mu.Unlock()

	currentValue := store.current.counters[counterName]
	if currentValue <= sentValue {
		store.current.counters[counterName] = 0
		return
	}

	store.current.counters[counterName] = currentValue - sentValue
}
