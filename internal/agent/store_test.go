package agent

import (
	"reflect"
	"testing"
)

type metricsUpdate struct {
	gauges   map[string]float64
	counters map[string]int64
}

type metricsStoreTestCase struct {
	name    string
	updates []metricsUpdate
	want    metricsSnapshot
}

func TestMetricsStoreUpdate(t *testing.T) {
	tests := []metricsStoreTestCase{
		{
			name: "gauges are overwritten",
			updates: []metricsUpdate{
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
			want: metricsSnapshot{
				gauges: map[string]float64{
					"Alloc": 42,
				},
				counters: map[string]int64{},
			},
		},
		{
			name: "counters are accumulated",
			updates: []metricsUpdate{
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
			want: metricsSnapshot{
				gauges: map[string]float64{},
				counters: map[string]int64{
					"Requests": 15,
				},
			},
		},
		{
			name: "poll count is incremented after two updates",
			updates: []metricsUpdate{
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
			want: metricsSnapshot{
				gauges: map[string]float64{},
				counters: map[string]int64{
					"PollCount": 2,
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := newMetricsStore()

			for _, update := range test.updates {
				store.update(update.gauges, update.counters)
			}

			assertMetricsSnapshot(t, store.snapshot(), test.want)
		})
	}
}

func TestMetricsStoreSnapshotReturnsCopies(t *testing.T) {
	store := newMetricsStore()
	store.update(
		map[string]float64{
			"Alloc": 42,
		},
		map[string]int64{
			"PollCount": 1,
		},
	)

	snapshot := store.snapshot()
	snapshot.gauges["Alloc"] = 100
	snapshot.counters["PollCount"] = 100

	assertMetricsSnapshot(t, store.snapshot(), metricsSnapshot{
		gauges: map[string]float64{
			"Alloc": 42,
		},
		counters: map[string]int64{
			"PollCount": 1,
		},
	})
}

func assertMetricsSnapshot(t *testing.T, got metricsSnapshot, want metricsSnapshot) {
	t.Helper()

	if !reflect.DeepEqual(got.gauges, want.gauges) {
		t.Fatalf("gauges = %v, want %v", got.gauges, want.gauges)
	}

	if !reflect.DeepEqual(got.counters, want.counters) {
		t.Fatalf("counters = %v, want %v", got.counters, want.counters)
	}
}
