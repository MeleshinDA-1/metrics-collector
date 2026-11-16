package agent

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/MeleshinDA-1/metrics-collector/internal/hash"
	"github.com/MeleshinDA-1/metrics-collector/internal/model"
	"github.com/MeleshinDA-1/metrics-collector/internal/retry"
)

func TestIsRetriableSendError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "transport error",
			err:  &transportError{Err: errors.New("connection refused")},
			want: true,
		},
		{
			name: "wrapped transport error",
			err:  fmt.Errorf("send batch: %w", &transportError{Err: errors.New("connection refused")}),
			want: true,
		},
		{
			name: "unexpected status",
			err:  fmt.Errorf("unexpected status: %d", http.StatusBadRequest),
			want: false,
		},
		{
			name: "plain error",
			err:  errors.New("boom"),
			want: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := isRetriableSendError(test.err); got != test.want {
				t.Fatalf("isRetriableSendError = %v, want %v", got, test.want)
			}
		})
	}
}

func TestPostRebuildsBodyOnEveryAttempt(t *testing.T) {
	var (
		mutex          sync.Mutex
		receivedBodies [][]byte
	)

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		body, err := io.ReadAll(request.Body)
		if err != nil {
			t.Errorf("read request body: %v", err)
			response.WriteHeader(http.StatusBadRequest)
			return
		}

		mutex.Lock()
		receivedBodies = append(receivedBodies, body)
		attempt := len(receivedBodies)
		mutex.Unlock()

		if attempt == 1 {
			panic(http.ErrAbortHandler)
		}

		response.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := newMetricsSender(server.URL, time.Second, 1, "", retry.NewPolicy(time.Millisecond))
	metricValue := 42.5
	batch := []model.Metrics{{ID: "Alloc", MType: model.Gauge, Value: &metricValue}}

	if err := sender.sendBatch(context.Background(), batch); err != nil {
		t.Fatalf("sendBatch returned error: %v", err)
	}

	mutex.Lock()
	defer mutex.Unlock()

	if len(receivedBodies) != 2 {
		t.Fatalf("requests count = %d, want 2", len(receivedBodies))
	}
	if !bytes.Equal(receivedBodies[0], receivedBodies[1]) {
		t.Fatalf("retry sent a different body: %d bytes on the first attempt, %d bytes on the second",
			len(receivedBodies[0]), len(receivedBodies[1]))
	}

	metrics, err := decodeGzipMetricsBatch(bytes.NewReader(receivedBodies[1]))
	if err != nil {
		t.Fatalf("read compressed batch of the retried request: %v", err)
	}
	if !reflect.DeepEqual(metrics, batch) {
		t.Fatalf("retried request carried %v, want %v", metrics, batch)
	}
}

func TestPostSignsBodyWhenKeyIsSet(t *testing.T) {
	tests := []struct {
		name       string
		signingKey string
		wantSigned bool
	}{
		{
			name:       "key is set",
			signingKey: "supersecret",
			wantSigned: true,
		},
		{
			name:       "key is empty",
			signingKey: "",
			wantSigned: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var (
				mutex             sync.Mutex
				receivedBody      []byte
				receivedSignature string
			)

			server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
				body, err := io.ReadAll(request.Body)
				if err != nil {
					t.Errorf("read request body: %v", err)
					response.WriteHeader(http.StatusBadRequest)
					return
				}

				mutex.Lock()
				receivedBody = body
				receivedSignature = request.Header.Get(hash.Header)
				mutex.Unlock()

				response.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			sender := newMetricsSender(server.URL, time.Second, 1, test.signingKey, retry.DefaultPolicy())
			metricValue := 42.5
			batch := []model.Metrics{{ID: "Alloc", MType: model.Gauge, Value: &metricValue}}

			if err := sender.sendBatch(context.Background(), batch); err != nil {
				t.Fatalf("sendBatch returned error: %v", err)
			}

			mutex.Lock()
			defer mutex.Unlock()

			if !test.wantSigned {
				if receivedSignature != "" {
					t.Fatalf("request was signed with an empty key: %q", receivedSignature)
				}
				return
			}

			if receivedSignature == "" {
				t.Fatalf("request has no %s header", hash.Header)
			}

			gzipReader, err := gzip.NewReader(bytes.NewReader(receivedBody))
			if err != nil {
				t.Fatalf("read compressed request body: %v", err)
			}
			defer gzipReader.Close()

			payload, err := io.ReadAll(gzipReader)
			if err != nil {
				t.Fatalf("decompress request body: %v", err)
			}

			if !hash.Equal(payload, test.signingKey, receivedSignature) {
				t.Fatalf("signature %q does not match the uncompressed request body", receivedSignature)
			}
		})
	}
}
