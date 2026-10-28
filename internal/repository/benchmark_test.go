package repository

import (
	"context"
	"fmt"
	"testing"

	"github.com/MeleshinDA-1/metrics-collector/internal/model"
)

func BenchmarkMemStorage(b *testing.B) {
	for _, size := range []int{32, 1024} {
		b.Run(fmt.Sprint(size), func(b *testing.B) {
			ctx := context.Background()
			storage := NewMemStorage()
			batch := make([]model.Metrics, size)
			for i := range batch {
				value := float64(i) + 0.5
				batch[i] = model.Metrics{ID: fmt.Sprintf("metric-%04d", i), MType: model.Gauge, Value: &value}
			}
			if err := storage.UpdateBatch(ctx, batch); err != nil {
				b.Fatal(err)
			}
			b.Run("UpdateBatch", func(b *testing.B) {
				b.ReportAllocs()
				for range b.N {
					if err := storage.UpdateBatch(ctx, batch); err != nil {
						b.Fatal(err)
					}
				}
			})
			b.Run("GetGauge", func(b *testing.B) {
				b.ReportAllocs()
				for range b.N {
					if _, ok, err := storage.GetGauge(ctx, batch[0].ID); err != nil || !ok {
						b.Fatalf("gauge: found %v, error %v", ok, err)
					}
				}
			})
			b.Run("Snapshot", func(b *testing.B) {
				b.ReportAllocs()
				for range b.N {
					if snapshot, err := storage.Snapshot(ctx); err != nil || len(snapshot.Gauges) != size {
						b.Fatalf("snapshot: %d gauges, error %v", len(snapshot.Gauges), err)
					}
				}
			})
		})
	}
}
