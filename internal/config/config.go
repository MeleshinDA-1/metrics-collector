package config

import (
	"flag"
	"fmt"
	"io"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
)

type AgentConfig struct {
	ServerAddress  string        `env:"ADDRESS" flag:"a,default=localhost:8080" usage:"HTTP server address"`
	ReportInterval time.Duration `env:"REPORT_INTERVAL" flag:"r,default=10s" usage:"metrics report interval in seconds"`
	PollInterval   time.Duration `env:"POLL_INTERVAL" flag:"p,default=2s" usage:"metrics poll interval in seconds"`
}

type ServerConfig struct {
	Address         string `env:"ADDRESS" flag:"a,default=localhost:8080" usage:"HTTP server address"`
	StoreInterval   int    `env:"STORE_INTERVAL" flag:"i,default=300" usage:"metrics store interval in seconds"`
	FileStoragePath string `env:"FILE_STORAGE_PATH" flag:"f,default=metricsDataDefault" usage:"metrics storage path"`
	Restore         bool   `env:"RESTORE" flag:"r,default=true" usage:"should load metrics data from storage"`
}

func ParseAgentConfig() (AgentConfig, error) {
	return ParseConfig[AgentConfig](os.Args[1:])
}

func ParseServerConfig() (ServerConfig, error) {
	return ParseConfig[ServerConfig](os.Args[1:])
}

func ParseConfig[T any](args []string) (T, error) {
	cfg, err := parseConfigFromFlags[T](args)
	if err != nil {
		return *new(T), err
	}

	if err := parseConfigFromEnv(&cfg); err != nil {
		return *new(T), err
	}

	return cfg, nil
}

func validateEnvTags[T any]() error {
	typeValue := reflect.TypeOf((*T)(nil)).Elem()

	for _, field := range reflect.VisibleFields(typeValue) {
		envTag := field.Tag.Get("env")
		if envTag == "" {
			return fmt.Errorf("type %v: field %s has no env tag", typeValue, field.Name)
		}
	}

	return nil
}

func parseConfigFromEnv[T any](cfg *T) error {
	if err := validateEnvTags[T](); err != nil {
		return err
	}

	err := env.ParseWithOptions(cfg, env.Options{
		FuncMap: map[reflect.Type]env.ParserFunc{
			reflect.TypeOf(time.Duration(0)): func(s string) (interface{}, error) {
				return parseDuration(s)
			},
		}})
	if err != nil {
		return fmt.Errorf("parse config from environment: %w", err)
	}

	return nil
}

func parseConfigFromFlags[T any](args []string) (T, error) {
	var cfg T

	typeValue := reflect.TypeOf(cfg)
	flagSet := flag.NewFlagSet(typeValue.Name(), flag.ContinueOnError)
	flagSet.SetOutput(io.Discard)

	for _, field := range reflect.VisibleFields(typeValue) {
		if err := registerFlag(flagSet, field, &cfg); err != nil {
			return cfg, err
		}
	}

	if err := flagSet.Parse(args); err != nil {
		return cfg, err
	}

	return cfg, nil
}

func registerFlag[T any](flagSet *flag.FlagSet, field reflect.StructField, obj *T) error {
	paramName, defaultValue := parseTagFlagForConfig(field)
	usage := parseTagUsageForConfig(field)
	fieldValue := reflect.ValueOf(obj).Elem().FieldByIndex(field.Index)

	switch field.Type {
	case reflect.TypeOf(""):
		flagSet.StringVar(
			fieldValue.Addr().Interface().(*string),
			paramName,
			defaultValue,
			usage,
		)
	case reflect.TypeOf(time.Duration(0)):
		duration, err := parseDuration(defaultValue)
		if err != nil {
			return fmt.Errorf("field %s: parse default duration %q: %w", field.Name, defaultValue, err)
		}
		fieldValue.SetInt(int64(duration))
		flagSet.Func(paramName, usage, func(value string) error {
			duration, err := parseDuration(value)
			if err != nil {
				return err
			}
			fieldValue.SetInt(int64(duration))
			return nil
		})
	case reflect.TypeOf(0):
		value, err := strconv.Atoi(defaultValue)
		if err != nil {
			return fmt.Errorf("field %s: parse default int %q: %w", field.Name, defaultValue, err)
		}
		flagSet.IntVar(
			fieldValue.Addr().Interface().(*int),
			paramName,
			value,
			usage,
		)
	case reflect.TypeOf(false):
		value, err := strconv.ParseBool(defaultValue)
		if err != nil {
			return fmt.Errorf("field %s: parse default bool %q: %w", field.Name, defaultValue, err)
		}
		flagSet.BoolVar(
			fieldValue.Addr().Interface().(*bool),
			paramName,
			value,
			usage,
		)
	default:
		return fmt.Errorf("type %v: unsupported flag field %q", field.Type, field.Name)
	}

	return nil
}

func parseDuration(value string) (time.Duration, error) {
	seconds, err := strconv.Atoi(value)
	if err == nil {
		return time.Duration(seconds) * time.Second, nil
	}

	return time.ParseDuration(value)
}

func parseTagFlagForConfig(field reflect.StructField) (string, string) {
	tagValue := field.Tag.Get("flag")
	split := strings.SplitN(tagValue, ",", 2)
	if len(split) != 2 {
		panic(fmt.Sprintf(
			"type field %s: invalid flag tag %q, expected name,default=value",
			field.Name,
			tagValue,
		))
	}

	paramName := split[0]
	defaultValue, ok := strings.CutPrefix(split[1], "default=")
	if !ok {
		panic(fmt.Sprintf(
			"type field %s: invalid flag tag %q, expected name,default=value",
			field.Name,
			tagValue,
		))
	}
	return paramName, defaultValue
}

func parseTagUsageForConfig(field reflect.StructField) string {
	return field.Tag.Get("usage")
}
