package config

import (
	"flag"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
)

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

func parseConfigFromEnv[T any](cfg *T) error {
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
	parsedTag, err := parseTagFlagForConfig(field)
	if err != nil {
		return err
	}
	paramName := parsedTag.name
	defaultValue := parsedTag.defaultValue
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

type flagTag struct {
	name         string
	defaultValue string
}

func parseTagFlagForConfig(field reflect.StructField) (flagTag, error) {
	tagValue := field.Tag.Get("flag")
	paramName, defaultTag, hasSeparator := strings.Cut(tagValue, ",")
	defaultValue, hasDefault := strings.CutPrefix(defaultTag, "default=")
	if !hasSeparator || !hasDefault {
		return flagTag{}, fmt.Errorf(
			"type field %s: invalid flag tag %q; expected format %q",
			field.Name,
			tagValue,
			"<name>,default=<value>",
		)
	}

	return flagTag{
		name:         paramName,
		defaultValue: defaultValue,
	}, nil
}

func parseTagUsageForConfig(field reflect.StructField) string {
	return field.Tag.Get("usage")
}
