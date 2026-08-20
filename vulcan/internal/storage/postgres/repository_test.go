package postgres_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/autorix/platform/paging"
	"github.com/autorix/platform/pgtest"
	"github.com/autorix/vulcan/internal/core"
	"github.com/autorix/vulcan/internal/storage/postgres"
	"github.com/google/uuid"
)

// newTestRepo spins up a fresh, isolated Postgres container (via
// testcontainers) with migrations applied, and returns a Repository backed
// by it. Each call gets its own container for full test isolation.
func newTestRepo(t *testing.T) *postgres.Repository {
	t.Helper()
	pool := pgtest.StartPostgres(t, "../../../migrations")
	return postgres.NewRepository(pool)
}

func newTestKey(t *testing.T, suffix string) *core.APIKey {
	t.Helper()
	return &core.APIKey{
		ID:               uuid.New(),
		KeyPrefix:        "av_live",
		KeyHint:          "ab12cd",
		KeyHash:          "hash-" + suffix,
		RootSignatureKey: "root-signature-" + suffix,
		Name:             "test key " + suffix,
		OwnerID:          "owner-" + suffix,
		Scopes:           []string{"read:foo", "write:bar"},
		State:            "active",
	}
}

func TestCreateAndGetKeyByID(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	k := newTestKey(t, "create-get-by-id")
	if err := repo.CreateKey(ctx, k); err != nil {
		t.Fatalf("CreateKey() error = %v", err)
	}
	if k.CreatedAt.IsZero() {
		t.Error("CreateKey() did not populate CreatedAt")
	}
	if k.UpdatedAt.IsZero() {
		t.Error("CreateKey() did not populate UpdatedAt")
	}

	got, err := repo.GetKeyByID(ctx, k.ID)
	if err != nil {
		t.Fatalf("GetKeyByID() error = %v", err)
	}

	if got.ID != k.ID {
		t.Errorf("ID = %v, want %v", got.ID, k.ID)
	}
	if got.KeyPrefix != k.KeyPrefix {
		t.Errorf("KeyPrefix = %q, want %q", got.KeyPrefix, k.KeyPrefix)
	}
	if got.KeyHint != k.KeyHint {
		t.Errorf("KeyHint = %q, want %q", got.KeyHint, k.KeyHint)
	}
	if got.KeyHash != k.KeyHash {
		t.Errorf("KeyHash = %q, want %q", got.KeyHash, k.KeyHash)
	}
	if got.RootSignatureKey != k.RootSignatureKey {
		t.Errorf("RootSignatureKey = %q, want %q", got.RootSignatureKey, k.RootSignatureKey)
	}
	if got.Name != k.Name {
		t.Errorf("Name = %q, want %q", got.Name, k.Name)
	}
	if got.OwnerID != k.OwnerID {
		t.Errorf("OwnerID = %q, want %q", got.OwnerID, k.OwnerID)
	}
	if len(got.Scopes) != len(k.Scopes) {
		t.Fatalf("Scopes = %v, want %v", got.Scopes, k.Scopes)
	}
	for i := range k.Scopes {
		if got.Scopes[i] != k.Scopes[i] {
			t.Errorf("Scopes[%d] = %q, want %q", i, got.Scopes[i], k.Scopes[i])
		}
	}
	if got.State != "active" {
		t.Errorf("State = %q, want %q", got.State, "active")
	}
	if got.LastUsedAt != nil {
		t.Errorf("LastUsedAt = %v, want nil", got.LastUsedAt)
	}
	if got.ExpiresAt != nil {
		t.Errorf("ExpiresAt = %v, want nil", got.ExpiresAt)
	}
}

func TestCreateKey_DefaultsNilScopesToEmptySlice(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	k := newTestKey(t, "nil-scopes")
	k.Scopes = nil
	if err := repo.CreateKey(ctx, k); err != nil {
		t.Fatalf("CreateKey() error = %v", err)
	}

	got, err := repo.GetKeyByID(ctx, k.ID)
	if err != nil {
		t.Fatalf("GetKeyByID() error = %v", err)
	}
	if got.Scopes == nil || len(got.Scopes) != 0 {
		t.Errorf("Scopes = %v, want empty non-nil slice", got.Scopes)
	}
}

func TestCreateKey_ExpiresAtPersisted(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	k := newTestKey(t, "expires-at")
	expiry := time.Now().Add(24 * time.Hour).UTC().Truncate(time.Microsecond)
	k.ExpiresAt = &expiry
	if err := repo.CreateKey(ctx, k); err != nil {
		t.Fatalf("CreateKey() error = %v", err)
	}

	got, err := repo.GetKeyByID(ctx, k.ID)
	if err != nil {
		t.Fatalf("GetKeyByID() error = %v", err)
	}
	if got.ExpiresAt == nil {
		t.Fatal("ExpiresAt = nil, want non-nil")
	}
	if !got.ExpiresAt.Equal(expiry) {
		t.Errorf("ExpiresAt = %v, want %v", got.ExpiresAt, expiry)
	}
}

func TestCreateKey_DuplicateKeyHashViolatesUniqueConstraint(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	k1 := newTestKey(t, "dup")
	if err := repo.CreateKey(ctx, k1); err != nil {
		t.Fatalf("first CreateKey() error = %v", err)
	}

	k2 := newTestKey(t, "dup") // same KeyHash "hash-dup" -> violates UNIQUE(key_hash)
	k2.ID = uuid.New()
	err := repo.CreateKey(ctx, k2)
	if err == nil {
		t.Fatal("second CreateKey() with duplicate key_hash: error = nil, want a unique-violation error")
	}
	if errors.Is(err, postgres.ErrNotFound) {
		t.Errorf("second CreateKey() error = %v, want a plain wrapped Postgres error, not ErrNotFound", err)
	}

	// Verify no partial/duplicate row exists beyond the original.
	all, _, err := repo.ListKeys(ctx, 20, "")
	if err != nil {
		t.Fatalf("ListKeys() error = %v", err)
	}
	count := 0
	for _, k := range all {
		if k.KeyHash == "hash-dup" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("rows with key_hash %q = %d, want 1 (failed insert must not leave partial data)", "hash-dup", count)
	}
}

func TestGetKeyByID_NotFound(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	got, err := repo.GetKeyByID(ctx, uuid.New())
	if got != nil {
		t.Errorf("GetKeyByID() key = %v, want nil", got)
	}
	if !errors.Is(err, postgres.ErrNotFound) {
		t.Errorf("GetKeyByID() error = %v, want ErrNotFound", err)
	}
}

func TestGetKeyByHash(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	k := newTestKey(t, "get-by-hash")
	if err := repo.CreateKey(ctx, k); err != nil {
		t.Fatalf("CreateKey() error = %v", err)
	}

	got, err := repo.GetKeyByHash(ctx, k.KeyHash)
	if err != nil {
		t.Fatalf("GetKeyByHash() error = %v", err)
	}
	if got.ID != k.ID {
		t.Errorf("ID = %v, want %v", got.ID, k.ID)
	}
}

func TestGetKeyByHash_NotFound(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	got, err := repo.GetKeyByHash(ctx, "does-not-exist")
	if got != nil {
		t.Errorf("GetKeyByHash() key = %v, want nil", got)
	}
	if !errors.Is(err, postgres.ErrNotFound) {
		t.Errorf("GetKeyByHash() error = %v, want ErrNotFound", err)
	}
}

// GetKeyByHash filters on state = 'active', so a revoked key must not be
// resolvable by hash even though it still exists in the table.
func TestGetKeyByHash_ExcludesRevokedKeys(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	k := newTestKey(t, "hash-revoked")
	if err := repo.CreateKey(ctx, k); err != nil {
		t.Fatalf("CreateKey() error = %v", err)
	}
	if err := repo.RevokeKey(ctx, k.ID); err != nil {
		t.Fatalf("RevokeKey() error = %v", err)
	}

	got, err := repo.GetKeyByHash(ctx, k.KeyHash)
	if got != nil {
		t.Errorf("GetKeyByHash() after revoke: key = %v, want nil", got)
	}
	if !errors.Is(err, postgres.ErrNotFound) {
		t.Errorf("GetKeyByHash() after revoke: error = %v, want ErrNotFound", err)
	}
}

func TestRevokeKey(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	k := newTestKey(t, "revoke")
	if err := repo.CreateKey(ctx, k); err != nil {
		t.Fatalf("CreateKey() error = %v", err)
	}

	if err := repo.RevokeKey(ctx, k.ID); err != nil {
		t.Fatalf("RevokeKey() error = %v", err)
	}

	got, err := repo.GetKeyByID(ctx, k.ID)
	if err != nil {
		t.Fatalf("GetKeyByID() error = %v", err)
	}
	if got.State != "revoked" {
		t.Errorf("State = %q, want %q", got.State, "revoked")
	}
	if !got.UpdatedAt.After(k.UpdatedAt) && !got.UpdatedAt.Equal(k.UpdatedAt) {
		t.Errorf("UpdatedAt = %v, want >= original CreatedAt %v", got.UpdatedAt, k.UpdatedAt)
	}
}

// RevokeKey issues an UPDATE with no rows-affected check, so revoking a
// non-existent id is observed to silently succeed with a nil error rather
// than surfacing ErrNotFound. This documents the real, current behavior.
func TestRevokeKey_NonExistentIDReturnsErrNotFound(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	err := repo.RevokeKey(ctx, uuid.New())
	if !errors.Is(err, postgres.ErrNotFound) {
		t.Fatalf("RevokeKey() on non-existent id: error = %v, want ErrNotFound", err)
	}
}

func TestUpdateLastUsed(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	k := newTestKey(t, "last-used")
	if err := repo.CreateKey(ctx, k); err != nil {
		t.Fatalf("CreateKey() error = %v", err)
	}

	before, err := repo.GetKeyByID(ctx, k.ID)
	if err != nil {
		t.Fatalf("GetKeyByID() error = %v", err)
	}
	if before.LastUsedAt != nil {
		t.Fatalf("LastUsedAt before update = %v, want nil", before.LastUsedAt)
	}

	repo.UpdateLastUsed(ctx, k.ID) // no return value to check

	after, err := repo.GetKeyByID(ctx, k.ID)
	if err != nil {
		t.Fatalf("GetKeyByID() error = %v", err)
	}
	if after.LastUsedAt == nil {
		t.Fatal("LastUsedAt after update = nil, want non-nil")
	}
}

func TestListKeys(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	if all, _, err := repo.ListKeys(ctx, 20, ""); err != nil {
		t.Fatalf("ListKeys() on empty table: error = %v", err)
	} else if len(all) != 0 {
		t.Fatalf("ListKeys() on empty table = %d keys, want 0", len(all))
	}

	k1 := newTestKey(t, "list-1")
	if err := repo.CreateKey(ctx, k1); err != nil {
		t.Fatalf("CreateKey() error = %v", err)
	}
	// Ensure a distinct created_at ordering (created_at has second-level
	// column precision expectations in ORDER BY DESC).
	time.Sleep(10 * time.Millisecond)
	k2 := newTestKey(t, "list-2")
	if err := repo.CreateKey(ctx, k2); err != nil {
		t.Fatalf("CreateKey() error = %v", err)
	}

	all, hasMore, err := repo.ListKeys(ctx, 20, "")
	if err != nil {
		t.Fatalf("ListKeys() error = %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("ListKeys() = %d keys, want 2", len(all))
	}
	if hasMore {
		t.Errorf("ListKeys() hasMore = true, want false (both fit in one page)")
	}
	// ORDER BY created_at DESC -> most recently created first.
	if all[0].ID != k2.ID {
		t.Errorf("ListKeys()[0].ID = %v, want %v (most recent first)", all[0].ID, k2.ID)
	}
	if all[1].ID != k1.ID {
		t.Errorf("ListKeys()[1].ID = %v, want %v", all[1].ID, k1.ID)
	}
}

// TestListKeys_CursorPagination proves ListKeys paginates via a real SQL
// keyset: a page smaller than the full set returns exactly N rows plus a
// cursor, and following that cursor returns the rest with no duplicate and
// no skipped row.
func TestListKeys_CursorPagination(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	const total = 5
	ids := make(map[uuid.UUID]bool, total)
	for i := 0; i < total; i++ {
		k := newTestKey(t, fmt.Sprintf("cursor-%d", i))
		if err := repo.CreateKey(ctx, k); err != nil {
			t.Fatalf("CreateKey() error = %v", err)
		}
		ids[k.ID] = true
		time.Sleep(2 * time.Millisecond)
	}

	const pageSize = 2
	var all []core.APIKey
	cursor := ""
	for i := 0; i < 10; i++ { // bounded loop guard against an infinite pagination bug
		page, hasMore, err := repo.ListKeys(ctx, pageSize, cursor)
		if err != nil {
			t.Fatalf("ListKeys(cursor=%q) error = %v", cursor, err)
		}
		all = append(all, page...)
		if !hasMore {
			break
		}
		if len(page) == 0 {
			t.Fatalf("hasMore=true but page was empty")
		}
		last := page[len(page)-1]
		cursor = paging.EncodeCursor(last.CreatedAt.Format(time.RFC3339Nano) + "|" + last.ID.String())
	}

	if len(all) != total {
		t.Fatalf("expected %d total keys across pages, got %d", total, len(all))
	}
	seen := map[uuid.UUID]bool{}
	for _, k := range all {
		if seen[k.ID] {
			t.Fatalf("duplicate key %v across pages", k.ID)
		}
		seen[k.ID] = true
		delete(ids, k.ID)
	}
	if len(ids) != 0 {
		t.Fatalf("some keys were skipped across pages: %v", ids)
	}
}

func TestUpdateKey(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	k := newTestKey(t, "update")
	k.Description = "initial description"
	if err := repo.CreateKey(ctx, k); err != nil {
		t.Fatalf("CreateKey() error = %v", err)
	}

	newName := "updated name"
	newDesc := "updated description"
	newScopes := []string{"admin:all", "read:logs"}
	newExpiry := time.Now().Add(48 * time.Hour).UTC().Truncate(time.Microsecond)

	updated, err := repo.UpdateKey(ctx, k.ID, core.UpdateKeyRequest{
		Name:        &newName,
		Description: &newDesc,
		Scopes:      &newScopes,
		ExpiresAt:   &newExpiry,
	})
	if err != nil {
		t.Fatalf("UpdateKey() error = %v", err)
	}

	if updated.Name != newName {
		t.Errorf("Name = %q, want %q", updated.Name, newName)
	}
	if updated.Description != newDesc {
		t.Errorf("Description = %q, want %q", updated.Description, newDesc)
	}
	if len(updated.Scopes) != 2 || updated.Scopes[0] != "admin:all" {
		t.Errorf("Scopes = %v, want %v", updated.Scopes, newScopes)
	}
	if updated.ExpiresAt == nil || !updated.ExpiresAt.Equal(newExpiry) {
		t.Errorf("ExpiresAt = %v, want %v", updated.ExpiresAt, newExpiry)
	}

	// Verify persistence
	fetched, err := repo.GetKeyByID(ctx, k.ID)
	if err != nil {
		t.Fatalf("GetKeyByID() error = %v", err)
	}
	if fetched.Name != newName || fetched.Description != newDesc {
		t.Errorf("Persisted key name/description mismatch: %v", fetched)
	}
}

func TestUpdateKey_NotFound(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	name := "does-not-matter"
	_, err := repo.UpdateKey(ctx, uuid.New(), core.UpdateKeyRequest{Name: &name})
	if !errors.Is(err, postgres.ErrNotFound) {
		t.Fatalf("UpdateKey() for nonexistent key: error = %v, want ErrNotFound", err)
	}
}

func TestRotateKey_AndGracePeriodLookup(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	k := newTestKey(t, "rotate")
	if err := repo.CreateKey(ctx, k); err != nil {
		t.Fatalf("CreateKey() error = %v", err)
	}

	newHash := "new-hash-rotate"
	newHint := "9999"
	newRootKey := "new-root-rotate"
	gracePeriodExpiresAt := time.Now().Add(24 * time.Hour).UTC().Truncate(time.Microsecond)

	rotated, err := repo.RotateKey(ctx, k.ID, newHash, newHint, newRootKey, &gracePeriodExpiresAt)
	if err != nil {
		t.Fatalf("RotateKey() error = %v", err)
	}

	if rotated.KeyHash != newHash {
		t.Errorf("KeyHash = %q, want %q", rotated.KeyHash, newHash)
	}
	if rotated.KeyHint != newHint {
		t.Errorf("KeyHint = %q, want %q", rotated.KeyHint, newHint)
	}
	if rotated.RootSignatureKey != newRootKey {
		t.Errorf("RootSignatureKey = %q, want %q", rotated.RootSignatureKey, newRootKey)
	}
	if rotated.PrevKeyHash != k.KeyHash {
		t.Errorf("PrevKeyHash = %q, want %q", rotated.PrevKeyHash, k.KeyHash)
	}
	if rotated.PrevRootSignatureKey != k.RootSignatureKey {
		t.Errorf("PrevRootSignatureKey = %q, want %q", rotated.PrevRootSignatureKey, k.RootSignatureKey)
	}

	// 1. New key hash resolves
	byNew, err := repo.GetKeyByHash(ctx, newHash)
	if err != nil {
		t.Fatalf("GetKeyByHash(newHash) error = %v", err)
	}
	if byNew.ID != k.ID {
		t.Errorf("GetKeyByHash(newHash) ID = %v, want %v", byNew.ID, k.ID)
	}

	// 2. Prev key hash resolves within grace period
	byPrev, err := repo.GetKeyByHash(ctx, k.KeyHash)
	if err != nil {
		t.Fatalf("GetKeyByHash(prevHash) during grace period error = %v", err)
	}
	if byPrev.ID != k.ID {
		t.Errorf("GetKeyByHash(prevHash) ID = %v, want %v", byPrev.ID, k.ID)
	}

	// 3. Expired grace period does not resolve prev hash
	expiredGrace := time.Now().Add(-1 * time.Hour).UTC()
	_, err = repo.RotateKey(ctx, k.ID, "hash-v3", "3333", "root-v3", &expiredGrace)
	if err != nil {
		t.Fatalf("second RotateKey() error = %v", err)
	}
	_, err = repo.GetKeyByHash(ctx, newHash) // newHash is now the previous hash, but grace is expired
	if !errors.Is(err, postgres.ErrNotFound) {
		t.Errorf("GetKeyByHash(prevHash) with expired grace period: error = %v, want ErrNotFound", err)
	}
}

func TestRecordUsage(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	k := newTestKey(t, "usage")
	if err := repo.CreateKey(ctx, k); err != nil {
		t.Fatalf("CreateKey() error = %v", err)
	}

	if err := repo.RecordUsage(ctx, k.ID, "192.168.1.100"); err != nil {
		t.Fatalf("RecordUsage() error = %v", err)
	}

	got, err := repo.GetKeyByID(ctx, k.ID)
	if err != nil {
		t.Fatalf("GetKeyByID() error = %v", err)
	}
	if got.CallCount != 1 {
		t.Errorf("CallCount = %d, want 1", got.CallCount)
	}
	if got.LastSourceIP != "192.168.1.100" {
		t.Errorf("LastSourceIP = %q, want 192.168.1.100", got.LastSourceIP)
	}
	if got.LastUsedAt == nil {
		t.Error("LastUsedAt is nil after RecordUsage")
	}

	// Second usage increments count
	if err := repo.RecordUsage(ctx, k.ID, "10.0.0.1"); err != nil {
		t.Fatalf("second RecordUsage() error = %v", err)
	}
	got2, err := repo.GetKeyByID(ctx, k.ID)
	if err != nil {
		t.Fatalf("GetKeyByID() error = %v", err)
	}
	if got2.CallCount != 2 {
		t.Errorf("CallCount = %d, want 2", got2.CallCount)
	}
	if got2.LastSourceIP != "10.0.0.1" {
		t.Errorf("LastSourceIP = %q, want 10.0.0.1", got2.LastSourceIP)
	}
}

func TestScopesCatalogue_CRUD(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// 1. Initial list is empty
	scopes, err := repo.ListScopes(ctx)
	if err != nil {
		t.Fatalf("ListScopes() error = %v", err)
	}
	if len(scopes) != 0 {
		t.Fatalf("ListScopes() initial len = %d, want 0", len(scopes))
	}

	// 2. Create scope
	s1 := &core.Scope{
		Name:        "vulcan:read",
		Description: "Read Vulcan resources",
	}
	if err := repo.CreateScope(ctx, s1); err != nil {
		t.Fatalf("CreateScope() error = %v", err)
	}
	if s1.CreatedAt.IsZero() {
		t.Error("CreateScope() did not populate CreatedAt")
	}

	s2 := &core.Scope{
		Name:        "vulcan:write",
		Description: "Write Vulcan resources",
	}
	if err := repo.CreateScope(ctx, s2); err != nil {
		t.Fatalf("CreateScope() 2 error = %v", err)
	}

	// 3. List scopes returns both ordered by name
	scopes, err = repo.ListScopes(ctx)
	if err != nil {
		t.Fatalf("ListScopes() error = %v", err)
	}
	if len(scopes) != 2 {
		t.Fatalf("ListScopes() len = %d, want 2", len(scopes))
	}
	if scopes[0].Name != "vulcan:read" || scopes[1].Name != "vulcan:write" {
		t.Errorf("ListScopes() unexpected order: %v", scopes)
	}

	// 4. Delete scope
	if err := repo.DeleteScope(ctx, "vulcan:read"); err != nil {
		t.Fatalf("DeleteScope() error = %v", err)
	}

	// 5. Delete nonexistent returns ErrNotFound
	if err := repo.DeleteScope(ctx, "vulcan:read"); !errors.Is(err, postgres.ErrNotFound) {
		t.Errorf("DeleteScope() nonexistent error = %v, want ErrNotFound", err)
	}

	// 6. List scopes returns remaining
	scopes, err = repo.ListScopes(ctx)
	if err != nil {
		t.Fatalf("ListScopes() error = %v", err)
	}
	if len(scopes) != 1 || scopes[0].Name != "vulcan:write" {
		t.Errorf("ListScopes() after delete: %v", scopes)
	}
}

