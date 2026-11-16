package audit

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/MeleshinDA-1/metrics-collector/internal/retry"
)

func TestPublisherContinuesAfterHTTPFailure(t *testing.T) {
	for _, test := range []struct {
		name     string
		statuses []int
		attempts int
		wantErr  bool
	}{
		{"success", []int{204}, 1, false},
		{"recovery", []int{500, 502, 503, 204}, 4, false},
		{"exhausted", []int{503}, 4, true},
		{"redirect", []int{307}, 1, true},
		{"client error", []int{400}, 1, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			received := make(chan Event, 8)
			var calls atomic.Int32
			receiver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost || r.URL.RequestURI() != "/audit?source=metrics" || r.Header.Get("Content-Type") != "application/json" {
					t.Errorf("unexpected audit request: %s %s, content type %q", r.Method, r.URL, r.Header.Get("Content-Type"))
				}
				var event Event
				if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
					t.Errorf("decode audit: %v", err)
				}
				select {
				case received <- event:
				default:
					t.Error("unexpected extra audit request")
				}
				w.Header().Set("Location", "/redirected")
				w.WriteHeader(test.statuses[int(calls.Add(1)-1)%len(test.statuses)])
			}))
			defer receiver.Close()
			remote, err := NewHTTPObserver(receiver.URL + "/audit?source=metrics")
			if err != nil {
				t.Fatal(err)
			}
			remote.retryPolicy = retry.NewPolicy(0, 0, 0)
			path := filepath.Join(t.TempDir(), "audit.jsonl")
			file, err := NewFileObserver(path)
			if err != nil {
				t.Fatal(err)
			}
			defer file.Close()
			results := make(chan error, 2)
			publisher := NewPublisher(observerFunc(func(ctx context.Context, event Event) error {
				err := remote.Notify(ctx, event)
				results <- err
				return err
			}), file)
			defer closePublisher(t, publisher)
			want := []Event{
				{Timestamp: 123, Metrics: []string{"Alloc", "Frees"}, IPAddress: "192.0.2.1"},
				{Timestamp: 124, Metrics: []string{"PollCount"}, IPAddress: "::1"},
			}
			var wantHTTP []Event
			for _, event := range want {
				if err := publisher.Publish(event); err != nil {
					t.Fatalf("publish error = %v", err)
				}
				for range test.attempts {
					wantHTTP = append(wantHTTP, event)
				}
			}
			closePublisher(t, publisher)
			for range want {
				if err := <-results; (err != nil) != test.wantErr {
					t.Fatalf("notify error = %v, want error = %v", err, test.wantErr)
				}
			}
			var gotHTTP []Event
			for range len(received) {
				gotHTTP = append(gotHTTP, <-received)
			}
			if !reflect.DeepEqual(gotHTTP, wantHTTP) {
				t.Fatalf("HTTP events = %+v, want %+v", gotHTTP, wantHTTP)
			}
			if got := readEvents(t, path); !reflect.DeepEqual(got, want) {
				t.Fatalf("file events = %+v, want %+v", got, want)
			}
		})
	}
}

func TestFileObserverConcurrentAppendAndReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.jsonl")
	for cycle := range 2 {
		observer, err := NewFileObserver(path)
		if err != nil {
			t.Fatal(err)
		}
		var group sync.WaitGroup
		for i := range 30 {
			group.Go(func() {
				event := Event{Timestamp: int64(cycle*30 + i), Metrics: []string{"PollCount"}, IPAddress: "127.0.0.1"}
				if err := observer.Notify(context.Background(), event); err != nil {
					t.Errorf("append event: %v", err)
				}
			})
		}
		group.Wait()
		if err := observer.Close(); err != nil {
			t.Fatal(err)
		}
	}
	events := readEvents(t, path)
	seen := make(map[int64]bool)
	for _, event := range events {
		seen[event.Timestamp] = true
	}
	if len(events) != 60 || len(seen) != 60 {
		t.Fatalf("events = %d, unique = %d, want 60 each", len(events), len(seen))
	}
}

func TestHTTPObserverTimeout(t *testing.T) {
	release := make(chan struct{})
	var calls atomic.Int32
	receiver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		<-release
	}))
	defer receiver.Close()
	defer close(release)
	observer, err := NewHTTPObserver(receiver.URL)
	if err != nil {
		t.Fatal(err)
	}
	observer.client.Timeout = 100 * time.Millisecond
	observer.retryPolicy = retry.NewPolicy(0, 0, 0)
	err = observer.Notify(context.Background(), Event{Metrics: []string{"Alloc"}})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v, want audit HTTP client timeout", err)
	}
	if got := calls.Load(); got != 4 {
		t.Fatalf("requests = %d, want 4", got)
	}
}

func TestHTTPObserverRejectsInvalidURL(t *testing.T) {
	for _, address := range []string{"", "/audit", "ftp://example.com", "http://", "://"} {
		if _, err := NewHTTPObserver(address); err == nil {
			t.Errorf("URL %q: expected error", address)
		}
	}
}

func TestFileFailureDoesNotStopHTTPObserver(t *testing.T) {
	received := make(chan Event, 1)
	receiver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var event Event
		if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
			t.Error(err)
		}
		received <- event
	}))
	defer receiver.Close()
	remote, err := NewHTTPObserver(receiver.URL)
	if err != nil {
		t.Fatal(err)
	}
	file, err := NewFileObserver(filepath.Join(t.TempDir(), "audit.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	event := Event{Timestamp: time.Now().Unix(), Metrics: []string{"Alloc"}, IPAddress: "127.0.0.1"}
	publisher := NewPublisher(file, remote)
	defer closePublisher(t, publisher)
	if err := publisher.Publish(event); err != nil {
		t.Fatal(err)
	}
	closePublisher(t, publisher)
	select {
	case got := <-received:
		if !reflect.DeepEqual(got, event) {
			t.Fatalf("event = %+v, want %+v", got, event)
		}
	default:
		t.Fatal("HTTP observer did not receive event after file failure")
	}
}

func readEvents(t *testing.T, path string) []Event {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	var events []Event
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var event Event
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			t.Fatalf("invalid JSON line %q: %v", scanner.Text(), err)
		}
		events = append(events, event)
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	return events
}
