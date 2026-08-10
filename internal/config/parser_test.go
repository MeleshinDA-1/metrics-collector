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
