package log_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/autorix/platform/log"
)

func TestNew_EmitsJSONWithAgreedFieldVocabulary(t *testing.T) {
	var buf bytes.Buffer
	logger := log.New(log.Config{
		Engine:     "ego",
		InstanceID: "ego-7f8c",
		Level:      "info",
	}, &buf)

	logger.Info("session created", "request_id", "req-1", "tenant_id", "acme", "actor", "user-42")

	var entry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("expected valid JSON log line, got %q: %v", buf.String(), err)
	}

	for _, field := range []struct{ key, want string }{
		{"engine", "ego"},
		{"instance_id", "ego-7f8c"},
		{"request_id", "req-1"},
		{"tenant_id", "acme"},
		{"actor", "user-42"},
	} {
		if entry[field.key] != field.want {
			t.Errorf("expected %s=%q, got %v", field.key, field.want, entry[field.key])
		}
	}
	if entry["msg"] != "session created" {
		t.Errorf("expected msg field, got %v", entry["msg"])
	}
}

func TestNew_LevelFiltersBelowConfigured(t *testing.T) {
	var buf bytes.Buffer
	logger := log.New(log.Config{Engine: "ego", Level: "warn"}, &buf)

	logger.Info("should be filtered out")
	logger.Warn("should appear")

	out := buf.String()
	if strings.Contains(out, "should be filtered out") {
		t.Fatalf("expected info-level log to be filtered at warn level, got: %s", out)
	}
	if !strings.Contains(out, "should appear") {
		t.Fatalf("expected warn-level log to appear, got: %s", out)
	}
}

func TestNew_DefaultsToInfoOnUnrecognizedLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := log.New(log.Config{Engine: "ego", Level: "not-a-real-level"}, &buf)

	logger.Info("visible at default info level")

	if !strings.Contains(buf.String(), "visible at default info level") {
		t.Fatalf("expected info level to be the fallback default")
	}
}

func TestNew_ReturnsAStandardSlogLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := log.New(log.Config{Engine: "ego", Level: "info"}, &buf)

	if logger == nil {
		t.Fatal("expected New to return a non-nil *slog.Logger")
	}
}
