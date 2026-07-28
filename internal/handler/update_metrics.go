package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	models "github.com/MeleshinDA-1/metrics-collector/internal/model"
)

func (handler *MetricsHandler) UpdateMetrics(res http.ResponseWriter, req *http.Request) {
	if err := handler.updateMetrics(req); err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	res.WriteHeader(http.StatusOK)
}

func (handler *MetricsHandler) updateMetrics(req *http.Request) error {
	metricType := pathValue(req, "metricType")
	metricName := pathValue(req, "metricName")
	metricValue := pathValue(req, "metricValue")

	switch metricType {
	case "counter":
		value, err := strconv.ParseInt(metricValue, 10, 64)
		if err != nil {
			return err
		}
		handler.storage.AddCounter(metricName, value)
	case "gauge":
		value, err := strconv.ParseFloat(metricValue, 64)
		if err != nil {
			return err
		}
		handler.storage.SetGauge(metricName, value)
	default:
		return fmt.Errorf("Unknown metric type \"%s\"", metricType)
	}

	return nil
}

func (handler *MetricsHandler) UpdateMetricsJson(res http.ResponseWriter, req *http.Request) {
	requestMetric, err := handler.updateMetricsJSON(req)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	writeMetricsJSON(res, requestMetric)
}

func (handler *MetricsHandler) updateMetricsJSON(req *http.Request) (models.Metrics, error) {
	var requestMetric models.Metrics
	if err := json.NewDecoder(req.Body).Decode(&requestMetric); err != nil {
		return models.Metrics{}, err
	}

	switch requestMetric.MType {
	case "counter":
		if requestMetric.Delta == nil {
			return models.Metrics{}, fmt.Errorf("delta is required")
		}
		handler.storage.AddCounter(requestMetric.ID, *requestMetric.Delta)
		value, _ := handler.storage.GetCounter(requestMetric.ID)
		requestMetric.Delta = &value
		requestMetric.Value = nil
	case "gauge":
		if requestMetric.Value == nil {
			return models.Metrics{}, fmt.Errorf("value is required")
		}
		handler.storage.SetGauge(requestMetric.ID, *requestMetric.Value)
		requestMetric.Delta = nil
	default:
		return models.Metrics{}, fmt.Errorf("Unknown metric type \"%s\"", requestMetric.MType)
	}

	return requestMetric, nil
}

func writeMetricsJSON(res http.ResponseWriter, metric models.Metrics) {
	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(res).Encode(metric); err != nil {
		return
	}
}
