package audit

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type observerFunc func(context.Context, Event) error

func (f observerFunc) Notify(ctx context.Context, event Event) error {
	return f(ctx, event)
}

func closePublisher(t *testing.T, p *Publisher) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := p.Close(ctx); err != nil {
		t.Errorf("close publisher: %v", err)
	}
}

func receiveEvent(t *testing.T, events <-chan Event) Event {
	t.Helper()
	select {
	case event := <-events:
		return event
	case <-time.After(3 * time.Second):
		t.Fatal("observer did not receive event")
		return Event{}
	}
}

func TestPublisherQueuesAreIndependent(t *testing.T) {
	slowEvents := make(chan Event, observerQueueSize+1)
	fastEvents := make(chan Event, 1)
	release := make(chan struct{}, observerQueueSize+1)
	slow := observerFunc(func(ctx context.Context, event Event) error {
		slowEvents <- event
		select {
		case <-release:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})
	fast := observerFunc(func(_ context.Context, event Event) error {
		fastEvents <- event
		return nil
	})
	p := NewPublisher(slow, fast)
	defer closePublisher(t, p)
	publish := func(event Event, wantErr bool) {
		t.Helper()
		result := make(chan error, 1)
		go func() { result <- p.Publish(event) }()
		select {
		case err := <-result:
			if (err != nil) != wantErr {
				t.Fatalf("publish %d: error = %v, want error = %v", event.Timestamp, err, wantErr)
			}
		case <-time.After(3 * time.Second):
			t.Fatal("Publish blocked on a slow observer")
		}
		if got := receiveEvent(t, fastEvents); got.Timestamp != event.Timestamp {
			t.Fatalf("fast event = %d, want %d", got.Timestamp, event.Timestamp)
		}
	}
	for cycle := range 2 {
		first := int64(cycle * (observerQueueSize + 2))
		event := Event{Timestamp: first, Metrics: []string{"Alloc"}}
		publish(event, false)
		event.Metrics[0] = "changed after Publish"
		if got := receiveEvent(t, slowEvents); got.Timestamp != first || got.Metrics[0] != "Alloc" {
			t.Fatalf("first slow event = %+v, want timestamp %d and Alloc", got, first)
		}
		for i := 1; i <= observerQueueSize; i++ {
			publish(Event{Timestamp: first + int64(i)}, false)
		}
		publish(Event{Timestamp: first + observerQueueSize + 1}, true)
		select {
		case event := <-slowEvents:
			t.Fatalf("concurrent notification of slow observer: %+v", event)
		default:
		}
		for range observerQueueSize + 1 {
			release <- struct{}{}
		}
		for i := 1; i <= observerQueueSize; i++ {
			if got := receiveEvent(t, slowEvents); got.Timestamp != first+int64(i) {
				t.Fatalf("slow event = %d, want %d", got.Timestamp, first+int64(i))
			}
		}
	}
	closePublisher(t, p)
}

func TestPublisherCloseCancelsHTTP(t *testing.T) {
	started := make(chan struct{}, 1)
	canceled := make(chan struct{}, 1)
	receiver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		started <- struct{}{}
		<-r.Context().Done()
		canceled <- struct{}{}
	}))
	defer receiver.Close()
	remote, err := NewHTTPObserver(receiver.URL)
	if err != nil {
		t.Fatal(err)
	}
	p := NewPublisher(remote)
	defer closePublisher(t, p)
	if err := p.Publish(Event{}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("audit request did not start")
	}
	if err := p.Publish(Event{Timestamp: 1}); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	if err := p.Close(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("close error = %v, want deadline exceeded", err)
	}
	select {
	case <-canceled:
	case <-time.After(3 * time.Second):
		t.Fatal("HTTP request was not canceled on shutdown")
	}
	if err := p.Publish(Event{}); err == nil {
		t.Fatal("closed publisher accepted an event")
	}
	select {
	case <-started:
		t.Fatal("queued event was sent after shutdown deadline")
	default:
	}
}

func TestPublisherConcurrentPublishAndClose(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.jsonl")
	file, err := NewFileObserver(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	p := NewPublisher(file)
	defer closePublisher(t, p)
	var accepted atomic.Int32
	if err := p.Publish(Event{Timestamp: -1}); err != nil {
		t.Fatal(err)
	}
	accepted.Store(1)
	var group sync.WaitGroup
	start := make(chan struct{})
	for worker := range 8 {
		group.Go(func() {
			<-start
			for i := range 64 {
				if err := p.Publish(Event{Timestamp: int64(worker*64 + i)}); err == nil {
					accepted.Add(1)
				}
			}
		})
	}
	for range 2 {
		group.Go(func() {
			<-start
			closePublisher(t, p)
		})
	}
	close(start)
	group.Wait()
	if got := len(readEvents(t, path)); got != int(accepted.Load()) {
		t.Fatalf("file events = %d, accepted = %d", got, accepted.Load())
	}
}
