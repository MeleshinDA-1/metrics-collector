package config

import (
	"flag"
	"fmt"
	"io"
	"os"
	"time"
)

const (
	DefaultServerAddress  = "localhost:8080"
	DefaultReportInterval = 10 * time.Second
	DefaultPollInterval   = 2 * time.Second
)

type ServerConfig struct {
	Address string
}

type AgentConfig struct {
	ServerAddress  string
	ReportInterval time.Duration
	PollInterval   time.Duration
}

func ParseServerConfig(args []string) (ServerConfig, error) {
	flagSet := flag.NewFlagSet("server", flag.ContinueOnError)
	flagSet.SetOutput(io.Discard)

	address := flagSet.String("a", DefaultServerAddress, "HTTP server address")

	if err := flagSet.Parse(args); err != nil {
		return ServerConfig{}, err
	}

	return ServerConfig{
		Address: *address,
	}, nil
}

func MustParseServerConfig() ServerConfig {
	config, err := ParseServerConfig(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	return config
}

func ParseAgentConfig(args []string) (AgentConfig, error) {
	flagSet := flag.NewFlagSet("agent", flag.ContinueOnError)
	flagSet.SetOutput(io.Discard)

	address := flagSet.String("a", DefaultServerAddress, "HTTP server address")
	reportInterval := flagSet.Int("r", int(DefaultReportInterval/time.Second), "metrics report interval in seconds")
	pollInterval := flagSet.Int("p", int(DefaultPollInterval/time.Second), "metrics poll interval in seconds")

	if err := flagSet.Parse(args); err != nil {
		return AgentConfig{}, err
	}

	return AgentConfig{
		ServerAddress:  *address,
		ReportInterval: time.Duration(*reportInterval) * time.Second,
		PollInterval:   time.Duration(*pollInterval) * time.Second,
	}, nil
}

func MustParseAgentConfig() AgentConfig {
	config, err := ParseAgentConfig(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	return config
}
