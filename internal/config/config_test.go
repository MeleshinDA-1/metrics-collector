package config

import (
	"testing"
	"time"
)

func TestParseServerConfig(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantAddress string
		wantError   bool
	}{
		{
			name:        "default address",
			wantAddress: DefaultServerAddress,
		},
		{
			name:        "custom address",
			args:        []string{"-a", "localhost:8888"},
			wantAddress: "localhost:8888",
		},
		{
			name:      "unknown flag",
			args:      []string{"-unknown"},
			wantError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config, err := ParseServerConfig(test.args)
			if test.wantError {
				if err == nil {
					t.Fatal("error = nil, want non-nil error")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if config.Address != test.wantAddress {
				t.Fatalf("address = %q, want %q", config.Address, test.wantAddress)
			}
		})
	}
}

func TestParseAgentConfig(t *testing.T) {
	tests := []struct {
		name               string
		args               []string
		wantServerAddress  string
		wantReportInterval time.Duration
		wantPollInterval   time.Duration
		wantError          bool
	}{
		{
			name:               "defaults",
			wantServerAddress:  DefaultServerAddress,
			wantReportInterval: DefaultReportInterval,
			wantPollInterval:   DefaultPollInterval,
		},
		{
			name:               "custom values",
			args:               []string{"-a", "localhost:8888", "-r", "3", "-p", "1"},
			wantServerAddress:  "localhost:8888",
			wantReportInterval: 3 * time.Second,
			wantPollInterval:   time.Second,
		},
		{
			name:      "unknown flag",
			args:      []string{"-unknown"},
			wantError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config, err := ParseAgentConfig(test.args)
			if test.wantError {
				if err == nil {
					t.Fatal("error = nil, want non-nil error")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if config.ServerAddress != test.wantServerAddress {
				t.Fatalf("server address = %q, want %q", config.ServerAddress, test.wantServerAddress)
			}
			if config.ReportInterval != test.wantReportInterval {
				t.Fatalf("report interval = %s, want %s", config.ReportInterval, test.wantReportInterval)
			}
			if config.PollInterval != test.wantPollInterval {
				t.Fatalf("poll interval = %s, want %s", config.PollInterval, test.wantPollInterval)
			}
		})
	}
}
