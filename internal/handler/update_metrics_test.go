package handler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
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
	mux := http.NewServeMux()
	mux.HandleFunc("/update/{metricType}/{metricName}/{metricValue}", UpdateMetrics)

	request := httptest.NewRequest(http.MethodPost, "/update/counter/PollCount", nil)
	response := httptest.NewRecorder()

	mux.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
	}
}

func assertUpdateMetricsStatus(t *testing.T, request updateMetricsRequest, wantStatusCode int) {
	t.Helper()

	response := handleUpdateMetrics(request)

	if response.Code != wantStatusCode {
		t.Fatalf("status = %d, want %d", response.Code, wantStatusCode)
	}
}

func handleUpdateMetrics(request updateMetricsRequest) *httptest.ResponseRecorder {
	target := fmt.Sprintf(
		"/update/%s/%s/%s",
		request.metricType,
		request.metricName,
		request.metricValue,
	)
	req := httptest.NewRequest(request.method, target, nil)
	req.SetPathValue("metricType", request.metricType)
	req.SetPathValue("metricValue", request.metricValue)

	response := httptest.NewRecorder()
	UpdateMetrics(response, req)

	return response
}
