package handler

import (
	"html/template"
	"net/http"
	"strconv"
)

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
