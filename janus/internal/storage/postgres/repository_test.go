package postgres_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/autorix/janus/internal/core"
	"github.com/autorix/janus/internal/storage/postgres"
	"github.com/autorix/platform/paging"
	"github.com/autorix/platform/pgtest"
	"github.com/google/uuid"
)

func newTestRepo(t *testing.T) *postgres.Repository {
	t.Helper()
	pool := pgtest.StartPostgres(t, "../../../migrations")
	return postgres.NewRepository(pool)
}

func newTestClient(id string) *core.OAuth2Client {
	return &core.OAuth2Client{
		ID:               id,
		ClientName:       "Test Client",
		ClientSecretHash: "hashed-secret",
		GrantTypes:       []string{"authorization_code"},
		ResponseTypes:    []string{"code"},
		RedirectURIs:     []string{"https://example.com/callback"},
		Scopes:           []string{"openid", "offline_access"},
		IsPublic:         false,
	}
}

func TestCreateAndGetClient(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	client := newTestClient("client-" + uuid.NewString())
	if err := repo.CreateClient(ctx, client); err != nil {
		t.Fatalf("CreateClient() error = %v", err)
	}

	if client.CreatedAt.IsZero() {
		t.Error("CreateClient() did not populate CreatedAt")
	}
	if client.UpdatedAt.IsZero() {
		t.Error("CreateClient() did not populate UpdatedAt")
	}

	got, err := repo.GetClient(ctx, client.ID)
	if err != nil {
		t.Fatalf("GetClient() error = %v", err)
	}

	if got.ID != client.ID {
		t.Errorf("GetClient() ID = %q, want %q", got.ID, client.ID)
	}
	if got.ClientName != client.ClientName {
		t.Errorf("GetClient() ClientName = %q, want %q", got.ClientName, client.ClientName)
	}
	if got.ClientSecretHash != client.ClientSecretHash {
		t.Errorf("GetClient() ClientSecretHash = %q, want %q", got.ClientSecretHash, client.ClientSecretHash)
	}
	if got.IsPublic != client.IsPublic {
		t.Errorf("GetClient() IsPublic = %v, want %v", got.IsPublic, client.IsPublic)
	}
	if len(got.GrantTypes) != 1 || got.GrantTypes[0] != "authorization_code" {
		t.Errorf("GetClient() GrantTypes = %v, want [authorization_code]", got.GrantTypes)
	}
	if len(got.RedirectURIs) != 1 || got.RedirectURIs[0] != "https://example.com/callback" {
		t.Errorf("GetClient() RedirectURIs = %v, want [https://example.com/callback]", got.RedirectURIs)
	}
}

// TestCreateClient_DefaultsNilSlicesToEmpty verifies the repository normalizes
// nil slice fields to empty slices before insert (the NOT NULL columns would
// otherwise reject a nil array in some drivers/paths).
func TestCreateClient_DefaultsNilSlicesToEmpty(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	client := &core.OAuth2Client{
		ID:         "client-" + uuid.NewString(),
		ClientName: "Nil Slices Client",
		IsPublic:   true,
		// GrantTypes, ResponseTypes, RedirectURIs, Scopes intentionally left nil.
	}

	if err := repo.CreateClient(ctx, client); err != nil {
		t.Fatalf("CreateClient() error = %v", err)
	}

	got, err := repo.GetClient(ctx, client.ID)
	if err != nil {
		t.Fatalf("GetClient() error = %v", err)
	}

	if got.GrantTypes == nil || len(got.GrantTypes) != 0 {
		t.Errorf("GetClient() GrantTypes = %v, want empty slice", got.GrantTypes)
	}
	if got.RedirectURIs == nil || len(got.RedirectURIs) != 0 {
		t.Errorf("GetClient() RedirectURIs = %v, want empty slice", got.RedirectURIs)
	}
}

// TestCreateClient_DuplicateID exercises the real Postgres primary-key
// constraint on oauth2_clients.id (the client_id). The repository does not
// translate this into a sentinel error; it wraps the raw pgx error.
func TestCreateClient_DuplicateID(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	id := "client-" + uuid.NewString()
	first := newTestClient(id)
	if err := repo.CreateClient(ctx, first); err != nil {
		t.Fatalf("CreateClient() first insert error = %v", err)
	}

	second := newTestClient(id)
	err := repo.CreateClient(ctx, second)
	if err == nil {
		t.Fatal("CreateClient() with duplicate ID: want error, got nil")
	}
	if errors.Is(err, postgres.ErrNotFound) {
		t.Errorf("CreateClient() duplicate ID error = %v, unexpectedly wrapped as ErrNotFound", err)
	}
	// The repository does not translate this to a typed conflict error; it
	// returns a wrapped generic error from the failed INSERT. Assert only
	// that an error surfaces and the second client's timestamps were left
	// unset (CreateClient only stamps CreatedAt/UpdatedAt after a
	// successful Exec).
	if !second.CreatedAt.IsZero() {
		t.Error("CreateClient() on failure unexpectedly set CreatedAt on the failed insert's struct")
	}
}

func TestGetClient_NotFound(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	got, err := repo.GetClient(ctx, "does-not-exist")
	if !errors.Is(err, postgres.ErrNotFound) {
		t.Fatalf("GetClient() error = %v, want ErrNotFound", err)
	}
	if got != nil {
		t.Errorf("GetClient() on not found = %+v, want nil", got)
	}
}

func TestListClients(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// No clients yet.
	empty, hasMore, err := repo.ListClients(ctx, 20, "")
	if err != nil {
		t.Fatalf("ListClients() error = %v", err)
	}
	if len(empty) != 0 {
		t.Errorf("ListClients() on empty table = %d clients, want 0", len(empty))
	}
	if hasMore {
		t.Errorf("ListClients() on empty table: hasMore = true, want false")
	}

	c1 := newTestClient("client-a-" + uuid.NewString())
	c2 := newTestClient("client-b-" + uuid.NewString())
	if err := repo.CreateClient(ctx, c1); err != nil {
		t.Fatalf("CreateClient(c1) error = %v", err)
	}
	// Ensure a distinguishable created_at ordering (ORDER BY created_at DESC).
	time.Sleep(10 * time.Millisecond)
	if err := repo.CreateClient(ctx, c2); err != nil {
		t.Fatalf("CreateClient(c2) error = %v", err)
	}

	clients, hasMore, err := repo.ListClients(ctx, 20, "")
	if err != nil {
		t.Fatalf("ListClients() error = %v", err)
	}
	if len(clients) != 2 {
		t.Fatalf("ListClients() = %d clients, want 2", len(clients))
	}
	if hasMore {
		t.Errorf("ListClients() hasMore = true, want false (both fit in one page)")
	}
	// Most-recently created first.
	if clients[0].ID != c2.ID {
		t.Errorf("ListClients()[0].ID = %q, want %q (most recent first)", clients[0].ID, c2.ID)
	}
	if clients[1].ID != c1.ID {
		t.Errorf("ListClients()[1].ID = %q, want %q", clients[1].ID, c1.ID)
	}
}

// TestListClients_CursorPagination proves ListClients paginates via a real
// SQL keyset: a page smaller than the full set returns exactly N rows plus
// a cursor, and following that cursor returns the rest with no duplicate
// and no skipped row.
func TestListClients_CursorPagination(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	const total = 5
	ids := make(map[string]bool, total)
	for i := 0; i < total; i++ {
		c := newTestClient(fmt.Sprintf("cursor-client-%d-%s", i, uuid.NewString()))
		if err := repo.CreateClient(ctx, c); err != nil {
			t.Fatalf("CreateClient(%s) error = %v", c.ID, err)
		}
		ids[c.ID] = true
		time.Sleep(2 * time.Millisecond)
	}

	const pageSize = 2
	var all []core.OAuth2Client
	cursor := ""
	for i := 0; i < 10; i++ { // bounded loop guard against an infinite pagination bug
		page, hasMore, err := repo.ListClients(ctx, pageSize, cursor)
		if err != nil {
			t.Fatalf("ListClients(cursor=%q) error = %v", cursor, err)
		}
		all = append(all, page...)
		if !hasMore {
			break
		}
		if len(page) == 0 {
			t.Fatalf("hasMore=true but page was empty")
		}
		last := page[len(page)-1]
		cursor = paging.EncodeCursor(last.CreatedAt.Format(time.RFC3339Nano) + "|" + last.ID)
	}

	if len(all) != total {
		t.Fatalf("expected %d total clients across pages, got %d", total, len(all))
	}
	seen := map[string]bool{}
	for _, c := range all {
		if seen[c.ID] {
			t.Fatalf("duplicate client %q across pages", c.ID)
		}
		seen[c.ID] = true
		delete(ids, c.ID)
	}
	if len(ids) != 0 {
		t.Fatalf("some clients were skipped across pages: %v", ids)
	}
}

func newTestGrant(t *testing.T, ctx context.Context, repo *postgres.Repository, clientID string) *core.Grant {
	t.Helper()
	client := newTestClient(clientID)
	if err := repo.CreateClient(ctx, client); err != nil {
		t.Fatalf("CreateClient() error = %v", err)
	}
	return &core.Grant{
		CodeHash:            "code-hash-" + uuid.NewString(),
		ClientID:            client.ID,
		Subject:             "user-123",
		Scopes:              []string{"openid"},
		RedirectURI:         "https://example.com/callback",
		CodeChallenge:       "challenge",
		CodeChallengeMethod: "S256",
		ExpiresAt:           time.Now().Add(5 * time.Minute).UTC(),
		Consumed:            false,
	}
}

func TestCreateAndConsumeGrant(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	grant := newTestGrant(t, ctx, repo, "client-"+uuid.NewString())
	if err := repo.CreateGrant(ctx, grant); err != nil {
		t.Fatalf("CreateGrant() error = %v", err)
	}

	consumed, err := repo.ConsumeGrant(ctx, grant.CodeHash)
	if err != nil {
		t.Fatalf("ConsumeGrant() error = %v", err)
	}

	if consumed.ClientID != grant.ClientID {
		t.Errorf("ConsumeGrant() ClientID = %q, want %q", consumed.ClientID, grant.ClientID)
	}
	if consumed.Subject != grant.Subject {
		t.Errorf("ConsumeGrant() Subject = %q, want %q", consumed.Subject, grant.Subject)
	}
	if !consumed.Consumed {
		t.Error("ConsumeGrant() Consumed = false, want true")
	}
	if consumed.RedirectURI != grant.RedirectURI {
		t.Errorf("ConsumeGrant() RedirectURI = %q, want %q", consumed.RedirectURI, grant.RedirectURI)
	}
}

// TestConsumeGrant_AlreadyConsumed proves the atomic UPDATE ... WHERE
// consumed = false guard actually prevents a second consumption of the same
// authorization code (replay protection).
func TestConsumeGrant_AlreadyConsumed(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	grant := newTestGrant(t, ctx, repo, "client-"+uuid.NewString())
	if err := repo.CreateGrant(ctx, grant); err != nil {
		t.Fatalf("CreateGrant() error = %v", err)
	}

	if _, err := repo.ConsumeGrant(ctx, grant.CodeHash); err != nil {
		t.Fatalf("ConsumeGrant() first call error = %v", err)
	}

	// Second consumption attempt must fail as ErrNotFound: the row exists
	// but no longer matches "consumed = false".
	second, err := repo.ConsumeGrant(ctx, grant.CodeHash)
	if !errors.Is(err, postgres.ErrNotFound) {
		t.Fatalf("ConsumeGrant() second call error = %v, want ErrNotFound", err)
	}
	if second != nil {
		t.Errorf("ConsumeGrant() second call = %+v, want nil", second)
	}
}

// TestConsumeGrant_Expired proves an expired-but-unconsumed grant is treated
// identically to a missing one (expires_at > CURRENT_TIMESTAMP guard).
func TestConsumeGrant_Expired(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	grant := newTestGrant(t, ctx, repo, "client-"+uuid.NewString())
	grant.ExpiresAt = time.Now().Add(-1 * time.Minute).UTC()
	if err := repo.CreateGrant(ctx, grant); err != nil {
		t.Fatalf("CreateGrant() error = %v", err)
	}

	got, err := repo.ConsumeGrant(ctx, grant.CodeHash)
	if !errors.Is(err, postgres.ErrNotFound) {
		t.Fatalf("ConsumeGrant() on expired grant error = %v, want ErrNotFound", err)
	}
	if got != nil {
		t.Errorf("ConsumeGrant() on expired grant = %+v, want nil", got)
	}
}

func TestConsumeGrant_NotFound(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	got, err := repo.ConsumeGrant(ctx, "no-such-code-hash")
	if !errors.Is(err, postgres.ErrNotFound) {
		t.Fatalf("ConsumeGrant() error = %v, want ErrNotFound", err)
	}
	if got != nil {
		t.Errorf("ConsumeGrant() on not found = %+v, want nil", got)
	}
}

// TestCreateGrant_UnknownClient exercises the real foreign-key constraint on
// oauth2_grants.client_id -> oauth2_clients.id.
func TestCreateGrant_UnknownClient(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	grant := &core.Grant{
		CodeHash:            "code-hash-" + uuid.NewString(),
		ClientID:            "no-such-client",
		Subject:             "user-123",
		Scopes:              []string{"openid"},
		RedirectURI:         "https://example.com/callback",
		CodeChallenge:       "challenge",
		CodeChallengeMethod: "S256",
		ExpiresAt:           time.Now().Add(5 * time.Minute).UTC(),
		Consumed:            false,
	}

	err := repo.CreateGrant(ctx, grant)
	if err == nil {
		t.Fatal("CreateGrant() with unknown client_id: want error, got nil")
	}
	if errors.Is(err, postgres.ErrNotFound) {
		t.Errorf("CreateGrant() unknown client_id error = %v, unexpectedly wrapped as ErrNotFound", err)
	}

	// The grant must not have been persisted; consuming it must report not found.
	got, consumeErr := repo.ConsumeGrant(ctx, grant.CodeHash)
	if !errors.Is(consumeErr, postgres.ErrNotFound) {
		t.Fatalf("ConsumeGrant() after failed CreateGrant() error = %v, want ErrNotFound", consumeErr)
	}
	if got != nil {
		t.Errorf("ConsumeGrant() after failed CreateGrant() = %+v, want nil", got)
	}
}

func TestUpdateAndDeleteClient(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	client := newTestClient("client-update-" + uuid.NewString())
	if err := repo.CreateClient(ctx, client); err != nil {
		t.Fatalf("CreateClient() error = %v", err)
	}

	// Update fields
	client.ClientName = "Updated Client Name"
	client.Scopes = []string{"openid", "profile", "email"}
	client.IsPublic = true
	if err := repo.UpdateClient(ctx, client); err != nil {
		t.Fatalf("UpdateClient() error = %v", err)
	}

	got, err := repo.GetClient(ctx, client.ID)
	if err != nil {
		t.Fatalf("GetClient() error = %v", err)
	}
	if got.ClientName != "Updated Client Name" {
		t.Errorf("GetClient() ClientName = %q, want %q", got.ClientName, "Updated Client Name")
	}
	if !got.IsPublic {
		t.Error("GetClient() IsPublic = false, want true")
	}
	if len(got.Scopes) != 3 {
		t.Errorf("GetClient() Scopes = %v, want 3 scopes", got.Scopes)
	}

	// Delete client
	if err := repo.DeleteClient(ctx, client.ID); err != nil {
		t.Fatalf("DeleteClient() error = %v", err)
	}

	// Verify not found after deletion
	_, err = repo.GetClient(ctx, client.ID)
	if !errors.Is(err, postgres.ErrNotFound) {
		t.Fatalf("GetClient() after DeleteClient error = %v, want ErrNotFound", err)
	}

	// Delete non-existent returns ErrNotFound
	if err := repo.DeleteClient(ctx, "non-existent-client"); !errors.Is(err, postgres.ErrNotFound) {
		t.Errorf("DeleteClient() non-existent error = %v, want ErrNotFound", err)
	}
}

func TestRotateClientSecret(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	client := newTestClient("client-rotate-" + uuid.NewString())
	client.ClientSecretHash = "original-secret-hash"
	if err := repo.CreateClient(ctx, client); err != nil {
		t.Fatalf("CreateClient() error = %v", err)
	}

	overlapExpiry := time.Now().Add(24 * time.Hour).UTC()
	newSecretHash := "new-rotated-secret-hash"

	if err := repo.RotateClientSecret(ctx, client.ID, newSecretHash, &overlapExpiry); err != nil {
		t.Fatalf("RotateClientSecret() error = %v", err)
	}

	got, err := repo.GetClient(ctx, client.ID)
	if err != nil {
		t.Fatalf("GetClient() error = %v", err)
	}

	if got.ClientSecretHash != newSecretHash {
		t.Errorf("GetClient() ClientSecretHash = %q, want %q", got.ClientSecretHash, newSecretHash)
	}
	if got.PreviousSecretHash != "original-secret-hash" {
		t.Errorf("GetClient() PreviousSecretHash = %q, want %q", got.PreviousSecretHash, "original-secret-hash")
	}
	if got.PreviousSecretExpiresAt == nil {
		t.Fatal("GetClient() PreviousSecretExpiresAt is nil")
	}
	if got.PreviousSecretExpiresAt.Sub(overlapExpiry).Abs() > time.Second {
		t.Errorf("GetClient() PreviousSecretExpiresAt = %v, want %v", got.PreviousSecretExpiresAt, overlapExpiry)
	}
}

func TestListGrants(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	clientID1 := "client-g1-" + uuid.NewString()
	clientID2 := "client-g2-" + uuid.NewString()

	c1 := newTestClient(clientID1)
	c2 := newTestClient(clientID2)
	if err := repo.CreateClient(ctx, c1); err != nil {
		t.Fatalf("CreateClient(c1) error = %v", err)
	}
	if err := repo.CreateClient(ctx, c2); err != nil {
		t.Fatalf("CreateClient(c2) error = %v", err)
	}

	// Active grant for client 1, user A
	g1 := &core.Grant{
		CodeHash:    "code-1-" + uuid.NewString(),
		ClientID:    clientID1,
		Subject:     "user-a",
		Scopes:      []string{"openid"},
		RedirectURI: "https://example.com/callback",
		ExpiresAt:   time.Now().Add(10 * time.Minute).UTC(),
		Consumed:    false,
	}
	// Consumed grant for client 1, user A
	g2 := &core.Grant{
		CodeHash:    "code-2-" + uuid.NewString(),
		ClientID:    clientID1,
		Subject:     "user-a",
		Scopes:      []string{"openid"},
		RedirectURI: "https://example.com/callback",
		ExpiresAt:   time.Now().Add(10 * time.Minute).UTC(),
		Consumed:    true,
	}
	// Active grant for client 2, user B
	g3 := &core.Grant{
		CodeHash:    "code-3-" + uuid.NewString(),
		ClientID:    clientID2,
		Subject:     "user-b",
		Scopes:      []string{"profile"},
		RedirectURI: "https://example.com/callback",
		ExpiresAt:   time.Now().Add(10 * time.Minute).UTC(),
		Consumed:    false,
	}

	_ = repo.CreateGrant(ctx, g1)
	time.Sleep(5 * time.Millisecond)
	_ = repo.CreateGrant(ctx, g2)
	time.Sleep(5 * time.Millisecond)
	_ = repo.CreateGrant(ctx, g3)

	// Filter by clientID1
	grants, _, err := repo.ListGrants(ctx, clientID1, "", false, 10, "")
	if err != nil {
		t.Fatalf("ListGrants(clientID1) error = %v", err)
	}
	if len(grants) != 2 {
		t.Errorf("ListGrants(clientID1) got %d grants, want 2", len(grants))
	}

	// Filter active only for clientID1
	activeGrants, _, err := repo.ListGrants(ctx, clientID1, "", true, 10, "")
	if err != nil {
		t.Fatalf("ListGrants(clientID1, active=true) error = %v", err)
	}
	if len(activeGrants) != 1 {
		t.Errorf("ListGrants(clientID1, active=true) got %d grants, want 1", len(activeGrants))
	}
	if len(activeGrants) > 0 && activeGrants[0].CodeHash != g1.CodeHash {
		t.Errorf("ListGrants() active grant code_hash = %q, want %q", activeGrants[0].CodeHash, g1.CodeHash)
	}

	// Filter by subject
	userBGrants, _, err := repo.ListGrants(ctx, "", "user-b", false, 10, "")
	if err != nil {
		t.Fatalf("ListGrants(user-b) error = %v", err)
	}
	if len(userBGrants) != 1 || userBGrants[0].ClientID != clientID2 {
		t.Errorf("ListGrants(user-b) got %v, want clientID2", userBGrants)
	}
}

func TestTokenRevocationAndRecords(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	client := newTestClient("client-tok-" + uuid.NewString())
	if err := repo.CreateClient(ctx, client); err != nil {
		t.Fatalf("CreateClient() error = %v", err)
	}

	tokenHash := "token-hash-" + uuid.NewString()

	// Initial check: not revoked
	revoked, err := repo.IsTokenRevoked(ctx, tokenHash)
	if err != nil {
		t.Fatalf("IsTokenRevoked() before revocation error = %v", err)
	}
	if revoked {
		t.Error("IsTokenRevoked() before revocation = true, want false")
	}

	// Revoke token
	record := &core.TokenRecord{
		TokenHash: tokenHash,
		ClientID:  client.ID,
		Subject:   "user-100",
		TokenType: "access_token",
		Scopes:    []string{"openid", "email"},
		ExpiresAt: time.Now().Add(1 * time.Hour).UTC(),
	}
	if err := repo.RevokeToken(ctx, record); err != nil {
		t.Fatalf("RevokeToken() error = %v", err)
	}

	// Check revocation again: should be true
	revoked, err = repo.IsTokenRevoked(ctx, tokenHash)
	if err != nil {
		t.Fatalf("IsTokenRevoked() after revocation error = %v", err)
	}
	if !revoked {
		t.Error("IsTokenRevoked() after revocation = false, want true")
	}

	// Get token record
	rec, err := repo.GetTokenRecord(ctx, tokenHash)
	if err != nil {
		t.Fatalf("GetTokenRecord() error = %v", err)
	}
	if rec.ClientID != client.ID {
		t.Errorf("GetTokenRecord() ClientID = %q, want %q", rec.ClientID, client.ID)
	}
	if !rec.Revoked {
		t.Error("GetTokenRecord() Revoked = false, want true")
	}
}

func TestScopeCatalogueCRUD(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	scope := &core.OAuth2Scope{
		Name:        "profile:read",
		Description: "Read user profile info",
		Claims:      []string{"name", "nickname", "picture"},
	}

	// 1. Create scope
	if err := repo.CreateScope(ctx, scope); err != nil {
		t.Fatalf("CreateScope() error = %v", err)
	}
	if scope.CreatedAt.IsZero() {
		t.Error("CreateScope() did not populate CreatedAt")
	}

	// 2. Get scope
	got, err := repo.GetScope(ctx, scope.Name)
	if err != nil {
		t.Fatalf("GetScope() error = %v", err)
	}
	if got.Name != scope.Name {
		t.Errorf("GetScope() Name = %q, want %q", got.Name, scope.Name)
	}
	if got.Description != scope.Description {
		t.Errorf("GetScope() Description = %q, want %q", got.Description, scope.Description)
	}
	if len(got.Claims) != 3 || got.Claims[0] != "name" {
		t.Errorf("GetScope() Claims = %v, want [name nickname picture]", got.Claims)
	}

	// 3. List scopes
	scopes, err := repo.ListScopes(ctx)
	if err != nil {
		t.Fatalf("ListScopes() error = %v", err)
	}
	if len(scopes) == 0 {
		t.Error("ListScopes() returned 0 scopes, expected at least 1")
	}

	// 4. Delete scope
	if err := repo.DeleteScope(ctx, scope.Name); err != nil {
		t.Fatalf("DeleteScope() error = %v", err)
	}

	// 5. Get scope after deletion returns ErrNotFound
	_, err = repo.GetScope(ctx, scope.Name)
	if !errors.Is(err, postgres.ErrNotFound) {
		t.Fatalf("GetScope() after delete error = %v, want ErrNotFound", err)
	}
}

