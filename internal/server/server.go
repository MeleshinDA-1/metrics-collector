package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/MeleshinDA-1/metrics-collector/internal/config"
	"github.com/MeleshinDA-1/metrics-collector/internal/handler"
	"github.com/MeleshinDA-1/metrics-collector/internal/repository"
)

func Run(serverConfig config.ServerConfig) error {
	ConfigureLogger()

	storage := repository.NewMemStorage()
	fileRepository := &repository.FileMetricsRepository{
		FilePath: serverConfig.FileStoragePath,
	}

	if serverConfig.Restore {
		if err := fileRepository.Restore(storage); err != nil {
			return fmt.Errorf("restore metrics: %w", err)
		}
	}

	if serverConfig.StoreInterval == 0 {
		storage.MetricsRepository = fileRepository
	}

	metricsHandler := handler.NewMetricsHandler(storage)
	router := handler.NewRouter(metricsHandler)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if serverConfig.StoreInterval > 0 {
		flusher := NewFlusher(storage, fileRepository)
		go func() {
			if err := flusher.Start(ctx, serverConfig); err != nil {
				slog.Error("unable to save metrics", "error", err)
			}
		}()
	}

	return http.ListenAndServe(serverConfig.Address, router)
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
