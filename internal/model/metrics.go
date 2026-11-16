// Пакет model описывает типы метрик агента, HTTP API и хранилища.
package model

const (
	// Counter обозначает целочисленную метрику, накапливающую приращения.
	Counter = "counter"
	// Gauge обозначает вещественную метрику, заменяемую при обновлении.
	Gauge = "gauge"
)

// Metrics представляет метрику в JSON. Указатели отличают отсутствие значения
// от нуля: обновление counter требует Delta, обновление gauge требует Value.
type Metrics struct {
	ID    string `json:"id"`
	MType string `json:"type"`
	// Delta содержит приращение счётчика при обновлении или итог при чтении.
	Delta *int64   `json:"delta,omitempty"`
	Value *float64 `json:"value,omitempty"`
	// Hash оставлен для совместимости JSON; подпись запроса передаётся HTTP-заголовком.
	Hash string `json:"hash,omitempty"`
}

// MetricsSnapshot содержит значения gauge и итоги counter по именам.
type MetricsSnapshot struct {
	Gauges   map[string]float64
	Counters map[string]int64
}
