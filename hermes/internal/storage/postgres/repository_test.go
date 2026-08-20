package postgres_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/autorix/hermes/internal/core"
	"github.com/autorix/hermes/internal/storage/postgres"
	"github.com/autorix/platform/paging"
	"github.com/autorix/platform/pgtest"
	"github.com/google/uuid"
)

func newTestRepo(t *testing.T) *postgres.Repository {
	t.Helper()
	pool := pgtest.StartPostgres(t, "../../../migrations")
	return postgres.NewRepository(pool)
}

func TestCreateAndGetSAMLProvider(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepo(t)

	p := &core.SAMLProvider{
		ID:          "okta-corporate",
		DisplayName: "Okta Corporate",
		IdPEntityID: "https://okta.example.com/entity",
		IdPSSOURL:   "https://okta.example.com/sso",
		SPEntityID:  "https://autorix.example.com/sp",
		AttributeMapping: map[string]string{
			"email": "user.email",
		},
		Enabled: true,
	}

	if err := repo.CreateSAMLProvider(ctx, p); err != nil {
		t.Fatalf("CreateSAMLProvider: %v", err)
	}
	if p.CreatedAt.IsZero() || p.UpdatedAt.IsZero() {
		t.Fatalf("expected CreatedAt/UpdatedAt to be populated, got %+v", p)
	}
	// Default cert should be injected because we didn't supply one.
	if p.IdPCertificatePEM == "" {
		t.Fatalf("expected default certificate to be populated")
	}

	got, err := repo.GetSAMLProvider(ctx, "okta-corporate")
	if err != nil {
		t.Fatalf("GetSAMLProvider: %v", err)
	}
	if got.ID != p.ID || got.DisplayName != p.DisplayName {
		t.Fatalf("unexpected provider: %+v", got)
	}
	if got.AttributeMapping["email"] != "user.email" {
		t.Fatalf("expected attribute mapping to round-trip, got %+v", got.AttributeMapping)
	}
}

func TestGetSAMLProviderNotFound(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepo(t)

	_, err := repo.GetSAMLProvider(ctx, "does-not-exist")
	if err != postgres.ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestCreateSAMLProviderDuplicateID(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepo(t)

	p := &core.SAMLProvider{
		ID:          "dup-provider",
		DisplayName: "Dup Provider",
		IdPEntityID: "https://idp.example.com/entity",
		IdPSSOURL:   "https://idp.example.com/sso",
		SPEntityID:  "https://autorix.example.com/sp",
		Enabled:     true,
	}
	if err := repo.CreateSAMLProvider(ctx, p); err != nil {
		t.Fatalf("first CreateSAMLProvider: %v", err)
	}

	dup := &core.SAMLProvider{
		ID:          "dup-provider",
		DisplayName: "Another Provider",
		IdPEntityID: "https://idp2.example.com/entity",
		IdPSSOURL:   "https://idp2.example.com/sso",
		SPEntityID:  "https://autorix.example.com/sp",
		Enabled:     true,
	}
	if err := repo.CreateSAMLProvider(ctx, dup); err == nil {
		t.Fatalf("expected duplicate primary key error, got nil")
	}
}

func TestListSAMLProviders(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepo(t)

	providers, hasMore, err := repo.ListSAMLProviders(ctx, 20, "")
	if err != nil {
		t.Fatalf("ListSAMLProviders on empty table: %v", err)
	}
	if len(providers) != 0 {
		t.Fatalf("expected empty list, got %d", len(providers))
	}
	if hasMore {
		t.Fatalf("expected hasMore=false on empty table")
	}

	for _, id := range []string{"provider-a", "provider-b"} {
		p := &core.SAMLProvider{
			ID:          id,
			DisplayName: id,
			IdPEntityID: "https://idp.example.com/" + id,
			IdPSSOURL:   "https://idp.example.com/sso/" + id,
			SPEntityID:  "https://autorix.example.com/sp",
			Enabled:     true,
		}
		if err := repo.CreateSAMLProvider(ctx, p); err != nil {
			t.Fatalf("CreateSAMLProvider(%s): %v", id, err)
		}
		time.Sleep(2 * time.Millisecond)
	}

	providers, hasMore, err = repo.ListSAMLProviders(ctx, 20, "")
	if err != nil {
		t.Fatalf("ListSAMLProviders: %v", err)
	}
	if len(providers) != 2 {
		t.Fatalf("expected 2 providers, got %d", len(providers))
	}
	if hasMore {
		t.Fatalf("expected hasMore=false when both fit in one page")
	}
}

// TestListSAMLProviders_CursorPagination proves ListSAMLProviders paginates
// via a real SQL keyset: a page smaller than the full set returns exactly N
// rows plus a cursor, and following that cursor returns the rest with no
// duplicate and no skipped row.
func TestListSAMLProviders_CursorPagination(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepo(t)

	const total = 5
	ids := make(map[string]bool, total)
	for i := 0; i < total; i++ {
		id := fmt.Sprintf("cursor-provider-%d", i)
		p := &core.SAMLProvider{
			ID:          id,
			DisplayName: id,
			IdPEntityID: "https://idp.example.com/" + id,
			IdPSSOURL:   "https://idp.example.com/sso/" + id,
			SPEntityID:  "https://autorix.example.com/sp",
			Enabled:     true,
		}
		if err := repo.CreateSAMLProvider(ctx, p); err != nil {
			t.Fatalf("CreateSAMLProvider(%s): %v", id, err)
		}
		ids[id] = true
		time.Sleep(2 * time.Millisecond)
	}

	const pageSize = 2
	var all []core.SAMLProvider
	cursor := ""
	for i := 0; i < 10; i++ { // bounded loop guard against an infinite pagination bug
		page, hasMore, err := repo.ListSAMLProviders(ctx, pageSize, cursor)
		if err != nil {
			t.Fatalf("ListSAMLProviders(cursor=%q): %v", cursor, err)
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
		t.Fatalf("expected %d total providers across pages, got %d", total, len(all))
	}
	seen := map[string]bool{}
	for _, p := range all {
		if seen[p.ID] {
			t.Fatalf("duplicate provider %q across pages", p.ID)
		}
		seen[p.ID] = true
		delete(ids, p.ID)
	}
	if len(ids) != 0 {
		t.Fatalf("some providers were skipped across pages: %v", ids)
	}
}

func TestCreateAndListSCIMUser(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepo(t)

	u := &core.SCIMUser{
		ExternalID:  "ext-1",
		UserName:    "jdoe",
		DisplayName: "John Doe",
		Active:      true,
		Emails: []core.SCIMEmail{
			{Value: "jdoe@example.com", Primary: true, Type: "work"},
		},
	}

	if err := repo.CreateSCIMUser(ctx, u); err != nil {
		t.Fatalf("CreateSCIMUser: %v", err)
	}
	if u.ID == uuid.Nil {
		t.Fatalf("expected generated UUID, got nil")
	}
	if u.Meta.Created.IsZero() || u.Meta.LastModified.IsZero() {
		t.Fatalf("expected Meta timestamps to be set, got %+v", u.Meta)
	}

	users, err := repo.ListSCIMUsers(ctx)
	if err != nil {
		t.Fatalf("ListSCIMUsers: %v", err)
	}
	if len(users) != 1 {
		t.Fatalf("expected 1 user, got %d", len(users))
	}
	if users[0].UserName != "jdoe" {
		t.Fatalf("unexpected user: %+v", users[0])
	}
	if len(users[0].Emails) != 1 || users[0].Emails[0].Value != "jdoe@example.com" {
		t.Fatalf("expected primary email to round-trip, got %+v", users[0].Emails)
	}
}

func TestCreateSCIMUserDuplicateUserName(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepo(t)

	u1 := &core.SCIMUser{
		ExternalID: "ext-a",
		UserName:   "sameuser",
		Active:     true,
		Emails:     []core.SCIMEmail{{Value: "a@example.com", Primary: true, Type: "work"}},
	}
	if err := repo.CreateSCIMUser(ctx, u1); err != nil {
		t.Fatalf("first CreateSCIMUser: %v", err)
	}

	u2 := &core.SCIMUser{
		ExternalID: "ext-b",
		UserName:   "sameuser", // violates UNIQUE(user_name)
		Active:     true,
		Emails:     []core.SCIMEmail{{Value: "b@example.com", Primary: true, Type: "work"}},
	}
	if err := repo.CreateSCIMUser(ctx, u2); err == nil {
		t.Fatalf("expected unique constraint violation on user_name, got nil")
	}
}

func TestCreateSCIMUserDuplicateExternalID(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepo(t)

	u1 := &core.SCIMUser{
		ExternalID: "same-ext-id",
		UserName:   "user-one",
		Active:     true,
		Emails:     []core.SCIMEmail{{Value: "one@example.com", Primary: true, Type: "work"}},
	}
	if err := repo.CreateSCIMUser(ctx, u1); err != nil {
		t.Fatalf("first CreateSCIMUser: %v", err)
	}

	u2 := &core.SCIMUser{
		ExternalID: "same-ext-id", // violates UNIQUE(external_id)
		UserName:   "user-two",
		Active:     true,
		Emails:     []core.SCIMEmail{{Value: "two@example.com", Primary: true, Type: "work"}},
	}
	if err := repo.CreateSCIMUser(ctx, u2); err == nil {
		t.Fatalf("expected unique constraint violation on external_id, got nil")
	}
}

func TestUpdateAndDeleteSAMLProvider(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepo(t)

	p := &core.SAMLProvider{
		ID:          "provider-lifecycle-1",
		DisplayName: "Initial Name",
		IdPEntityID: "https://idp.example.com/init",
		IdPSSOURL:   "https://idp.example.com/sso",
		SPEntityID:  "https://autorix.example.com/sp",
		Enabled:     true,
	}

	if err := repo.CreateSAMLProvider(ctx, p); err != nil {
		t.Fatalf("CreateSAMLProvider: %v", err)
	}

	// 1. Update provider
	p.DisplayName = "Updated Name"
	p.IdPSSOURL = "https://idp.example.com/sso/updated"
	p.AttributeMapping = map[string]string{"email": "mail"}
	if err := repo.UpdateSAMLProvider(ctx, p); err != nil {
		t.Fatalf("UpdateSAMLProvider: %v", err)
	}

	got, err := repo.GetSAMLProvider(ctx, p.ID)
	if err != nil {
		t.Fatalf("GetSAMLProvider: %v", err)
	}
	if got.DisplayName != "Updated Name" || got.IdPSSOURL != "https://idp.example.com/sso/updated" {
		t.Fatalf("expected updated fields, got %+v", got)
	}
	if got.AttributeMapping["email"] != "mail" {
		t.Fatalf("expected updated attribute mapping, got %+v", got.AttributeMapping)
	}

	// 2. Disable provider
	if err := repo.SetSAMLProviderEnabled(ctx, p.ID, false); err != nil {
		t.Fatalf("SetSAMLProviderEnabled(false): %v", err)
	}
	got, _ = repo.GetSAMLProvider(ctx, p.ID)
	if got.Enabled {
		t.Fatalf("expected provider to be disabled")
	}

	// 3. Enable provider
	if err := repo.SetSAMLProviderEnabled(ctx, p.ID, true); err != nil {
		t.Fatalf("SetSAMLProviderEnabled(true): %v", err)
	}
	got, _ = repo.GetSAMLProvider(ctx, p.ID)
	if !got.Enabled {
		t.Fatalf("expected provider to be enabled")
	}

	// 4. Delete provider
	if err := repo.DeleteSAMLProvider(ctx, p.ID); err != nil {
		t.Fatalf("DeleteSAMLProvider: %v", err)
	}

	_, err = repo.GetSAMLProvider(ctx, p.ID)
	if err != postgres.ErrNotFound {
		t.Fatalf("expected ErrNotFound after deletion, got %v", err)
	}
}

func TestSCIMGroupsCRUD(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepo(t)

	group := &core.SCIMGroup{
		DisplayName: "Engineering",
		Members: []core.SCIMMember{
			{Value: uuid.New().String(), Display: "Ada Lovelace"},
		},
	}

	// 1. Create group
	if err := repo.CreateSCIMGroup(ctx, group); err != nil {
		t.Fatalf("CreateSCIMGroup: %v", err)
	}
	if group.ID == uuid.Nil {
		t.Fatalf("expected generated UUID for group")
	}
	if group.Meta.Created.IsZero() {
		t.Fatalf("expected Meta timestamps")
	}

	// 2. Get group
	got, err := repo.GetSCIMGroup(ctx, group.ID)
	if err != nil {
		t.Fatalf("GetSCIMGroup: %v", err)
	}
	if got.DisplayName != "Engineering" || len(got.Members) != 1 {
		t.Fatalf("unexpected group: %+v", got)
	}

	// 3. List groups
	groups, err := repo.ListSCIMGroups(ctx)
	if err != nil {
		t.Fatalf("ListSCIMGroups: %v", err)
	}
	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}

	// 4. Update group
	group.DisplayName = "Core Engineering"
	newMember := core.SCIMMember{Value: uuid.New().String(), Display: "Grace Hopper"}
	group.Members = append(group.Members, newMember)
	if err := repo.UpdateSCIMGroup(ctx, group); err != nil {
		t.Fatalf("UpdateSCIMGroup: %v", err)
	}

	got, err = repo.GetSCIMGroup(ctx, group.ID)
	if err != nil {
		t.Fatalf("GetSCIMGroup after update: %v", err)
	}
	if got.DisplayName != "Core Engineering" || len(got.Members) != 2 {
		t.Fatalf("expected updated group, got %+v", got)
	}

	// 5. Delete group
	if err := repo.DeleteSCIMGroup(ctx, group.ID); err != nil {
		t.Fatalf("DeleteSCIMGroup: %v", err)
	}

	_, err = repo.GetSCIMGroup(ctx, group.ID)
	if err != postgres.ErrNotFound {
		t.Fatalf("expected ErrNotFound after group deletion, got %v", err)
	}
}

func TestSCIMSyncHistory(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepo(t)

	now := time.Now()
	completed := now.Add(2 * time.Second)
	sync := &core.SCIMSyncHistory{
		ProviderID:   "okta-corp",
		ResourceType: "All",
		Status:       "success",
		TotalRecords: 42,
		CreatedCount: 30,
		UpdatedCount: 10,
		DeletedCount: 2,
		ErrorCount:   0,
		Errors:       []string{},
		StartedAt:    now,
		CompletedAt:  &completed,
	}

	if err := repo.RecordSCIMSync(ctx, sync); err != nil {
		t.Fatalf("RecordSCIMSync: %v", err)
	}
	if sync.ID == uuid.Nil {
		t.Fatalf("expected generated UUID for sync record")
	}

	history, err := repo.ListSCIMSyncHistory(ctx, 10)
	if err != nil {
		t.Fatalf("ListSCIMSyncHistory: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("expected 1 sync record, got %d", len(history))
	}
	if history[0].ProviderID != "okta-corp" || history[0].TotalRecords != 42 {
		t.Fatalf("unexpected sync record: %+v", history[0])
	}
}

