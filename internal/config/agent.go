package config

import (
	"os"
	"time"
)

type AgentConfig struct {
	ServerAddress  string        `env:"ADDRESS" flag:"a,default=localhost:8080" usage:"HTTP server address"`
	ReportInterval time.Duration `env:"REPORT_INTERVAL" flag:"r,default=10s" usage:"metrics report interval in seconds"`
	PollInterval   time.Duration `env:"POLL_INTERVAL" flag:"p,default=2s" usage:"metrics poll interval in seconds"`
	Key            string        `env:"KEY" flag:"k,default=" usage:"key used to sign outgoing requests"`
}

func ParseAgentConfig() (AgentConfig, error) {
	return ParseConfig[AgentConfig](os.Args[1:])
}
