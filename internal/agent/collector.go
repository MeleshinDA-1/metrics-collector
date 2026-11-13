package agent

import (
	"context"
	"log/slog"
	"math/rand"
	"runtime"
	"time"
)

type metricsCollector func(context.Context) (metricsSnapshot, error)

func runRuntimeCollector(ctx context.Context, pollInterval time.Duration) <-chan metricsSnapshot {
	return runCollector(ctx, pollInterval, collectRuntimeMetrics)
}

func runCollector(
	ctx context.Context,
	pollInterval time.Duration,
	collect metricsCollector,
) <-chan metricsSnapshot {
	collected := make(chan metricsSnapshot)

	go func() {
		defer close(collected)

		ticker := time.NewTicker(pollInterval)
		defer ticker.Stop()

		for {
			snapshot, err := collect(ctx)
			if err != nil {
				slog.Error("unable to collect metrics", "error", err)
			} else {
				select {
				case collected <- snapshot:
				case <-ctx.Done():
					return
				}
			}

			select {
			case <-ticker.C:
			case <-ctx.Done():
				return
			}
		}
	}()

	return collected
}

func collectRuntimeMetrics(context.Context) (metricsSnapshot, error) {
	return metricsSnapshot{
		gauges:   collectGauges(),
		counters: collectCounters(),
	}, nil
}

func collectCounters() map[string]int64 {
	return map[string]int64{
		"PollCount": 1,
	}
}

func collectGauges() map[string]float64 {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return map[string]float64{
		"Alloc":         float64(memStats.Alloc),
		"BuckHashSys":   float64(memStats.BuckHashSys),
		"Frees":         float64(memStats.Frees),
		"GCCPUFraction": memStats.GCCPUFraction,
		"GCSys":         float64(memStats.GCSys),
		"HeapAlloc":     float64(memStats.HeapAlloc),
		"HeapIdle":      float64(memStats.HeapIdle),
		"HeapInuse":     float64(memStats.HeapInuse),
		"HeapObjects":   float64(memStats.HeapObjects),
		"HeapReleased":  float64(memStats.HeapReleased),
		"HeapSys":       float64(memStats.HeapSys),
		"LastGC":        float64(memStats.LastGC),
		"Lookups":       float64(memStats.Lookups),
		"MCacheInuse":   float64(memStats.MCacheInuse),
		"MCacheSys":     float64(memStats.MCacheSys),
		"MSpanInuse":    float64(memStats.MSpanInuse),
		"MSpanSys":      float64(memStats.MSpanSys),
		"Mallocs":       float64(memStats.Mallocs),
		"NextGC":        float64(memStats.NextGC),
		"NumForcedGC":   float64(memStats.NumForcedGC),
		"NumGC":         float64(memStats.NumGC),
		"OtherSys":      float64(memStats.OtherSys),
		"PauseTotalNs":  float64(memStats.PauseTotalNs),
		"StackInuse":    float64(memStats.StackInuse),
		"StackSys":      float64(memStats.StackSys),
		"Sys":           float64(memStats.Sys),
		"TotalAlloc":    float64(memStats.TotalAlloc),
		"RandomValue":   float64(rand.Intn(100)),
	}
}
