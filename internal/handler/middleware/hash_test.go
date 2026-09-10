package middleware

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MeleshinDA-1/metrics-collector/internal/hash"
)

const signingTestKey = "supersecret"

func echoHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	})
}

func TestSigningMiddlewareRequestSignature(t *testing.T) {
	body := []byte(`[{"id":"Alloc","type":"gauge","value":42}]`)

	tests := []struct {
		name           string
		signature      string
		setSignature   bool
		wantStatusCode int
	}{
		{
			name:           "valid signature",
			signature:      hash.Sign(body, signingTestKey),
			setSignature:   true,
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "signature of other data",
			signature:      hash.Sign([]byte("other body"), signingTestKey),
			setSignature:   true,
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "signature made with another key",
			signature:      hash.Sign(body, "another key"),
			setSignature:   true,
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "no signature at all",
			setSignature:   false,
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "empty signature",
			signature:      "",
			setSignature:   true,
			wantStatusCode: http.StatusOK,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/updates/", bytes.NewReader(body))
			if test.setSignature {
				request.Header.Set(hash.Header, test.signature)
			}
			response := httptest.NewRecorder()

			SigningMiddleware(signingTestKey)(echoHandler()).ServeHTTP(response, request)

			if response.Code != test.wantStatusCode {
				t.Fatalf("status code = %d, want %d", response.Code, test.wantStatusCode)
			}

			signature := response.Header().Get(hash.Header)
			if !hash.Equal(response.Body.Bytes(), signingTestKey, signature) {
				t.Fatalf("response signature %q does not match the response body %q",
					signature, response.Body.Bytes())
			}
		})
	}
}

func TestSigningMiddlewareSignsResponse(t *testing.T) {
	body := []byte(`[{"id":"Alloc","type":"gauge","value":42}]`)

	tests := []struct {
		name        string
		signRequest bool
	}{
		{
			name:        "signed request",
			signRequest: true,
		},
		{
			name:        "unsigned request",
			signRequest: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/updates/", bytes.NewReader(body))
			if test.signRequest {
				request.Header.Set(hash.Header, hash.Sign(body, signingTestKey))
			}
			response := httptest.NewRecorder()

			SigningMiddleware(signingTestKey)(echoHandler()).ServeHTTP(response, request)

			if response.Code != http.StatusOK {
				t.Fatalf("status code = %d, want %d", response.Code, http.StatusOK)
			}
			if !bytes.Equal(response.Body.Bytes(), body) {
				t.Fatalf("response body = %q, want %q", response.Body.Bytes(), body)
			}

			signature := response.Header().Get(hash.Header)
			if signature == "" {
				t.Fatalf("response has no %s header", hash.Header)
			}
			if !hash.Equal(response.Body.Bytes(), signingTestKey, signature) {
				t.Fatalf("response signature %q does not match the response body", signature)
			}
		})
	}
}

func TestSigningMiddlewareKeepsHandlerStatusCode(t *testing.T) {
	for _, wantStatusCode := range []int{http.StatusNotFound, http.StatusInternalServerError} {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(wantStatusCode)
			_, _ = w.Write([]byte("metric not found"))
		})

		request := httptest.NewRequest(http.MethodGet, "/value/gauge/Unknown", nil)
		response := httptest.NewRecorder()

		SigningMiddleware(signingTestKey)(handler).ServeHTTP(response, request)

		if response.Code != wantStatusCode {
			t.Fatalf("status code = %d, want %d", response.Code, wantStatusCode)
		}
		if body := response.Body.String(); body != "metric not found" {
			t.Fatalf("response body = %q, want %q", body, "metric not found")
		}

		signature := response.Header().Get(hash.Header)
		if !hash.Equal(response.Body.Bytes(), signingTestKey, signature) {
			t.Fatalf("response signature %q does not match the response body", signature)
		}
	}
}

func TestSigningMiddlewareWithoutKeyIsDisabled(t *testing.T) {
	body := []byte(`[{"id":"Alloc","type":"gauge","value":42}]`)

	request := httptest.NewRequest(http.MethodPost, "/updates/", bytes.NewReader(body))
	request.Header.Set(hash.Header, "deadbeef")
	response := httptest.NewRecorder()

	SigningMiddleware("")(echoHandler()).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", response.Code, http.StatusOK)
	}
	if signature := response.Header().Get(hash.Header); signature != "" {
		t.Fatalf("response signed with an empty key: %q", signature)
	}
}

func TestSigningMiddlewareSignsUncompressedBody(t *testing.T) {
	metrics := []byte(`[{"id":"Alloc","type":"gauge","value":42}]`)

	var compressed bytes.Buffer
	gzipWriter := gzip.NewWriter(&compressed)
	if _, err := gzipWriter.Write(metrics); err != nil {
		t.Fatalf("compress request body: %v", err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatalf("close gzip writer: %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, "/updates/", bytes.NewReader(compressed.Bytes()))
	request.Header.Set("Content-Encoding", "gzip")
	request.Header.Set("Accept-Encoding", "gzip")
	request.Header.Set(hash.Header, hash.Sign(metrics, signingTestKey))
	response := httptest.NewRecorder()

	CompressingMiddleware(SigningMiddleware(signingTestKey)(echoHandler())).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", response.Code, http.StatusOK)
	}
	if encoding := response.Header().Get("Content-Encoding"); encoding != "gzip" {
		t.Fatalf("content encoding = %q, want %q", encoding, "gzip")
	}

	gzipReader, err := gzip.NewReader(bytes.NewReader(response.Body.Bytes()))
	if err != nil {
		t.Fatalf("read compressed response: %v", err)
	}
	defer gzipReader.Close()

	decompressed, err := io.ReadAll(gzipReader)
	if err != nil {
		t.Fatalf("decompress response: %v", err)
	}
	if !bytes.Equal(decompressed, metrics) {
		t.Fatalf("response body = %q, want %q", decompressed, metrics)
	}

	signature := response.Header().Get(hash.Header)
	if !hash.Equal(decompressed, signingTestKey, signature) {
		t.Fatalf("response signature %q does not match the uncompressed response body", signature)
	}
}
