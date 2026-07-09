package handler

import (
	"fmt"
	"html/template"
	"net/http"
	"strconv"

	models "github.com/MeleshinDA-1/metrics-collector/internal/model"
	"github.com/gorilla/mux"
)

type MetricsSnapshot = models.MetricsSnapshot

var metricsListTemplate = template.Must(template.New("metrics-list").Parse(`
<html>
<body>
<ul>
{{range .Gauges}}
	<li>{{.Name}}: {{.Value}}</li>
{{end}}
{{range .Counters}}
	<li>{{.Name}}: {{.Value}}</li>
{{end}}
</ul>
</body>
</html>
`))

type metricView struct {
	Name  string
	Value string
}

type MetricsStorage interface {
	SetGauge(name string, value float64)
	AddCounter(name string, delta int64)
	GetGauge(name string) (float64, bool)
	GetCounter(name string) (int64, bool)
	Snapshot() MetricsSnapshot
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
	metrics := handler.storage.Snapshot()

	data := struct {
		Gauges   []metricView
		Counters []metricView
	}{
		Gauges:   make([]metricView, 0, len(metrics.Gauges)),
		Counters: make([]metricView, 0, len(metrics.Counters)),
	}

	for name, value := range metrics.Gauges {
		data.Gauges = append(data.Gauges, metricView{
			Name:  name,
			Value: strconv.FormatFloat(value, 'f', -1, 64),
		})
	}
	for name, value := range metrics.Counters {
		data.Counters = append(data.Counters, metricView{
			Name:  name,
			Value: strconv.FormatInt(value, 10),
		})
	}

	res.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := metricsListTemplate.Execute(res, data); err != nil {
		http.Error(res, "failed to render metrics", http.StatusInternalServerError)
	}
}

func pathValue(req *http.Request, name string) string {
	return mux.Vars(req)[name]
}
