package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/MeleshinDA-1/metrics-collector/internal/repository"
	"github.com/gorilla/mux"
)

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
