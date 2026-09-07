package agent

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
)

func runSystemCollector(ctx context.Context, pollInterval time.Duration) <-chan metricsSnapshot {
	return runCollector(ctx, pollInterval, collectSystemMetrics)
}

func collectSystemMetrics(ctx context.Context) (metricsSnapshot, error) {
	virtualMemory, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		return metricsSnapshot{}, fmt.Errorf("read virtual memory stats: %w", err)
	}

	utilization, err := cpu.PercentWithContext(ctx, 0, true)
	if err != nil {
		return metricsSnapshot{}, fmt.Errorf("read cpu utilization: %w", err)
	}

	gauges := map[string]float64{
		"TotalMemory": float64(virtualMemory.Total),
		"FreeMemory":  float64(virtualMemory.Free),
	}
	for index, percent := range utilization {
		gauges["CPUutilization"+strconv.Itoa(index+1)] = percent
	}

	return metricsSnapshot{gauges: gauges}, nil
}
