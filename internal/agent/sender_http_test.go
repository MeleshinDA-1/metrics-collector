package agent

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/MeleshinDA-1/metrics-collector/internal/model"
)

func TestIsRetriableSendError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "transport error",
			err:  &transportError{Err: errors.New("connection refused")},
			want: true,
		},
		{
			name: "wrapped transport error",
			err:  fmt.Errorf("send batch: %w", &transportError{Err: errors.New("connection refused")}),
			want: true,
		},
		{
			name: "unexpected status",
			err:  fmt.Errorf("unexpected status: %d", http.StatusBadRequest),
			want: false,
		},
		{
			name: "plain error",
			err:  errors.New("boom"),
			want: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := isRetriableSendError(test.err); got != test.want {
				t.Fatalf("isRetriableSendError = %v, want %v", got, test.want)
			}
		})
	}
}

func TestPostRebuildsBodyOnEveryAttempt(t *testing.T) {
	receivedBodies := make([]int, 0, 2)
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		metrics, err := decodeGzipMetricsBatch(request)
		if err != nil {
			t.Errorf("read compressed batch: %v", err)
			response.WriteHeader(http.StatusBadRequest)
			return
		}
		receivedBodies = append(receivedBodies, len(metrics))

		response.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := newMetricsSender(server.URL, time.Second)
	metricValue := 42.5
	batch := []model.Metrics{{ID: "Alloc", MType: model.Gauge, Value: &metricValue}}

	for attempt := 0; attempt < 2; attempt++ {
		if err := sender.sendBatch(batch); err != nil {
			t.Fatalf("sendBatch returned error: %v", err)
		}
	}

	if len(receivedBodies) != 2 {
		t.Fatalf("requests count = %d, want 2", len(receivedBodies))
	}
	for i, count := range receivedBodies {
		if count != 1 {
			t.Fatalf("request %d carried %d metrics, want 1", i, count)
		}
	}
}
