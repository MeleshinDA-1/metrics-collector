package agent

import "sync"

type metricStorage struct {
	mutex      sync.Mutex
	allMetrics allMetrics
}

type allMetrics struct {
	gauges   map[string]float64
	counters map[string]int64
}

func newMetricStorage() *metricStorage {
	return &metricStorage{
		allMetrics: allMetrics{
			gauges:   make(map[string]float64),
			counters: make(map[string]int64),
		},
	}
}

func (metricStorage *metricStorage) updateMetrics(gauges map[string]float64, counters map[string]int64) {
	metricStorage.mutex.Lock()
	defer metricStorage.mutex.Unlock()

	metricStorage.allMetrics.gauges = gauges
	for metricName, metricValue := range counters {
		metricStorage.allMetrics.counters[metricName] += metricValue
	}
}

func (metricStorage *metricStorage) snapshot() allMetrics {
	metricStorage.mutex.Lock()
	defer metricStorage.mutex.Unlock()

	gauges := make(map[string]float64, len(metricStorage.allMetrics.gauges))
	for metricName, metricValue := range metricStorage.allMetrics.gauges {
		gauges[metricName] = metricValue
	}

	counters := make(map[string]int64, len(metricStorage.allMetrics.counters))
	for metricName, metricValue := range metricStorage.allMetrics.counters {
		counters[metricName] = metricValue
	}

	return allMetrics{
		gauges:   gauges,
		counters: counters,
	}
}
