package health_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/autorix/platform/health"
)

func TestChecker_ReadyWithNoChecks(t *testing.T) {
	c := health.NewChecker()

	ok, results := c.Ready(context.Background())

	if !ok {
		t.Fatalf("expected ready=true with no registered checks, got false (results=%v)", results)
	}
	if len(results) != 0 {
		t.Fatalf("expected no results, got %v", results)
	}
}

func TestChecker_ReadyAllPass(t *testing.T) {
	c := health.NewChecker()
	c.Register("postgres", func(ctx context.Context) error { return nil })
	c.Register("cache", func(ctx context.Context) error { return nil })

	ok, results := c.Ready(context.Background())

	if !ok {
		t.Fatalf("expected ready=true, got false (results=%v)", results)
	}
	if results["postgres"] != "ok" || results["cache"] != "ok" {
		t.Fatalf("expected all checks ok, got %v", results)
	}
}

func TestChecker_ReadyOneFails(t *testing.T) {
	c := health.NewChecker()
	c.Register("postgres", func(ctx context.Context) error { return errors.New("connection refused") })

	ok, results := c.Ready(context.Background())

	if ok {
		t.Fatalf("expected ready=false when a check fails")
	}
	if results["postgres"] != "error: connection refused" {
		t.Fatalf("expected failing check message, got %q", results["postgres"])
	}
}

func TestChecker_ReadyRespectsContextTimeout(t *testing.T) {
	c := health.NewChecker()
	c.Register("slow", func(ctx context.Context) error {
		select {
		case <-time.After(200 * time.Millisecond):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	ok, results := c.Ready(ctx)

	if ok {
		t.Fatalf("expected ready=false when check exceeds context deadline")
	}
	if results["slow"] == "ok" {
		t.Fatalf("expected slow check to fail on timeout, got ok")
	}
}

func TestHandler_Alive_AlwaysHealthyRegardlessOfChecks(t *testing.T) {
	c := health.NewChecker()
	c.Register("postgres", func(ctx context.Context) error { return errors.New("down") })
	h := health.NewHandler(c, func() health.Info { return health.Info{Engine: "ego"} })

	req := httptest.NewRequest(http.MethodGet, "/health/alive", nil)
	rec := httptest.NewRecorder()
	h.Alive(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("invalid json body: %v", err)
	}
	if body["status"] != "alive" {
		t.Fatalf(`expected status="alive", got %q`, body["status"])
	}
}

func TestHandler_Ready_200WhenAllChecksPass(t *testing.T) {
	c := health.NewChecker()
	c.Register("postgres", func(ctx context.Context) error { return nil })
	h := health.NewHandler(c, func() health.Info { return health.Info{Engine: "ego"} })

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	rec := httptest.NewRecorder()
	h.Ready(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var body struct {
		Status string            `json:"status"`
		Checks map[string]string `json:"checks"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("invalid json body: %v", err)
	}
	if body.Status != "ready" {
		t.Fatalf(`expected status="ready", got %q`, body.Status)
	}
	if body.Checks["postgres"] != "ok" {
		t.Fatalf("expected postgres check ok, got %v", body.Checks)
	}
}

func TestHandler_Ready_503WhenACheckFails(t *testing.T) {
	c := health.NewChecker()
	c.Register("postgres", func(ctx context.Context) error { return errors.New("pool exhausted") })
	h := health.NewHandler(c, func() health.Info { return health.Info{Engine: "ego"} })

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	rec := httptest.NewRecorder()
	h.Ready(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
	var body struct {
		Status string            `json:"status"`
		Checks map[string]string `json:"checks"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("invalid json body: %v", err)
	}
	if body.Status != "not_ready" {
		t.Fatalf(`expected status="not_ready", got %q`, body.Status)
	}
}

func TestHandler_Info_ReportsProvidedFields(t *testing.T) {
	c := health.NewChecker()
	started := time.Now().Add(-90 * time.Second)
	h := health.NewHandler(c, func() health.Info {
		return health.Info{
			Engine:        "ego",
			Version:       "1.4.0",
			BuildSHA:      "abc123",
			SchemaVersion: "7",
			Capabilities:  []string{"credential.argon2id", "session"},
			InstanceID:    "ego-7f8c",
			StartedAt:     started,
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/info", nil)
	rec := httptest.NewRecorder()
	h.Info(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var body map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("invalid json body: %v", err)
	}
	if body["engine"] != "ego" || body["version"] != "1.4.0" || body["build_sha"] != "abc123" {
		t.Fatalf("unexpected /info body: %v", body)
	}
	if body["schema_version"] != "7" || body["instance_id"] != "ego-7f8c" {
		t.Fatalf("unexpected /info body: %v", body)
	}
	uptime, ok := body["uptime_seconds"].(float64)
	if !ok || uptime < 89 {
		t.Fatalf("expected uptime_seconds >= 89, got %v", body["uptime_seconds"])
	}
}

func TestHandler_RegisterRoutes_MountsAllThree(t *testing.T) {
	c := health.NewChecker()
	h := health.NewHandler(c, func() health.Info { return health.Info{Engine: "ego"} })
	mux := http.NewServeMux()
	h.Register(mux)

	for _, path := range []string{"/health/alive", "/health/ready", "/info"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code == http.StatusNotFound {
			t.Fatalf("expected %s to be registered, got 404", path)
		}
	}
}
