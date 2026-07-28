package handler

import (
	"net/http"

	"github.com/gorilla/mux"
)

type MetricsEndpoints interface {
	ListMetrics(http.ResponseWriter, *http.Request)
	UpdateMetrics(http.ResponseWriter, *http.Request)
	UpdateMetricsJson(http.ResponseWriter, *http.Request)
	ValueMetrics(http.ResponseWriter, *http.Request)
	ValueMetricsJson(http.ResponseWriter, *http.Request)
}

func NewRouter(metricsEndpoints MetricsEndpoints) http.Handler {
	router := mux.NewRouter()

	router.HandleFunc("/", metricsEndpoints.ListMetrics).Methods(http.MethodGet) // localhost:8080/

	router.HandleFunc("/update", metricsEndpoints.UpdateMetricsJson).Methods(http.MethodPost)
	router.HandleFunc("/update/", metricsEndpoints.UpdateMetricsJson).Methods(http.MethodPost)
	router.HandleFunc("/value", metricsEndpoints.ValueMetricsJson).Methods(http.MethodPost)
	router.HandleFunc("/value/", metricsEndpoints.ValueMetricsJson).Methods(http.MethodPost)

	router.HandleFunc("/update/{metricType}/{metricName}/{metricValue}", metricsEndpoints.UpdateMetrics).Methods(http.MethodPost)
	router.HandleFunc("/value/{metricType}/{metricName}", metricsEndpoints.ValueMetrics).Methods(http.MethodGet)

	return LoggingMiddleware(CompressingMiddleware(router))
}
