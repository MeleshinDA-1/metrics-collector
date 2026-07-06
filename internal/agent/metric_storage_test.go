package agent

import (
	"reflect"
	"testing"
)

type metricStorageUpdate struct {
	gauges   map[string]float64
	counters map[string]int64
}

type metricStorageCase struct {
	name    string
	updates []metricStorageUpdate
	want    allMetrics
}

func TestMetricStorageUpdateMetrics(t *testing.T) {
	tests := []metricStorageCase{
		{
			name: "gauges are overwritten",
			updates: []metricStorageUpdate{
				{
					gauges: map[string]float64{
						"Alloc": 10,
					},
				},
				{
					gauges: map[string]float64{
						"Alloc": 42,
					},
				},
			},
			want: allMetrics{
				gauges: map[string]float64{
					"Alloc": 42,
				},
				counters: map[string]int64{},
			},
		},
		{
			name: "counters are accumulated",
			updates: []metricStorageUpdate{
				{
					counters: map[string]int64{
						"Requests": 10,
					},
				},
				{
					counters: map[string]int64{
						"Requests": 5,
					},
				},
			},
			want: allMetrics{
				gauges: map[string]float64{},
				counters: map[string]int64{
					"Requests": 15,
				},
			},
		},
		{
			name: "poll count is incremented after two updates",
			updates: []metricStorageUpdate{
				{
					counters: map[string]int64{
						"PollCount": 1,
					},
				},
				{
					counters: map[string]int64{
						"PollCount": 1,
					},
				},
			},
			want: allMetrics{
				gauges: map[string]float64{},
				counters: map[string]int64{
					"PollCount": 2,
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			storage := newMetricStorage()

			for _, update := range test.updates {
				storage.updateMetrics(update.gauges, update.counters)
			}

			assertAllMetrics(t, storage.snapshot(), test.want)
		})
	}
}

func TestMetricStorageSnapshotReturnsCopies(t *testing.T) {
	storage := newMetricStorage()
	storage.updateMetrics(
		map[string]float64{
			"Alloc": 42,
		},
		map[string]int64{
			"PollCount": 1,
		},
	)

	snapshot := storage.snapshot()
	snapshot.gauges["Alloc"] = 100
	snapshot.counters["PollCount"] = 100

	assertAllMetrics(t, storage.snapshot(), allMetrics{
		gauges: map[string]float64{
			"Alloc": 42,
		},
		counters: map[string]int64{
			"PollCount": 1,
		},
	})
}

func assertAllMetrics(t *testing.T, got allMetrics, want allMetrics) {
	t.Helper()

	if !reflect.DeepEqual(got.gauges, want.gauges) {
		t.Fatalf("gauges = %v, want %v", got.gauges, want.gauges)
	}

	if !reflect.DeepEqual(got.counters, want.counters) {
		t.Fatalf("counters = %v, want %v", got.counters, want.counters)
	}
}
