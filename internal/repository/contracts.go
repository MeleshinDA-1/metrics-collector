// Пакет repository хранит метрики в памяти и PostgreSQL, сохраняет файловые снимки.
package repository

import (
	"context"

	"github.com/MeleshinDA-1/metrics-collector/internal/model"
)

// MetricsSnapshotProvider предоставляет метрики для сохранения и списка.
type MetricsSnapshotProvider interface {
	Snapshot(ctx context.Context) (model.MetricsSnapshot, error)
}

// MetricsWriter обновляет метрики.
type MetricsWriter interface {
	// SetGauge создаёт или заменяет gauge name.
	SetGauge(ctx context.Context, name string, value float64) error
	// AddCounter прибавляет delta к name и возвращает итог; новый счётчик начинается с нуля.
	AddCounter(ctx context.Context, name string, delta int64) (int64, error)
}

// MetricsRepository сохраняет снимок поставщика метрик.
type MetricsRepository interface {
	Flush(ctx context.Context, provider MetricsSnapshotProvider) error
}
