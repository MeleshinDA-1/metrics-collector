package server

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/MeleshinDA-1/metrics-collector/internal/handler"
	"github.com/MeleshinDA-1/metrics-collector/internal/repository"
)

func Run(address string) error {
	ConfigureLogger()

	storage := repository.NewMemStorage()
	metricsHandler := handler.NewMetricsHandler(storage)
	router := handler.NewRouter(metricsHandler)

	return http.ListenAndServe(address, router)
}

func ConfigureLogger() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil)).With(
		"service", "metrics-collector",
		"env", "prod",
		"region", "ru-central1",
	)
	slog.SetDefault(logger)

	logger.Info("started", "port", 8080)
}
