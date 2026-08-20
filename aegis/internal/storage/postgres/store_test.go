package postgres_test

import (
	"context"
	"net/http/httptest"
	"testing"

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
	return store
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
