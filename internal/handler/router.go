package handler

import (
	"net/http"

	"github.com/gorilla/mux"
)

func NewRouter(metricsHandler *MetricsHandler) http.Handler {
	router := mux.NewRouter()

	router.HandleFunc("/", metricsHandler.ListMetrics).Methods(http.MethodGet) // localhost:8080/

	router.HandleFunc("/update", metricsHandler.UpdateMetricsJson).Methods(http.MethodPost)
	router.HandleFunc("/update/", metricsHandler.UpdateMetricsJson).Methods(http.MethodPost)
	router.HandleFunc("/value", metricsHandler.ValueMetricsJson).Methods(http.MethodPost)
	router.HandleFunc("/value/", metricsHandler.ValueMetricsJson).Methods(http.MethodPost)

	router.HandleFunc("/update/{metricType}/{metricName}/{metricValue}", metricsHandler.UpdateMetrics).Methods(http.MethodPost)
	router.HandleFunc("/value/{metricType}/{metricName}", metricsHandler.ValueMetrics).Methods(http.MethodGet)

	return LoggingMiddleware(CompressingMiddleware(router))
}
