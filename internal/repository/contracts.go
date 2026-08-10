package repository

import "github.com/MeleshinDA-1/metrics-collector/internal/model"

type MetricsSnapshotProvider interface {
	Snapshot() model.MetricsSnapshot
}

type MetricsRepository interface {
	Flush(MetricsSnapshotProvider) error
}
