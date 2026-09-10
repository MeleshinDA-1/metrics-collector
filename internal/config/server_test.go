package config

import "testing"

func TestParseServerConfig(t *testing.T) {
	t.Setenv("DATABASE_DSN", "")
	t.Setenv("KEY", "")

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
				"-d", "postgres://flag-dsn",
				"-k", "supersecret",
			},
			wantConfig: ServerConfig{
				Address:                  "localhost:8888",
				StoreInterval:            0,
				FileStoragePath:          "metrics.json",
				Restore:                  false,
				PostgresConnectionString: "postgres://flag-dsn",
				Key:                      "supersecret",
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
	t.Setenv("DATABASE_DSN", "postgres://environment-dsn")
	t.Setenv("KEY", "environment-key")

	config, err := ParseConfig[ServerConfig]([]string{
		"-a", "localhost:8888",
		"-i", "10",
		"-f", "metrics.json",
		"-r=false",
		"-d", "postgres://flag-dsn",
		"-k", "flag-key",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := ServerConfig{
		Address:                  "localhost:8888",
		StoreInterval:            15,
		FileStoragePath:          "metrics.json",
		Restore:                  false,
		PostgresConnectionString: "postgres://environment-dsn",
		Key:                      "environment-key",
	}
	if config != want {
		t.Fatalf("config = %+v, want %+v", config, want)
	}
}
