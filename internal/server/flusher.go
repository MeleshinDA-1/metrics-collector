package server

import (
	"context"
	"log/slog"
	"time"

	"github.com/MeleshinDA-1/metrics-collector/internal/config"
	"github.com/MeleshinDA-1/metrics-collector/internal/repository"
)

type Flusher struct {
	storage           repository.MetricsSnapshotProvider
	metricsRepository repository.MetricsRepository
}

func NewFlusher(
	storage repository.MetricsSnapshotProvider,
	metricsRepository repository.MetricsRepository,
) *Flusher {
	if metricsRepository == nil {
		panic("metrics repository is required")
	}

	f := &Flusher{
		storage:           storage,
		metricsRepository: metricsRepository,
	}
	return f
}

func (f *Flusher) Start(
	ctx context.Context,
	serverConfig config.ServerConfig,
) error {
	if serverConfig.StoreInterval <= 0 {
		return nil
	}

	ticker := time.NewTicker(time.Duration(serverConfig.StoreInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := f.metricsRepository.Flush(f.storage); err != nil {
				slog.Error("unable to save metrics", "error", err)
			}

		case <-ctx.Done():
			return nil
		}
	}
}
