package main

import (
	"github.com/MeleshinDA-1/metrics-collector/internal/agent"
	"github.com/MeleshinDA-1/metrics-collector/internal/config"
)

func main() {
	agentConfig := config.MustParseAgentConfig()

	agent.Run(agentConfig.ServerAddress, agentConfig.PollInterval, agentConfig.ReportInterval)
}
