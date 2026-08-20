// Package health implements Autorix's uniform health, readiness and identity
// contract (P1-S1): GET /health/alive, GET /health/ready and GET /info,
// identical across every engine.
//
// Liveness never consults a dependency — it answers as long as the process
// can schedule a goroutine. Readiness runs every registered check and fails
// closed: any single failing check makes the whole engine "not_ready".
package health

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// CheckFunc is one named readiness check (e.g. "postgres"). It must respect
// ctx's deadline and return promptly when it is exceeded.
type CheckFunc func(ctx context.Context) error

// Info is the payload served on GET /info.
type Info struct {
	Engine        string   `json:"engine"`
	Version       string   `json:"version"`
	BuildSHA      string   `json:"build_sha"`
	SchemaVersion string   `json:"schema_version"`
	Capabilities  []string `json:"capabilities"`
	InstanceID    string   `json:"instance_id"`
	StartedAt     time.Time
}

// InfoProvider returns the current Info snapshot, evaluated on every request
// so uptime and instance metadata stay accurate.
type InfoProvider func() Info

// Checker owns the set of named readiness checks for one engine.
type Checker struct {
	checks map[string]CheckFunc
}

// NewChecker returns an empty Checker. An engine with no dependencies is
// ready by construction — register checks only for things that can fail.
func NewChecker() *Checker {
	return &Checker{checks: make(map[string]CheckFunc)}
}

// Register adds a named readiness check. Registering the same name twice
// overwrites the previous check.
func (c *Checker) Register(name string, fn CheckFunc) {
	c.checks[name] = fn
}

// Ready runs every registered check against ctx and reports whether all of
// them passed, alongside a per-check result: "ok" or "error: <cause>".
func (c *Checker) Ready(ctx context.Context) (bool, map[string]string) {
	results := make(map[string]string, len(c.checks))
	ok := true
	for name, fn := range c.checks {
		if err := fn(ctx); err != nil {
			results[name] = "error: " + err.Error()
			ok = false
			continue
		}
		results[name] = "ok"
	}
	return ok, results
}

// Handler serves the three HTTP endpoints of the contract over one Checker
// and InfoProvider.
type Handler struct {
	checker *Checker
	info    InfoProvider
}

// NewHandler wires a Checker and an InfoProvider into a Handler.
func NewHandler(checker *Checker, info InfoProvider) *Handler {
	return &Handler{checker: checker, info: info}
}

// Register mounts GET /health/alive, GET /health/ready and GET /info on mux.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /health/alive", h.Alive)
	mux.HandleFunc("GET /health/ready", h.Ready)
	mux.HandleFunc("GET /info", h.Info)
}

// Alive answers process liveness. It never consults a dependency and never
// fails while the process itself is scheduling requests.
func (h *Handler) Alive(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "alive"})
}

// Ready runs the registered checks and reports 200 "ready" only when every
// one of them passes; otherwise 503 "not_ready" with the per-check causes.
func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	ok, results := h.checker.Ready(r.Context())

	status := "ready"
	code := http.StatusOK
	if !ok {
		status = "not_ready"
		code = http.StatusServiceUnavailable
	}

	writeJSON(w, code, map[string]interface{}{
		"status": status,
		"checks": results,
	})
}

// Info reports engine identity: type, version, build SHA, schema version,
// declared capabilities, instance id and uptime.
func (h *Handler) Info(w http.ResponseWriter, r *http.Request) {
	info := h.info()
	uptime := time.Duration(0)
	if !info.StartedAt.IsZero() {
		uptime = time.Since(info.StartedAt)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"engine":         info.Engine,
		"version":        info.Version,
		"build_sha":      info.BuildSHA,
		"schema_version": info.SchemaVersion,
		"capabilities":   info.Capabilities,
		"instance_id":    info.InstanceID,
		"uptime_seconds": uptime.Seconds(),
	})
}

func writeJSON(w http.ResponseWriter, code int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(body)
}
