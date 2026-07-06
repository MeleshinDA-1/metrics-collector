package agent

import "time"

func Run(serverAddress string, pollInterval time.Duration, reportInterval time.Duration) {
	metricStorage := newMetricStorage()
	gatherMetricsInternal(metricStorage)

	metricSender := newMetricSender(serverAddress, reportInterval)

	go gatherMetrics(metricStorage, pollInterval)
	go metricSender.sendMetrics(metricStorage)
	select {}
}
