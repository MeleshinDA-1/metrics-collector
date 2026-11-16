package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/MeleshinDA-1/metrics-collector/internal/handler"
	"github.com/MeleshinDA-1/metrics-collector/internal/handler/health"
	"github.com/MeleshinDA-1/metrics-collector/internal/handler/metrics"
	"github.com/MeleshinDA-1/metrics-collector/internal/repository"
)

func exampleRouter(pinger health.DBPinger) http.Handler {
	return handler.NewRouter(
		metrics.NewMetricsHandler(repository.NewMemStorage()),
		health.NewPingHandler(pinger),
		"",
	)
}

func exampleRequest(router http.Handler, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, req)
	return response
}

// Обновление и чтение счётчика через параметры пути.
func ExampleNewRouter_text() {
	router := exampleRouter(nil)
	response := exampleRequest(router, http.MethodPost, "/update/counter/PollCount/3", "")
	fmt.Println(response.Code)
	response = exampleRequest(router, http.MethodGet, "/value/counter/PollCount", "")
	fmt.Println(response.Code, response.Body.String())

	// Output:
	// 200
	// 200 3
}

// Обновление и чтение gauge через JSON.
func ExampleNewRouter_json() {
	router := exampleRouter(nil)
	response := exampleRequest(router, http.MethodPost, "/update/", `{"id":"Alloc","type":"gauge","value":1.5}`)
	fmt.Println(response.Code, strings.TrimSpace(response.Body.String()))
	response = exampleRequest(router, http.MethodPost, "/value/", `{"id":"Alloc","type":"gauge"}`)
	fmt.Println(response.Code, strings.TrimSpace(response.Body.String()))

	// Output:
	// 200 {"id":"Alloc","type":"gauge","value":1.5}
	// 200 {"id":"Alloc","type":"gauge","value":1.5}
}

// Пакетное обновление возвращает исходный массив метрик.
func ExampleNewRouter_batch() {
	router := exampleRouter(nil)
	batch := `[{"id":"Alloc","type":"gauge","value":1.5},{"id":"PollCount","type":"counter","delta":2}]`
	response := exampleRequest(router, http.MethodPost, "/updates/", batch)
	fmt.Println(response.Code, strings.TrimSpace(response.Body.String()))

	// Output:
	// 200 [{"id":"Alloc","type":"gauge","value":1.5},{"id":"PollCount","type":"counter","delta":2}]
}

// Получение HTML-списка сохранённых метрик.
func ExampleNewRouter_list() {
	router := exampleRouter(nil)
	response := exampleRequest(router, http.MethodPost, "/update/gauge/Alloc/1.5", "")
	fmt.Println(response.Code)
	response = exampleRequest(router, http.MethodGet, "/", "")
	fmt.Println(response.Code, response.Header().Get("Content-Type"))
	fmt.Println(strings.Join(strings.Fields(response.Body.String()), " "))

	// Output:
	// 200
	// 200 text/html; charset=utf-8
	// <html> <body> <ul> <li>Alloc: 1.5</li> </ul> </body> </html>
}

// examplePinger заменяет подключение к БД в примере.
type examplePinger struct{}

func (examplePinger) Ping(context.Context) error { return nil }

// Проверка доступности БД.
func ExampleNewRouter_ping() {
	router := exampleRouter(examplePinger{})
	response := exampleRequest(router, http.MethodGet, "/ping", "")
	fmt.Println(response.Code)

	// Output:
	// 200
}
