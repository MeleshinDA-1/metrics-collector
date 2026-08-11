package agent

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/MeleshinDA-1/metrics-collector/internal/model"
)

type metricsSender struct {
	httpClient     *http.Client
	updatesURL     string
	reportInterval time.Duration
}

func newMetricsSender(serverAddress string, reportInterval time.Duration) *metricsSender {
	return &metricsSender{
		httpClient: &http.Client{
			Timeout: requestTimeout,
		},
		updatesURL:     normalizeServerAddress(serverAddress) + "/updates/",
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

	metrics := make([]model.Metrics, 0, len(snapshot.gauges)+len(snapshot.counters))
	for metricName, metricValue := range snapshot.gauges {
		value := metricValue
		metrics = append(metrics, model.Metrics{
			ID:    metricName,
			MType: model.Gauge,
			Value: &value,
		})
	}
	for metricName, metricValue := range snapshot.counters {
		delta := metricValue
		metrics = append(metrics, model.Metrics{
			ID:    metricName,
			MType: model.Counter,
			Delta: &delta,
		})
	}

	if len(metrics) == 0 {
		return
	}

	if err := sender.sendBatch(metrics); err != nil {
		slog.Error("failed to send metrics batch", "count", len(metrics), "err", err)
		return
	}

	for metricName, metricValue := range snapshot.counters {
		store.acknowledgeCounter(metricName, metricValue)
	}
}

func (sender *metricsSender) sendBatch(metrics []model.Metrics) error {
	body, err := buildRequestBody(
		withGzipCompression(encodeJSON(metrics)),
	)
	if err != nil {
		return err
	}

	return sender.post(sender.updatesURL, body)
}
