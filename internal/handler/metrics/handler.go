package metrics

import (
	"context"
	"net/http"

	"github.com/MeleshinDA-1/metrics-collector/internal/model"
	"github.com/gorilla/mux"
)

type MetricsSnapshot = model.MetricsSnapshot

type MetricsStorage interface {
	SetGauge(ctx context.Context, name string, value float64) error
	AddCounter(ctx context.Context, name string, delta int64) (int64, error)
	UpdateBatch(ctx context.Context, metrics []model.Metrics) error
	GetGauge(ctx context.Context, name string) (float64, bool, error)
	GetCounter(ctx context.Context, name string) (int64, bool, error)
	Snapshot(ctx context.Context) (MetricsSnapshot, error)
}

type MetricsHandler struct {
	storage MetricsStorage
}

func NewMetricsHandler(storage MetricsStorage) *MetricsHandler {
	return &MetricsHandler{
		storage: storage,
	}
}

func pathValue(req *http.Request, name string) string {
	return mux.Vars(req)[name]
}

func writeError(res http.ResponseWriter, status int, err error) {
	if status >= http.StatusInternalServerError {
		http.Error(res, http.StatusText(status), status)
		return
	}

	http.Error(res, err.Error(), status)
}
