package main

import (
	"net/http"

	"github.com/MeleshinDA-1/golang-practicum-alice/internal/handler"
)

func main() {
	startCollectorServer()
}

func startCollectorServer() {
	http.HandleFunc("/update/{metricType}/{metricName}/{metricValue}", handler.UpdateMetrics)
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		panic(err)
	}
}
