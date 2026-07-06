package main

import (
	"net/http"

	"github.com/MeleshinDA-1/golang-practicum-alice/internal/handler"
	"github.com/MeleshinDA-1/golang-practicum-alice/internal/repository"
	"github.com/gorilla/mux"
)

func main() {
	startCollectorServer()
}

func startCollectorServer() {
	storage := repository.NewMemStorage()
	metricsHandler := handler.NewMetricsHandler(storage)

	router := mux.NewRouter()
	router.HandleFunc("/", metricsHandler.ListMetrics).Methods(http.MethodGet)
	router.HandleFunc("/update/{metricType}/{metricName}/{metricValue}", metricsHandler.UpdateMetrics).Methods(http.MethodPost)
	router.HandleFunc("/value/{metricType}/{metricName}", metricsHandler.ValueMetrics).Methods(http.MethodGet)

	err := http.ListenAndServe(":8080", router)
	if err != nil {
		panic(err)
	}
}
