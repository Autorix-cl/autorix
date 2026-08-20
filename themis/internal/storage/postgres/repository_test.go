package postgres_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/autorix/platform/paging"
	"github.com/autorix/themis/internal/core"
	"github.com/autorix/themis/internal/storage/postgres"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

var sharedRepo *postgres.Repository

func TestMain(m *testing.M) {
	ctx := context.Background()
	container, err := tcpostgres.Run(ctx, "postgres:16-alpine",
		tcpostgres.WithDatabase("autorix_test"),
		tcpostgres.WithUsername("autorix"),
		tcpostgres.WithPassword("autorix_test_password"),
		testcontainers.WithWaitStrategy(wait.ForListeningPort("5432/tcp")),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to start test container: %v\n", err)
		os.Exit(1)
	}

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get dsn: %v\n", err)
		_ = container.Terminate(ctx)
		os.Exit(1)
	}

	if err := applyMigrations(ctx, dsn, "../../../migrations"); err != nil {
		fmt.Fprintf(os.Stderr, "failed to apply migrations: %v\n", err)
		_ = container.Terminate(ctx)
		os.Exit(1)
	}

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to parse pool config: %v\n", err)
		_ = container.Terminate(ctx)
		os.Exit(1)
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create pool: %v\n", err)
		_ = container.Terminate(ctx)
		os.Exit(1)
	}

	sharedRepo = postgres.NewRepository(pool)

	code := m.Run()

	pool.Close()
	_ = container.Terminate(context.Background())
	os.Exit(code)
}

func applyMigrations(ctx context.Context, dsn, dir string) error {
	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		return fmt.Errorf("parsing DSN: %w", err)
	}
	cfg.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

	var conn *pgx.Conn
	for attempt := 0; attempt < 20; attempt++ {
		conn, err = pgx.ConnectConfig(ctx, cfg)
		if err == nil {
			break
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(150 * time.Millisecond):
		}
	}
	if err != nil {
		return fmt.Errorf("connecting: %w", err)
	}
	defer func() { _ = conn.Close(ctx) }()

	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("reading migrations dir: %w", err)
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".up.sql") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)

	for _, name := range files {
		sqlBytes, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return fmt.Errorf("reading %s: %w", name, err)
		}
		if _, err := conn.Exec(ctx, string(sqlBytes)); err != nil {
			return fmt.Errorf("applying %s: %w", name, err)
		}
	}
	return nil
}

func newTestRepo(t *testing.T) *postgres.Repository {
	t.Helper()
	if sharedRepo == nil {
		t.Fatal("sharedRepo is not initialized")
	}
	return sharedRepo
}

func TestCreateAndGetByID(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepo(t)

	p := &core.Policy{
		TenantID:    "tenant-a",
		Name:        "allow-admins",
		Description: "allow admins to do anything",
		Expression:  `request.role == "admin"`,
		Priority:    10,
		Enabled:     true,
		Labels:      map[string]string{"team": "platform"},
	}

	if err := repo.Create(ctx, p); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if p.ID == "" {
		t.Fatalf("expected DB-generated ID to be populated")
	}
	if p.CreatedAt.IsZero() || p.UpdatedAt.IsZero() {
		t.Fatalf("expected CreatedAt/UpdatedAt populated, got %+v", p)
	}

	got, err := repo.GetByID(ctx, "tenant-a", p.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Name != "allow-admins" || got.Expression != p.Expression {
		t.Fatalf("unexpected policy: %+v", got)
	}
	if got.Labels["team"] != "platform" {
		t.Fatalf("expected labels to round-trip, got %+v", got.Labels)
	}
}

func TestGetByIDNotFound(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepo(t)

	_, err := repo.GetByID(ctx, "tenant-a", "00000000-0000-0000-0000-000000000000")
	if err == nil {
		t.Fatalf("expected error for nonexistent policy, got nil")
	}
}

func TestGetByIDWrongTenantIsNotFound(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepo(t)

	p := &core.Policy{
		TenantID:   "tenant-a",
		Name:       "scoped-policy",
		Expression: `true`,
		Priority:   0,
		Enabled:    true,
	}
	if err := repo.Create(ctx, p); err != nil {
		t.Fatalf("Create: %v", err)
	}

	_, err := repo.GetByID(ctx, "tenant-b", p.ID)
	if err == nil {
		t.Fatalf("expected error fetching policy under wrong tenant, got nil")
	}
}

func TestCreateDuplicateNameSameTenant(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepo(t)

	p1 := &core.Policy{
		TenantID:   "tenant-a",
		Name:       "dup-policy",
		Expression: `true`,
		Priority:   0,
		Enabled:    true,
	}
	if err := repo.Create(ctx, p1); err != nil {
		t.Fatalf("first Create: %v", err)
	}

	p2 := &core.Policy{
		TenantID:   "tenant-a",
		Name:       "dup-policy", // violates uq_tenant_policy_name
		Expression: `false`,
		Priority:   1,
		Enabled:    true,
	}
	if err := repo.Create(ctx, p2); err == nil {
		t.Fatalf("expected unique constraint violation for duplicate (tenant_id, name), got nil")
	}
}

func TestCreateSameNameDifferentTenantsAllowed(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepo(t)

	p1 := &core.Policy{TenantID: "tenant-a", Name: "shared-name", Expression: `true`, Enabled: true}
	p2 := &core.Policy{TenantID: "tenant-b", Name: "shared-name", Expression: `true`, Enabled: true}

	if err := repo.Create(ctx, p1); err != nil {
		t.Fatalf("Create tenant-a: %v", err)
	}
	if err := repo.Create(ctx, p2); err != nil {
		t.Fatalf("Create tenant-b (same name, different tenant): %v", err)
	}
}

func TestListWithEnabledOnlyAndLabelFilter(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepo(t)

	policies := []*core.Policy{
		{TenantID: "tenant-list-filter", Name: "p1", Expression: `true`, Priority: 2, Enabled: true, Labels: map[string]string{"env": "prod"}},
		{TenantID: "tenant-list-filter", Name: "p2", Expression: `true`, Priority: 1, Enabled: false, Labels: map[string]string{"env": "prod"}},
		{TenantID: "tenant-list-filter", Name: "p3", Expression: `true`, Priority: 3, Enabled: true, Labels: map[string]string{"env": "staging"}},
	}
	for _, p := range policies {
		if err := repo.Create(ctx, p); err != nil {
			t.Fatalf("Create(%s): %v", p.Name, err)
		}
	}

	all, err := repo.List(ctx, core.ListFilter{TenantID: "tenant-list-filter"})
	if err != nil {
		t.Fatalf("List(all): %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("expected 3 policies, got %d", len(all))
	}
	// ORDER BY priority ASC: p2(1), p1(2), p3(3)
	if all[0].Name != "p2" || all[1].Name != "p1" || all[2].Name != "p3" {
		t.Fatalf("unexpected priority ordering: %v", []string{all[0].Name, all[1].Name, all[2].Name})
	}

	enabledOnly, err := repo.List(ctx, core.ListFilter{TenantID: "tenant-list-filter", EnabledOnly: true})
	if err != nil {
		t.Fatalf("List(enabledOnly): %v", err)
	}
	if len(enabledOnly) != 2 {
		t.Fatalf("expected 2 enabled policies, got %d", len(enabledOnly))
	}

	prodOnly, err := repo.List(ctx, core.ListFilter{TenantID: "tenant-list-filter", Labels: map[string]string{"env": "prod"}})
	if err != nil {
		t.Fatalf("List(labels=env:prod): %v", err)
	}
	if len(prodOnly) != 2 {
		t.Fatalf("expected 2 prod-labeled policies, got %d", len(prodOnly))
	}
}

// TestListPage_CursorPagination proves ListPage paginates via a real SQL
// keyset over the same ordering List uses (priority ASC, created_at ASC,
// tie-broken by id): a page smaller than the full set returns exactly N
// rows plus a cursor, and following that cursor returns the rest with no
// duplicate and no skipped row.
func TestListPage_CursorPagination(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepo(t)

	const total = 5
	names := make(map[string]bool, total)
	for i := 0; i < total; i++ {
		name := fmt.Sprintf("cursor-policy-%d", i)
		p := &core.Policy{TenantID: "tenant-cursor", Name: name, Expression: `true`, Priority: int32(i), Enabled: true} // #nosec G115 -- i is bounded by the small test loop
		if err := repo.Create(ctx, p); err != nil {
			t.Fatalf("Create(%s): %v", name, err)
		}
		names[name] = true
	}

	const pageSize = 2
	var all []core.Policy
	cursor := ""
	for i := 0; i < 10; i++ { // bounded loop guard against an infinite pagination bug
		page, hasMore, err := repo.ListPage(ctx, core.ListFilter{TenantID: "tenant-cursor"}, pageSize, cursor)
		if err != nil {
			t.Fatalf("ListPage(cursor=%q): %v", cursor, err)
		}
		all = append(all, page...)
		if !hasMore {
			break
		}
		if len(page) == 0 {
			t.Fatalf("hasMore=true but page was empty")
		}
		last := page[len(page)-1]
		cursor = paging.EncodeCursor(fmt.Sprintf("%d|%s|%s", last.Priority, last.CreatedAt.Format(time.RFC3339Nano), last.ID))
	}

	if len(all) != total {
		t.Fatalf("expected %d total policies across pages, got %d", total, len(all))
	}
	seen := map[string]bool{}
	for i, p := range all {
		if seen[p.Name] {
			t.Fatalf("duplicate policy %q across pages", p.Name)
		}
		seen[p.Name] = true
		delete(names, p.Name)
		want := fmt.Sprintf("cursor-policy-%d", i)
		if p.Name != want {
			t.Fatalf("page order broken: position %d = %q, want %q", i, p.Name, want)
		}
	}
	if len(names) != 0 {
		t.Fatalf("some policies were skipped across pages: %v", names)
	}
}

func TestListPage_EmptyPageHasMoreFalse(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepo(t)

	page, hasMore, err := repo.ListPage(ctx, core.ListFilter{TenantID: "tenant-empty"}, 20, "")
	if err != nil {
		t.Fatalf("ListPage: %v", err)
	}
	if len(page) != 0 {
		t.Fatalf("expected 0 policies, got %d", len(page))
	}
	if hasMore {
		t.Fatalf("expected hasMore=false on empty tenant")
	}
}

func TestUpdatePolicy(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepo(t)

	p := &core.Policy{
		TenantID:   "tenant-a",
		Name:       "to-update",
		Expression: `true`,
		Priority:   0,
		Enabled:    true,
	}
	if err := repo.Create(ctx, p); err != nil {
		t.Fatalf("Create: %v", err)
	}

	p.Description = "updated description"
	p.Expression = `request.role == "editor"`
	p.Priority = 5
	p.Enabled = false
	p.Labels = map[string]string{"updated": "true"}

	if err := repo.Update(ctx, p); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, err := repo.GetByID(ctx, "tenant-a", p.ID)
	if err != nil {
		t.Fatalf("GetByID after update: %v", err)
	}
	if got.Description != "updated description" || got.Expression != p.Expression || got.Priority != 5 || got.Enabled != false {
		t.Fatalf("update did not persist correctly: %+v", got)
	}
	if got.Labels["updated"] != "true" {
		t.Fatalf("expected updated labels, got %+v", got.Labels)
	}
}

func TestUpdateNonExistentPolicy(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepo(t)

	p := &core.Policy{
		ID:         "00000000-0000-0000-0000-000000000000",
		TenantID:   "tenant-a",
		Name:       "ghost",
		Expression: `true`,
	}

	if err := repo.Update(ctx, p); err == nil {
		t.Fatalf("expected error updating nonexistent policy, got nil")
	}
}

func TestDeletePolicy(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepo(t)

	p := &core.Policy{
		TenantID:   "tenant-a",
		Name:       "to-delete",
		Expression: `true`,
		Enabled:    true,
	}
	if err := repo.Create(ctx, p); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := repo.Delete(ctx, "tenant-a", p.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err := repo.GetByID(ctx, "tenant-a", p.ID)
	if err == nil {
		t.Fatalf("expected GetByID to fail after delete, got nil error")
	}
}

func TestDeleteNonExistentPolicy(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepo(t)

	err := repo.Delete(ctx, "tenant-a", "00000000-0000-0000-0000-000000000000")
	if err == nil {
		t.Fatalf("expected error deleting nonexistent policy, got nil")
	}
}

func TestPolicyVersioningAndRollback(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepo(t)

	p := &core.Policy{
		TenantID:    "tenant-versioning",
		Name:        "versioned-policy",
		Description: "v1 description",
		Expression:  `request.role == "viewer"`,
		Priority:    1,
		Enabled:     true,
		Labels:      map[string]string{"v": "1"},
	}

	if err := repo.Create(ctx, p); err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Verify initial version 1 created
	versions, err := repo.ListVersions(ctx, "tenant-versioning", p.ID)
	if err != nil {
		t.Fatalf("ListVersions: %v", err)
	}
	if len(versions) != 1 {
		t.Fatalf("expected 1 initial version, got %d", len(versions))
	}
	if versions[0].Version != 1 || versions[0].Expression != p.Expression {
		t.Fatalf("unexpected version 1: %+v", versions[0])
	}

	// Update policy -> should create version 2
	p.Description = "v2 description"
	p.Expression = `request.role == "editor"`
	p.Labels = map[string]string{"v": "2"}
	if err := repo.Update(ctx, p); err != nil {
		t.Fatalf("Update: %v", err)
	}

	versions, err = repo.ListVersions(ctx, "tenant-versioning", p.ID)
	if err != nil {
		t.Fatalf("ListVersions after update: %v", err)
	}
	if len(versions) != 2 {
		t.Fatalf("expected 2 versions, got %d", len(versions))
	}
	if versions[0].Version != 2 || versions[1].Version != 1 {
		t.Fatalf("expected versions descending [2, 1], got [%d, %d]", versions[0].Version, versions[1].Version)
	}

	// Rollback to version 1 -> restores v1 expression and creates version 3
	rolledBack, err := repo.Rollback(ctx, "tenant-versioning", p.ID, 1)
	if err != nil {
		t.Fatalf("Rollback: %v", err)
	}
	if rolledBack.Expression != `request.role == "viewer"` || rolledBack.Description != "v1 description" {
		t.Fatalf("rollback did not restore v1 state: %+v", rolledBack)
	}

	versions, err = repo.ListVersions(ctx, "tenant-versioning", p.ID)
	if err != nil {
		t.Fatalf("ListVersions after rollback: %v", err)
	}
	if len(versions) != 3 {
		t.Fatalf("expected 3 versions after rollback, got %d", len(versions))
	}
	if versions[0].Version != 3 || versions[0].Expression != `request.role == "viewer"` {
		t.Fatalf("unexpected version 3 snapshot: %+v", versions[0])
	}
}

func TestPolicyFixturesCRUD(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepo(t)

	p := &core.Policy{
		TenantID:   "tenant-fixtures",
		Name:       "fixtures-policy",
		Expression: `request.role == "admin"`,
		Enabled:    true,
	}
	if err := repo.Create(ctx, p); err != nil {
		t.Fatalf("Create policy: %v", err)
	}

	f1 := &core.PolicyFixture{
		PolicyID:       p.ID,
		TenantID:       "tenant-fixtures",
		Name:           "admin-allow",
		Description:    "admin role should pass",
		Payload:        map[string]interface{}{"request": map[string]interface{}{"role": "admin"}},
		ExpectedResult: true,
	}
	if err := repo.CreateFixture(ctx, f1); err != nil {
		t.Fatalf("CreateFixture(f1): %v", err)
	}
	if f1.ID == "" || f1.CreatedAt.IsZero() {
		t.Fatalf("expected ID and CreatedAt populated on fixture: %+v", f1)
	}

	f2 := &core.PolicyFixture{
		PolicyID:       p.ID,
		TenantID:       "tenant-fixtures",
		Name:           "viewer-deny",
		Description:    "viewer role should fail",
		Payload:        map[string]interface{}{"request": map[string]interface{}{"role": "viewer"}},
		ExpectedResult: false,
	}
	if err := repo.CreateFixture(ctx, f2); err != nil {
		t.Fatalf("CreateFixture(f2): %v", err)
	}

	fixtures, err := repo.ListFixtures(ctx, "tenant-fixtures", p.ID)
	if err != nil {
		t.Fatalf("ListFixtures: %v", err)
	}
	if len(fixtures) != 2 {
		t.Fatalf("expected 2 fixtures, got %d", len(fixtures))
	}

	// Delete f1
	if err := repo.DeleteFixture(ctx, "tenant-fixtures", p.ID, f1.ID); err != nil {
		t.Fatalf("DeleteFixture: %v", err)
	}

	fixturesAfter, err := repo.ListFixtures(ctx, "tenant-fixtures", p.ID)
	if err != nil {
		t.Fatalf("ListFixtures after delete: %v", err)
	}
	if len(fixturesAfter) != 1 || fixturesAfter[0].ID != f2.ID {
		t.Fatalf("expected only f2 remaining, got %+v", fixturesAfter)
	}
}
