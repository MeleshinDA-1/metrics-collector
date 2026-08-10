package agent

import (
	"compress/gzip"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/MeleshinDA-1/metrics-collector/internal/model"
)

type capturedMetricRequest struct {
	method          string
	path            string
	mediaType       string
	contentEncoding string
	metric          model.Metrics
}

func TestMetricsSenderSendMetric(t *testing.T) {
	capturedRequests := make([]capturedMetricRequest, 0, 1)
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		metric, err := decodeGzipMetric(request)
		if err != nil {
			t.Errorf("read compressed metric: %v", err)
			response.WriteHeader(http.StatusBadRequest)
			return
		}

		capturedRequests = append(capturedRequests, capturedMetricRequest{
			method:          request.Method,
			path:            request.URL.Path,
			mediaType:       request.Header.Get("Content-Type"),
			contentEncoding: request.Header.Get("Content-Encoding"),
			metric:          metric,
		})

		response.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := newMetricsSender(server.URL, time.Second)
	metricValue := 42.5
	metric := model.Metrics{
		ID:    "Alloc",
		MType: model.Gauge,
		Value: &metricValue,
	}

	err := sender.sendMetric(metric)
	if err != nil {
		t.Fatalf("sendMetric returned error: %v", err)
	}

	if len(capturedRequests) != 1 {
		t.Fatalf("requests count = %d, want %d", len(capturedRequests), 1)
	}

	want := capturedMetricRequest{
		method:          http.MethodPost,
		path:            "/update",
		mediaType:       "application/json",
		contentEncoding: "gzip",
		metric:          metric,
	}
	if !reflect.DeepEqual(capturedRequests[0], want) {
		t.Fatalf("request = %+v, want %+v", capturedRequests[0], want)
	}
}

func TestMetricsSenderSendMetricReturnsErrorOnUnexpectedStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	sender := newMetricsSender(server.URL, time.Second)

	err := sender.sendMetric(model.Metrics{})
	if err == nil {
		t.Fatal("sendMetric returned nil error, want non-nil error")
	}
}

func TestMetricsSenderSendMetrics(t *testing.T) {
	receivedMetrics := make([]model.Metrics, 0, 2)
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		metric, err := decodeGzipMetric(request)
		if err != nil {
			t.Errorf("read compressed metric: %v", err)
			response.WriteHeader(http.StatusBadRequest)
			return
		}
		receivedMetrics = append(receivedMetrics, metric)

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

	sender := newMetricsSender(server.URL, time.Second)
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
}

func decodeGzipMetric(request *http.Request) (model.Metrics, error) {
	reader, err := gzip.NewReader(request.Body)
	if err != nil {
		return model.Metrics{}, err
	}
	defer reader.Close()

	var metric model.Metrics
	if err := json.NewDecoder(reader).Decode(&metric); err != nil {
		return model.Metrics{}, err
	}

	return metric, nil
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

	sender := newMetricsSender(server.URL, time.Second)
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
