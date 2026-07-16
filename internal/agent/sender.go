package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	models "github.com/MeleshinDA-1/metrics-collector/internal/model"
)

const (
	requestTimeout = 3 * time.Second
)

type metricSender struct {
	client         *http.Client
	urlBase        string
	reportInterval time.Duration
}

func newMetricSender(serverAddress string, reportInterval time.Duration) *metricSender {
	return &metricSender{
		client: &http.Client{
			Timeout: requestTimeout,
		},
		urlBase:        normalizeServerAddress(serverAddress) + "/update",
		reportInterval: reportInterval,
	}
}

func (metricSender *metricSender) sendMetrics(metricStorage *metricStorage) {
	for {
		metricSender.sendMetricsInternal(metricStorage)
		time.Sleep(metricSender.reportInterval)
	}
}

func (metricSender *metricSender) sendMetricsInternal(metricStorage *metricStorage) {
	metrics := metricStorage.snapshot()
	var requestMetric models.Metrics
	requestMetric.MType = "gauge"
	for metricName, metricValue := range metrics.gauges {
		requestMetric.ID = metricName
		requestMetric.Value = &metricValue
		err := metricSender.sendMetric(requestMetric)
		if err != nil {
			slog.Error("failed to send gauge", "metric", metricName, "err", err)
		}
	}

	metricSender.sendCounters(metricStorage, metrics)
}

func (metricSender *metricSender) sendCounters(metricStorage *metricStorage, metrics allMetrics) {
	var requestMetric models.Metrics
	requestMetric.MType = "counter"
	for metricName, metricValue := range metrics.counters {
		requestMetric.ID = metricName
		requestMetric.Delta = &metricValue
		err := metricSender.sendMetric(requestMetric)
		if err != nil {
			slog.Error("failed to send counter", "metric", metricName, "err", err)
			continue
		}

		metricStorage.markCounterSent(metricName, metricValue)
	}
}

func (metricStorage *metricStorage) markCounterSent(counterName string, valueSent int64) {
	metricStorage.mutex.Lock()
	defer metricStorage.mutex.Unlock()

	currentValue := metricStorage.allMetrics.counters[counterName]
	if currentValue <= valueSent {
		metricStorage.allMetrics.counters[counterName] = 0
		return
	}

	metricStorage.allMetrics.counters[counterName] = currentValue - valueSent
}

func (metricSender *metricSender) sendMetric(requestMetric models.Metrics) error {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(requestMetric); err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, url, &buf)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	resp, err := metricSender.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	return nil
}

func normalizeServerAddress(serverAddress string) string {
	serverAddress = strings.TrimRight(serverAddress, "/")
	if strings.HasPrefix(serverAddress, "http://") || strings.HasPrefix(serverAddress, "https://") {
		return serverAddress
	}

	return "http://" + serverAddress
}
