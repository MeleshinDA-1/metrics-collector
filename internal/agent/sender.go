package agent

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	reportInterval = 10 * time.Second
	requestTimeout = 3 * time.Second
)

type metricSender struct {
	client  *http.Client
	urlBase string
}

func newMetricSender(serverAddress string) *metricSender {
	return &metricSender{
		client: &http.Client{
			Timeout: requestTimeout,
		},
		urlBase: strings.TrimRight(serverAddress, "/") + "/update/%s/%s/%s",
	}
}

func (metricSender *metricSender) sendMetrics(metricStorage *metricStorage) {
	for {
		metricSender.sendMetricsInternal(metricStorage)
		time.Sleep(reportInterval)
	}
}

func (metricSender *metricSender) sendMetricsInternal(metricStorage *metricStorage) {
	metrics := metricStorage.snapshot()

	for metricName, metricValue := range metrics.gauges {
		err := metricSender.sendMetric("gauge", metricName, strconv.FormatFloat(metricValue, 'f', -1, 64))
		if err != nil {
			fmt.Printf("failed to send gauge %s: %v\n", metricName, err)
		}
	}

	for metricName, metricValue := range metrics.counters {
		err := metricSender.sendMetric("counter", metricName, strconv.FormatInt(metricValue, 10))
		if err != nil {
			fmt.Printf("failed to send counter %s: %v\n", metricName, err)
		}
	}
}

func (metricSender *metricSender) sendMetric(metricType string, metricName string, metricValue string) error {
	url := fmt.Sprintf(metricSender.urlBase, metricType, metricName, metricValue)
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "text/plain")
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
