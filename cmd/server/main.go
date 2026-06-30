package main

import (
	"fmt"
	"net/http"
	"strconv"
)

func main() {
	startCollectorServer()
}

func startCollectorServer() {
	http.HandleFunc("/update/{metricType}/{metricName}/{metricValue}", updateMetrics)
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		panic(err)
	}
}

func updateMetrics(res http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(res, "Only POST allowed", http.StatusMethodNotAllowed)
		return
	}

	metricType := req.PathValue("metricType")
	metricValue := req.PathValue("metricValue")

	switch metricType {
	case "counter":
		_, err := strconv.ParseInt(metricValue, 10, 64)
		if err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}
	case "gauge":
		_, err := strconv.ParseFloat(metricValue, 64)
		if err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}
	default:
		http.Error(res, fmt.Sprintf("Unknown metric type \"%s\"", metricType), http.StatusBadRequest)
		return
	}

	res.WriteHeader(http.StatusOK)
}
