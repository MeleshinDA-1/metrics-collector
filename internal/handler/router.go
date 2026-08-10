package handler

import (
	"net/http"

	"github.com/MeleshinDA-1/metrics-collector/internal/handler/middleware"
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

	registerMetricsRoutes(router, metricsEndpoints)

	return middleware.LoggingMiddleware(middleware.CompressingMiddleware(router))
}

func registerMetricsRoutes(r *mux.Router, metricsEndpoints MetricsEndpoints) {
	r.HandleFunc("/", metricsEndpoints.ListMetrics).Methods(http.MethodGet)

	r.HandleFunc("/update", metricsEndpoints.UpdateMetricsJson).Methods(http.MethodPost)
	r.HandleFunc("/update/", metricsEndpoints.UpdateMetricsJson).Methods(http.MethodPost)
	r.HandleFunc("/value", metricsEndpoints.ValueMetricsJson).Methods(http.MethodPost)
	r.HandleFunc("/value/", metricsEndpoints.ValueMetricsJson).Methods(http.MethodPost)

	r.HandleFunc("/update/{metricType}/{metricName}/{metricValue}", metricsEndpoints.UpdateMetrics).Methods(http.MethodPost)
	r.HandleFunc("/value/{metricType}/{metricName}", metricsEndpoints.ValueMetrics).Methods(http.MethodGet)
}
