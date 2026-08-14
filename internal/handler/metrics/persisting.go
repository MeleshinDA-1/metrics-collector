package metrics

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
	if status, err := handler.updateMetrics(req); err != nil {
		writeError(res, status, err)
		return
	}
	if err := handler.metricsRepository.Flush(req.Context(), handler.storage); err != nil {
		writeError(res, http.StatusInternalServerError, err)
		return
	}

	res.WriteHeader(http.StatusOK)
}

func (handler *PersistingMetricsHandler) UpdateMetricsJSON(res http.ResponseWriter, req *http.Request) {
	requestMetric, status, err := handler.updateMetricsJSON(req)
	if err != nil {
		writeError(res, status, err)
		return
	}
	if err := handler.metricsRepository.Flush(req.Context(), handler.storage); err != nil {
		writeError(res, http.StatusInternalServerError, err)
		return
	}

	writeMetricsJSON(res, requestMetric)
}

func (handler *PersistingMetricsHandler) UpdateBatchMetricsJSON(res http.ResponseWriter, req *http.Request) {
	requestMetrics, status, err := handler.updateBatchMetricsJSON(req)
	if err != nil {
		writeError(res, status, err)
		return
	}
	if err := handler.metricsRepository.Flush(req.Context(), handler.storage); err != nil {
		writeError(res, http.StatusInternalServerError, err)
		return
	}

	writeMetricsBatchJSON(res, requestMetrics)
}
