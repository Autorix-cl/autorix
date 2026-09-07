package postgres_test

import (
	"context"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/autorix/aegis/internal/core"
	"github.com/autorix/aegis/internal/storage/postgres"
	"github.com/autorix/platform/pgtest"
)

func newTestStore(t *testing.T) *postgres.PostgresStore {
	t.Helper()
	pool := pgtest.StartPostgres(t, "../../../migrations")
	store, err := postgres.NewPostgresStore(context.Background(), pool)
	if err != nil {
		t.Fatalf("failed to create postgres store: %v", err)
	}
	t.Cleanup(store.Close)
	return store
}

func testRule(id, path string) core.Rule {
	return core.Rule{ID: id, Description: id, Match: core.MatchConfig{URL: path, Methods: []string{"GET"}}, Upstream: core.UpstreamConfig{URL: "http://backend.internal"}}
}

func TestPostgresStore_CRUDAndMatching(t *testing.T) {
	store := newTestStore(t)

	r1 := core.Rule{
		ID:          "rule-public",
		Description: "Public API",
		OrderIdx:    10,
		Match: core.MatchConfig{
			URL:     "/public/<.*>",
			Methods: []string{"GET"},
		},
		Authenticators: []core.HandlerConfig{{Handler: "anonymous"}},
		Authorizer:     core.HandlerConfig{Handler: "allow"},
		Upstream:       core.UpstreamConfig{URL: "http://public.service"},
	}

	created, err := store.Create(r1)
	if err != nil {
		t.Fatalf("Create rule: %v", err)
	}
	if created.ID != "rule-public" {
		t.Errorf("expected ID rule-public, got %s", created.ID)
	}

	// Test Get
	got, err := store.Get("rule-public")
	if err != nil {
		t.Fatalf("Get rule: %v", err)
	}
	if got.Description != "Public API" || got.Match.URL != "/public/<.*>" {
		t.Errorf("unexpected rule fetched: %+v", got)
	}

	// Test Match
	req := httptest.NewRequest("GET", "/public/users", nil)
	matched, err := store.Match(req)
	if err != nil {
		t.Fatalf("Match error: %v", err)
	}
	if matched.ID != "rule-public" {
		t.Errorf("expected match ID rule-public, got %s", matched.ID)
	}

	// Test Update
	got.Description = "Updated Public API"
	updated, err := store.Update("rule-public", got)
	if err != nil {
		t.Fatalf("Update error: %v", err)
	}
	if updated.Description != "Updated Public API" {
		t.Errorf("expected updated description, got %s", updated.Description)
	}

	// Test Delete
	if err := store.Delete("rule-public"); err != nil {
		t.Fatalf("Delete error: %v", err)
	}
	_, err = store.Get("rule-public")
	if err == nil {
		t.Fatalf("expected error after delete, got nil")
	}
}

func TestPostgresStore_Reorder(t *testing.T) {
	store := newTestStore(t)

	rules := []core.Rule{
		{
			ID:       "catch-all",
			OrderIdx: 10,
			Match:    core.MatchConfig{URL: "/api/<.*>", Methods: []string{"GET"}},
			Upstream: core.UpstreamConfig{URL: "http://backend/all"},
		},
		{
			ID:       "specific",
			OrderIdx: 20,
			Match:    core.MatchConfig{URL: "/api/special", Methods: []string{"GET"}},
			Upstream: core.UpstreamConfig{URL: "http://backend/special"},
		},
	}

	for _, r := range rules {
		if _, err := store.Create(r); err != nil {
			t.Fatalf("Create(%s): %v", r.ID, err)
		}
	}

	// Initially catch-all (order_idx=10) matches /api/special first
	req := httptest.NewRequest("GET", "/api/special", nil)
	m, err := store.Match(req)
	if err != nil || m.ID != "catch-all" {
		t.Fatalf("expected initial match catch-all, got %s (err: %v)", m.ID, err)
	}

	// Reorder so 'specific' comes before 'catch-all'
	if err := store.Reorder([]string{"specific", "catch-all"}); err != nil {
		t.Fatalf("Reorder error: %v", err)
	}

	// Now specific must match /api/special
	m2, err := store.Match(req)
	if err != nil || m2.ID != "specific" {
		t.Fatalf("expected match specific after reorder, got %s (err: %v)", m2.ID, err)
	}
}

func TestPostgresStore_RollbackAndVersions(t *testing.T) {
	store := newTestStore(t)

	// Step 1: Create rule v1
	_, err := store.Create(core.Rule{
		ID:       "rule-v1",
		Match:    core.MatchConfig{URL: "/v1", Methods: []string{"GET"}},
		Upstream: core.UpstreamConfig{URL: "http://v1"},
	})
	if err != nil {
		t.Fatalf("Create rule-v1: %v", err)
	}

	// Step 2: Create rule v2
	_, err = store.Create(core.Rule{
		ID:       "rule-v2",
		Match:    core.MatchConfig{URL: "/v2", Methods: []string{"GET"}},
		Upstream: core.UpstreamConfig{URL: "http://v2"},
	})
	if err != nil {
		t.Fatalf("Create rule-v2: %v", err)
	}

	versions, err := store.GetVersions()
	if err != nil {
		t.Fatalf("GetVersions: %v", err)
	}
	if len(versions) < 2 {
		t.Fatalf("expected at least 2 versions, got %d", len(versions))
	}

	// Rollback to version 1 (which only had rule-v1)
	v1Num := versions[len(versions)-1].Version
	if err := store.Rollback(v1Num); err != nil {
		t.Fatalf("Rollback to version %d failed: %v", v1Num, err)
	}

	// Verify only rule-v1 is present
	list := store.List()
	if len(list) != 1 || list[0].ID != "rule-v1" {
		t.Fatalf("expected only rule-v1 after rollback, got %+v", list)
	}
}

func TestPostgresStore_ImportExport(t *testing.T) {
	store := newTestStore(t)

	importedRules := []core.Rule{
		{
			ID:          "imp-1",
			Description: "Imported 1",
			Match:       core.MatchConfig{URL: "/imp1", Methods: []string{"GET"}},
			Upstream:    core.UpstreamConfig{URL: "http://imp1"},
		},
		{
			ID:          "imp-2",
			Description: "Imported 2",
			Match:       core.MatchConfig{URL: "/imp2", Methods: []string{"POST"}},
			Upstream:    core.UpstreamConfig{URL: "http://imp2"},
		},
	}

	if err := store.Import(importedRules); err != nil {
		t.Fatalf("Import error: %v", err)
	}

	exported := store.Export()
	if len(exported) != 2 {
		t.Fatalf("expected 2 exported rules, got %d", len(exported))
	}
	if exported[0].ID != "imp-1" || exported[1].ID != "imp-2" {
		t.Fatalf("unexpected exported rules: %+v", exported)
	}
}

func TestPostgresStore_BootstrapSeedsOnceAcrossConcurrentStartsAndRestart(t *testing.T) {
	pool := pgtest.StartPostgres(t, "../../../migrations")
	ctx := context.Background()
	seedPath := writeBootstrapRules(t, `
- id: "seed-health"
  description: "bootstrap health"
  match:
    url: "/health"
    methods: ["GET"]
  authenticators: []
  authorizer: {}
  mutators: []
  upstream:
    url: "http://health.internal"
- id: "seed-api"
  description: "bootstrap api"
  match:
    url: "/api/<.*>"
    methods: ["GET"]
  authenticators: []
  authorizer: {}
  mutators: []
  upstream:
    url: "http://api.internal"
`)

	const replicas = 4
	stores := make(chan *postgres.PostgresStore, replicas)
	errs := make(chan error, replicas)
	var wg sync.WaitGroup
	for range replicas {
		wg.Add(1)
		go func() {
			defer wg.Done()
			store, err := postgres.NewPostgresStoreWithBootstrap(ctx, pool, seedPath)
			if err == nil {
				stores <- store
			}
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	close(stores)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent bootstrap: %v", err)
		}
	}

	var count, versions int
	if err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM rules").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM rule_versions").Scan(&versions); err != nil {
		t.Fatal(err)
	}
	if count != 2 || versions != 1 {
		t.Fatalf("rules=%d versions=%d, want exactly 2 seeded rules and one snapshot", count, versions)
	}

	var initialized []*postgres.PostgresStore
	for store := range stores {
		initialized = append(initialized, store)
	}
	if len(initialized) == 0 {
		t.Fatal("no store initialized")
	}
	first := initialized[0]
	t.Cleanup(first.Close)
	for _, store := range initialized[1:] {
		store.Close()
	}
	managed, err := first.Get("seed-health")
	if err != nil {
		t.Fatal(err)
	}
	managed.Description = "operator managed"
	if _, err := first.Update(managed.ID, managed); err != nil {
		t.Fatal(err)
	}

	replacementPath := writeBootstrapRules(t, `
- id: "replacement"
  description: "must not replace operator rules"
  match:
    url: "/replacement"
    methods: ["GET"]
  authenticators: []
  authorizer: {}
  mutators: []
  upstream:
    url: "http://replacement.internal"
`)
	restarted, err := postgres.NewPostgresStoreWithBootstrap(ctx, pool, replacementPath)
	if err != nil {
		t.Fatalf("restart bootstrap: %v", err)
	}
	t.Cleanup(restarted.Close)
	got, err := restarted.Get("seed-health")
	if err != nil || got.Description != "operator managed" {
		t.Fatalf("existing rules changed on restart: rule=%+v err=%v", got, err)
	}
	if _, err := restarted.Get("replacement"); err == nil {
		t.Fatal("restart imported replacement bootstrap rules over existing database rules")
	}
}

func TestPostgresStore_PropagatesRuleChangesAndStopsOnClose(t *testing.T) {
	pool := pgtest.StartPostgres(t, "../../../migrations")
	ctx := context.Background()
	writer, err := postgres.NewPostgresStore(ctx, pool)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(writer.Close)
	reader, err := postgres.NewPostgresStore(ctx, pool)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(reader.Close)
	if _, err := writer.Create(testRule("replica-visible", "/replica")); err != nil {
		t.Fatalf("writer create: %v", err)
	}
	waitForRule(t, reader, "replica-visible")
	reader.Close()
	if _, err := writer.Create(testRule("after-close", "/after-close")); err != nil {
		t.Fatalf("writer create after close: %v", err)
	}
	time.Sleep(150 * time.Millisecond)
	if _, err := reader.Get("after-close"); err == nil {
		t.Fatal("closed store reloaded from postgres notification")
	}
}

func waitForRule(t *testing.T, store *postgres.PostgresStore, id string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := store.Get(id); err == nil {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for propagated rule %q", id)
}

func writeBootstrapRules(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "rules.yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write bootstrap rules: %v", err)
	}
	return path
}
