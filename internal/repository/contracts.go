package repository

import "github.com/MeleshinDA-1/metrics-collector/internal/model"

type MetricsSnapshotProvider interface {
	Snapshot() (model.MetricsSnapshot, error)
}

type MetricsWriter interface {
	SetGauge(name string, value float64) error
	AddCounter(name string, delta int64) (int64, error)
}

type MetricsRepository interface {
	Flush(MetricsSnapshotProvider) error
}
