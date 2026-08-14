package health

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MeleshinDA-1/metrics-collector/internal/handler"
	"github.com/MeleshinDA-1/metrics-collector/internal/handler/metrics"
	"github.com/MeleshinDA-1/metrics-collector/internal/repository"
)

type pingerStub struct {
	err error
}

func (p pingerStub) Ping(context.Context) error {
	return p.err
}

func TestPingDB(t *testing.T) {
	tests := []struct {
		name           string
		pinger         DBPinger
		wantStatusCode int
	}{
		{
			name:           "database is available",
			pinger:         pingerStub{},
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "database is unavailable",
			pinger:         pingerStub{err: errors.New("connection refused")},
			wantStatusCode: http.StatusInternalServerError,
		},
		{
			name:           "database is not configured",
			pinger:         nil,
			wantStatusCode: http.StatusInternalServerError,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			metricsHandler := metrics.NewMetricsHandler(repository.NewMemStorage())
			request := httptest.NewRequest(http.MethodGet, "/ping", nil)
			response := httptest.NewRecorder()

			handler.NewRouter(metricsHandler, NewPingHandler(test.pinger)).ServeHTTP(response, request)

			if response.Code != test.wantStatusCode {
				t.Fatalf("status = %d, want %d", response.Code, test.wantStatusCode)
			}
		})
	}
}
