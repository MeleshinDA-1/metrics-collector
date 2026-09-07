package agent

import (
	"context"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/MeleshinDA-1/metrics-collector/internal/model"
	"github.com/MeleshinDA-1/metrics-collector/internal/retry"
)

type metricsSender struct {
	httpClient     *http.Client
	updatesURL     string
	reportInterval time.Duration
	rateLimit      int
	signingKey     string
	retryPolicy    retry.Policy
}

func newMetricsSender(
	serverAddress string,
	reportInterval time.Duration,
	rateLimit int,
	signingKey string,
	retryPolicy retry.Policy,
) *metricsSender {
	if rateLimit < 1 {
		rateLimit = 1
	}

	return &metricsSender{
		httpClient: &http.Client{
			Timeout: requestTimeout,
		},
		updatesURL:     normalizeServerAddress(serverAddress) + "/updates/",
		reportInterval: reportInterval,
		rateLimit:      rateLimit,
		signingKey:     signingKey,
		retryPolicy:    retryPolicy,
	}
}

func (sender *metricsSender) run(ctx context.Context, store *metricsStore) {
	reports := make(chan struct{}, sender.rateLimit)

	var workers sync.WaitGroup
	for worker := 1; worker <= sender.rateLimit; worker++ {
		workers.Add(1)
		go func() {
			defer workers.Done()

			sender.runWorker(ctx, worker, reports, store)
		}()
	}

	sender.scheduleReports(ctx, reports)
	workers.Wait()
}

func (sender *metricsSender) runWorker(
	ctx context.Context,
	worker int,
	reports <-chan struct{},
	store *metricsStore,
) {
	for {
		select {
		case <-ctx.Done():
			return
		case _, ok := <-reports:
			if !ok {
				return
			}

			if err := sender.report(ctx, store); err != nil {
				slog.Error("failed to report metrics", "worker", worker, "error", err)
			}
		}
	}
}

func (sender *metricsSender) scheduleReports(ctx context.Context, reports chan<- struct{}) {
	defer close(reports)

	ticker := time.NewTicker(sender.reportInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			select {
			case reports <- struct{}{}:
			case <-ctx.Done():
				return
			}
		}
	}
}

func (sender *metricsSender) report(ctx context.Context, store *metricsStore) error {
	snapshot := store.drain()

	metrics := buildBatch(snapshot)
	if len(metrics) == 0 {
		return nil
	}

	if err := sender.sendBatch(ctx, metrics); err != nil {
		store.restoreCounters(snapshot.counters)
		return err
	}

	return nil
}

func buildBatch(snapshot metricsSnapshot) []model.Metrics {
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

	return metrics
}

func (sender *metricsSender) sendBatch(ctx context.Context, metrics []model.Metrics) error {
	body, err := buildRequestBody(
		withGzipCompression(encodeJSON(metrics)),
	)
	if err != nil {
		return err
	}

	return sender.post(ctx, sender.updatesURL, body)
}
