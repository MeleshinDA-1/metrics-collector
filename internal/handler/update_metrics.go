package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	models "github.com/MeleshinDA-1/metrics-collector/internal/model"
)

func (handler *MetricsHandler) UpdateMetrics(res http.ResponseWriter, req *http.Request) {
	metricType := pathValue(req, "metricType")
	metricName := pathValue(req, "metricName")
	metricValue := pathValue(req, "metricValue")

	switch metricType {
	case "counter":
		value, err := strconv.ParseInt(metricValue, 10, 64)
		if err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}
		handler.storage.AddCounter(metricName, value)
	case "gauge":
		value, err := strconv.ParseFloat(metricValue, 64)
		if err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}
		handler.storage.SetGauge(metricName, value)
	default:
		http.Error(res, fmt.Sprintf("Unknown metric type \"%s\"", metricType), http.StatusBadRequest)
		return
	}
	if err := handler.saveMetrics(); err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	res.WriteHeader(http.StatusOK)
}

func (handler *MetricsHandler) UpdateMetricsJson(res http.ResponseWriter, req *http.Request) {
	var requestMetric models.Metrics
	if err := json.NewDecoder(req.Body).Decode(&requestMetric); err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	switch requestMetric.MType {
	case "counter":
		if requestMetric.Delta == nil {
			http.Error(res, "value is required", http.StatusBadRequest)
			return
		}
		handler.storage.AddCounter(requestMetric.ID, *requestMetric.Delta)
		value, _ := handler.storage.GetCounter(requestMetric.ID)
		requestMetric.Delta = &value
		requestMetric.Value = nil
	case "gauge":
		if requestMetric.Value == nil {
			http.Error(res, "delta is required", http.StatusBadRequest)
			return
		}
		handler.storage.SetGauge(requestMetric.ID, *requestMetric.Value)
		requestMetric.Delta = nil
	default:
		http.Error(res, fmt.Sprintf("Unknown metric type \"%s\"", requestMetric.MType), http.StatusBadRequest)
		return
	}
	if err := handler.saveMetrics(); err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(res).Encode(requestMetric); err != nil {
		return
	}
}
