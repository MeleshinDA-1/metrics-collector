package agent

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
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

	sender := newMetricsSender(server.URL, time.Second, retry.DefaultPolicy())

	err := sender.sendBatch(context.Background(), []model.Metrics{{ID: "Alloc", MType: model.Gauge}})
	if err == nil {
		t.Fatal("sendBatch returned nil error, want non-nil error")
	}
}

func TestMetricsSenderSendMetrics(t *testing.T) {
	receivedMetrics := make([]model.Metrics, 0, 2)
	requestCount := 0
	requestMethod := ""
	requestPath := ""
	mediaType := ""
	contentEncoding := ""
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		metrics, err := decodeGzipMetricsBatch(request.Body)
		if err != nil {
			t.Errorf("read compressed batch: %v", err)
			response.WriteHeader(http.StatusBadRequest)
			return
		}
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

	sender := newMetricsSender(server.URL, time.Second, retry.DefaultPolicy())
	sender.sendMetrics(store)

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
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requestCount++
		response.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := newMetricsSender(server.URL, time.Second, retry.DefaultPolicy())
	sender.sendMetrics(newMetricsStore())

	if requestCount != 0 {
		t.Fatalf("requests count = %d, want 0", requestCount)
	}
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

func TestMetricsSenderKeepsCounterAfterFailedSend(t *testing.T) {
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

	sender := newMetricsSender(server.URL, time.Second, retry.DefaultPolicy())
	sender.sendMetrics(store)

	assertMetricsSnapshot(t, store.snapshot(), metricsSnapshot{
		gauges: map[string]float64{},
		counters: map[string]int64{
			"PollCount": 2,
		},
	})
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
