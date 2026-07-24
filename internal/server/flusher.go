package server

import (
	"context"
	"log/slog"
	"time"

	"github.com/MeleshinDA-1/metrics-collector/internal/config"
	"github.com/MeleshinDA-1/metrics-collector/internal/repository"
)

type Flusher struct {
	memStorage        *repository.MemStorage
	metricsRepository repository.MetricsRepository
}

func NewFlusher(storage *repository.MemStorage, metricsRepository repository.MetricsRepository) *Flusher {
	f := &Flusher{
		memStorage:        storage,
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
			if err := f.metricsRepository.Flush(f.memStorage); err != nil {
				slog.Error("unable to save metrics", "error", err)
			}

		case <-ctx.Done():
			return nil
		}
	}
}
