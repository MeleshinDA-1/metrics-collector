package config

import (
	"testing"
	"time"
)

func TestParseAgentConfig(t *testing.T) {
	t.Setenv("KEY", "")

	tests := []struct {
		name               string
		args               []string
		wantServerAddress  string
		wantReportInterval time.Duration
		wantPollInterval   time.Duration
		wantKey            string
		wantError          bool
	}{
		{
			name:               "defaults",
			wantServerAddress:  "localhost:8080",
			wantReportInterval: 10 * time.Second,
			wantPollInterval:   2 * time.Second,
		},
		{
			name:               "custom values",
			args:               []string{"-a", "localhost:8888", "-r", "3", "-p", "1", "-k", "supersecret"},
			wantServerAddress:  "localhost:8888",
			wantReportInterval: 3 * time.Second,
			wantPollInterval:   time.Second,
			wantKey:            "supersecret",
		},
		{
			name:      "unknown flag",
			args:      []string{"-unknown"},
			wantError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config, err := ParseConfig[AgentConfig](test.args)
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
			if config.Key != test.wantKey {
				t.Fatalf("key = %q, want %q", config.Key, test.wantKey)
			}
		})
	}
}

func TestParseAgentConfigEnvironmentIntervalsInSeconds(t *testing.T) {
	t.Setenv("ADDRESS", "localhost:8888")
	t.Setenv("REPORT_INTERVAL", "5")
	t.Setenv("POLL_INTERVAL", "1")
	t.Setenv("KEY", "supersecret")

	config, err := ParseConfig[AgentConfig](nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if config.ReportInterval != 5*time.Second {
		t.Fatalf("report interval = %s, want %s", config.ReportInterval, 5*time.Second)
	}
	if config.PollInterval != time.Second {
		t.Fatalf("poll interval = %s, want %s", config.PollInterval, time.Second)
	}
	if config.Key != "supersecret" {
		t.Fatalf("key = %q, want %q", config.Key, "supersecret")
	}
}
