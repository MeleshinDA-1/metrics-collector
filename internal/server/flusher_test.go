package server

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/MeleshinDA-1/metrics-collector/internal/config"
	"github.com/MeleshinDA-1/metrics-collector/internal/repository"
)

type flakyMetricsRepository struct {
	calls      int
	secondCall chan struct{}
}

func (repo *flakyMetricsRepository) Flush(context.Context, repository.MetricsSnapshotProvider) error {
	repo.calls++
	if repo.calls == 1 {
		return errors.New("temporary error")
	}

	repo.secondCall <- struct{}{}
	return nil
}

func TestFlusherContinuesAfterFlushError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	metricsRepository := &flakyMetricsRepository{
		secondCall: make(chan struct{}, 1),
	}
	flusher := NewFlusher(repository.NewMemStorage(), metricsRepository)
	errCh := make(chan error, 1)

	go func() {
		errCh <- flusher.Start(ctx, config.ServerConfig{StoreInterval: 1})
	}()

	select {
	case <-metricsRepository.secondCall:
		cancel()
	case err := <-errCh:
		t.Fatalf("flusher stopped after the first error: %v", err)
	case <-time.After(3 * time.Second):
		t.Fatal("second flush was not attempted")
	}

	if err := <-errCh; err != nil {
		t.Fatalf("flusher returned an error: %v", err)
	}
	if metricsRepository.calls < 2 {
		t.Fatalf("flush calls = %d, want at least 2", metricsRepository.calls)
	}
}
