package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/MeleshinDA-1/metrics-collector/internal/config"
	"github.com/MeleshinDA-1/metrics-collector/internal/handler"
	"github.com/MeleshinDA-1/metrics-collector/internal/handler/health"
	"github.com/MeleshinDA-1/metrics-collector/internal/handler/metrics"
	"github.com/MeleshinDA-1/metrics-collector/internal/repository"
	"github.com/MeleshinDA-1/metrics-collector/internal/retry"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func Run(serverConfig config.ServerConfig) error {
	ConfigureLogger(serverConfig)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var pinger health.DBPinger
	var metricsEndpoints handler.MetricsEndpoints

	switch {
	case serverConfig.PostgresConnectionString != "":
		if err := runMigrations(
			serverConfig.PostgresConnectionString,
		); err != nil {
			return fmt.Errorf("run database migrations: %w", err)
		}

		pool, err := initDB(
			serverConfig.PostgresConnectionString,
		)
		if err != nil {
			return fmt.Errorf("init database: %w", err)
		}
		defer pool.Close()

		pinger = pool
		metricsEndpoints = metrics.NewMetricsHandler(
			repository.NewDBMetricStorage(pool, retry.DefaultPolicy()),
		)

	default:
		storage := repository.NewMemStorage()
		metricsHandler := metrics.NewMetricsHandler(storage)
		metricsEndpoints = metricsHandler

		if serverConfig.FileStoragePath != "" {
			fileRepository := &repository.FileMetricsRepository{
				FilePath: serverConfig.FileStoragePath,
			}

			if serverConfig.Restore {
				if err := fileRepository.Restore(ctx, storage); err != nil {
					return fmt.Errorf("restore metrics: %w", err)
				}
			}

			if serverConfig.StoreInterval == 0 {
				metricsEndpoints = metrics.NewPersistingMetricsHandler(metricsHandler, fileRepository)
			} else {
				flusher := NewFlusher(storage, fileRepository)
				go func() {
					if err := flusher.Start(ctx, serverConfig); err != nil {
						slog.Error("unable to save metrics", "error", err)
					}
				}()
			}
		}
	}

	router := handler.NewRouter(metricsEndpoints, health.NewPingHandler(pinger))

	return http.ListenAndServe(serverConfig.Address, router)
}

func runMigrations(connectionString string) error {
	databaseURL, err := url.Parse(connectionString)
	if err != nil {
		return fmt.Errorf("parse database URL: %w", err)
	}

	databaseURL.Scheme = "pgx5"

	migration, err := migrate.New(
		"file://migrations",
		databaseURL.String(),
	)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}
	defer migration.Close()

	if err := migration.Up(); err != nil &&
		!errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("apply migrations: %w", err)
	}

	return nil
}
