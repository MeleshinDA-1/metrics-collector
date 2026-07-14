package config

import (
	"reflect"
	"testing"
	"time"
)

func TestParseTagFlagForConfigPanicsOnInvalidTag(t *testing.T) {
	tests := []struct {
		name string
		tag  reflect.StructTag
	}{
		{name: "missing separator", tag: `flag:"a"`},
		{name: "missing default prefix", tag: `flag:"a,localhost:8080"`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			field := reflect.StructField{Name: "Address", Tag: test.tag}

			defer func() {
				if recover() == nil {
					t.Fatal("parseTagFlagForConfig did not panic")
				}
			}()

			parseTagFlagForConfig(field)
		})
	}
}

func TestParseConfigReturnsInvalidEnvironmentError(t *testing.T) {
	t.Setenv("ADDRESS", "localhost:8080")
	t.Setenv("REPORT_INTERVAL", "invalid")
	t.Setenv("POLL_INTERVAL", "2s")

	_, err := ParseConfig[AgentConfig](nil)
	if err == nil {
		t.Fatal("ParseConfig error = nil, want non-nil")
	}
}

func TestParseConfigReturnsInvalidDefaultDurationError(t *testing.T) {
	type invalidConfig struct {
		Interval time.Duration `flag:"i,default=invalid" usage:"invalid interval"`
	}

	_, err := parseConfigFromFlags[invalidConfig](nil)
	if err == nil {
		t.Fatal("parseConfigFromFlags error = nil, want non-nil")
	}
}

func TestParseConfigReturnsMissingEnvTagError(t *testing.T) {
	type invalidConfig struct {
		Address string `flag:"a,default=localhost:8080"`
	}

	_, err := ParseConfig[invalidConfig](nil)
	if err == nil {
		t.Fatal("ParseConfig error = nil, want non-nil")
	}
}

func TestParseConfigReturnsUnsupportedFlagTypeError(t *testing.T) {
	type invalidConfig struct {
		Count int `env:"COUNT" flag:"c,default=1"`
	}

	_, err := ParseConfig[invalidConfig](nil)
	if err == nil {
		t.Fatal("ParseConfig error = nil, want non-nil")
	}
}

func TestParseServerConfig(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantAddress string
		wantError   bool
	}{
		{
			name:        "default address",
			wantAddress: "localhost:8080",
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
			config, err := ParseConfig[ServerConfig](test.args)
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
			wantServerAddress:  "localhost:8080",
			wantReportInterval: 10 * time.Second,
			wantPollInterval:   2 * time.Second,
		},
		{
			name:               "custom values",
			args:               []string{"-a", "localhost:8888", "-r", "3s", "-p", "1s"},
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
		})
	}
}
