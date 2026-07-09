package handler

import (
	"net/http"

	"github.com/gorilla/mux"
)

func NewRouter(metricsHandler *MetricsHandler) http.Handler {
	router := mux.NewRouter()
	router.HandleFunc("/", metricsHandler.ListMetrics).Methods(http.MethodGet)
	router.HandleFunc("/update/{metricType}/{metricName}/{metricValue}", metricsHandler.UpdateMetrics).Methods(http.MethodPost)
	router.HandleFunc("/value/{metricType}/{metricName}", metricsHandler.ValueMetrics).Methods(http.MethodGet)

	return router
}
