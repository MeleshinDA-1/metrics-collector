package config

import "testing"

func TestParseServerConfig(t *testing.T) {
	t.Setenv("DATABASE_DSN", "")
	t.Setenv("KEY", "")
	t.Setenv("AUDIT_FILE", "")
	t.Setenv("AUDIT_URL", "")

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
				"--audit-file", "audit.jsonl",
				"--audit-url", "http://localhost:9000/audit",
			},
			wantConfig: ServerConfig{
				Address:                  "localhost:8888",
				StoreInterval:            0,
				FileStoragePath:          "metrics.json",
				Restore:                  false,
				PostgresConnectionString: "postgres://flag-dsn",
				Key:                      "supersecret",
				AuditFile:                "audit.jsonl",
				AuditURL:                 "http://localhost:9000/audit",
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
	t.Setenv("AUDIT_FILE", "env-audit.jsonl")
	t.Setenv("AUDIT_URL", "http://localhost:9001/audit")

	config, err := ParseConfig[ServerConfig]([]string{
		"-a", "localhost:8888",
		"-i", "10",
		"-f", "metrics.json",
		"-r=false",
		"-d", "postgres://flag-dsn",
		"-k", "flag-key",
		"--audit-file", "flag-audit.jsonl",
		"--audit-url", "http://localhost:9000/audit",
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
		AuditFile:                "env-audit.jsonl",
		AuditURL:                 "http://localhost:9001/audit",
	}
	if config != want {
		t.Fatalf("config = %+v, want %+v", config, want)
	}
}
