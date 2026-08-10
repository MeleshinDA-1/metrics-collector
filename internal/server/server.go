package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/MeleshinDA-1/metrics-collector/internal/config"
	"github.com/MeleshinDA-1/metrics-collector/internal/handler"
	"github.com/MeleshinDA-1/metrics-collector/internal/handler/health"
	"github.com/MeleshinDA-1/metrics-collector/internal/handler/metrics"
	"github.com/MeleshinDA-1/metrics-collector/internal/repository"
)

func Run(serverConfig config.ServerConfig) error {
	ConfigureLogger(serverConfig)

	var pinger health.DBPinger
	if serverConfig.PostgresConnectionString != "" {
		pool, err := initDb(serverConfig.PostgresConnectionString)
		if err != nil {
			return fmt.Errorf("init database: %w", err)
		}
		defer pool.Close()

		pinger = pool
	}

	storage := repository.NewMemStorage()
	fileRepository := &repository.FileMetricsRepository{
		FilePath: serverConfig.FileStoragePath,
	}

	if serverConfig.Restore {
		if err := fileRepository.Restore(storage); err != nil {
			return fmt.Errorf("restore metrics: %w", err)
		}
	}

	metricsHandler := metrics.NewMetricsHandler(storage)
	var metricsEndpoints handler.MetricsEndpoints = metricsHandler
	if serverConfig.StoreInterval == 0 {
		metricsEndpoints = metrics.NewPersistingMetricsHandler(metricsHandler, fileRepository)
	}

	pingEndpoints := health.NewPingHandler(pinger)
	router := handler.NewRouter(metricsEndpoints, pingEndpoints)

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
