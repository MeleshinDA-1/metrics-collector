package config

import (
	"testing"
	"time"
)

func TestParseAgentConfig(t *testing.T) {
	t.Setenv("KEY", "")
	t.Setenv("RATE_LIMIT", "")

	tests := []struct {
		name               string
		args               []string
		wantServerAddress  string
		wantReportInterval time.Duration
		wantPollInterval   time.Duration
		wantKey            string
		wantRateLimit      int
		wantError          bool
	}{
		{
			name:               "defaults",
			wantServerAddress:  "localhost:8080",
			wantReportInterval: 10 * time.Second,
			wantPollInterval:   2 * time.Second,
			wantRateLimit:      1,
		},
		{
			name:               "custom values",
			args:               []string{"-a", "localhost:8888", "-r", "3", "-p", "1", "-k", "supersecret", "-l", "5"},
			wantServerAddress:  "localhost:8888",
			wantReportInterval: 3 * time.Second,
			wantPollInterval:   time.Second,
			wantKey:            "supersecret",
			wantRateLimit:      5,
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
			if config.RateLimit != test.wantRateLimit {
				t.Fatalf("rate limit = %d, want %d", config.RateLimit, test.wantRateLimit)
			}
		})
	}
}

func TestParseAgentConfigEnvironmentIntervalsInSeconds(t *testing.T) {
	t.Setenv("ADDRESS", "localhost:8888")
	t.Setenv("REPORT_INTERVAL", "5")
	t.Setenv("POLL_INTERVAL", "1")
	t.Setenv("KEY", "supersecret")
	t.Setenv("RATE_LIMIT", "7")

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
	if config.RateLimit != 7 {
		t.Fatalf("rate limit = %d, want %d", config.RateLimit, 7)
	}
}

func TestAgentConfigValidate(t *testing.T) {
	valid := AgentConfig{
		ServerAddress:  "localhost:8080",
		ReportInterval: 10 * time.Second,
		PollInterval:   2 * time.Second,
		RateLimit:      1,
	}

	tests := []struct {
		name      string
		config    AgentConfig
		wantError bool
	}{
		{
			name:   "valid config",
			config: valid,
		},
		{
			name: "zero poll interval",
			config: func() AgentConfig {
				config := valid
				config.PollInterval = 0
				return config
			}(),
			wantError: true,
		},
		{
			name: "negative poll interval",
			config: func() AgentConfig {
				config := valid
				config.PollInterval = -time.Second
				return config
			}(),
			wantError: true,
		},
		{
			name: "zero report interval",
			config: func() AgentConfig {
				config := valid
				config.ReportInterval = 0
				return config
			}(),
			wantError: true,
		},
		{
			name: "zero rate limit",
			config: func() AgentConfig {
				config := valid
				config.RateLimit = 0
				return config
			}(),
			wantError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.config.validate()
			if test.wantError && err == nil {
				t.Fatal("error = nil, want non-nil error")
			}
			if !test.wantError && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
