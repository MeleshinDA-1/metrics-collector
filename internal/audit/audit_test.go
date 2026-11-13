package audit

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestPublisherContinuesAfterHTTPFailure(t *testing.T) {
	for _, status := range []int{http.StatusNoContent, http.StatusInternalServerError, http.StatusTemporaryRedirect} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			received := make(chan Event, 4)
			receiver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost || r.URL.RequestURI() != "/audit?source=metrics" || r.Header.Get("Content-Type") != "application/json" {
					t.Errorf("unexpected audit request: %s %s, content type %q", r.Method, r.URL, r.Header.Get("Content-Type"))
				}
				var event Event
				if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
					t.Errorf("decode audit: %v", err)
				}
				received <- event
				w.Header().Set("Location", "/redirected")
				w.WriteHeader(status)
			}))
			defer receiver.Close()
			remote, err := NewHTTPObserver(receiver.URL + "/audit?source=metrics")
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(t.TempDir(), "audit.jsonl")
			file, err := NewFileObserver(path)
			if err != nil {
				t.Fatal(err)
			}
			defer file.Close()
			publisher := NewPublisher(remote, file)
			want := []Event{
				{Timestamp: 123, Metrics: []string{"Alloc", "Frees"}, IPAddress: "192.0.2.1"},
				{Timestamp: 124, Metrics: []string{"PollCount"}, IPAddress: "::1"},
			}
			for _, event := range want {
				err := publisher.Publish(context.Background(), event)
				if (err != nil) != (status != http.StatusNoContent) {
					t.Fatalf("receiver status %d: publish error = %v", status, err)
				}
			}
			var gotHTTP []Event
			for range len(received) {
				gotHTTP = append(gotHTTP, <-received)
			}
			if !reflect.DeepEqual(gotHTTP, want) {
				t.Fatalf("HTTP events = %+v, want %+v", gotHTTP, want)
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
	receiver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-release
	}))
	defer receiver.Close()
	defer close(release)
	observer, err := NewHTTPObserver(receiver.URL)
	if err != nil {
		t.Fatal(err)
	}
	err = observer.Notify(context.Background(), Event{Metrics: []string{"Alloc"}})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v, want audit HTTP client timeout", err)
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
	if err := NewPublisher(file, remote).Publish(context.Background(), event); err == nil {
		t.Fatal("expected closed file error")
	}
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
