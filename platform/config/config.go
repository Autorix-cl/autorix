// Package config loads engine configuration from environment variables into
// a typed struct via `env`/`envDefault`/`required` tags (P1-S1-T2... P1-S2-T1),
// replacing the scattered os.Getenv-with-inline-default pattern duplicated in
// every engine's main.go. Loading fails fast at boot: a missing required
// variable or an unparsable value is a startup error, never a zero value
// silently propagated into running code.
package config

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"time"
)

// Load populates dst — a pointer to a struct — from environment variables
// named by each field's `env` tag. `envDefault` supplies a value used only
// when the variable is unset; `required:"true"` makes an unset variable
// with no default a load error. Supported field types: string, int, bool,
// time.Duration.
func Load(dst interface{}) error {
	v := reflect.ValueOf(dst)
	if v.Kind() != reflect.Pointer || v.IsNil() || v.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("config: Load requires a non-nil pointer to a struct, got %T", dst)
	}

	elem := v.Elem()
	t := elem.Type()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		envName := field.Tag.Get("env")
		if envName == "" {
			continue
		}

		raw, isSet := os.LookupEnv(envName)
		if !isSet {
			if def, hasDefault := field.Tag.Lookup("envDefault"); hasDefault {
				raw = def
			} else if field.Tag.Get("required") == "true" {
				return fmt.Errorf("config: required environment variable %s is not set", envName)
			} else {
				continue
			}
		}

		if err := setField(elem.Field(i), envName, raw); err != nil {
			return err
		}
	}

	return nil
}

func setField(field reflect.Value, envName, raw string) error {
	switch field.Interface().(type) {
	case time.Duration:
		d, err := time.ParseDuration(raw)
		if err != nil {
			return fmt.Errorf("config: %s=%q is not a valid duration: %w", envName, raw, err)
		}
		field.Set(reflect.ValueOf(d))
		return nil
	}

	switch field.Kind() {
	case reflect.String:
		field.SetString(raw)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return fmt.Errorf("config: %s=%q is not a valid integer: %w", envName, raw, err)
		}
		field.SetInt(n)
	case reflect.Bool:
		b, err := strconv.ParseBool(raw)
		if err != nil {
			return fmt.Errorf("config: %s=%q is not a valid boolean: %w", envName, raw, err)
		}
		field.SetBool(b)
	default:
		return fmt.Errorf("config: field for %s has unsupported type %s", envName, field.Type())
	}
	return nil
}
