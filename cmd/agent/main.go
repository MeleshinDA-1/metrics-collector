package main

import (
	"log"

	"github.com/MeleshinDA-1/metrics-collector/internal/agent"
	"github.com/MeleshinDA-1/metrics-collector/internal/config"
)

func main() {
	agentConfig, err := config.ParseAgentConfigFromArgs()
	if err != nil {
		log.Fatal(err)
	}

	agent.Run(agentConfig.ServerAddress, agentConfig.PollInterval, agentConfig.ReportInterval)
}
