package middleware

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"

	"github.com/MeleshinDA-1/metrics-collector/internal/hash"
)

type signingResponseWriter struct {
	http.ResponseWriter
	signingKey string
	statusCode int
	body       bytes.Buffer
}

func (w *signingResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
}

func (w *signingResponseWriter) Write(data []byte) (int, error) {
	return w.body.Write(data)
}

func (w *signingResponseWriter) flush() {
	w.Header().Set(hash.Header, hash.Sign(w.body.Bytes(), w.signingKey))
	w.ResponseWriter.WriteHeader(w.statusCode)

	if _, err := w.ResponseWriter.Write(w.body.Bytes()); err != nil {
		slog.Error("unable to write signed response", "error", err)
	}
}

func SigningMiddleware(signingKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if signingKey == "" {
			return next
		}

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			signingWriter := &signingResponseWriter{
				ResponseWriter: w,
				signingKey:     signingKey,
				statusCode:     http.StatusOK,
			}

			serveVerified(signingWriter, r, next, signingKey)

			signingWriter.flush()
		})
	}
}

func serveVerified(w http.ResponseWriter, r *http.Request, next http.Handler, signingKey string) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("unable to read request body", "error", err)
		http.Error(w, "unable to read request body", http.StatusBadRequest)
		return
	}
	r.Body = io.NopCloser(bytes.NewReader(body))

	signature := r.Header.Get(hash.Header)
	if signature != "" && !hash.Equal(body, signingKey, signature) {
		http.Error(w, "request signature mismatch", http.StatusBadRequest)
		return
	}

	next.ServeHTTP(w, r)
}
