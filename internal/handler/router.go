package handler

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

func NewRouter(metricsHandler *MetricsHandler) http.Handler {
	router := mux.NewRouter()

	router.HandleFunc("/", metricsHandler.ListMetrics).Methods(http.MethodGet)
	router.HandleFunc("/update/{metricType}/{metricName}/{metricValue}", metricsHandler.UpdateMetrics).Methods(http.MethodPost)
	router.HandleFunc("/value/{metricType}/{metricName}", metricsHandler.ValueMetrics).Methods(http.MethodGet)

	return LoggingMiddleware(router)
}

type ResponseInterceptor struct {
	responseWriter http.ResponseWriter
	statusCode     int
	size           int
}

func NewResponseInterceptor(responseWriter http.ResponseWriter) *ResponseInterceptor {
	statusCode := http.StatusOK
	return &ResponseInterceptor{responseWriter: responseWriter, statusCode: statusCode, size: 0}
}

func (interceptor *ResponseInterceptor) Header() http.Header {
	return interceptor.responseWriter.Header()
}

func (interceptor *ResponseInterceptor) Write(arg []byte) (int, error) {
	size, err := interceptor.responseWriter.Write(arg)
	interceptor.size += size
	return size, err
}

func (interceptor *ResponseInterceptor) WriteHeader(statusCode int) {
	interceptor.statusCode = statusCode
	interceptor.responseWriter.WriteHeader(statusCode)
}

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		var requestID string
		var err error
		requestID = r.Header.Get("X-Request-ID")
		if requestID == "" {
			if requestID, err = GenerateID(16); err != nil {
				slog.Info("Unable to generate request ID", "error", err)
			}
		}

		interceptor := NewResponseInterceptor(w)
		next.ServeHTTP(interceptor, r)

		slog.Info("request completed",
			"method", r.Method,
			"requestUri", r.RequestURI,
			"requestID", requestID,
			"size", interceptor.size,
			"statusCode", interceptor.statusCode,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}

const maxIDBytes = 64

func GenerateID(byteLength int) (string, error) {
	if byteLength <= 0 {
		return "", fmt.Errorf("id length must be positive")
	}

	if byteLength > maxIDBytes {
		return "", fmt.Errorf("id length must not exceed %d bytes", maxIDBytes)
	}

	bytes := make([]byte, byteLength)

	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate random bytes: %w", err)
	}

	return hex.EncodeToString(bytes), nil
}
