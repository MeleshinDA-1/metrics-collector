package audit

import (
	"context"
	"crypto/x509"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/MeleshinDA-1/metrics-collector/internal/retry"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestHTTPObserverTransportErrors(t *testing.T) {
	for _, test := range []struct {
		name      string
		err       error
		wantRetry bool
	}{
		{"connection refused", &net.OpError{Op: "dial", Net: "tcp", Err: syscall.ECONNREFUSED}, true},
		{"connection reset", &net.OpError{Op: "read", Net: "tcp", Err: syscall.ECONNRESET}, true},
		{"temporary DNS failure", &net.DNSError{Err: "server failure", IsTemporary: true}, true},
		{"unknown host", &net.DNSError{Err: "no such host", IsNotFound: true}, false},
		{"EOF", io.EOF, true},
		{"unexpected EOF", io.ErrUnexpectedEOF, true},
		{"certificate", x509.UnknownAuthorityError{}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			observer, err := NewHTTPObserver("https://example.com/audit")
			if err != nil {
				t.Fatal(err)
			}
			observer.retryPolicy = retry.NewPolicy(0, 0, 0)
			calls := 0
			observer.client.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
				defer req.Body.Close()
				calls++
				if calls == 1 {
					return nil, test.err
				}
				return &http.Response{StatusCode: http.StatusNoContent, Body: http.NoBody}, nil
			})
			err = observer.Notify(context.Background(), Event{Metrics: []string{"Alloc"}})
			wantCalls := 1
			if test.wantRetry {
				wantCalls = 2
			}
			if calls != wantCalls || (err == nil) != test.wantRetry {
				t.Fatalf("calls = %d, error = %v; want calls = %d, success = %v", calls, err, wantCalls, test.wantRetry)
			}
			if !test.wantRetry && !errors.Is(err, test.err) {
				t.Fatalf("error = %v, want %v", err, test.err)
			}
		})
	}
}

func TestHTTPObserverContext(t *testing.T) {
	for _, canceled := range []bool{false, true} {
		name := "deadline while waiting to retry"
		if canceled {
			name = "canceled before sending"
		}
		t.Run(name, func(t *testing.T) {
			observer, err := NewHTTPObserver("http://example.com/audit")
			if err != nil {
				t.Fatal(err)
			}
			observer.retryPolicy = retry.NewPolicy(time.Hour)
			calls := 0
			observer.client.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
				defer req.Body.Close()
				calls++
				return &http.Response{StatusCode: http.StatusServiceUnavailable, Body: http.NoBody}, nil
			})
			ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
			defer cancel()
			wantCalls := 1
			if canceled {
				cancel()
				wantCalls = 0
			}
			err = observer.Notify(ctx, Event{})
			if ctx.Err() == nil || !errors.Is(err, ctx.Err()) || calls != wantCalls {
				t.Fatalf("calls = %d, error = %v; want calls = %d, error = %v", calls, err, wantCalls, ctx.Err())
			}
			if !canceled && !strings.Contains(err.Error(), "503") {
				t.Fatalf("error = %v, want the failed HTTP status as well as the deadline", err)
			}
		})
	}
}
