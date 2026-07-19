package agent

import "time"

func Run(serverAddress string, pollInterval time.Duration, reportInterval time.Duration) {
	store := newMetricsStore()
	collectMetrics(store)

	sender := newMetricsSender(serverAddress, reportInterval)

	go runCollector(store, pollInterval)
	sender.run(store)
}
