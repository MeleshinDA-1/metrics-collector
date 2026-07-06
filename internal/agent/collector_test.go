package agent

import "testing"

func TestGatherCounter(t *testing.T) {
	counters := gatherCounter()

	if counters["PollCount"] != 1 {
		t.Fatalf("PollCount = %d, want %d", counters["PollCount"], 1)
	}
}

func TestGatherGauge(t *testing.T) {
	gauges := gatherGauge()
	requiredMetrics := []string{
		"Alloc",
		"BuckHashSys",
		"Frees",
		"GCCPUFraction",
		"GCSys",
		"HeapAlloc",
		"HeapIdle",
		"HeapInuse",
		"HeapObjects",
		"HeapReleased",
		"HeapSys",
		"LastGC",
		"Lookups",
		"MCacheInuse",
		"MCacheSys",
		"MSpanInuse",
		"MSpanSys",
		"Mallocs",
		"NextGC",
		"NumForcedGC",
		"NumGC",
		"OtherSys",
		"PauseTotalNs",
		"StackInuse",
		"StackSys",
		"Sys",
		"TotalAlloc",
		"RandomValue",
	}

	for _, metricName := range requiredMetrics {
		if _, ok := gauges[metricName]; !ok {
			t.Fatalf("metric %q was not gathered", metricName)
		}
	}
}

func TestGatherMetricsInternal(t *testing.T) {
	storage := newMetricStorage()

	gatherMetricsInternal(storage)
	metrics := storage.snapshot()

	if metrics.counters["PollCount"] != 1 {
		t.Fatalf("PollCount = %d, want %d", metrics.counters["PollCount"], 1)
	}

	if _, ok := metrics.gauges["Alloc"]; !ok {
		t.Fatal("metric \"Alloc\" was not gathered")
	}
}
