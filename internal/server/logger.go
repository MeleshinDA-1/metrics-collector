package server

import (
	"log/slog"
	"os"

	"github.com/MeleshinDA-1/metrics-collector/internal/config"
)

func ConfigureLogger(config config.ServerConfig) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil)).With(
		"service", "metrics-collector",
		"env", "prod",
		"region", "ru-central1",
	)
	slog.SetDefault(logger)

	logger.Info("started", "address", config.Address)
}
