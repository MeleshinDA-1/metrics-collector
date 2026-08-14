package metrics

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/MeleshinDA-1/metrics-collector/internal/repository"
)

func TestListMetrics(t *testing.T) {
	storage := repository.NewMemStorage()
	storage.SetGauge(context.Background(), "Alloc", 42.5)
	storage.AddCounter(context.Background(), "PollCount", 2)
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

func TestListMetricsEscapesHTML(t *testing.T) {
	storage := repository.NewMemStorage()
	storage.SetGauge(context.Background(), `<script>alert("x")</script>`, 1)
	metricsHandler := NewMetricsHandler(storage)

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()

	metricsHandler.ListMetrics(response, request)

	bodyText := response.Body.String()
	if strings.Contains(bodyText, `<script>alert("x")</script>`) {
		t.Fatalf("body contains unescaped HTML: %q", bodyText)
	}
	if !strings.Contains(bodyText, "&lt;script&gt;") {
		t.Fatalf("body does not contain escaped metric name: %q", bodyText)
	}
}
