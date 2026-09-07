package agent

import (
	"github.com/MeleshinDA-1/metrics-collector/internal/config"
	"github.com/MeleshinDA-1/metrics-collector/internal/retry"
)

func Run(agentConfig config.AgentConfig) {
	store := newMetricsStore()
	collectMetrics(store)

	sender := newMetricsSender(
		agentConfig.ServerAddress,
		agentConfig.ReportInterval,
		agentConfig.Key,
		retry.DefaultPolicy(),
	)

	go runCollector(store, agentConfig.PollInterval)
	sender.run(store)
}
