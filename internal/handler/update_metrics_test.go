package handler

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/MeleshinDA-1/golang-practicum-alice/internal/repository"
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

func TestValueMetrics(t *testing.T) {
	storage := repository.NewMemStorage()
	storage.SetGauge("Alloc", 42.5)
	storage.AddCounter("PollCount", 2)
	metricsHandler := NewMetricsHandler(storage)

	tests := []struct {
		name       string
		target     string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "existing gauge",
			target:     "/value/gauge/Alloc",
			wantStatus: http.StatusOK,
			wantBody:   "42.5",
		},
		{
			name:       "existing counter",
			target:     "/value/counter/PollCount",
			wantStatus: http.StatusOK,
			wantBody:   "2",
		},
		{
			name:       "unknown metric",
			target:     "/value/gauge/Unknown",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, test.target, nil)
			request = mux.SetURLVars(request, map[string]string{
				"metricType": strings.Split(test.target, "/")[2],
				"metricName": strings.Split(test.target, "/")[3],
			})
			response := httptest.NewRecorder()

			metricsHandler.ValueMetrics(response, request)

			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, test.wantStatus)
			}

			if test.wantBody != "" && response.Body.String() != test.wantBody {
				t.Fatalf("body = %q, want %q", response.Body.String(), test.wantBody)
			}
		})
	}
}

func TestListMetrics(t *testing.T) {
	storage := repository.NewMemStorage()
	storage.SetGauge("Alloc", 42.5)
	storage.AddCounter("PollCount", 2)
	metricsHandler := NewMetricsHandler(storage)

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()

	metricsHandler.ListMetrics(response, request)
	result := response.Result()
	defer result.Body.Close()
	body, err := io.ReadAll(result.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}

	if result.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", result.StatusCode, http.StatusOK)
	}

	bodyText := string(body)
	for _, value := range []string{"Alloc", "42.5", "PollCount", "2"} {
		if !strings.Contains(bodyText, value) {
			t.Fatalf("body %q does not contain %q", bodyText, value)
		}
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

func handleUpdateMetricsWithHandler(metricsHandler *MetricsHandler, request updateMetricsRequest) *httptest.ResponseRecorder {
	target := fmt.Sprintf(
		"/update/%s/%s/%s",
		request.metricType,
		request.metricName,
		request.metricValue,
	)
	req := httptest.NewRequest(request.method, target, nil)
	req = mux.SetURLVars(req, map[string]string{
		"metricType":  request.metricType,
		"metricName":  request.metricName,
		"metricValue": request.metricValue,
	})

	response := httptest.NewRecorder()
	metricsHandler.UpdateMetrics(response, req)

	return response
}
