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

		next.ServeHTTP(w, r)

		slog.Info("request completed",
			"method", r.Method,
			"path", r.URL.Path,
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
