package config

import (
	"reflect"
	"testing"
	"time"
)

func TestParseTagFlagForConfigReturnsErrorOnInvalidTag(t *testing.T) {
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

			_, err := parseTagFlagForConfig(field)
			if err == nil {
				t.Fatal("parseTagFlagForConfig error = nil, want non-nil")
			}
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

func TestConfigFieldsHaveEnvTags(t *testing.T) {
	configTypes := []reflect.Type{
		reflect.TypeOf(AgentConfig{}),
		reflect.TypeOf(ServerConfig{}),
	}

	for _, configType := range configTypes {
		t.Run(configType.Name(), func(t *testing.T) {
			for _, field := range reflect.VisibleFields(configType) {
				if field.Tag.Get("env") == "" {
					t.Errorf("%s.%s has no env tag", configType.Name(), field.Name)
				}
			}
		})
	}
}

func TestParseConfigReturnsUnsupportedFlagTypeError(t *testing.T) {
	type invalidConfig struct {
		Count float64 `env:"COUNT" flag:"c,default=1"`
	}

	_, err := ParseConfig[invalidConfig](nil)
	if err == nil {
		t.Fatal("ParseConfig error = nil, want non-nil")
	}
}

func TestParseServerConfig(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantConfig ServerConfig
		wantError  bool
	}{
		{
			name: "defaults",
			wantConfig: ServerConfig{
				Address:         "localhost:8080",
				StoreInterval:   300,
				FileStoragePath: "metricsDataDefault",
				Restore:         true,
			},
		},
		{
			name: "custom values",
			args: []string{
				"-a", "localhost:8888",
				"-i", "0",
				"-f", "metrics.json",
				"-r=false",
			},
			wantConfig: ServerConfig{
				Address:         "localhost:8888",
				StoreInterval:   0,
				FileStoragePath: "metrics.json",
				Restore:         false,
			},
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
			if config != test.wantConfig {
				t.Fatalf("config = %+v, want %+v", config, test.wantConfig)
			}
		})
	}
}

func TestParseServerConfigEnvironmentOverridesFlags(t *testing.T) {
	t.Setenv("ADDRESS", "")
	t.Setenv("STORE_INTERVAL", "15")
	t.Setenv("FILE_STORAGE_PATH", "")
	t.Setenv("RESTORE", "")

	config, err := ParseConfig[ServerConfig]([]string{
		"-a", "localhost:8888",
		"-i", "10",
		"-f", "metrics.json",
		"-r=false",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := ServerConfig{
		Address:         "localhost:8888",
		StoreInterval:   15,
		FileStoragePath: "metrics.json",
		Restore:         false,
	}
	if config != want {
		t.Fatalf("config = %+v, want %+v", config, want)
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

func TestParseAgentConfigEnvironmentIntervalsInSeconds(t *testing.T) {
	t.Setenv("ADDRESS", "localhost:8888")
	t.Setenv("REPORT_INTERVAL", "5")
	t.Setenv("POLL_INTERVAL", "1")

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
}
