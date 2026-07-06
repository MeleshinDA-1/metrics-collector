package handler

import (
	"fmt"
	"html"
	"net/http"
	"strconv"

	"github.com/MeleshinDA-1/golang-practicum-alice/internal/repository"
	"github.com/gorilla/mux"
)

type MetricsStorage interface {
	SetGauge(name string, value float64)
	AddCounter(name string, delta int64)
	GetGauge(name string) (float64, bool)
	GetCounter(name string) (int64, bool)
	Snapshot() repository.MetricsSnapshot
}

type MetricsHandler struct {
	storage MetricsStorage
}

func NewMetricsHandler(storage MetricsStorage) *MetricsHandler {
	return &MetricsHandler{
		storage: storage,
	}
}

func (handler *MetricsHandler) UpdateMetrics(res http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(res, "Only POST allowed", http.StatusMethodNotAllowed)
		return
	}

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

	res.WriteHeader(http.StatusOK)
}

func (handler *MetricsHandler) ValueMetrics(res http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(res, "Only GET allowed", http.StatusMethodNotAllowed)
		return
	}

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

func (handler *MetricsHandler) ListMetrics(res http.ResponseWriter, req *http.Request) {
	if req.URL.Path != "/" {
		http.NotFound(res, req)
		return
	}

	if req.Method != http.MethodGet {
		http.Error(res, "Only GET allowed", http.StatusMethodNotAllowed)
		return
	}

	metrics := handler.storage.Snapshot()

	res.Header().Set("Content-Type", "text/html")
	_, _ = res.Write([]byte("<html><body><ul>"))
	for name, value := range metrics.Gauges {
		_, _ = fmt.Fprintf(res, "<li>%s: %s</li>", html.EscapeString(name), strconv.FormatFloat(value, 'f', -1, 64))
	}
	for name, value := range metrics.Counters {
		_, _ = fmt.Fprintf(res, "<li>%s: %s</li>", html.EscapeString(name), strconv.FormatInt(value, 10))
	}
	_, _ = res.Write([]byte("</ul></body></html>"))
}

func pathValue(req *http.Request, name string) string {
	if value := req.PathValue(name); value != "" {
		return value
	}

	return mux.Vars(req)[name]
}
