package metrics

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MeleshinDA-1/metrics-collector/internal/handler"
	"github.com/MeleshinDA-1/metrics-collector/internal/handler/health"
	"github.com/MeleshinDA-1/metrics-collector/internal/model"
	"github.com/MeleshinDA-1/metrics-collector/internal/repository"
)

type metricsRepositorySpy struct {
	flushCalls int
	snapshot   model.MetricsSnapshot
}

func (repo *metricsRepositorySpy) Flush(ctx context.Context, storage repository.MetricsSnapshotProvider) error {
	repo.flushCalls++

	snapshot, err := storage.Snapshot(ctx)
	if err != nil {
		return err
	}
	repo.snapshot = snapshot

	return nil
}

func TestUpdateMetricsSavesSynchronously(t *testing.T) {
	fileRepository := &repository.FileMetricsRepository{
		FilePath: filepath.Join(t.TempDir(), "metrics.json"),
	}
	storage := repository.NewMemStorage()
	metricsHandler := NewPersistingMetricsHandler(NewMetricsHandler(storage), fileRepository)

	response := handleUpdateMetricsWithHandler(metricsHandler, updateMetricsRequest{
		method:      http.MethodPost,
		metricType:  "counter",
		metricName:  "PollCount",
		metricValue: "2",
	})
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}

	restoredStorage := repository.NewMemStorage()
	if err := fileRepository.Restore(context.Background(), restoredStorage); err != nil {
		t.Fatalf("restore metrics: %v", err)
	}

	value, ok, err := restoredStorage.GetCounter(context.Background(), "PollCount")
	if err != nil {
		t.Fatalf("get counter: %v", err)
	}
	if !ok {
		t.Fatal("counter PollCount not found")
	}
	if value != 2 {
		t.Fatalf("counter PollCount = %d, want 2", value)
	}
}

func TestUpdateMetricsJSONSavesSynchronously(t *testing.T) {
	storage := repository.NewMemStorage()
	metricsRepository := &metricsRepositorySpy{}
	metricsHandler := NewPersistingMetricsHandler(NewMetricsHandler(storage), metricsRepository)

	request := httptest.NewRequest(
		http.MethodPost,
		"/update",
		strings.NewReader(`{"id":"PollCount","type":"counter","delta":2}`),
	)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.NewRouter(metricsHandler, health.NewPingHandler(nil)).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if metricsRepository.flushCalls != 1 {
		t.Fatalf("flush calls = %d, want 1", metricsRepository.flushCalls)
	}
	if value := metricsRepository.snapshot.Counters["PollCount"]; value != 2 {
		t.Fatalf("persisted counter PollCount = %d, want 2", value)
	}
}
