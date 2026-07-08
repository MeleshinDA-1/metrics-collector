package agent

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
)

type receivedMetricRequest struct {
	method      string
	path        string
	contentType string
}

func TestMetricSenderSendMetric(t *testing.T) {
	receivedRequests := make([]receivedMetricRequest, 0, 1)
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		receivedRequests = append(receivedRequests, receivedMetricRequest{
			method:      request.Method,
			path:        request.URL.Path,
			contentType: request.Header.Get("Content-Type"),
		})

		response.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := newMetricSender(server.URL, time.Second)

	err := sender.sendMetric("gauge", "Alloc", "42.5")
	if err != nil {
		t.Fatalf("sendMetric returned error: %v", err)
	}

	if len(receivedRequests) != 1 {
		t.Fatalf("requests count = %d, want %d", len(receivedRequests), 1)
	}

	want := receivedMetricRequest{
		method:      http.MethodPost,
		path:        "/update/gauge/Alloc/42.5",
		contentType: "text/plain",
	}
	if receivedRequests[0] != want {
		t.Fatalf("request = %+v, want %+v", receivedRequests[0], want)
	}
}

func TestMetricSenderSendMetricReturnsErrorOnUnexpectedStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	sender := newMetricSender(server.URL, time.Second)

	err := sender.sendMetric("gauge", "Alloc", "42.5")
	if err == nil {
		t.Fatal("sendMetric returned nil error, want non-nil error")
	}
}

func TestMetricSenderSendMetricsInternal(t *testing.T) {
	receivedPaths := make([]string, 0, 2)
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		receivedPaths = append(receivedPaths, request.URL.Path)

		response.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	storage := newMetricStorage()
	storage.updateMetrics(
		map[string]float64{
			"Alloc": 42.5,
		},
		map[string]int64{
			"PollCount": 2,
		},
	)

	sender := newMetricSender(server.URL, time.Second)
	sender.sendMetricsInternal(storage)

	wantPaths := map[string]bool{
		"/update/gauge/Alloc/42.5":    true,
		"/update/counter/PollCount/2": true,
	}
	gotPaths := map[string]bool{}
	for _, path := range receivedPaths {
		gotPaths[path] = true
	}

	if !reflect.DeepEqual(gotPaths, wantPaths) {
		t.Fatalf("paths = %v, want %v", gotPaths, wantPaths)
	}
}

func TestMetricSenderKeepsCounterAfterFailedSend(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	storage := newMetricStorage()
	storage.updateMetrics(
		nil,
		map[string]int64{
			"PollCount": 2,
		},
	)

	sender := newMetricSender(server.URL, time.Second)
	sender.sendMetricsInternal(storage)

	assertAllMetrics(t, storage.snapshot(), allMetrics{
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
