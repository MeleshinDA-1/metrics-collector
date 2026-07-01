package agent

const defaultServerAddress = "http://localhost:8080"

func Run() {
	metricStorage := newMetricStorage()
	gatherMetricsInternal(metricStorage)

	metricSender := newMetricSender(defaultServerAddress)

	go gatherMetrics(metricStorage)
	go metricSender.sendMetrics(metricStorage)
	select {}
}
