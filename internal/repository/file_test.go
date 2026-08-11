package repository

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/MeleshinDA-1/metrics-collector/internal/model"
)

func TestFileMetricsRepositoryFlushAndRestore(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "metrics.json")
	repository := &FileMetricsRepository{FilePath: filePath}
	storage := NewMemStorage()
	storage.SetGauge("Alloc", 42.5)
	storage.SetGauge("ZeroGauge", 0)
	storage.AddCounter("PollCount", 2)
	storage.AddCounter("ZeroCounter", 0)

	if err := repository.Flush(storage); err != nil {
		t.Fatalf("flush metrics: %v", err)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("read metrics file: %v", err)
	}

	var metrics []model.Metrics
	if err := json.Unmarshal(data, &metrics); err != nil {
		t.Fatalf("decode metrics: %v", err)
	}
	if len(metrics) != 4 {
		t.Fatalf("metrics count = %d, want 4", len(metrics))
	}

	restoredStorage := NewMemStorage()
	if err := repository.Restore(restoredStorage); err != nil {
		t.Fatalf("restore metrics: %v", err)
	}

	assertGauge(t, restoredStorage, "Alloc", 42.5)
	assertGauge(t, restoredStorage, "ZeroGauge", 0)
	assertCounter(t, restoredStorage, "PollCount", 2)
	assertCounter(t, restoredStorage, "ZeroCounter", 0)
}

func TestFileMetricsRepositoryRestoreEmptyStorage(t *testing.T) {
	tests := []struct {
		name       string
		createFile bool
	}{
		{name: "missing file"},
		{name: "empty file", createFile: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			filePath := filepath.Join(t.TempDir(), "metrics.json")
			if test.createFile {
				if err := os.WriteFile(filePath, nil, 0600); err != nil {
					t.Fatalf("create metrics file: %v", err)
				}
			}

			repository := &FileMetricsRepository{FilePath: filePath}
			storage := NewMemStorage()
			if err := repository.Restore(storage); err != nil {
				t.Fatalf("restore metrics: %v", err)
			}
		})
	}
}

func assertGauge(t *testing.T, storage *MemStorage, name string, want float64) {
	t.Helper()

	value, ok, err := storage.GetGauge(name)
	if err != nil {
		t.Fatalf("get gauge %q: %v", name, err)
	}
	if !ok {
		t.Fatalf("gauge %q not found", name)
	}
	if value != want {
		t.Fatalf("gauge %q = %v, want %v", name, value, want)
	}
}

func assertCounter(t *testing.T, storage *MemStorage, name string, want int64) {
	t.Helper()

	value, ok, err := storage.GetCounter(name)
	if err != nil {
		t.Fatalf("get counter %q: %v", name, err)
	}
	if !ok {
		t.Fatalf("counter %q not found", name)
	}
	if value != want {
		t.Fatalf("counter %q = %v, want %v", name, value, want)
	}
}
