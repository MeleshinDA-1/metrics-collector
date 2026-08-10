package agent

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/MeleshinDA-1/metrics-collector/internal/model"
)

type metricsSender struct {
	httpClient     *http.Client
	updateURL      string
	reportInterval time.Duration
}

func newMetricsSender(serverAddress string, reportInterval time.Duration) *metricsSender {
	return &metricsSender{
		httpClient: &http.Client{
			Timeout: requestTimeout,
		},
		updateURL:      normalizeServerAddress(serverAddress) + "/update",
		reportInterval: reportInterval,
	}
}

func (sender *metricsSender) run(store *metricsStore) {
	for {
		sender.sendMetrics(store)
		time.Sleep(sender.reportInterval)
	}
}

func (sender *metricsSender) sendMetrics(store *metricsStore) {
	snapshot := store.snapshot()
	metric := model.Metrics{MType: model.Gauge}
	for metricName, metricValue := range snapshot.gauges {
		metric.ID = metricName
		metric.Value = &metricValue
		err := sender.sendMetric(metric)
		if err != nil {
			slog.Error("failed to send gauge", "metric", metricName, "err", err)
		}
	}

	sender.sendCounters(store, snapshot.counters)
}

func (sender *metricsSender) sendCounters(store *metricsStore, counters map[string]int64) {
	metric := model.Metrics{MType: model.Counter}
	for metricName, metricValue := range counters {
		metric.ID = metricName
		metric.Delta = &metricValue
		err := sender.sendMetric(metric)
		if err != nil {
			slog.Error("failed to send counter", "metric", metricName, "err", err)
			continue
		}

		store.acknowledgeCounter(metricName, metricValue)
	}
}

func (sender *metricsSender) sendMetric(metric model.Metrics) error {
	body, err := buildRequestBody(
		withGzipCompression(encodeJSON(metric)),
	)
	if err != nil {
		return err
	}

	return sender.postUpdate(body)
}
