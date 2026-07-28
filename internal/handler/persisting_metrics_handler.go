package handler

import (
	"net/http"

	"github.com/MeleshinDA-1/metrics-collector/internal/repository"
)

type PersistingMetricsHandler struct {
	*MetricsHandler
	metricsRepository repository.MetricsRepository
}

func NewPersistingMetricsHandler(
	metricsHandler *MetricsHandler,
	metricsRepository repository.MetricsRepository,
) *PersistingMetricsHandler {
	if metricsRepository == nil {
		panic("metrics repository is required")
	}

	return &PersistingMetricsHandler{
		MetricsHandler:    metricsHandler,
		metricsRepository: metricsRepository,
	}
}

func (handler *PersistingMetricsHandler) UpdateMetrics(res http.ResponseWriter, req *http.Request) {
	if err := handler.updateMetrics(req); err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}
	if err := handler.metricsRepository.Flush(handler.storage); err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	res.WriteHeader(http.StatusOK)
}

func (handler *PersistingMetricsHandler) UpdateMetricsJson(res http.ResponseWriter, req *http.Request) {
	requestMetric, err := handler.updateMetricsJSON(req)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}
	if err := handler.metricsRepository.Flush(handler.storage); err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	writeMetricsJSON(res, requestMetric)
}
