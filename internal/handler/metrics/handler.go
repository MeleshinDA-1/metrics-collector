package metrics

import (
	"net/http"

	"github.com/MeleshinDA-1/metrics-collector/internal/model"
	"github.com/gorilla/mux"
)

type MetricsSnapshot = model.MetricsSnapshot

type MetricsStorage interface {
	SetGauge(name string, value float64) error
	AddCounter(name string, delta int64) error
	GetGauge(name string) (float64, bool, error)
	GetCounter(name string) (int64, bool, error)
	Snapshot() (MetricsSnapshot, error)
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
