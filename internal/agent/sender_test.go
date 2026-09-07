package agent

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/MeleshinDA-1/metrics-collector/internal/model"
	"github.com/MeleshinDA-1/metrics-collector/internal/retry"
)

func TestMetricsSenderSendBatchReturnsErrorOnUnexpectedStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	sender := newMetricsSender(server.URL, time.Second, 1, "", retry.DefaultPolicy())

	err := sender.sendBatch(context.Background(), []model.Metrics{{ID: "Alloc", MType: model.Gauge}})
	if err == nil {
		t.Fatal("sendBatch returned nil error, want non-nil error")
	}
}

func TestMetricsSenderReport(t *testing.T) {
	var (
		mutex           sync.Mutex
		receivedMetrics []model.Metrics
		requestCount    int
		requestMethod   string
		requestPath     string
		mediaType       string
		contentEncoding string
	)

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		metrics, err := decodeGzipMetricsBatch(request.Body)
		if err != nil {
			t.Errorf("read compressed batch: %v", err)
			response.WriteHeader(http.StatusBadRequest)
			return
		}

		mutex.Lock()
		defer mutex.Unlock()

		requestCount++
		requestMethod = request.Method
		requestPath = request.URL.Path
		mediaType = request.Header.Get("Content-Type")
		contentEncoding = request.Header.Get("Content-Encoding")
		receivedMetrics = append(receivedMetrics, metrics...)

		response.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	store := newMetricsStore()
	store.update(
		map[string]float64{
			"Alloc": 42.5,
		},
		map[string]int64{
			"PollCount": 2,
		},
	)

	sender := newMetricsSender(server.URL, time.Second, 1, "", retry.DefaultPolicy())
	if err := sender.report(context.Background(), store); err != nil {
		t.Fatalf("report returned error: %v", err)
	}

	mutex.Lock()
	defer mutex.Unlock()

	gaugeValue := 42.5
	counterDelta := int64(2)
	wantMetrics := map[string]model.Metrics{
		"Alloc": {
			ID:    "Alloc",
			MType: model.Gauge,
			Value: &gaugeValue,
		},
		"PollCount": {
			ID:    "PollCount",
			MType: model.Counter,
			Delta: &counterDelta,
		},
	}
	gotMetrics := make(map[string]model.Metrics, len(receivedMetrics))
	for _, metric := range receivedMetrics {
		gotMetrics[metric.ID] = metric
	}

	if !reflect.DeepEqual(gotMetrics, wantMetrics) {
		t.Fatalf("metrics = %v, want %v", gotMetrics, wantMetrics)
	}
	if requestCount != 1 {
		t.Fatalf("requests count = %d, want 1", requestCount)
	}
	if requestMethod != http.MethodPost {
		t.Fatalf("method = %q, want %q", requestMethod, http.MethodPost)
	}
	if requestPath != "/updates/" {
		t.Fatalf("path = %q, want %q", requestPath, "/updates/")
	}
	if mediaType != "application/json" {
		t.Fatalf("content type = %q, want %q", mediaType, "application/json")
	}
	if contentEncoding != "gzip" {
		t.Fatalf("content encoding = %q, want %q", contentEncoding, "gzip")
	}
}

func TestMetricsSenderSkipsEmptyBatch(t *testing.T) {
	var requestCount atomic.Int64

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requestCount.Add(1)
		response.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := newMetricsSender(server.URL, time.Second, 1, "", retry.DefaultPolicy())
	if err := sender.report(context.Background(), newMetricsStore()); err != nil {
		t.Fatalf("report returned error: %v", err)
	}

	if got := requestCount.Load(); got != 0 {
		t.Fatalf("requests count = %d, want 0", got)
	}
}

func TestMetricsSenderReportSendsCounterDeltaOnce(t *testing.T) {
	var (
		mutex    sync.Mutex
		batches  [][]model.Metrics
		received = func(metrics []model.Metrics) {
			mutex.Lock()
			defer mutex.Unlock()

			batches = append(batches, metrics)
		}
	)

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		metrics, err := decodeGzipMetricsBatch(request.Body)
		if err != nil {
			t.Errorf("read compressed batch: %v", err)
			response.WriteHeader(http.StatusBadRequest)
			return
		}

		received(metrics)
		response.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	store := newMetricsStore()
	store.update(map[string]float64{"Alloc": 1}, map[string]int64{"PollCount": 3})

	sender := newMetricsSender(server.URL, time.Second, 1, "", retry.DefaultPolicy())
	if err := sender.report(context.Background(), store); err != nil {
		t.Fatalf("first report returned error: %v", err)
	}
	if err := sender.report(context.Background(), store); err != nil {
		t.Fatalf("second report returned error: %v", err)
	}

	mutex.Lock()
	defer mutex.Unlock()

	if len(batches) != 2 {
		t.Fatalf("requests count = %d, want 2", len(batches))
	}
	if counterDelta(batches[0], "PollCount") != 3 {
		t.Fatalf("first report delta = %d, want 3", counterDelta(batches[0], "PollCount"))
	}
	if counterDelta(batches[1], "PollCount") != 0 {
		t.Fatalf("second report resent the delta: %d", counterDelta(batches[1], "PollCount"))
	}
}

func counterDelta(metrics []model.Metrics, metricName string) int64 {
	for _, metric := range metrics {
		if metric.ID == metricName && metric.Delta != nil {
			return *metric.Delta
		}
	}

	return 0
}

func decodeGzipMetricsBatch(body io.Reader) ([]model.Metrics, error) {
	reader, err := gzip.NewReader(body)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	var metrics []model.Metrics
	if err := json.NewDecoder(reader).Decode(&metrics); err != nil {
		return nil, err
	}

	return metrics, nil
}

func TestMetricsSenderKeepsCounterAfterFailedReport(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	store := newMetricsStore()
	store.update(
		nil,
		map[string]int64{
			"PollCount": 2,
		},
	)

	sender := newMetricsSender(server.URL, time.Second, 1, "", retry.DefaultPolicy())
	if err := sender.report(context.Background(), store); err == nil {
		t.Fatal("report returned nil error, want non-nil error")
	}

	assertMetricsSnapshot(t, storeState(store), metricsSnapshot{
		gauges: map[string]float64{},
		counters: map[string]int64{
			"PollCount": 2,
		},
	})
}

func TestMetricsSenderRunRespectsRateLimit(t *testing.T) {
	tests := []struct {
		name      string
		rateLimit int
	}{
		{
			name:      "single worker serializes requests",
			rateLimit: 1,
		},
		{
			name:      "several workers report in parallel",
			rateLimit: 3,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var inFlight, maxInFlight atomic.Int64

			server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
				current := inFlight.Add(1)
				for {
					observed := maxInFlight.Load()
					if current <= observed || maxInFlight.CompareAndSwap(observed, current) {
						break
					}
				}

				time.Sleep(20 * time.Millisecond)
				inFlight.Add(-1)

				response.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			store := newMetricsStore()
			store.update(map[string]float64{"Alloc": 1}, nil)

			sender := newMetricsSender(server.URL, time.Millisecond, test.rateLimit, "", retry.DefaultPolicy())

			ctx, cancel := context.WithCancel(context.Background())
			stopped := make(chan struct{})
			go func() {
				defer close(stopped)

				sender.run(ctx, store)
			}()

			time.Sleep(300 * time.Millisecond)
			cancel()

			select {
			case <-stopped:
			case <-time.After(5 * time.Second):
				t.Fatal("run did not return after the context was cancelled")
			}

			got := maxInFlight.Load()
			if got == 0 {
				t.Fatal("no requests were sent")
			}
			if got > int64(test.rateLimit) {
				t.Fatalf("requests in flight = %d, want at most %d", got, test.rateLimit)
			}
			if test.rateLimit > 1 && got < 2 {
				t.Fatalf("requests in flight = %d, want the pool to report in parallel", got)
			}
		})
	}
}

func TestNewMetricsSenderClampsRateLimit(t *testing.T) {
	for _, rateLimit := range []int{-1, 0} {
		sender := newMetricsSender("localhost:8080", time.Second, rateLimit, "", retry.DefaultPolicy())
		if sender.rateLimit != 1 {
			t.Fatalf("rate limit for %d = %d, want 1", rateLimit, sender.rateLimit)
		}
	}
}

func TestNormalizeServerAddress(t *testing.T) {
	tests := []struct {
		name          string
		serverAddress string
		want          string
	}{
		{
			name:          "address without scheme",
			serverAddress: "localhost:8080",
			want:          "http://localhost:8080",
		},
		{
			name:          "address with scheme",
			serverAddress: "http://localhost:8080",
			want:          "http://localhost:8080",
		},
		{
			name:          "address with trailing slash",
			serverAddress: "http://localhost:8080/",
			want:          "http://localhost:8080",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := normalizeServerAddress(test.serverAddress)
			if got != test.want {
				t.Fatalf("address = %q, want %q", got, test.want)
			}
		})
	}
}
