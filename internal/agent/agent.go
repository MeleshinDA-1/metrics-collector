package agent

import (
	"context"
	"os/signal"
	"sync"
	"syscall"

	"github.com/MeleshinDA-1/metrics-collector/internal/config"
	"github.com/MeleshinDA-1/metrics-collector/internal/retry"
	"golang.org/x/sync/errgroup"
)

func Run(agentConfig config.AgentConfig) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	store := newMetricsStore()

	collected := fanIn(
		runRuntimeCollector(ctx, agentConfig.PollInterval),
		runSystemCollector(ctx, agentConfig.PollInterval),
	)

	sender := newMetricsSender(
		agentConfig.ServerAddress,
		agentConfig.ReportInterval,
		agentConfig.RateLimit,
		agentConfig.Key,
		retry.DefaultPolicy(),
	)

	group, groupCtx := errgroup.WithContext(ctx)

	group.Go(func() error {
		storeCollectedMetrics(store, collected)

		return nil
	})

	group.Go(func() error {
		sender.run(groupCtx, store)

		return nil
	})

	return group.Wait()
}

func storeCollectedMetrics(store *metricsStore, collected <-chan metricsSnapshot) {
	for snapshot := range collected {
		store.update(snapshot.gauges, snapshot.counters)
	}
}

func fanIn(collectors ...<-chan metricsSnapshot) <-chan metricsSnapshot {
	merged := make(chan metricsSnapshot)

	var running sync.WaitGroup
	for _, collected := range collectors {
		running.Add(1)
		go func() {
			defer running.Done()

			for snapshot := range collected {
				merged <- snapshot
			}
		}()
	}

	go func() {
		running.Wait()
		close(merged)
	}()

	return merged
}
