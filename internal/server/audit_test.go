package server

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/MeleshinDA-1/metrics-collector/internal/audit"
	"github.com/MeleshinDA-1/metrics-collector/internal/config"
)

func TestRunDrainsAuditBeforeClosingFile(t *testing.T) {
	previousLogger := slog.Default()
	defer slog.SetDefault(previousLogger)
	release := make(chan struct{})
	releaseHTTP := sync.OnceFunc(func() { close(release) })
	received := make(chan audit.Event, 3)
	receiver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var event audit.Event
		if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
			t.Error(err)
		}
		received <- event
		select {
		case <-release:
			w.WriteHeader(http.StatusNoContent)
		case <-r.Context().Done():
		}
	}))
	defer receiver.Close()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "audit.jsonl")
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	var runErr error
	go func() {
		runErr = Run(ctx, config.ServerConfig{Address: address, AuditFile: path, AuditURL: receiver.URL})
		close(done)
	}()
	defer func() {
		cancel()
		releaseHTTP()
		select {
		case <-done:
		case <-time.After(8 * time.Second):
			t.Error("server did not stop")
		}
	}()
	client := &http.Client{Timeout: time.Second}
	defer client.CloseIdleConnections()
	deadline := time.Now().Add(3 * time.Second)
	for {
		response, err := client.Get("http://" + address + "/")
		if err == nil {
			response.Body.Close()
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("server did not start: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}
	for range 3 {
		response, err := client.Post("http://"+address+"/update/counter/PollCount/1", "text/plain", nil)
		if err != nil {
			t.Fatalf("update waited for blocked audit receiver: %v", err)
		}
		response.Body.Close()
		if response.StatusCode != http.StatusOK {
			t.Fatalf("update status = %d", response.StatusCode)
		}
	}
	response, err := client.Get("http://" + address + "/value/counter/PollCount")
	if err != nil {
		t.Fatal(err)
	}
	value, err := io.ReadAll(response.Body)
	response.Body.Close()
	if err != nil || string(value) != "3" {
		t.Fatalf("counter = %q, error = %v; want 3", value, err)
	}
	select {
	case <-received:
	case <-time.After(time.Second):
		t.Fatal("HTTP audit did not start")
	}
	cancel()
	select {
	case <-done:
		t.Fatalf("server stopped before audit delivery finished: %v", runErr)
	case <-time.After(50 * time.Millisecond):
	}
	releaseHTTP()
	select {
	case <-done:
		if runErr != nil {
			t.Fatal(runErr)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("server did not finish after receiver recovered")
	}
	if len(received) != 2 {
		t.Fatalf("remaining HTTP audit events = %d, want 2", len(received))
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("file events = %d, want 3: %s", len(lines), data)
	}
	for _, line := range lines {
		var event audit.Event
		if err := json.Unmarshal([]byte(line), &event); err != nil || len(event.Metrics) != 1 || event.Metrics[0] != "PollCount" {
			t.Fatalf("file event = %s, error = %v", line, err)
		}
	}
}
