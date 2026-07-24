package handler

import (
	"net/http"

	models "github.com/MeleshinDA-1/metrics-collector/internal/model"
	"github.com/gorilla/mux"
)

type MetricsSnapshot = models.MetricsSnapshot

type MetricsStorage interface {
	SetGauge(name string, value float64)
	AddCounter(name string, delta int64)
	GetGauge(name string) (float64, bool)
	GetCounter(name string) (int64, bool)
	Snapshot() MetricsSnapshot
}

type MetricsSaver interface {
	Save() error
}

type MetricsHandler struct {
	storage MetricsStorage
}

func NewMetricsHandler(storage MetricsStorage) *MetricsHandler {
	return &MetricsHandler{
		storage: storage,
	}
}

func (handler *MetricsHandler) saveMetrics() error {
	storage, ok := handler.storage.(MetricsSaver)
	if !ok {
		return nil
	}

	return storage.Save()
}

func pathValue(req *http.Request, name string) string {
	return mux.Vars(req)[name]
}
