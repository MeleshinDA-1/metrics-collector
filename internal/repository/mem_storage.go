package repository

type MemStorage struct {
	Gauges   map[string]float64
	Counters map[string]int64
}
