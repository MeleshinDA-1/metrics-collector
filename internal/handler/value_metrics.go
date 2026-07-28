package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	models "github.com/MeleshinDA-1/metrics-collector/internal/model"
)

func (handler *MetricsHandler) ValueMetricsJson(res http.ResponseWriter, req *http.Request) {
	var requestMetric models.Metrics
	if err := json.NewDecoder(req.Body).Decode(&requestMetric); err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	var responseMetric = models.Metrics{
		ID:    requestMetric.ID,
		MType: requestMetric.MType,
		Delta: nil,
		Value: nil,
	}

	res.Header().Set("Content-Type", "application/json")

	switch requestMetric.MType {
	case "counter":
		value, ok := handler.storage.GetCounter(requestMetric.ID)
		if !ok {
			http.Error(res, "Metric not found", http.StatusNotFound)
			return
		}

		responseMetric.Delta = &value
		if err := json.NewEncoder(res).Encode(responseMetric); err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
	case "gauge":
		value, ok := handler.storage.GetGauge(requestMetric.ID)
		if !ok {
			http.Error(res, "Metric not found", http.StatusNotFound)
			return
		}

		responseMetric.Value = &value
		if err := json.NewEncoder(res).Encode(responseMetric); err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
	default:
		http.Error(res, fmt.Sprintf("Unknown metric type \"%s\"", requestMetric.MType), http.StatusNotFound)
	}
}

func (handler *MetricsHandler) ValueMetrics(res http.ResponseWriter, req *http.Request) {
	metricType := pathValue(req, "metricType")
	metricName := pathValue(req, "metricName")

	switch metricType {
	case "counter":
		value, ok := handler.storage.GetCounter(metricName)
		if !ok {
			http.Error(res, "Metric not found", http.StatusNotFound)
			return
		}
		res.Header().Set("Content-Type", "text/plain")
		_, _ = res.Write([]byte(strconv.FormatInt(value, 10)))
	case "gauge":
		value, ok := handler.storage.GetGauge(metricName)
		if !ok {
			http.Error(res, "Metric not found", http.StatusNotFound)
			return
		}
		res.Header().Set("Content-Type", "text/plain")
		_, _ = res.Write([]byte(strconv.FormatFloat(value, 'f', -1, 64)))
	default:
		http.Error(res, fmt.Sprintf("Unknown metric type \"%s\"", metricType), http.StatusNotFound)
	}
}
