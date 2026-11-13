package agent

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/MeleshinDA-1/metrics-collector/internal/handler"
	"github.com/MeleshinDA-1/metrics-collector/internal/handler/health"
	"github.com/MeleshinDA-1/metrics-collector/internal/handler/metrics"
	"github.com/MeleshinDA-1/metrics-collector/internal/hash"
	"github.com/MeleshinDA-1/metrics-collector/internal/model"
	"github.com/MeleshinDA-1/metrics-collector/internal/repository"
)

func benchmarkSnapshot() metricsSnapshot {
	names := []string{
		"Alloc", "BuckHashSys", "Frees", "GCCPUFraction", "GCSys", "HeapAlloc",
		"HeapIdle", "HeapInuse", "HeapObjects", "HeapReleased", "HeapSys",
		"LastGC", "Lookups", "MCacheInuse", "MCacheSys", "MSpanInuse", "MSpanSys",
		"Mallocs", "NextGC", "NumForcedGC", "NumGC", "OtherSys", "PauseTotalNs",
		"StackInuse", "StackSys", "Sys", "TotalAlloc", "RandomValue",
		"TotalMemory", "FreeMemory", "CPUutilization1", "CPUutilization2",
	}
	snapshot := metricsSnapshot{gauges: make(map[string]float64, len(names)), counters: map[string]int64{"PollCount": 1}}
	for i, name := range names {
		snapshot.gauges[name] = float64((i+1)*1024) + 0.5
	}
	return snapshot
}

func BenchmarkBuildBatch(b *testing.B) {
	snapshot := benchmarkSnapshot()
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		batch := buildBatch(snapshot)
		if len(batch) != 33 {
			b.Fatalf("batch length = %d, want 33", len(batch))
		}
	}
}

func BenchmarkEncodeBatch(b *testing.B) {
	batch := buildBatch(benchmarkSnapshot())
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		payload, err := buildRequestBody(encodeJSON(batch))
		if err != nil {
			b.Fatal(err)
		}
		if _, err := buildRequestBody(withGzipCompression(writeRaw(payload))); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMetricsPipeline exercises the agent encoding and the complete server
// router in process. One operation updates 33 metrics, then reads the counter.
// Fixed metric names keep the live dataset bounded across all iterations.
func BenchmarkMetricsPipeline(b *testing.B) {
	previousLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError})))
	b.Cleanup(func() { slog.SetDefault(previousLogger) })
	storage := repository.NewMemStorage()
	router := handler.NewRouter(metrics.NewMetricsHandler(storage), health.NewPingHandler(nil), "benchmark-key")
	snapshot := benchmarkSnapshot()
	var lastResponse *httptest.ResponseRecorder
	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		batch := buildBatch(snapshot)
		payload, err := buildRequestBody(encodeJSON(batch))
		if err != nil {
			b.Fatal(err)
		}
		body, err := buildRequestBody(withGzipCompression(writeRaw(payload)))
		if err != nil {
			b.Fatal(err)
		}
		request := httptest.NewRequest(http.MethodPost, "/updates/", bytes.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Content-Encoding", "gzip")
		request.Header.Set("Accept-Encoding", "gzip")
		request.Header.Set(hash.Header, hash.Sign(payload, "benchmark-key"))
		lastResponse = httptest.NewRecorder()
		router.ServeHTTP(lastResponse, request)
		if lastResponse.Code != http.StatusOK {
			b.Fatalf("update: status %d, body %s", lastResponse.Code, lastResponse.Body)
		}
		read := httptest.NewRecorder()
		router.ServeHTTP(read, httptest.NewRequest(http.MethodGet, "/value/counter/PollCount", nil))
		if want := strconv.Itoa(i + 1); read.Code != http.StatusOK || read.Body.String() != want {
			b.Fatalf("counter: status %d, body %s, want %s", read.Code, read.Body, want)
		}
	}
	b.StopTimer()
	if lastResponse.Header().Get("Content-Encoding") != "gzip" {
		b.Fatal("response was not gzip-compressed")
	}
	reader, err := gzip.NewReader(lastResponse.Body)
	if err != nil {
		b.Fatal(err)
	}
	defer reader.Close()
	decoded, err := io.ReadAll(reader)
	if err != nil {
		b.Fatal(err)
	}
	if !hash.Equal(decoded, "benchmark-key", lastResponse.Header().Get(hash.Header)) {
		b.Fatal("response signature mismatch")
	}
	var received []model.Metrics
	if err := json.Unmarshal(decoded, &received); err != nil || len(received) != 33 {
		b.Fatalf("response: %d metrics, error %v", len(received), err)
	}
	stored, err := storage.Snapshot(context.Background())
	if err != nil || len(stored.Gauges) != 32 || stored.Counters["PollCount"] != int64(b.N) {
		b.Fatalf("stored metrics after workload: %+v, error %v", stored, err)
	}
}
