package repository

import (
	"fmt"

	"github.com/MeleshinDA-1/metrics-collector/internal/model"
)

func validateBatch(metrics []model.Metrics) error {
	for _, metric := range metrics {
		switch metric.MType {
		case model.Gauge:
			if metric.Value == nil {
				return fmt.Errorf("gauge %q has no value", metric.ID)
			}
		case model.Counter:
			if metric.Delta == nil {
				return fmt.Errorf("counter %q has no delta", metric.ID)
			}
		default:
			return fmt.Errorf("unknown metric type %q", metric.MType)
		}
	}

	return nil
}
