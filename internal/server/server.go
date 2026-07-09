package server

import (
	"net/http"

	"github.com/MeleshinDA-1/metrics-collector/internal/handler"
	"github.com/MeleshinDA-1/metrics-collector/internal/repository"
)

func Run(address string) error {
	storage := repository.NewMemStorage()
	metricsHandler := handler.NewMetricsHandler(storage)
	router := handler.NewRouter(metricsHandler)

	return http.ListenAndServe(address, router)
}
