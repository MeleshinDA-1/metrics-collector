package main

import (
	"net/http"

	"github.com/MeleshinDA-1/metrics-collector/internal/config"
	"github.com/MeleshinDA-1/metrics-collector/internal/handler"
	"github.com/MeleshinDA-1/metrics-collector/internal/repository"
	"github.com/gorilla/mux"
)

func main() {
	serverConfig := config.MustParseServerConfig()

	startCollectorServer(serverConfig.Address)
}

func startCollectorServer(address string) {
	storage := repository.NewMemStorage()
	metricsHandler := handler.NewMetricsHandler(storage)

	router := mux.NewRouter()
	router.HandleFunc("/", metricsHandler.ListMetrics).Methods(http.MethodGet)
	router.HandleFunc("/update/{metricType}/{metricName}/{metricValue}", metricsHandler.UpdateMetrics).Methods(http.MethodPost)
	router.HandleFunc("/value/{metricType}/{metricName}", metricsHandler.ValueMetrics).Methods(http.MethodGet)

	err := http.ListenAndServe(address, router)
	if err != nil {
		panic(err)
	}
}
