package server

import (
	"context"
	"log/slog"
	"time"

	"github.com/MeleshinDA-1/metrics-collector/internal/config"
	"github.com/MeleshinDA-1/metrics-collector/internal/repository"
)

// Flusher периодически сохраняет снимки метрик.
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

// Start сохраняет снимок каждые StoreInterval секунд до отмены ctx.
// Ошибки журналируются, запись продолжается. При интервале <= 0 возвращается сразу.
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
			if err := f.metricsRepository.Flush(ctx, f.storage); err != nil {
				slog.Error("unable to save metrics", "error", err)
			}

		case <-ctx.Done():
			return nil
		}
	}
}
