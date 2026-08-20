package postgres_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/autorix/ego/internal/core"
	"github.com/autorix/ego/internal/storage/postgres"
	"github.com/autorix/platform/paging"
	"github.com/autorix/platform/pgtest"
	"github.com/google/uuid"
)

const migrationsDir = "../../../migrations"

// newTestRepo starts an isolated Postgres container via pgtest, migrated
// and ready, and wraps its pool in a Repository. pgtest.StartPostgres
// applies migrations over its own internal connection and returns a pool
// using pgx's normal (extended/cache-statement) protocol, so a []byte
// parameter against a jsonb column — e.g. CreateIdentityWithPassword's
// marshaled traits — encodes correctly, exactly as it does for every real
// caller.
func newTestRepo(t *testing.T) *postgres.Repository {
	t.Helper()
	pool := pgtest.StartPostgres(t, migrationsDir)
	return postgres.NewRepository(pool)
}

func TestCreateIdentityWithPassword_HappyPath(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	traits := map[string]interface{}{
		"email": "alice@example.com",
		"name":  "Alice",
	}

	identity, err := repo.CreateIdentityWithPassword(ctx, traits, "argon2id-hash")
	if err != nil {
		t.Fatalf("CreateIdentityWithPassword: unexpected error: %v", err)
	}
	if identity.ID == uuid.Nil {
		t.Fatal("CreateIdentityWithPassword: expected non-nil identity ID")
	}
	if identity.SchemaID != "default" {
		t.Errorf("SchemaID = %q, want %q", identity.SchemaID, "default")
	}
	if identity.State != "active" {
		t.Errorf("State = %q, want %q", identity.State, "active")
	}
	if identity.Traits["email"] != "alice@example.com" {
		t.Errorf("Traits[email] = %v, want alice@example.com", identity.Traits["email"])
	}

	// Verify it can actually be found back through the public lookup path.
	found, err := repo.FindIdentityByIdentifier(ctx, "alice@example.com")
	if err != nil {
		t.Fatalf("FindIdentityByIdentifier: unexpected error: %v", err)
	}
	if found.ID != identity.ID {
		t.Errorf("found.ID = %v, want %v", found.ID, identity.ID)
	}

	// Verify the password credential was stored and is retrievable.
	hash, err := repo.GetPasswordCredential(ctx, identity.ID)
	if err != nil {
		t.Fatalf("GetPasswordCredential: unexpected error: %v", err)
	}
	if hash != "argon2id-hash" {
		t.Errorf("hash = %q, want %q", hash, "argon2id-hash")
	}
}

func TestCreateIdentityWithPassword_MissingEmail(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	_, err := repo.CreateIdentityWithPassword(ctx, map[string]interface{}{"name": "NoEmail"}, "hash")
	if err == nil {
		t.Fatal("expected error when traits lack an email, got nil")
	}
}

// TestCreateIdentityWithPassword_DuplicateEmail exercises the genuine
// Postgres-driven error path: the second insert violates the UNIQUE
// constraint on identity_verifiable_addresses.value. repository.go does not
// translate this into ErrAlreadyExists (that sentinel is declared but never
// returned by CreateIdentityWithPassword) — it just wraps the raw pgx/pg
// error. This test documents the real observed behavior rather than an
// assumed one.
func TestCreateIdentityWithPassword_DuplicateEmail(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	traits := map[string]interface{}{"email": "dupe@example.com"}

	if _, err := repo.CreateIdentityWithPassword(ctx, traits, "hash-1"); err != nil {
		t.Fatalf("first CreateIdentityWithPassword: unexpected error: %v", err)
	}

	_, err := repo.CreateIdentityWithPassword(ctx, traits, "hash-2")
	if err == nil {
		t.Fatal("expected an error on duplicate email, got nil")
	}
	if errors.Is(err, postgres.ErrAlreadyExists) {
		t.Fatal("unexpectedly got ErrAlreadyExists: repository.go does not currently map " +
			"the unique_violation to this sentinel for CreateIdentityWithPassword")
	}

	// Exactly one identity must exist for this email; the failed second
	// attempt's transaction must have rolled back the identity row it
	// inserted before failing on the verifiable-address insert.
	identities, _, err := repo.ListIdentities(ctx, postgres.ListIdentitiesFilter{Limit: 20})
	if err != nil {
		t.Fatalf("ListIdentities: unexpected error: %v", err)
	}
	count := 0
	for _, i := range identities {
		if i.Traits["email"] == "dupe@example.com" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected exactly 1 identity for dupe@example.com after failed duplicate insert, got %d "+
			"(indicates the transaction did not roll back cleanly)", count)
	}
}

func TestFindIdentityByIdentifier_NotFound(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	_, err := repo.FindIdentityByIdentifier(ctx, "nobody@example.com")
	if !errors.Is(err, postgres.ErrNotFound) {
		t.Fatalf("FindIdentityByIdentifier: err = %v, want ErrNotFound", err)
	}
}

func TestGetPasswordCredential_NotFound(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	_, err := repo.GetPasswordCredential(ctx, uuid.New())
	if !errors.Is(err, postgres.ErrNotFound) {
		t.Fatalf("GetPasswordCredential: err = %v, want ErrNotFound", err)
	}
}

func TestCreateSession_And_GetSessionByTokenHash(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	identity, err := repo.CreateIdentityWithPassword(ctx, map[string]interface{}{
		"email": "session-user@example.com",
	}, "hash")
	if err != nil {
		t.Fatalf("CreateIdentityWithPassword: unexpected error: %v", err)
	}

	session := &core.Session{
		ID:              uuid.New(),
		IdentityID:      identity.ID,
		TokenHash:       "token-hash-abc",
		ExpiresAt:       time.Now().Add(time.Hour),
		AuthenticatedAt: time.Now(),
	}
	if err := repo.CreateSession(ctx, session); err != nil {
		t.Fatalf("CreateSession: unexpected error: %v", err)
	}

	got, err := repo.GetSessionByTokenHash(ctx, "token-hash-abc")
	if err != nil {
		t.Fatalf("GetSessionByTokenHash: unexpected error: %v", err)
	}
	if got.ID != session.ID {
		t.Errorf("got.ID = %v, want %v", got.ID, session.ID)
	}
	if got.IdentityID != identity.ID {
		t.Errorf("got.IdentityID = %v, want %v", got.IdentityID, identity.ID)
	}
	if got.Identity == nil {
		t.Fatal("expected joined Identity to be populated")
	}
	if got.Identity.Traits["email"] != "session-user@example.com" {
		t.Errorf("Identity.Traits[email] = %v, want session-user@example.com", got.Identity.Traits["email"])
	}
}

func TestGetSessionByTokenHash_NotFound(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	_, err := repo.GetSessionByTokenHash(ctx, "does-not-exist")
	if !errors.Is(err, postgres.ErrNotFound) {
		t.Fatalf("GetSessionByTokenHash: err = %v, want ErrNotFound", err)
	}
}

// TestGetSessionByTokenHash_Expired proves the query's expiry filter
// (expires_at > CURRENT_TIMESTAMP) is real: an expired session is treated
// exactly like a missing one.
func TestGetSessionByTokenHash_Expired(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	identity, err := repo.CreateIdentityWithPassword(ctx, map[string]interface{}{
		"email": "expired-user@example.com",
	}, "hash")
	if err != nil {
		t.Fatalf("CreateIdentityWithPassword: unexpected error: %v", err)
	}

	session := &core.Session{
		ID:              uuid.New(),
		IdentityID:      identity.ID,
		TokenHash:       "expired-token",
		ExpiresAt:       time.Now().Add(-time.Hour),
		AuthenticatedAt: time.Now().Add(-2 * time.Hour),
	}
	if err := repo.CreateSession(ctx, session); err != nil {
		t.Fatalf("CreateSession: unexpected error: %v", err)
	}

	_, err = repo.GetSessionByTokenHash(ctx, "expired-token")
	if !errors.Is(err, postgres.ErrNotFound) {
		t.Fatalf("GetSessionByTokenHash(expired): err = %v, want ErrNotFound", err)
	}
}

func TestDeleteSessionByTokenHash(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	identity, err := repo.CreateIdentityWithPassword(ctx, map[string]interface{}{
		"email": "delete-session@example.com",
	}, "hash")
	if err != nil {
		t.Fatalf("CreateIdentityWithPassword: unexpected error: %v", err)
	}

	session := &core.Session{
		ID:              uuid.New(),
		IdentityID:      identity.ID,
		TokenHash:       "token-to-delete",
		ExpiresAt:       time.Now().Add(time.Hour),
		AuthenticatedAt: time.Now(),
	}
	if err := repo.CreateSession(ctx, session); err != nil {
		t.Fatalf("CreateSession: unexpected error: %v", err)
	}

	if err := repo.DeleteSessionByTokenHash(ctx, "token-to-delete"); err != nil {
		t.Fatalf("DeleteSessionByTokenHash: unexpected error: %v", err)
	}

	_, err = repo.GetSessionByTokenHash(ctx, "token-to-delete")
	if !errors.Is(err, postgres.ErrNotFound) {
		t.Fatalf("expected ErrNotFound after deletion, got %v", err)
	}

	// Deleting an already-deleted / nonexistent session must not error
	// (DELETE with no matching rows is a no-op in Postgres).
	if err := repo.DeleteSessionByTokenHash(ctx, "token-to-delete"); err != nil {
		t.Fatalf("DeleteSessionByTokenHash (second time): unexpected error: %v", err)
	}
}

func TestListIdentities(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	if identities, _, err := repo.ListIdentities(ctx, postgres.ListIdentitiesFilter{Limit: 20}); err != nil {
		t.Fatalf("ListIdentities (empty): unexpected error: %v", err)
	} else if len(identities) != 0 {
		t.Fatalf("expected 0 identities initially, got %d", len(identities))
	}

	for _, email := range []string{"list-a@example.com", "list-b@example.com", "list-c@example.com"} {
		if _, err := repo.CreateIdentityWithPassword(ctx, map[string]interface{}{"email": email}, "hash"); err != nil {
			t.Fatalf("CreateIdentityWithPassword(%s): unexpected error: %v", email, err)
		}
	}

	identities, hasMore, err := repo.ListIdentities(ctx, postgres.ListIdentitiesFilter{Limit: 20})
	if err != nil {
		t.Fatalf("ListIdentities: unexpected error: %v", err)
	}
	if len(identities) != 3 {
		t.Fatalf("expected 3 identities, got %d", len(identities))
	}
	if hasMore {
		t.Fatalf("expected hasMore=false when all rows fit in one page")
	}
}

// TestListIdentities_CursorPagination proves the cursor is a real SQL
// keyset, not an in-memory slice: a page smaller than the full set returns
// exactly N rows plus a cursor, and following that cursor returns the rest
// with no duplicate and no skipped row.
func TestListIdentities_CursorPagination(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	const total = 5
	created := make(map[string]bool, total)
	for i := 0; i < total; i++ {
		email := fmt.Sprintf("cursor-%d@example.com", i)
		if _, err := repo.CreateIdentityWithPassword(ctx, map[string]interface{}{"email": email}, "hash"); err != nil {
			t.Fatalf("CreateIdentityWithPassword(%s): unexpected error: %v", email, err)
		}
		created[email] = true
	}

	const pageSize = 2
	page1, hasMore1, err := repo.ListIdentities(ctx, postgres.ListIdentitiesFilter{Limit: pageSize})
	if err != nil {
		t.Fatalf("ListIdentities page1: unexpected error: %v", err)
	}
	if len(page1) != pageSize {
		t.Fatalf("page1: expected %d identities, got %d", pageSize, len(page1))
	}
	if !hasMore1 {
		t.Fatalf("page1: expected hasMore=true")
	}

	last := page1[len(page1)-1]
	cursor := paging.EncodeCursor(last.CreatedAt.Format(time.RFC3339Nano) + "|" + last.ID.String())

	page2, hasMore2, err := repo.ListIdentities(ctx, postgres.ListIdentitiesFilter{Limit: pageSize, Cursor: cursor})
	if err != nil {
		t.Fatalf("ListIdentities page2: unexpected error: %v", err)
	}
	if len(page2) != pageSize {
		t.Fatalf("page2: expected %d identities, got %d", pageSize, len(page2))
	}
	if !hasMore2 {
		t.Fatalf("page2: expected hasMore=true")
	}

	last2 := page2[len(page2)-1]
	cursor2 := paging.EncodeCursor(last2.CreatedAt.Format(time.RFC3339Nano) + "|" + last2.ID.String())
	page3, hasMore3, err := repo.ListIdentities(ctx, postgres.ListIdentitiesFilter{Limit: pageSize, Cursor: cursor2})
	if err != nil {
		t.Fatalf("ListIdentities page3: unexpected error: %v", err)
	}
	if len(page3) != total-2*pageSize {
		t.Fatalf("page3: expected %d identities, got %d", total-2*pageSize, len(page3))
	}
	if hasMore3 {
		t.Fatalf("page3: expected hasMore=false, exhausted the set")
	}

	seen := map[uuid.UUID]bool{}
	all := append(append(page1, page2...), page3...)
	if len(all) != total {
		t.Fatalf("expected %d total identities across pages, got %d", total, len(all))
	}
	for _, i := range all {
		if seen[i.ID] {
			t.Fatalf("duplicate identity %v across pages", i.ID)
		}
		seen[i.ID] = true
		delete(created, i.Traits["email"].(string))
	}
	if len(created) != 0 {
		t.Fatalf("some identities were skipped across pages: %v", created)
	}
}

// TestCreateIdentityWithPassword_TransactionRollback proves that a
// mid-transaction failure (duplicate verifiable address, the third and
// last statement in the transaction) leaves no partial data committed: no
// identity row and no credential row survive the failed attempt.
func TestCreateIdentityWithPassword_TransactionRollback(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	traits := map[string]interface{}{"email": "rollback@example.com"}
	first, err := repo.CreateIdentityWithPassword(ctx, traits, "hash-1")
	if err != nil {
		t.Fatalf("first CreateIdentityWithPassword: unexpected error: %v", err)
	}

	// This second call inserts a new identity + credential successfully,
	// then fails on the verifiable-address insert (duplicate value). The
	// whole transaction — including the identity and credential rows just
	// inserted in *this* attempt — must roll back.
	if _, err := repo.CreateIdentityWithPassword(ctx, traits, "hash-2"); err == nil {
		t.Fatal("expected error on duplicate verifiable address, got nil")
	}

	identities, _, err := repo.ListIdentities(ctx, postgres.ListIdentitiesFilter{Limit: 20})
	if err != nil {
		t.Fatalf("ListIdentities: unexpected error: %v", err)
	}
	if len(identities) != 1 {
		t.Fatalf("expected exactly 1 identity after rolled-back duplicate attempt, got %d", len(identities))
	}
	if identities[0].ID != first.ID {
		t.Fatalf("surviving identity ID = %v, want the first attempt's ID %v", identities[0].ID, first.ID)
	}

	// The credential from the failed second attempt must not have survived
	// either: only the original hash-1 credential should be gettable, and
	// it must still be exactly hash-1 (not overwritten or duplicated).
	hash, err := repo.GetPasswordCredential(ctx, first.ID)
	if err != nil {
		t.Fatalf("GetPasswordCredential: unexpected error: %v", err)
	}
	if hash != "hash-1" {
		t.Fatalf("hash = %q, want %q (rolled-back attempt's credential must not have persisted)", hash, "hash-1")
	}
}

func TestGetIdentityByID(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	traits := map[string]interface{}{
		"email": "bob@example.com",
		"name":  "Bob",
	}
	created, err := repo.CreateIdentityWithPassword(ctx, traits, "hash-bob")
	if err != nil {
		t.Fatalf("CreateIdentityWithPassword: %v", err)
	}

	found, err := repo.GetIdentityByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetIdentityByID: unexpected error: %v", err)
	}
	if found.ID != created.ID {
		t.Errorf("found.ID = %v, want %v", found.ID, created.ID)
	}
	if found.State != core.StateActive {
		t.Errorf("found.State = %q, want %q", found.State, core.StateActive)
	}
	if found.Traits["email"] != "bob@example.com" {
		t.Errorf("found.Traits[email] = %v, want bob@example.com", found.Traits["email"])
	}

	// Test non-existent ID
	_, err = repo.GetIdentityByID(ctx, uuid.New())
	if !errors.Is(err, postgres.ErrNotFound) {
		t.Errorf("expected ErrNotFound for non-existent ID, got %v", err)
	}
}

func TestUpdateIdentity(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	traits := map[string]interface{}{
		"email": "carol@example.com",
		"name":  "Carol",
	}
	created, err := repo.CreateIdentityWithPassword(ctx, traits, "hash-carol")
	if err != nil {
		t.Fatalf("CreateIdentityWithPassword: %v", err)
	}

	// 1. Update traits and state
	newTraits := map[string]interface{}{
		"email": "carol.new@example.com",
		"name":  "Carol Danvers",
	}
	newState := core.StateInactive
	updated, err := repo.UpdateIdentity(ctx, created.ID, newTraits, &newState, nil)
	if err != nil {
		t.Fatalf("UpdateIdentity: unexpected error: %v", err)
	}
	if updated.State != core.StateInactive {
		t.Errorf("updated.State = %q, want %q", updated.State, core.StateInactive)
	}
	if updated.Traits["email"] != "carol.new@example.com" {
		t.Errorf("updated.Traits[email] = %v, want carol.new@example.com", updated.Traits["email"])
	}

	// Verify email address was updated in verifiable addresses so new email is searchable
	found, err := repo.FindIdentityByIdentifier(ctx, "carol.new@example.com")
	if err != nil {
		t.Fatalf("FindIdentityByIdentifier with new email: %v", err)
	}
	if found.ID != created.ID {
		t.Errorf("found.ID = %v, want %v", found.ID, created.ID)
	}

	// Old email should not be found
	_, err = repo.FindIdentityByIdentifier(ctx, "carol@example.com")
	if !errors.Is(err, postgres.ErrNotFound) {
		t.Errorf("expected ErrNotFound for old email, got %v", err)
	}

	// 2. Reject invalid state
	invalidState := "invalid_state"
	_, err = repo.UpdateIdentity(ctx, created.ID, nil, &invalidState, nil)
	if err == nil {
		t.Fatal("expected error for invalid state, got nil")
	}

	// 3. Update non-existent identity
	_, err = repo.UpdateIdentity(ctx, uuid.New(), newTraits, &newState, nil)
	if !errors.Is(err, postgres.ErrNotFound) {
		t.Errorf("expected ErrNotFound for non-existent ID, got %v", err)
	}
}

func TestDeleteIdentity_SoftDelete(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	traits := map[string]interface{}{
		"email": "dave@example.com",
		"name":  "Dave",
	}
	created, err := repo.CreateIdentityWithPassword(ctx, traits, "hash-dave")
	if err != nil {
		t.Fatalf("CreateIdentityWithPassword: %v", err)
	}

	// Create a session for dave
	tokenHash := "dave-session-token-hash"
	sess := &core.Session{
		ID:              uuid.New(),
		IdentityID:      created.ID,
		TokenHash:       tokenHash,
		ExpiresAt:       time.Now().Add(1 * time.Hour),
		AuthenticatedAt: time.Now(),
	}
	if err := repo.CreateSession(ctx, sess); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	// Soft delete Dave
	if err := repo.DeleteIdentity(ctx, created.ID); err != nil {
		t.Fatalf("DeleteIdentity: unexpected error: %v", err)
	}

	// Verify Dave is soft-deleted: GetIdentityByID still returns record but with DeletedAt set and state inactive
	deleted, err := repo.GetIdentityByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetIdentityByID on soft-deleted: %v", err)
	}
	if deleted.DeletedAt == nil {
		t.Fatal("expected DeletedAt to be non-nil on soft-deleted identity")
	}
	if deleted.State != core.StateInactive {
		t.Errorf("deleted.State = %q, want %q", deleted.State, core.StateInactive)
	}

	// Active session for Dave must be revoked
	_, err = repo.GetSessionByTokenHash(ctx, tokenHash)
	if !errors.Is(err, postgres.ErrNotFound) {
		t.Errorf("expected session to be deleted/revoked, got %v", err)
	}

	// Dave should not be searchable by identifier or listed in active identities
	_, err = repo.FindIdentityByIdentifier(ctx, "dave@example.com")
	if !errors.Is(err, postgres.ErrNotFound) {
		t.Errorf("expected FindIdentityByIdentifier to return ErrNotFound for soft-deleted identity, got %v", err)
	}

	list, _, err := repo.ListIdentities(ctx, postgres.ListIdentitiesFilter{Limit: 50})
	if err != nil {
		t.Fatalf("ListIdentities: %v", err)
	}
	for _, id := range list {
		if id.ID == created.ID {
			t.Errorf("ListIdentities returned soft-deleted identity %v", created.ID)
		}
	}

	// Deleting again should return ErrNotFound
	err = repo.DeleteIdentity(ctx, created.ID)
	if !errors.Is(err, postgres.ErrNotFound) {
		t.Errorf("expected ErrNotFound on second DeleteIdentity, got %v", err)
	}
}

func TestListIdentities_Filters(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Seed 3 users with different traits and states
	user1, err := repo.CreateIdentityWithPassword(ctx, map[string]interface{}{
		"email":      "dev1@autorix.io",
		"department": "engineering",
		"role":       "admin",
	}, "hash-1")
	if err != nil {
		t.Fatalf("create user1: %v", err)
	}

	user2, err := repo.CreateIdentityWithPassword(ctx, map[string]interface{}{
		"email":      "sales1@autorix.io",
		"department": "sales",
		"role":       "viewer",
	}, "hash-2")
	if err != nil {
		t.Fatalf("create user2: %v", err)
	}

	user3, err := repo.CreateIdentityWithPassword(ctx, map[string]interface{}{
		"email":      "dev2@autorix.io",
		"department": "engineering",
		"role":       "developer",
	}, "hash-3")
	if err != nil {
		t.Fatalf("create user3: %v", err)
	}

	// Deactivate user3
	inactiveState := core.StateInactive
	_, err = repo.UpdateIdentity(ctx, user3.ID, nil, &inactiveState, nil)
	if err != nil {
		t.Fatalf("deactivate user3: %v", err)
	}

	// 1. Filter by state: active
	activeList, _, err := repo.ListIdentities(ctx, postgres.ListIdentitiesFilter{
		State: core.StateActive,
	})
	if err != nil {
		t.Fatalf("list active: %v", err)
	}
	if len(activeList) != 2 {
		t.Errorf("expected 2 active users, got %d", len(activeList))
	}

	// 2. Filter by state: inactive
	inactiveList, _, err := repo.ListIdentities(ctx, postgres.ListIdentitiesFilter{
		State: core.StateInactive,
	})
	if err != nil {
		t.Fatalf("list inactive: %v", err)
	}
	if len(inactiveList) != 1 || inactiveList[0].ID != user3.ID {
		t.Errorf("expected 1 inactive user (user3), got %v", inactiveList)
	}

	// 3. Filter by JSONB traits containment (@> idx_identities_traits_gin)
	engList, _, err := repo.ListIdentities(ctx, postgres.ListIdentitiesFilter{
		Traits: map[string]interface{}{"department": "engineering"},
	})
	if err != nil {
		t.Fatalf("list engineering: %v", err)
	}
	if len(engList) != 2 {
		t.Errorf("expected 2 engineering users (user1, user3), got %d", len(engList))
	}

	adminList, _, err := repo.ListIdentities(ctx, postgres.ListIdentitiesFilter{
		Traits: map[string]interface{}{"role": "admin"},
	})
	if err != nil {
		t.Fatalf("list admin: %v", err)
	}
	if len(adminList) != 1 || adminList[0].ID != user1.ID {
		t.Errorf("expected 1 admin user (user1), got %v", adminList)
	}

	// 4. Filter by text query substring
	queryList, _, err := repo.ListIdentities(ctx, postgres.ListIdentitiesFilter{
		Query: "sales1",
	})
	if err != nil {
		t.Fatalf("list query sales1: %v", err)
	}
	if len(queryList) != 1 || queryList[0].ID != user2.ID {
		t.Errorf("expected 1 user matching query 'sales1', got %v", queryList)
	}
}

func TestListActiveSessions(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	user, err := repo.CreateIdentityWithPassword(ctx, map[string]interface{}{"email": "sesslist@example.com"}, "hash")
	if err != nil {
		t.Fatalf("CreateIdentity: %v", err)
	}

	// 1 active session, 1 expired session
	activeSess := &core.Session{
		ID:              uuid.New(),
		IdentityID:      user.ID,
		TokenHash:       "active-sess-token-1",
		ExpiresAt:       time.Now().Add(1 * time.Hour),
		AuthenticatedAt: time.Now(),
	}
	if err := repo.CreateSession(ctx, activeSess); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	expiredSess := &core.Session{
		ID:              uuid.New(),
		IdentityID:      user.ID,
		TokenHash:       "expired-sess-token-1",
		ExpiresAt:       time.Now().Add(-1 * time.Hour),
		AuthenticatedAt: time.Now().Add(-2 * time.Hour),
	}
	if err := repo.CreateSession(ctx, expiredSess); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	sessions, hasMore, err := repo.ListActiveSessions(ctx, 10, "")
	if err != nil {
		t.Fatalf("ListActiveSessions: unexpected error: %v", err)
	}
	if hasMore {
		t.Errorf("expected hasMore=false, got true")
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 active session, got %d", len(sessions))
	}
	if sessions[0].ID != activeSess.ID {
		t.Errorf("sessions[0].ID = %v, want %v", sessions[0].ID, activeSess.ID)
	}
	if sessions[0].Identity == nil || sessions[0].Identity.ID != user.ID {
		t.Errorf("expected joined identity for session")
	}
}

func TestListActiveSessionsByIdentity(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	user1, _ := repo.CreateIdentityWithPassword(ctx, map[string]interface{}{"email": "u1@example.com"}, "hash")
	user2, _ := repo.CreateIdentityWithPassword(ctx, map[string]interface{}{"email": "u2@example.com"}, "hash")

	s1 := &core.Session{ID: uuid.New(), IdentityID: user1.ID, TokenHash: "u1-s1", ExpiresAt: time.Now().Add(time.Hour), AuthenticatedAt: time.Now()}
	s2 := &core.Session{ID: uuid.New(), IdentityID: user1.ID, TokenHash: "u1-s2", ExpiresAt: time.Now().Add(time.Hour), AuthenticatedAt: time.Now()}
	s3 := &core.Session{ID: uuid.New(), IdentityID: user2.ID, TokenHash: "u2-s1", ExpiresAt: time.Now().Add(time.Hour), AuthenticatedAt: time.Now()}
	_ = repo.CreateSession(ctx, s1)
	_ = repo.CreateSession(ctx, s2)
	_ = repo.CreateSession(ctx, s3)

	u1Sessions, err := repo.ListActiveSessionsByIdentity(ctx, user1.ID)
	if err != nil {
		t.Fatalf("ListActiveSessionsByIdentity: %v", err)
	}
	if len(u1Sessions) != 2 {
		t.Fatalf("expected 2 sessions for user1, got %d", len(u1Sessions))
	}

	u2Sessions, err := repo.ListActiveSessionsByIdentity(ctx, user2.ID)
	if err != nil {
		t.Fatalf("ListActiveSessionsByIdentity user2: %v", err)
	}
	if len(u2Sessions) != 1 {
		t.Fatalf("expected 1 session for user2, got %d", len(u2Sessions))
	}
}

func TestDeleteSessionByID_And_BulkDelete(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	user, _ := repo.CreateIdentityWithPassword(ctx, map[string]interface{}{"email": "del-sess@example.com"}, "hash")
	s1 := &core.Session{ID: uuid.New(), IdentityID: user.ID, TokenHash: "del-s1", ExpiresAt: time.Now().Add(time.Hour), AuthenticatedAt: time.Now()}
	s2 := &core.Session{ID: uuid.New(), IdentityID: user.ID, TokenHash: "del-s2", ExpiresAt: time.Now().Add(time.Hour), AuthenticatedAt: time.Now()}
	_ = repo.CreateSession(ctx, s1)
	_ = repo.CreateSession(ctx, s2)

	// 1. Delete single session by ID
	err := repo.DeleteSessionByID(ctx, s1.ID)
	if err != nil {
		t.Fatalf("DeleteSessionByID: %v", err)
	}
	_, err = repo.GetSessionByTokenHash(ctx, "del-s1")
	if !errors.Is(err, postgres.ErrNotFound) {
		t.Errorf("expected ErrNotFound for deleted session s1, got %v", err)
	}

	// Deleting nonexistent session returns ErrNotFound
	err = repo.DeleteSessionByID(ctx, s1.ID)
	if !errors.Is(err, postgres.ErrNotFound) {
		t.Errorf("expected ErrNotFound when deleting already deleted session, got %v", err)
	}

	// 2. Bulk delete sessions by Identity ID
	err = repo.DeleteSessionsByIdentityID(ctx, user.ID)
	if err != nil {
		t.Fatalf("DeleteSessionsByIdentityID: %v", err)
	}
	list, err := repo.ListActiveSessionsByIdentity(ctx, user.ID)
	if err != nil {
		t.Fatalf("ListActiveSessionsByIdentity after bulk delete: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("expected 0 sessions after bulk delete, got %d", len(list))
	}
}

func TestSetPasswordCredential_And_ListCredentials(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	user, err := repo.CreateIdentityWithPassword(ctx, map[string]interface{}{"email": "credtest@example.com"}, "initial-hash")
	if err != nil {
		t.Fatalf("CreateIdentity: %v", err)
	}

	// 1. Inspect initial credentials
	creds, err := repo.ListCredentialsByIdentity(ctx, user.ID)
	if err != nil {
		t.Fatalf("ListCredentialsByIdentity: %v", err)
	}
	if len(creds) != 1 {
		t.Fatalf("expected 1 initial credential, got %d", len(creds))
	}
	if creds[0].CredentialType != "password" {
		t.Errorf("creds[0].CredentialType = %q, want password", creds[0].CredentialType)
	}
	if creds[0].ForceRotation {
		t.Errorf("expected force_rotation=false initially")
	}

	// 2. Reset password with forced rotation
	err = repo.SetPasswordCredential(ctx, user.ID, "new-reset-hash", true)
	if err != nil {
		t.Fatalf("SetPasswordCredential: %v", err)
	}

	// Verify updated password hash
	hash, err := repo.GetPasswordCredential(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetPasswordCredential: %v", err)
	}
	if hash != "new-reset-hash" {
		t.Errorf("hash = %q, want new-reset-hash", hash)
	}

	// Verify force_rotation flag in inspection
	creds, err = repo.ListCredentialsByIdentity(ctx, user.ID)
	if err != nil {
		t.Fatalf("ListCredentialsByIdentity: %v", err)
	}
	if len(creds) != 1 {
		t.Fatalf("expected 1 credential, got %d", len(creds))
	}
	if !creds[0].ForceRotation {
		t.Errorf("expected force_rotation=true after reset")
	}
}

func TestRecoveryToken_Save_And_Get(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	user, err := repo.CreateIdentityWithPassword(ctx, map[string]interface{}{"email": "recoverytest@example.com"}, "hash")
	if err != nil {
		t.Fatalf("CreateIdentity: %v", err)
	}

	expiresAt := time.Now().Add(1 * time.Hour)
	err = repo.SaveRecoveryToken(ctx, user.ID, "hashed-recovery-token-xyz", expiresAt)
	if err != nil {
		t.Fatalf("SaveRecoveryToken: %v", err)
	}

	tokenHash, exp, err := repo.GetRecoveryToken(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetRecoveryToken: %v", err)
	}
	if tokenHash != "hashed-recovery-token-xyz" {
		t.Errorf("tokenHash = %q, want hashed-recovery-token-xyz", tokenHash)
	}
	if exp.Sub(expiresAt).Abs() > time.Second {
		t.Errorf("exp = %v, want %v", exp, expiresAt)
	}
}

func TestTOTPCredential_CRUD(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	user, err := repo.CreateIdentityWithPassword(ctx, map[string]interface{}{"email": "totptest@example.com"}, "hash")
	if err != nil {
		t.Fatalf("CreateIdentity: %v", err)
	}

	// 1. Initially no TOTP
	_, _, _, err = repo.GetTOTPCredential(ctx, user.ID)
	if !errors.Is(err, postgres.ErrNotFound) {
		t.Fatalf("expected ErrNotFound initially, got %v", err)
	}

	// 2. Set TOTP credential
	backupHashes := []string{"hash-code-1", "hash-code-2"}
	err = repo.SetTOTPCredential(ctx, user.ID, "JBSWY3DPEHPK3PXP", backupHashes, true)
	if err != nil {
		t.Fatalf("SetTOTPCredential: %v", err)
	}

	// 3. Get TOTP credential
	secret, codes, confirmed, err := repo.GetTOTPCredential(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetTOTPCredential: %v", err)
	}
	if secret != "JBSWY3DPEHPK3PXP" {
		t.Errorf("secret = %q, want JBSWY3DPEHPK3PXP", secret)
	}
	if len(codes) != 2 || codes[0] != "hash-code-1" {
		t.Errorf("codes = %v, want %v", codes, backupHashes)
	}
	if !confirmed {
		t.Errorf("expected confirmed=true")
	}

	// 4. Delete TOTP credential
	err = repo.DeleteTOTPCredential(ctx, user.ID)
	if err != nil {
		t.Fatalf("DeleteTOTPCredential: %v", err)
	}
	_, _, _, err = repo.GetTOTPCredential(ctx, user.ID)
	if !errors.Is(err, postgres.ErrNotFound) {
		t.Errorf("expected ErrNotFound after deletion, got %v", err)
	}
}

func TestSchemaCRUD(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	rawSchema := map[string]interface{}{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type":    "object",
		"properties": map[string]interface{}{
			"email": map[string]interface{}{"type": "string"},
			"name":  map[string]interface{}{"type": "string"},
		},
		"required": []interface{}{"email"},
	}

	// 1. Create Schema
	schema := &core.IdentitySchema{
		ID:     "customer",
		Name:   "Customer Identity",
		Schema: rawSchema,
	}
	created, err := repo.CreateSchema(ctx, schema)
	if err != nil {
		t.Fatalf("CreateSchema: %v", err)
	}
	if created.ID != "customer" || created.Version != 1 {
		t.Errorf("created schema id=%s, version=%d", created.ID, created.Version)
	}

	// 2. Get Schema by ID
	got, err := repo.GetSchemaByID(ctx, "customer")
	if err != nil {
		t.Fatalf("GetSchemaByID: %v", err)
	}
	if got.Name != "Customer Identity" {
		t.Errorf("got.Name = %q, want Customer Identity", got.Name)
	}

	// 3. List Schemas
	list, err := repo.ListSchemas(ctx)
	if err != nil {
		t.Fatalf("ListSchemas: %v", err)
	}
	if len(list) < 1 {
		t.Fatalf("expected at least 1 schema, got %d", len(list))
	}

	// 4. Update Schema
	newName := "Updated Customer Identity"
	updated, err := repo.UpdateSchema(ctx, "customer", &newName, nil)
	if err != nil {
		t.Fatalf("UpdateSchema: %v", err)
	}
	if updated.Name != "Updated Customer Identity" {
		t.Errorf("updated.Name = %q, want Updated Customer Identity", updated.Name)
	}
	if updated.Version != 2 {
		t.Errorf("expected version to increment to 2, got %d", updated.Version)
	}

	// 5. Delete Schema
	err = repo.DeleteSchema(ctx, "customer")
	if err != nil {
		t.Fatalf("DeleteSchema: %v", err)
	}
	_, err = repo.GetSchemaByID(ctx, "customer")
	if !errors.Is(err, postgres.ErrNotFound) {
		t.Errorf("expected ErrNotFound after schema deletion, got %v", err)
	}

	// Delete non-existent schema
	err = repo.DeleteSchema(ctx, "nonexistent")
	if !errors.Is(err, postgres.ErrNotFound) {
		t.Errorf("expected ErrNotFound when deleting nonexistent schema, got %v", err)
	}
}





