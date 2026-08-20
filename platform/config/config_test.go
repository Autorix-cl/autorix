package config_test

import (
	"testing"
	"time"

	"github.com/autorix/platform/config"
)

type sampleConfig struct {
	Port        string        `env:"PORT" envDefault:"4433"`
	DatabaseURL string        `env:"DATABASE_URL" required:"true"`
	LogLevel    string        `env:"LOG_LEVEL" envDefault:"info"`
	MaxConns    int           `env:"MAX_CONNS" envDefault:"10"`
	Debug       bool          `env:"DEBUG" envDefault:"false"`
	Timeout     time.Duration `env:"TIMEOUT" envDefault:"5s"`
}

func TestLoad_FillsDefaultsWhenUnset(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x")

	var cfg sampleConfig
	if err := config.Load(&cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Port != "4433" {
		t.Errorf("expected default port 4433, got %q", cfg.Port)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("expected default log level info, got %q", cfg.LogLevel)
	}
	if cfg.MaxConns != 10 {
		t.Errorf("expected default max conns 10, got %d", cfg.MaxConns)
	}
	if cfg.Debug != false {
		t.Errorf("expected default debug false, got %v", cfg.Debug)
	}
	if cfg.Timeout != 5*time.Second {
		t.Errorf("expected default timeout 5s, got %v", cfg.Timeout)
	}
}

func TestLoad_EnvOverridesDefault(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("PORT", "9999")
	t.Setenv("MAX_CONNS", "42")
	t.Setenv("DEBUG", "true")
	t.Setenv("TIMEOUT", "30s")

	var cfg sampleConfig
	if err := config.Load(&cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Port != "9999" {
		t.Errorf("expected overridden port 9999, got %q", cfg.Port)
	}
	if cfg.MaxConns != 42 {
		t.Errorf("expected overridden max conns 42, got %d", cfg.MaxConns)
	}
	if cfg.Debug != true {
		t.Errorf("expected overridden debug true, got %v", cfg.Debug)
	}
	if cfg.Timeout != 30*time.Second {
		t.Errorf("expected overridden timeout 30s, got %v", cfg.Timeout)
	}
}

func TestLoad_FailsFastWhenRequiredFieldMissing(t *testing.T) {
	var cfg sampleConfig
	err := config.Load(&cfg)
	if err == nil {
		t.Fatalf("expected an error when a required field has no env var and no default")
	}
}

func TestLoad_FailsWithClearMessageOnBadType(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("MAX_CONNS", "not-a-number")

	var cfg sampleConfig
	err := config.Load(&cfg)
	if err == nil {
		t.Fatalf("expected an error for an unparsable int field")
	}
	if !contains(err.Error(), "MAX_CONNS") {
		t.Fatalf("expected error to name the offending env var MAX_CONNS, got: %v", err)
	}
}

func TestLoad_RejectsNonPointer(t *testing.T) {
	var cfg sampleConfig
	err := config.Load(cfg)
	if err == nil {
		t.Fatalf("expected an error when dst is not a pointer to a struct")
	}
}

func contains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
