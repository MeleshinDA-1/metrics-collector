package repository

import (
	"context"

	"github.com/MeleshinDA-1/metrics-collector/internal/model"
)

type MetricsSnapshotProvider interface {
	Snapshot(ctx context.Context) (model.MetricsSnapshot, error)
}

type MetricsWriter interface {
	SetGauge(ctx context.Context, name string, value float64) error
	AddCounter(ctx context.Context, name string, delta int64) (int64, error)
}

type MetricsRepository interface {
	Flush(ctx context.Context, provider MetricsSnapshotProvider) error
}
