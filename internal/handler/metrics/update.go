package metrics

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/MeleshinDA-1/metrics-collector/internal/model"
)

func (handler *MetricsHandler) UpdateMetrics(res http.ResponseWriter, req *http.Request) {
	if status, err := handler.updateMetrics(req); err != nil {
		writeError(res, status, err)
		return
	}

	res.WriteHeader(http.StatusOK)
}

func (handler *MetricsHandler) updateMetrics(req *http.Request) (int, error) {
	metricType := pathValue(req, "metricType")
	metricName := pathValue(req, "metricName")
	metricValue := pathValue(req, "metricValue")

	switch metricType {
	case "counter":
		value, err := strconv.ParseInt(metricValue, 10, 64)
		if err != nil {
			return http.StatusBadRequest, err
		}
		if err := handler.storage.AddCounter(metricName, value); err != nil {
			return http.StatusInternalServerError, err
		}
	case "gauge":
		value, err := strconv.ParseFloat(metricValue, 64)
		if err != nil {
			return http.StatusBadRequest, err
		}
		if err := handler.storage.SetGauge(metricName, value); err != nil {
			return http.StatusInternalServerError, err
		}
	default:
		return http.StatusBadRequest, fmt.Errorf("Unknown metric type \"%s\"", metricType)
	}

	return http.StatusOK, nil
}

func (handler *MetricsHandler) UpdateMetricsJson(res http.ResponseWriter, req *http.Request) {
	requestMetric, status, err := handler.updateMetricsJSON(req)
	if err != nil {
		writeError(res, status, err)
		return
	}

	writeMetricsJSON(res, requestMetric)
}

func (handler *MetricsHandler) updateMetricsJSON(req *http.Request) (model.Metrics, int, error) {
	var requestMetric model.Metrics
	if err := json.NewDecoder(req.Body).Decode(&requestMetric); err != nil {
		return model.Metrics{}, http.StatusBadRequest, err
	}

	switch requestMetric.MType {
	case "counter":
		if requestMetric.Delta == nil {
			return model.Metrics{}, http.StatusBadRequest, fmt.Errorf("delta is required")
		}
		if err := handler.storage.AddCounter(requestMetric.ID, *requestMetric.Delta); err != nil {
			return model.Metrics{}, http.StatusInternalServerError, err
		}
		value, _, err := handler.storage.GetCounter(requestMetric.ID)
		if err != nil {
			return model.Metrics{}, http.StatusInternalServerError, err
		}
		requestMetric.Delta = &value
		requestMetric.Value = nil
	case "gauge":
		if requestMetric.Value == nil {
			return model.Metrics{}, http.StatusBadRequest, fmt.Errorf("value is required")
		}
		if err := handler.storage.SetGauge(requestMetric.ID, *requestMetric.Value); err != nil {
			return model.Metrics{}, http.StatusInternalServerError, err
		}
		requestMetric.Delta = nil
	default:
		return model.Metrics{}, http.StatusBadRequest, fmt.Errorf("Unknown metric type \"%s\"", requestMetric.MType)
	}

	return requestMetric, http.StatusOK, nil
}

func writeMetricsJSON(res http.ResponseWriter, metric model.Metrics) {
	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(res).Encode(metric); err != nil {
		return
	}
}
