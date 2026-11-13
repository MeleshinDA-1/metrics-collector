package metrics

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/MeleshinDA-1/metrics-collector/internal/model"
)

// ValueMetricsJSON обрабатывает POST /value/ с JSON-полями id и type.
// Возвращает 200 со значением в JSON, 400 при неверном JSON, 404 при неизвестном
// типе или отсутствии метрики и 500 при ошибке хранилища.
func (handler *MetricsHandler) ValueMetricsJSON(res http.ResponseWriter, req *http.Request) {
	var requestMetric model.Metrics
	if err := json.NewDecoder(req.Body).Decode(&requestMetric); err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	var responseMetric = model.Metrics{
		ID:    requestMetric.ID,
		MType: requestMetric.MType,
		Delta: nil,
		Value: nil,
	}

	res.Header().Set("Content-Type", "application/json")

	switch requestMetric.MType {
	case "counter":
		value, ok, err := handler.storage.GetCounter(req.Context(), requestMetric.ID)
		if err != nil {
			writeError(res, http.StatusInternalServerError, err)
			return
		}
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
		value, ok, err := handler.storage.GetGauge(req.Context(), requestMetric.ID)
		if err != nil {
			writeError(res, http.StatusInternalServerError, err)
			return
		}
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

// ValueMetrics обрабатывает GET /value/{metricType}/{metricName}; переменные задаёт роутер.
// Возвращает 200 с текстовым значением, 404 при отсутствии метрики или неизвестном типе,
// 500 при ошибке хранилища.
func (handler *MetricsHandler) ValueMetrics(res http.ResponseWriter, req *http.Request) {
	metricType := pathValue(req, "metricType")
	metricName := pathValue(req, "metricName")

	switch metricType {
	case "counter":
		value, ok, err := handler.storage.GetCounter(req.Context(), metricName)
		if err != nil {
			writeError(res, http.StatusInternalServerError, err)
			return
		}
		if !ok {
			http.Error(res, "Metric not found", http.StatusNotFound)
			return
		}
		res.Header().Set("Content-Type", "text/plain")
		_, _ = res.Write([]byte(strconv.FormatInt(value, 10)))
	case "gauge":
		value, ok, err := handler.storage.GetGauge(req.Context(), metricName)
		if err != nil {
			writeError(res, http.StatusInternalServerError, err)
			return
		}
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
