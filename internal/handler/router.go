package handler

import (
	"net/http"

	"github.com/MeleshinDA-1/metrics-collector/internal/handler/middleware"
	"github.com/gorilla/mux"
)

type MetricsEndpoints interface {
	ListMetrics(http.ResponseWriter, *http.Request)
	UpdateMetrics(http.ResponseWriter, *http.Request)
	UpdateMetricsJSON(http.ResponseWriter, *http.Request)
	UpdateBatchMetricsJSON(http.ResponseWriter, *http.Request)
	ValueMetrics(http.ResponseWriter, *http.Request)
	ValueMetricsJSON(http.ResponseWriter, *http.Request)
}

type PingEndpoints interface {
	PingDB(http.ResponseWriter, *http.Request)
}

func NewRouter(
	metricsEndpoints MetricsEndpoints,
	pingEndpoints PingEndpoints,
	signingKey string,
) http.Handler {
	router := mux.NewRouter()

	registerMetricsRoutes(router, metricsEndpoints)
	registerPingRoutes(router, pingEndpoints)

	return middleware.LoggingMiddleware(
		middleware.CompressingMiddleware(
			middleware.SigningMiddleware(signingKey)(router),
		),
	)
}

func registerMetricsRoutes(r *mux.Router, metricsEndpoints MetricsEndpoints) {
	r.HandleFunc("/", metricsEndpoints.ListMetrics).Methods(http.MethodGet)

	r.HandleFunc("/update", metricsEndpoints.UpdateMetricsJSON).Methods(http.MethodPost)
	r.HandleFunc("/update/", metricsEndpoints.UpdateMetricsJSON).Methods(http.MethodPost)
	r.HandleFunc("/updates/", metricsEndpoints.UpdateBatchMetricsJSON).Methods(http.MethodPost)
	r.HandleFunc("/value", metricsEndpoints.ValueMetricsJSON).Methods(http.MethodPost)
	r.HandleFunc("/value/", metricsEndpoints.ValueMetricsJSON).Methods(http.MethodPost)

	r.HandleFunc("/update/{metricType}/{metricName}/{metricValue}", metricsEndpoints.UpdateMetrics).Methods(http.MethodPost)
	r.HandleFunc("/value/{metricType}/{metricName}", metricsEndpoints.ValueMetrics).Methods(http.MethodGet)
}

func registerPingRoutes(r *mux.Router, pingEndpoints PingEndpoints) {
	r.HandleFunc("/ping", pingEndpoints.PingDB).Methods(http.MethodGet)
	r.HandleFunc("/ping/", pingEndpoints.PingDB).Methods(http.MethodGet)
}
