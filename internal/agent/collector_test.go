package agent

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"
)

var errCollectFailed = errors.New("collect failed")

func TestCollectCounters(t *testing.T) {
	counters := collectCounters()

	if counters["PollCount"] != 1 {
		t.Fatalf("PollCount = %d, want %d", counters["PollCount"], 1)
	}
}

func TestCollectGauges(t *testing.T) {
	gauges := collectGauges()
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

func TestCollectRuntimeMetrics(t *testing.T) {
	snapshot, err := collectRuntimeMetrics(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if snapshot.counters["PollCount"] != 1 {
		t.Fatalf("PollCount = %d, want %d", snapshot.counters["PollCount"], 1)
	}
	if _, ok := snapshot.gauges["Alloc"]; !ok {
		t.Fatal("metric \"Alloc\" was not gathered")
	}
}

func TestRunCollectorEmitsFirstSnapshotWithoutWaiting(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	collected := runCollector(ctx, time.Hour, func(context.Context) (metricsSnapshot, error) {
		return metricsSnapshot{gauges: map[string]float64{"Alloc": 42}}, nil
	})

	select {
	case snapshot := <-collected:
		if snapshot.gauges["Alloc"] != 42 {
			t.Fatalf("Alloc = %v, want %v", snapshot.gauges["Alloc"], 42.0)
		}
	case <-time.After(time.Second):
		t.Fatal("collector did not emit a snapshot")
	}
}

func TestRunCollectorPollsRepeatedlyAndStopsWithContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var polls int
	collected := runCollector(ctx, time.Millisecond, func(context.Context) (metricsSnapshot, error) {
		polls++
		return metricsSnapshot{gauges: map[string]float64{"Poll": float64(polls)}}, nil
	})

	for want := 1; want <= 3; want++ {
		select {
		case snapshot := <-collected:
			if snapshot.gauges["Poll"] != float64(want) {
				t.Fatalf("Poll = %v, want %v", snapshot.gauges["Poll"], want)
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("collector emitted only %d snapshots", want-1)
		}
	}

	cancel()

	deadline := time.After(2 * time.Second)
	for {
		select {
		case _, ok := <-collected:
			if !ok {
				return
			}
		case <-deadline:
			t.Fatal("collector did not stop after the context was cancelled")
		}
	}
}

func TestRunCollectorKeepsPollingAfterAnError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var polls int
	collected := runCollector(ctx, time.Millisecond, func(context.Context) (metricsSnapshot, error) {
		polls++
		if polls == 1 {
			return metricsSnapshot{}, errCollectFailed
		}

		return metricsSnapshot{gauges: map[string]float64{"Poll": float64(polls)}}, nil
	})

	select {
	case snapshot := <-collected:
		if snapshot.gauges["Poll"] == 0 {
			t.Fatalf("collector emitted the failed snapshot: %v", snapshot.gauges)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("collector stopped after a failed poll")
	}
}

func TestCollectSystemMetrics(t *testing.T) {
	snapshot, err := collectSystemMetrics(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, metricName := range []string{"TotalMemory", "FreeMemory", "CPUutilization1"} {
		if _, ok := snapshot.gauges[metricName]; !ok {
			t.Fatalf("metric %q was not gathered", metricName)
		}
	}
	if snapshot.gauges["TotalMemory"] <= 0 {
		t.Fatalf("TotalMemory = %v, want a positive value", snapshot.gauges["TotalMemory"])
	}

	utilizations := 0
	for metricName := range snapshot.gauges {
		if strings.HasPrefix(metricName, "CPUutilization") {
			utilizations++
		}
	}
	for index := 1; index <= utilizations; index++ {
		metricName := "CPUutilization" + strconv.Itoa(index)
		if _, ok := snapshot.gauges[metricName]; !ok {
			t.Fatalf("metric %q was not gathered, but %d CPUs were reported", metricName, utilizations)
		}
	}
}
