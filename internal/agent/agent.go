package agent

import (
	"time"

	"github.com/MeleshinDA-1/metrics-collector/internal/retry"
)

func Run(serverAddress string, pollInterval time.Duration, reportInterval time.Duration) {
	store := newMetricsStore()
	collectMetrics(store)

	sender := newMetricsSender(serverAddress, reportInterval, retry.DefaultPolicy())

	go runCollector(store, pollInterval)
	sender.run(store)
}
