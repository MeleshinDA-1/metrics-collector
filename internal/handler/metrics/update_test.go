package metrics

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/MeleshinDA-1/metrics-collector/internal/handler"
	"github.com/MeleshinDA-1/metrics-collector/internal/handler/health"
	"github.com/MeleshinDA-1/metrics-collector/internal/model"
	"github.com/MeleshinDA-1/metrics-collector/internal/repository"
	"github.com/gorilla/mux"
)

type updateMetricsRequest struct {
	method      string
	metricType  string
	metricName  string
	metricValue string
}

type updateMetricsWant struct {
	statusCode int
}

type updateMetricsCase struct {
	name    string
	request updateMetricsRequest
	want    updateMetricsWant
}

func TestUpdateMetrics(t *testing.T) {
	tests := []updateMetricsCase{
		{
			name: "valid counter",
			request: updateMetricsRequest{
				method:      http.MethodPost,
				metricType:  "counter",
				metricName:  "PollCount",
				metricValue: "42",
			},
			want: updateMetricsWant{
				statusCode: http.StatusOK,
			},
		},
		{
			name: "invalid counter value",
			request: updateMetricsRequest{
				method:      http.MethodPost,
				metricType:  "counter",
				metricName:  "PollCount",
				metricValue: "not-number",
			},
			want: updateMetricsWant{
				statusCode: http.StatusBadRequest,
			},
		},
		{
			name: "valid gauge",
			request: updateMetricsRequest{
				method:      http.MethodPost,
				metricType:  "gauge",
				metricName:  "Alloc",
				metricValue: "42.5",
			},
			want: updateMetricsWant{
				statusCode: http.StatusOK,
			},
		},
		{
			name: "invalid gauge value",
			request: updateMetricsRequest{
				method:      http.MethodPost,
				metricType:  "gauge",
				metricName:  "Alloc",
				metricValue: "not-number",
			},
			want: updateMetricsWant{
				statusCode: http.StatusBadRequest,
			},
		},
		{
			name: "unknown metric type",
			request: updateMetricsRequest{
				method:      http.MethodPost,
				metricType:  "unknown",
				metricName:  "PollCount",
				metricValue: "42",
			},
			want: updateMetricsWant{
				statusCode: http.StatusBadRequest,
			},
		},
		{
			name: "method not allowed",
			request: updateMetricsRequest{
				method:      http.MethodGet,
				metricType:  "counter",
				metricName:  "PollCount",
				metricValue: "42",
			},
			want: updateMetricsWant{
				statusCode: http.StatusMethodNotAllowed,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assertUpdateMetricsStatus(t, test.request, test.want.statusCode)
		})
	}
}

func TestUpdateMetricsRouteNotFound(t *testing.T) {
	metricsHandler := NewMetricsHandler(repository.NewMemStorage())
	router := mux.NewRouter()
	router.HandleFunc("/update/{metricType}/{metricName}/{metricValue}", metricsHandler.UpdateMetrics).Methods(http.MethodPost)

	request := httptest.NewRequest(http.MethodPost, "/update/counter/PollCount", nil)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
	}
}

func TestUpdateMetricsStoresValues(t *testing.T) {
	storage := repository.NewMemStorage()
	metricsHandler := NewMetricsHandler(storage)

	updateRequests := []updateMetricsRequest{
		{
			method:      http.MethodPost,
			metricType:  "counter",
			metricName:  "PollCount",
			metricValue: "2",
		},
		{
			method:      http.MethodPost,
			metricType:  "gauge",
			metricName:  "Alloc",
			metricValue: "42.5",
		},
	}

	for _, updateRequest := range updateRequests {
		response := handleUpdateMetricsWithHandler(metricsHandler, updateRequest)
		if response.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
		}
	}

	assertValueMetricsBody(t, metricsHandler, "/value/counter/PollCount", "counter", "PollCount", "2")
	assertValueMetricsBody(t, metricsHandler, "/value/gauge/Alloc", "gauge", "Alloc", "42.5")
}

func TestUpdateMetricsJSONResponse(t *testing.T) {
	metricsHandler := NewMetricsHandler(repository.NewMemStorage())
	var response *httptest.ResponseRecorder
	for range 2 {
		request := httptest.NewRequest(
			http.MethodPost,
			"/update",
			strings.NewReader(`{"id":"PollCount","type":"counter","delta":2}`),
		)
		request.Header.Set("Content-Type", "application/json")
		response = httptest.NewRecorder()
		handler.NewRouter(metricsHandler, health.NewPingHandler(nil), "").ServeHTTP(response, request)
	}

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if contentType := response.Header().Get("Content-Type"); contentType != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", contentType)
	}

	var metric model.Metrics
	if err := json.NewDecoder(response.Body).Decode(&metric); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if metric.Delta == nil || *metric.Delta != 4 {
		t.Fatalf("counter delta = %v, want 4", metric.Delta)
	}
}

func assertUpdateMetricsStatus(t *testing.T, request updateMetricsRequest, wantStatusCode int) {
	t.Helper()

	response := handleUpdateMetrics(request)

	if response.Code != wantStatusCode {
		t.Fatalf("status = %d, want %d", response.Code, wantStatusCode)
	}
}

func assertValueMetricsBody(t *testing.T, metricsHandler *MetricsHandler, target string, metricType string, metricName string, wantBody string) {
	t.Helper()

	request := httptest.NewRequest(http.MethodGet, target, nil)
	request = mux.SetURLVars(request, map[string]string{
		"metricType": metricType,
		"metricName": metricName,
	})
	response := httptest.NewRecorder()

	metricsHandler.ValueMetrics(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}

	if response.Body.String() != wantBody {
		t.Fatalf("body = %q, want %q", response.Body.String(), wantBody)
	}
}

func handleUpdateMetrics(request updateMetricsRequest) *httptest.ResponseRecorder {
	metricsHandler := NewMetricsHandler(repository.NewMemStorage())

	return handleUpdateMetricsWithHandler(metricsHandler, request)
}

func handleUpdateMetricsWithHandler(metricsHandler handler.MetricsEndpoints, request updateMetricsRequest) *httptest.ResponseRecorder {
	target := fmt.Sprintf(
		"/update/%s/%s/%s",
		request.metricType,
		request.metricName,
		request.metricValue,
	)
	req := httptest.NewRequest(request.method, target, nil)
	response := httptest.NewRecorder()
	handler.NewRouter(metricsHandler, health.NewPingHandler(nil), "").ServeHTTP(response, req)

	return response
}
