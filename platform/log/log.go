// Package log builds a structured JSON logger over log/slog, shared by
// every engine (P1-S2-T2). Every engine used the bare `log` package before
// this, which produces plain text no aggregator can parse. The agreed field
// vocabulary — engine, instance_id, request_id, tenant_id, actor — is
// attached as base attributes (engine, instance_id) or passed per call site
// (request_id, tenant_id, actor); this package does not enforce the latter
// three beyond documenting the convention, since they vary per log call.
package log

import (
	"io"
	"log/slog"
	"os"
)

// Config configures a logger's identity and verbosity.
type Config struct {
	// Engine is this process's engine name (e.g. "ego"), attached to every
	// log line.
	Engine string
	// InstanceID is this process's instance id, attached to every log line
	// when non-empty.
	InstanceID string
	// Level is one of "debug", "info", "warn", "error" (case-insensitive).
	// An unrecognized value defaults to "info" rather than failing boot over
	// a logging misconfiguration.
	Level string
}

// New builds a JSON slog.Logger with cfg's identity attached to every line.
// w defaults to os.Stdout when nil.
func New(cfg Config, w io.Writer) *slog.Logger {
	if w == nil {
		w = os.Stdout
	}

	handler := slog.NewJSONHandler(w, &slog.HandlerOptions{Level: parseLevel(cfg.Level)})
	logger := slog.New(handler)

	attrs := []any{"engine", cfg.Engine}
	if cfg.InstanceID != "" {
		attrs = append(attrs, "instance_id", cfg.InstanceID)
	}
	return logger.With(attrs...)
}

func parseLevel(level string) slog.Level {
	var l slog.Level
	if err := l.UnmarshalText([]byte(level)); err != nil {
		return slog.LevelInfo
	}
	return l
}
