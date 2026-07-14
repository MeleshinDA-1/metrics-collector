package config

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"reflect"
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
	Address string `env:"ADDRESS" flag:"a,default=localhost:8080" usage:"HTTP server address"`
}

func ParseAgentConfig() (AgentConfig, error) {
	return ParseConfig[AgentConfig](os.Args[1:])
}

func ParseServerConfig() (ServerConfig, error) {
	return ParseConfig[ServerConfig](os.Args[1:])
}

func ParseConfig[T any](args []string) (T, error) {
	cfg, ok, err := parseConfigFromEnv[T]()
	if err != nil {
		return *new(T), err
	}
	if ok {
		log.Println("Using config from env:", cfg)
		return cfg, nil
	}

	cfg, err = parseConfigFromFlags[T](args)
	if err != nil {
		return *new(T), err
	}

	return cfg, nil
}

func isAllEnvArgsFound[T any]() (bool, error) {
	typeValue := reflect.TypeOf((*T)(nil)).Elem()

	for _, field := range reflect.VisibleFields(typeValue) {
		envTag := field.Tag.Get("env")
		if envTag == "" {
			return false, fmt.Errorf("type %v: field %s has no env tag", typeValue, field.Name)
		}
		if !isEnvSet(envTag) {
			return false, nil
		}
	}

	return true, nil
}

func isEnvSet(name string) bool {
	_, ok := os.LookupEnv(name)
	return ok
}

func parseConfigFromEnv[T any]() (T, bool, error) {
	allEnvArgsFound, err := isAllEnvArgsFound[T]()
	if err != nil {
		return *new(T), false, err
	}
	if !allEnvArgsFound {
		return *new(T), false, nil
	}

	cfg, err := env.ParseAsWithOptions[T](env.Options{
		FuncMap: map[reflect.Type]env.ParserFunc{
			reflect.TypeOf(time.Duration(0)): func(s string) (interface{}, error) {
				return time.ParseDuration(s)
			},
		}})
	if err != nil {
		return *new(T), false, fmt.Errorf("parse config from environment: %w", err)
	}

	return cfg, true, nil
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
		duration, err := time.ParseDuration(defaultValue)
		if err != nil {
			return fmt.Errorf("field %s: parse default duration %q: %w", field.Name, defaultValue, err)
		}
		flagSet.DurationVar(
			fieldValue.Addr().Interface().(*time.Duration),
			paramName,
			duration,
			usage,
		)
	default:
		return fmt.Errorf("type %v: unsupported flag field %q", field.Type, field.Name)
	}

	return nil
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
