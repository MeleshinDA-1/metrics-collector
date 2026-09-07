package agent

import (
	"testing"
	"time"
)

func TestFanInMergesEveryCollector(t *testing.T) {
	first := make(chan metricsSnapshot, 1)
	second := make(chan metricsSnapshot, 1)

	first <- metricsSnapshot{gauges: map[string]float64{"Alloc": 1}}
	second <- metricsSnapshot{gauges: map[string]float64{"TotalMemory": 2048}}
	close(first)
	close(second)

	merged := fanIn(first, second)

	gauges := make(map[string]float64)
	for snapshot := range merged {
		for metricName, metricValue := range snapshot.gauges {
			gauges[metricName] = metricValue
		}
	}

	if gauges["Alloc"] != 1 || gauges["TotalMemory"] != 2048 {
		t.Fatalf("merged gauges = %v, want both collectors", gauges)
	}
}

func TestFanInClosesWhenEveryCollectorStops(t *testing.T) {
	first := make(chan metricsSnapshot)
	second := make(chan metricsSnapshot)

	merged := fanIn(first, second)

	close(first)
	select {
	case _, ok := <-merged:
		if !ok {
			t.Fatal("merged channel closed while a collector was still running")
		}
	case <-time.After(100 * time.Millisecond):
	}

	close(second)
	select {
	case _, ok := <-merged:
		if ok {
			t.Fatal("merged channel produced a value after every collector stopped")
		}
	case <-time.After(time.Second):
		t.Fatal("merged channel was not closed after every collector stopped")
	}
}

func TestStoreCollectedMetricsMergesIntoTheStore(t *testing.T) {
	collected := make(chan metricsSnapshot, 2)
	collected <- metricsSnapshot{
		gauges:   map[string]float64{"Alloc": 1},
		counters: map[string]int64{"PollCount": 1},
	}
	collected <- metricsSnapshot{
		gauges:   map[string]float64{"TotalMemory": 2048},
		counters: map[string]int64{"PollCount": 1},
	}
	close(collected)

	store := newMetricsStore()
	storeCollectedMetrics(store, collected)

	assertMetricsSnapshot(t, storeState(store), metricsSnapshot{
		gauges: map[string]float64{
			"Alloc":       1,
			"TotalMemory": 2048,
		},
		counters: map[string]int64{
			"PollCount": 2,
		},
	})
}
