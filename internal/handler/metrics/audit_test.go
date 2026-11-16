package metrics

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/MeleshinDA-1/metrics-collector/internal/audit"
	"github.com/MeleshinDA-1/metrics-collector/internal/handler"
	"github.com/MeleshinDA-1/metrics-collector/internal/handler/health"
	"github.com/MeleshinDA-1/metrics-collector/internal/hash"
	"github.com/MeleshinDA-1/metrics-collector/internal/model"
	"github.com/MeleshinDA-1/metrics-collector/internal/repository"
)

type auditRecorder struct {
	events   []audit.Event
	err      error
	received chan audit.Event
}

func (r *auditRecorder) Notify(_ context.Context, event audit.Event) error {
	r.events = append(r.events, event)
	if r.received != nil {
		r.received <- event
	}
	return r.err
}

func closeAudit(t *testing.T, handler *MetricsHandler) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := handler.Close(ctx); err != nil {
		t.Errorf("close audit: %v", err)
	}
}

func nextAuditEvent(t *testing.T, events <-chan audit.Event) audit.Event {
	t.Helper()
	select {
	case event := <-events:
		return event
	case <-time.After(3 * time.Second):
		t.Fatal("audit event was not delivered")
		return audit.Event{}
	}
}

func TestAuditSuccessfulUpdatesAndContinue(t *testing.T) {
	for _, persistent := range []bool{false, true} {
		name := "memory"
		if persistent {
			name = "synchronous file"
		}
		t.Run(name, func(t *testing.T) {
			storage := repository.NewMemStorage()
			observer := &auditRecorder{err: errors.New("receiver unavailable")}
			other := &auditRecorder{received: make(chan audit.Event, 6)}
			metricsHandler := NewMetricsHandler(storage, observer, other)
			defer closeAudit(t, metricsHandler)
			var endpoints handler.MetricsEndpoints = metricsHandler
			fileRepo := &repository.FileMetricsRepository{FilePath: filepath.Join(t.TempDir(), "metrics.json")}
			if persistent {
				endpoints = NewPersistingMetricsHandler(metricsHandler, fileRepo)
			}
			router := handler.NewRouter(endpoints, health.NewPingHandler(nil), "secret")
			updates := []struct {
				path, body, remote, ip string
				names                  []string
			}{
				{"/update/counter/PollCount/2", "", "192.0.2.1:1234", "192.0.2.1", []string{"PollCount"}},
				{"/update/", `{"id":"Alloc","type":"gauge","value":42.5}`, "[2001:db8::1]:5678", "2001:db8::1", []string{"Alloc"}},
				{"/updates/", `[{"id":"PollCount","type":"counter","delta":3},{"id":"Alloc","type":"gauge","value":43}]`, "192.0.2.2", "192.0.2.2", []string{"PollCount", "Alloc"}},
			}
			start := time.Now().Unix()
			for cycle := range 2 {
				for _, update := range updates {
					var body bytes.Buffer
					compressed := gzip.NewWriter(&body)
					if _, err := compressed.Write([]byte(update.body)); err != nil {
						t.Fatal(err)
					}
					if err := compressed.Close(); err != nil {
						t.Fatal(err)
					}
					request := httptest.NewRequest(http.MethodPost, update.path, &body)
					request.RemoteAddr = update.remote
					request.Header.Set("Content-Encoding", "gzip")
					request.Header.Set(hash.Header, hash.Sign([]byte(update.body), "secret"))
					request.Header.Set("X-Forwarded-For", "203.0.113.99")
					response := httptest.NewRecorder()
					router.ServeHTTP(response, request)
					if response.Code != http.StatusOK {
						t.Fatalf("cycle %d %s: status = %d, body = %s", cycle, update.path, response.Code, response.Body)
					}
					event := nextAuditEvent(t, other.received)
					if !reflect.DeepEqual(event.Metrics, update.names) || event.IPAddress != update.ip || event.Timestamp < start || event.Timestamp > time.Now().Unix() {
						t.Fatalf("%s: event = %+v, want names %v, IP %s, current timestamp", update.path, event, update.names, update.ip)
					}
				}
				response := httptest.NewRecorder()
				router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/value/counter/PollCount", nil))
				want := "5"
				if cycle == 1 {
					want = "10"
				}
				if response.Code != http.StatusOK || response.Body.String() != want {
					t.Fatalf("cycle %d: counter status = %d, value = %q, want %s", cycle, response.Code, response.Body.String(), want)
				}
			}
			closeAudit(t, metricsHandler)
			if len(observer.events) != 6 || !reflect.DeepEqual(observer.events, other.events) {
				t.Fatalf("expected six identical events per observer: first=%+v, second=%+v", observer.events, other.events)
			}
			if persistent {
				restored := repository.NewMemStorage()
				if err := fileRepo.Restore(context.Background(), restored); err != nil {
					t.Fatal(err)
				}
				value, ok, err := restored.GetCounter(context.Background(), "PollCount")
				if err != nil || !ok || value != 10 {
					t.Fatalf("persisted counter = %d, found = %v, error = %v; want 10", value, ok, err)
				}
			}
		})
	}
}

func TestAuditHTTPRetryAndContinue(t *testing.T) {
	var calls atomic.Int32
	received := make(chan audit.Event, 6)
	started := make(chan struct{})
	release := make(chan struct{})
	releaseHTTP := sync.OnceFunc(func() { close(release) })
	receiver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var event audit.Event
		if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
			t.Errorf("decode audit: %v", err)
		}
		select {
		case received <- event:
		default:
			t.Error("unexpected extra audit request")
		}
		call := calls.Add(1)
		if call == 1 {
			close(started)
			select {
			case <-release:
			case <-r.Context().Done():
				return
			}
		}
		if call%3 == 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer receiver.Close()
	remote, err := audit.NewHTTPObserver(receiver.URL)
	if err != nil {
		t.Fatal(err)
	}
	storage := repository.NewMemStorage()
	other := &auditRecorder{received: make(chan audit.Event, 4)}
	metricsHandler := NewMetricsHandler(storage, remote, other)
	defer func() {
		releaseHTTP()
		closeAudit(t, metricsHandler)
	}()
	router := handler.NewRouter(metricsHandler, health.NewPingHandler(nil), "")
	for update := range 4 {
		response := httptest.NewRecorder()
		done := make(chan struct{})
		go func() {
			router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/update/counter/PollCount/1", nil))
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatal("metrics update waited for HTTP audit delivery")
		}
		nextAuditEvent(t, other.received)
		value, ok, err := storage.GetCounter(context.Background(), "PollCount")
		if response.Code != http.StatusOK || err != nil || !ok || value != int64(update+1) {
			t.Fatalf("update %d: status = %d, counter = %d, found = %v, error = %v", update, response.Code, value, ok, err)
		}
	}
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("HTTP audit did not start")
	}
	releaseHTTP()
	closeAudit(t, metricsHandler)
	if got := calls.Load(); got != 6 || len(other.events) != 4 {
		t.Fatalf("HTTP attempts = %d, other observer events = %d; want 6 and 4", got, len(other.events))
	}
	for _, event := range []audit.Event{other.events[0], other.events[1], other.events[1], other.events[2], other.events[3], other.events[3]} {
		select {
		case got := <-received:
			if !reflect.DeepEqual(got, event) {
				t.Fatalf("HTTP event = %+v, want %+v", got, event)
			}
		default:
			t.Fatal("missing audit request")
		}
	}
}

func TestAuditSkipsRejectedRequests(t *testing.T) {
	observer := &auditRecorder{}
	metricsHandler := NewMetricsHandler(repository.NewMemStorage(), observer)
	defer closeAudit(t, metricsHandler)
	router := handler.NewRouter(metricsHandler, health.NewPingHandler(nil), "secret")
	for _, request := range []struct{ path, body string }{
		{"/update/counter/PollCount/no", ""},
		{"/update/", "{"},
		{"/update/", `{"id":"Alloc","type":"gauge"}`},
		{"/updates/", "[]"},
		{"/updates/", `[{"id":"Alloc","type":"gauge","value":1},{"id":"x","type":"invalid"}]`},
	} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, request.path, strings.NewReader(request.body)))
		if response.Code != http.StatusBadRequest {
			t.Fatalf("%s %s: status = %d, want 400", request.path, request.body, response.Code)
		}
	}
	request := httptest.NewRequest(http.MethodPost, "/update/counter/PollCount/1", nil)
	request.Header.Set(hash.Header, "invalid")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	closeAudit(t, metricsHandler)
	if response.Code != http.StatusBadRequest || len(observer.events) != 0 {
		t.Fatalf("bad signature status = %d, events after rejections = %+v", response.Code, observer.events)
	}
}

type failingAuditStorage struct{ MetricsStorage }

func (failingAuditStorage) AddCounter(context.Context, string, int64) (int64, error) {
	return 0, errors.New("storage unavailable")
}

func (failingAuditStorage) UpdateBatch(context.Context, []model.Metrics) error {
	return errors.New("storage unavailable")
}

func TestAuditSkipsStorageAndFlushFailures(t *testing.T) {
	for _, failure := range []string{"storage", "flush"} {
		t.Run(failure, func(t *testing.T) {
			observer := &auditRecorder{}
			var storage MetricsStorage = repository.NewMemStorage()
			if failure == "storage" {
				storage = failingAuditStorage{storage}
			}
			metricsHandler := NewMetricsHandler(storage, observer)
			defer closeAudit(t, metricsHandler)
			var endpoints handler.MetricsEndpoints = metricsHandler
			if failure == "flush" {
				endpoints = NewPersistingMetricsHandler(metricsHandler, &repository.FileMetricsRepository{FilePath: t.TempDir()})
			}
			router := handler.NewRouter(endpoints, health.NewPingHandler(nil), "")
			for _, update := range []struct{ path, body string }{
				{"/update/counter/PollCount/1", ""},
				{"/update/", `{"id":"PollCount","type":"counter","delta":1}`},
				{"/updates/", `[{"id":"PollCount","type":"counter","delta":1}]`},
			} {
				response := httptest.NewRecorder()
				router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, update.path, strings.NewReader(update.body)))
				if response.Code != http.StatusInternalServerError {
					t.Fatalf("%s: status = %d, want 500", update.path, response.Code)
				}
			}
			closeAudit(t, metricsHandler)
			if len(observer.events) != 0 {
				t.Fatalf("failed updates produced events: %+v", observer.events)
			}
		})
	}
}
