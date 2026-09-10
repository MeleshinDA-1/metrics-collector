package config

import (
	"fmt"
	"os"
	"time"
)

type AgentConfig struct {
	ServerAddress  string        `env:"ADDRESS" flag:"a,default=localhost:8080" usage:"HTTP server address"`
	ReportInterval time.Duration `env:"REPORT_INTERVAL" flag:"r,default=10s" usage:"metrics report interval in seconds"`
	PollInterval   time.Duration `env:"POLL_INTERVAL" flag:"p,default=2s" usage:"metrics poll interval in seconds"`
	Key            string        `env:"KEY" flag:"k,default=" usage:"key used to sign outgoing requests"`
	RateLimit      int           `env:"RATE_LIMIT" flag:"l,default=1" usage:"maximum number of simultaneous requests to the server"`
}

func ParseAgentConfig() (AgentConfig, error) {
	agentConfig, err := ParseConfig[AgentConfig](os.Args[1:])
	if err != nil {
		return AgentConfig{}, err
	}

	if err := agentConfig.validate(); err != nil {
		return AgentConfig{}, err
	}

	return agentConfig, nil
}

func (agentConfig AgentConfig) validate() error {
	if agentConfig.PollInterval <= 0 {
		return fmt.Errorf("poll interval must be positive, got %s", agentConfig.PollInterval)
	}
	if agentConfig.ReportInterval <= 0 {
		return fmt.Errorf("report interval must be positive, got %s", agentConfig.ReportInterval)
	}
	if agentConfig.RateLimit < 1 {
		return fmt.Errorf("rate limit must be at least 1, got %d", agentConfig.RateLimit)
	}

	return nil
}
