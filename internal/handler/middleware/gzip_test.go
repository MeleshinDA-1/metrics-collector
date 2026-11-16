package middleware

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"runtime"
	"sync"
	"testing"

	"github.com/MeleshinDA-1/metrics-collector/internal/hash"
)

func TestGzipRepeatedConcurrentResponses(t *testing.T) {
	handler := CompressingMiddleware(SigningMiddleware(signingTestKey)(echoHandler()))
	var group sync.WaitGroup
	for worker := range 8 {
		group.Go(func() {
			for cycle := range 6 {
				payload := bytes.Repeat([]byte(fmt.Sprintf("worker-%d-cycle-%d;", worker, cycle)), 1<<(cycle*3))
				request := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(payload))
				request.Header.Set(hash.Header, hash.Sign(payload, signingTestKey))
				if cycle%2 == 0 {
					request.Header.Set("Accept-Encoding", "gzip")
				}
				response := httptest.NewRecorder()
				handler.ServeHTTP(response, request)
				got := response.Body.Bytes()
				if cycle%2 == 0 {
					if response.Header().Get("Content-Encoding") != "gzip" {
						t.Error("missing gzip header")
						return
					}
					reader, err := gzip.NewReader(response.Body)
					if err != nil {
						t.Error(err)
						return
					}
					got, err = io.ReadAll(reader)
					reader.Close()
					if err != nil {
						t.Error(err)
						return
					}
				} else if response.Header().Get("Content-Encoding") != "" {
					t.Error("plain response inherited gzip header")
				}
				if response.Code != http.StatusOK || !bytes.Equal(got, payload) || !hash.Equal(got, signingTestKey, response.Header().Get(hash.Header)) {
					t.Errorf("worker %d cycle %d: status=%d, body/signature mismatch", worker, cycle, response.Code)
				}
				if cycle == 3 {
					runtime.GC()
				}
			}
		})
	}
	group.Wait()
}

type failedResponseWriter struct{ header http.Header }

func (w failedResponseWriter) Header() http.Header     { return w.header }
func (failedResponseWriter) WriteHeader(int)           {}
func (failedResponseWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func TestGzipResponseAfterWriteFailureAndEmptyBody(t *testing.T) {
	handler := CompressingMiddleware(echoHandler())
	request := func(body string) *http.Request {
		r := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))
		r.Header.Set("Accept-Encoding", "gzip")
		return r
	}
	handler.ServeHTTP(failedResponseWriter{header: make(http.Header)}, request("failed"))
	for _, payload := range []string{"", "after empty", "next request"} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request(payload))
		reader, err := gzip.NewReader(response.Body)
		if err != nil {
			t.Fatal(err)
		}
		got, err := io.ReadAll(reader)
		reader.Close()
		if err != nil || string(got) != payload {
			t.Fatalf("body = %q, error = %v, want %q", got, err, payload)
		}
	}
}
