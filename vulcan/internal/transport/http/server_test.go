package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/autorix/platform/cache"
	"github.com/autorix/platform/health"
	"github.com/autorix/platform/pgtest"
	"github.com/autorix/vulcan/internal/core"
	"github.com/autorix/vulcan/internal/macaroon"
	"github.com/autorix/vulcan/internal/storage/postgres"
	"github.com/google/uuid"
)

func newTestHealthHandler(failing bool) *health.Handler {
	checker := health.NewChecker()
	checker.Register("postgres", func(ctx context.Context) error {
		if failing {
			return errors.New("connection refused")
		}
		return nil
	})
	return health.NewHandler(checker, func() health.Info {
		return health.Info{Engine: "vulcan"}
	})
}

func TestHTTP_HealthAlive_Returns200(t *testing.T) {
	server := NewServer(nil, "https://api.autorix.io", newTestHealthHandler(false))
	router := server.Routes()

	req := httptest.NewRequest("GET", "/health/alive", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid json body: %v", err)
	}
	if body["status"] != "alive" {
		t.Fatalf(`expected status="alive", got %q`, body["status"])
	}
}

func TestHTTP_HealthReady_503WhenCheckFails(t *testing.T) {
	server := NewServer(nil, "https://api.autorix.io", newTestHealthHandler(true))
	router := server.Routes()

	req := httptest.NewRequest("GET", "/health/ready", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
}

func TestAttenuateHandler(t *testing.T) {
	server := NewServer(nil, "https://api.autorix.io", newTestHealthHandler(false))
	router := server.Routes()

	baseMacaroon := macaroon.New("https://api.autorix.io", "key-123", "secret-root-key")

	body, _ := json.Marshal(map[string]interface{}{
		"macaroon": baseMacaroon,
		"caveat":   "method = GET",
	})

	req := httptest.NewRequest("POST", "/keys/attenuate", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp core.Macaroon
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Caveats) != 1 {
		t.Errorf("expected 1 caveat, got %d", len(resp.Caveats))
	}

	if resp.Caveats[0].Predicate != "method = GET" {
		t.Errorf("expected caveat 'method = GET', got %s", resp.Caveats[0].Predicate)
	}
}

// newContractMux builds a real vulcan Server with its readiness check rigged
// to fail iff checkErr is non-nil, for the shared platform/health.Contract
// suite.
func newContractMux(checkErr error) http.Handler {
	server := NewServer(nil, "https://api.autorix.io", newTestHealthHandler(checkErr != nil))
	return server.Routes()
}

func TestHealthContract(t *testing.T) {
	health.Contract(t, newContractMux)
}

func TestAdmin_KeyLifecycle_AndRotation(t *testing.T) {
	pool := pgtest.StartPostgres(t, "../../../migrations")
	repo := postgres.NewRepository(pool)
	server := NewServer(repo, "https://api.autorix.io", newTestHealthHandler(false))
	router := server.Routes()

	// 1. Create a key via POST /keys
	createReqBody, _ := json.Marshal(core.CreateKeyRequest{
		Name:    "prod-backend",
		OwnerID: "team-platform",
		Scopes:  []string{"read:metrics", "write:deployments"},
		IsLive:  true,
	})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest("POST", "/keys", bytes.NewReader(createReqBody)))
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /keys failed: %d %s", rec.Code, rec.Body.String())
	}
	var created core.CreateKeyResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &created)
	keyID := created.APIKey.ID
	oldMacaroon := created.Macaroon

	// 2. GET /admin/keys/{id}
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest("GET", "/admin/keys/"+keyID.String(), nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /admin/keys/{id} failed: %d %s", rec.Code, rec.Body.String())
	}
	var getKey core.APIKey
	_ = json.Unmarshal(rec.Body.Bytes(), &getKey)
	if getKey.ID != keyID || getKey.Name != "prod-backend" {
		t.Errorf("GET /admin/keys/{id} unexpected key: %v", getKey)
	}

	// 3. PATCH /admin/keys/{id}
	newName := "prod-backend-v2"
	newDesc := "Production backend service key"
	newScopes := []string{"read:metrics", "write:deployments", "admin:all"}
	patchReqBody, _ := json.Marshal(core.UpdateKeyRequest{
		Name:        &newName,
		Description: &newDesc,
		Scopes:      &newScopes,
	})
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest("PATCH", "/admin/keys/"+keyID.String(), bytes.NewReader(patchReqBody)))
	if rec.Code != http.StatusOK {
		t.Fatalf("PATCH /admin/keys/{id} failed: %d %s", rec.Code, rec.Body.String())
	}
	var patchedKey core.APIKey
	_ = json.Unmarshal(rec.Body.Bytes(), &patchedKey)
	if patchedKey.Name != newName || patchedKey.Description != newDesc || len(patchedKey.Scopes) != 3 {
		t.Errorf("PATCH /admin/keys/{id} unexpected result: %v", patchedKey)
	}

	// 4. POST /admin/keys/{id}/rotate with grace period
	rotateReqBody, _ := json.Marshal(core.RotateKeyRequest{
		GracePeriod: "48h",
	})
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest("POST", "/admin/keys/"+keyID.String()+"/rotate", bytes.NewReader(rotateReqBody)))
	if rec.Code != http.StatusOK {
		t.Fatalf("POST /admin/keys/{id}/rotate failed: %d %s", rec.Code, rec.Body.String())
	}
	var rotated core.RotateKeyResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &rotated)
	if rotated.RawToken == "" || rotated.Macaroon == nil {
		t.Fatalf("POST /admin/keys/{id}/rotate did not return new token/macaroon: %v", rotated)
	}
	if rotated.APIKey.GracePeriodExpiresAt == nil {
		t.Fatalf("POST /admin/keys/{id}/rotate missing grace_period_expires_at: %v", rotated.APIKey)
	}

	// 5. Verify using OLD macaroon during grace period -> should succeed
	verifyOldBody, _ := json.Marshal(map[string]interface{}{
		"macaroon": oldMacaroon,
	})
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest("POST", "/keys/verify", bytes.NewReader(verifyOldBody)))
	if rec.Code != http.StatusOK {
		t.Fatalf("POST /keys/verify with old macaroon during grace period failed: %d %s", rec.Code, rec.Body.String())
	}

	// 6. Verify using NEW macaroon -> should succeed
	verifyNewBody, _ := json.Marshal(map[string]interface{}{
		"macaroon": rotated.Macaroon,
		"context": core.VerificationContext{
			IPAddress: "203.0.113.195",
		},
	})
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest("POST", "/keys/verify", bytes.NewReader(verifyNewBody)))
	if rec.Code != http.StatusOK {
		t.Fatalf("POST /keys/verify with new macaroon failed: %d %s", rec.Code, rec.Body.String())
	}

	// 7. Verify usage tracking recorded
	// Let background goroutine finish
	for i := 0; i < 20; i++ {
		k, err := repo.GetKeyByID(context.Background(), keyID)
		if err == nil && k.CallCount >= 2 {
			if k.LastSourceIP != "203.0.113.195" {
				t.Errorf("LastSourceIP = %q, want 203.0.113.195", k.LastSourceIP)
			}
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestAdmin_ScopesCatalogue_CRUD(t *testing.T) {
	pool := pgtest.StartPostgres(t, "../../../migrations")
	repo := postgres.NewRepository(pool)
	server := NewServer(repo, "https://api.autorix.io", newTestHealthHandler(false))
	router := server.Routes()

	// 1. GET /admin/scopes -> initial empty
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest("GET", "/admin/scopes", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /admin/scopes failed: %d %s", rec.Code, rec.Body.String())
	}
	var scopes []core.Scope
	_ = json.Unmarshal(rec.Body.Bytes(), &scopes)
	if len(scopes) != 0 {
		t.Fatalf("GET /admin/scopes initial count = %d, want 0", len(scopes))
	}

	// 2. POST /admin/scopes -> create scope
	createBody, _ := json.Marshal(core.CreateScopeRequest{
		Name:        "vulcan:admin",
		Description: "Full administrative access to Vulcan",
	})
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest("POST", "/admin/scopes", bytes.NewReader(createBody)))
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /admin/scopes failed: %d %s", rec.Code, rec.Body.String())
	}
	var createdScope core.Scope
	_ = json.Unmarshal(rec.Body.Bytes(), &createdScope)
	if createdScope.Name != "vulcan:admin" {
		t.Errorf("createdScope.Name = %q, want vulcan:admin", createdScope.Name)
	}

	// 3. GET /admin/scopes -> 1 scope
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest("GET", "/admin/scopes", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /admin/scopes 2 failed: %d", rec.Code)
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &scopes)
	if len(scopes) != 1 || scopes[0].Name != "vulcan:admin" {
		t.Fatalf("GET /admin/scopes count = %d, want 1 with vulcan:admin", len(scopes))
	}

	// 4. DELETE /admin/scopes/{name} -> delete scope
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest("DELETE", "/admin/scopes/vulcan:admin", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("DELETE /admin/scopes/{name} failed: %d %s", rec.Code, rec.Body.String())
	}

	// 5. DELETE /admin/scopes/{name} on nonexistent -> 404
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest("DELETE", "/admin/scopes/vulcan:admin", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("DELETE /admin/scopes/{name} non-existent expected 404, got %d", rec.Code)
	}
}

func TestHTTP_Metrics(t *testing.T) {
	server := NewServer(nil, "https://api.autorix.io", newTestHealthHandler(false))
	router := server.Routes()

	// Trigger a failed verification to exercise metric
	verifyReq := httptest.NewRequest("POST", "/keys/verify", strings.NewReader(`invalid-json`))
	verifyRec := httptest.NewRecorder()
	router.ServeHTTP(verifyRec, verifyReq)

	req := httptest.NewRequest("GET", "/metrics", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 from /metrics, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "autorix_vulcan_keys_verified_total") {
		t.Errorf("expected body to contain autorix_vulcan_keys_verified_total, got: %s", body)
	}
}

func TestVerify_WithCache(t *testing.T) {
	pool := pgtest.StartPostgres(t, "../../../migrations")
	repo := postgres.NewRepository(pool)
	server := NewServer(repo, "https://api.autorix.io", newTestHealthHandler(false))

	c := cache.NewMemoryCache()
	defer c.Close()
	server.SetCache(c)

	router := server.Routes()
	ctx := context.Background()

	// 1. Create a live key
	createPayload := `{"name":"test-cached-key","owner_id":"usr_test_123","prefix":"av_live","scopes":["read"]}`
	createRec := httptest.NewRecorder()
	createReq := httptest.NewRequest("POST", "/keys", strings.NewReader(createPayload))
	router.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create failed: %d %s", createRec.Code, createRec.Body.String())
	}

	var createResp core.CreateKeyResponse
	_ = json.Unmarshal(createRec.Body.Bytes(), &createResp)
	keyID := createResp.APIKey.ID.String()

	cacheKey := "vulcan:key:" + keyID

	// 2. Before verify, cache is empty
	_, err := c.Get(ctx, cacheKey)
	if err != cache.ErrNotFound {
		t.Fatalf("expected ErrNotFound in cache before first verify, got %v", err)
	}

	// 3. Verify macaroon -> populates cache
	verifyBody, _ := json.Marshal(map[string]interface{}{
		"macaroon": createResp.Macaroon,
	})
	verifyRec := httptest.NewRecorder()
	verifyReq := httptest.NewRequest("POST", "/keys/verify", bytes.NewReader(verifyBody))
	router.ServeHTTP(verifyRec, verifyReq)
	if verifyRec.Code != http.StatusOK {
		t.Fatalf("verify failed: %d %s", verifyRec.Code, verifyRec.Body.String())
	}

	// 4. Cache should now contain the key
	cachedData, err := c.Get(ctx, cacheKey)
	if err != nil {
		t.Fatalf("expected key to be in cache after verify: %v", err)
	}
	if len(cachedData) == 0 {
		t.Fatalf("expected cached data to be non-empty")
	}

	// 5. Revoke key -> must evict from cache
	revokeRec := httptest.NewRecorder()
	revokeReq := httptest.NewRequest("DELETE", "/keys/"+keyID, nil)
	router.ServeHTTP(revokeRec, revokeReq)
	if revokeRec.Code != http.StatusOK {
		t.Fatalf("revoke failed: %d %s", revokeRec.Code, revokeRec.Body.String())
	}

	// 6. Cache should be evicted immediately
	_, err = c.Get(ctx, cacheKey)
	if err != cache.ErrNotFound {
		t.Fatalf("expected key to be evicted from cache after revoke, got %v", err)
	}
}

type mockKeyRepo struct {
	mu      sync.Mutex
	keys    map[uuid.UUID]*core.APIKey
	getHits int
	lastIP  string
}

func newMockKeyRepo() *mockKeyRepo {
	return &mockKeyRepo{keys: make(map[uuid.UUID]*core.APIKey)}
}

func (m *mockKeyRepo) CreateKey(_ context.Context, k *core.APIKey) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := *k
	m.keys[k.ID] = &cp
	return nil
}

func (m *mockKeyRepo) GetKeyByID(_ context.Context, id uuid.UUID) (*core.APIKey, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.getHits++
	k, ok := m.keys[id]
	if !ok {
		return nil, postgres.ErrNotFound
	}
	cp := *k
	return &cp, nil
}

func (m *mockKeyRepo) UpdateKey(_ context.Context, id uuid.UUID, req core.UpdateKeyRequest) (*core.APIKey, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	k, ok := m.keys[id]
	if !ok {
		return nil, postgres.ErrNotFound
	}
	if k.State == "revoked" {
		return nil, postgres.ErrKeyRevoked
	}
	if req.Name != nil {
		k.Name = *req.Name
	}
	cp := *k
	return &cp, nil
}

func (m *mockKeyRepo) RotateKey(_ context.Context, id uuid.UUID, newHash, newHint, newRootKey string, gracePeriodExpiresAt *time.Time) (*core.APIKey, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	k, ok := m.keys[id]
	if !ok {
		return nil, postgres.ErrNotFound
	}
	k.PrevKeyHash = k.KeyHash
	k.PrevRootSignatureKey = k.RootSignatureKey
	k.KeyHash = newHash
	k.KeyHint = newHint
	k.RootSignatureKey = newRootKey
	k.GracePeriodExpiresAt = gracePeriodExpiresAt
	cp := *k
	return &cp, nil
}

func (m *mockKeyRepo) RevokeKey(_ context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	k, ok := m.keys[id]
	if !ok {
		return postgres.ErrNotFound
	}
	k.State = "revoked"
	return nil
}

func (m *mockKeyRepo) RecordUsage(_ context.Context, id uuid.UUID, sourceIP string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if k, ok := m.keys[id]; ok {
		k.CallCount++
		k.LastSourceIP = sourceIP
		m.lastIP = sourceIP
	}
	return nil
}

func (m *mockKeyRepo) ListKeys(_ context.Context, _ int, _ string) ([]core.APIKey, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]core.APIKey, 0, len(m.keys))
	for _, k := range m.keys {
		out = append(out, *k)
	}
	return out, false, nil
}

func (m *mockKeyRepo) ListScopes(_ context.Context) ([]core.Scope, error) {
	return make([]core.Scope, 0), nil
}

func (m *mockKeyRepo) CreateScope(_ context.Context, _ *core.Scope) error { return nil }
func (m *mockKeyRepo) DeleteScope(_ context.Context, _ string) error      { return nil }

func TestHandleVerify_CacheHitPreservesSecretAndRejectsEmptyKeyForgery(t *testing.T) {
	repo := newMockKeyRepo()
	keyID := uuid.New()
	rootSecret := "super-secret-root-signing-key-xyz"
	repo.keys[keyID] = &core.APIKey{
		ID:               keyID,
		KeyPrefix:        "av_live",
		KeyHint:          "1234",
		RootSignatureKey: rootSecret,
		Name:             "cached-key",
		OwnerID:          "owner-1",
		Scopes:           []string{"read"},
		State:            "active",
	}

	server := NewServer(nil, "https://api.autorix.io", newTestHealthHandler(false))
	server.repo = repo
	c := cache.NewMemoryCache()
	defer c.Close()
	server.SetCache(c)
	router := server.Routes()

	validMac := macaroon.New("https://api.autorix.io", keyID.String(), rootSecret)
	validBody, _ := json.Marshal(map[string]interface{}{"macaroon": validMac})

	// 1st verify: cache miss -> loads from repo and populates cache
	rec1 := httptest.NewRecorder()
	router.ServeHTTP(rec1, httptest.NewRequest("POST", "/keys/verify", bytes.NewReader(validBody)))
	if rec1.Code != http.StatusOK {
		t.Fatalf("first verify failed: %d %s", rec1.Code, rec1.Body.String())
	}
	if strings.Contains(rec1.Body.String(), rootSecret) {
		t.Fatalf("HTTP response leaked root_signature_key secret")
	}

	// 2nd verify: cache hit -> MUST succeed with preserved RootSignatureKey
	rec2 := httptest.NewRecorder()
	router.ServeHTTP(rec2, httptest.NewRequest("POST", "/keys/verify", bytes.NewReader(validBody)))
	if rec2.Code != http.StatusOK {
		t.Fatalf("second (cached) verify failed: %d %s", rec2.Code, rec2.Body.String())
	}
	if strings.Contains(rec2.Body.String(), rootSecret) {
		t.Fatalf("second HTTP response leaked root_signature_key secret")
	}
	repo.mu.Lock()
	hits := repo.getHits
	repo.mu.Unlock()
	if hits != 1 {
		t.Fatalf("expected exactly 1 repo GetKeyByID call (2nd call served from cache), got %d", hits)
	}

	// 3rd verify: forged macaroon signed with empty key "" MUST be rejected on cache hit
	forgedMac := macaroon.New("https://api.autorix.io", keyID.String(), "")
	forgedBody, _ := json.Marshal(map[string]interface{}{"macaroon": forgedMac})
	rec3 := httptest.NewRecorder()
	router.ServeHTTP(rec3, httptest.NewRequest("POST", "/keys/verify", bytes.NewReader(forgedBody)))
	if rec3.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for empty-key forged macaroon on cache hit, got %d %s", rec3.Code, rec3.Body.String())
	}
}

func TestHandleVerify_EnforcesServerClockAndNoPrevKeyFallbackOnCaveatFailure(t *testing.T) {
	repo := newMockKeyRepo()
	keyID := uuid.New()
	rootSecret := "current-secret"
	prevSecret := "previous-secret"
	graceEnd := time.Now().Add(time.Hour)
	repo.keys[keyID] = &core.APIKey{
		ID:                   keyID,
		KeyPrefix:            "av_live",
		RootSignatureKey:     rootSecret,
		PrevRootSignatureKey: prevSecret,
		GracePeriodExpiresAt: &graceEnd,
		State:                "active",
	}

	server := NewServer(nil, "https://api.autorix.io", newTestHealthHandler(false))
	server.repo = repo
	router := server.Routes()

	// Expired macaroon: client attempts to bypass by sending backdated context.now
	base := macaroon.New("https://api.autorix.io", keyID.String(), rootSecret)
	expiredAt := time.Now().Add(-10 * time.Minute).UTC().Format(time.RFC3339)
	expiredMac, err := macaroon.Attenuate(base, "time_before = "+expiredAt)
	if err != nil {
		t.Fatalf("Attenuate failed: %v", err)
	}

	backdatedBody, _ := json.Marshal(map[string]interface{}{
		"macaroon": expiredMac,
		"context": core.VerificationContext{
			Now: time.Now().Add(-1 * time.Hour).UTC(),
		},
	})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest("POST", "/keys/verify", bytes.NewReader(backdatedBody)))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 when client backdates context.now for expired macaroon, got %d %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "caveat") && !strings.Contains(rec.Body.String(), "expired") {
		t.Fatalf("expected caveat/expired error (not signature error from prev key fallback), got: %s", rec.Body.String())
	}
}

func TestHandleAdminRotateAndUpdate_Validation(t *testing.T) {
	repo := newMockKeyRepo()
	activeID := uuid.New()
	revokedID := uuid.New()
	expiredID := uuid.New()
	past := time.Now().Add(-time.Hour)

	repo.keys[activeID] = &core.APIKey{ID: activeID, KeyPrefix: "av_live", RootSignatureKey: "root", State: "active"}
	repo.keys[revokedID] = &core.APIKey{ID: revokedID, KeyPrefix: "av_live", RootSignatureKey: "root", State: "revoked"}
	repo.keys[expiredID] = &core.APIKey{ID: expiredID, KeyPrefix: "av_live", RootSignatureKey: "root", State: "active", ExpiresAt: &past}

	server := NewServer(nil, "https://api.autorix.io", newTestHealthHandler(false))
	server.repo = repo
	router := server.Routes()

	// 1. Chunked request (ContentLength = -1) with valid grace_period succeeds
	req := httptest.NewRequest("POST", "/admin/keys/"+activeID.String()+"/rotate", strings.NewReader(`{"grace_period":"2h"}`))
	req.ContentLength = -1
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("chunked rotate failed: %d %s", rec.Code, rec.Body.String())
	}

	// 2. Grace period > 168h rejected
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest("POST", "/admin/keys/"+activeID.String()+"/rotate", strings.NewReader(`{"grace_period":"200h"}`)))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for grace_period > 168h, got %d", rec.Code)
	}

	// 3. Negative grace period rejected
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest("POST", "/admin/keys/"+activeID.String()+"/rotate", strings.NewReader(`{"grace_period":"-1h"}`)))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for negative grace_period, got %d", rec.Code)
	}

	// 4. Rotate expired key rejected
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest("POST", "/admin/keys/"+expiredID.String()+"/rotate", nil))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for rotating expired key, got %d", rec.Code)
	}

	// 5. Update revoked key rejected
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest("PATCH", "/admin/keys/"+revokedID.String(), strings.NewReader(`{"name":"hack"}`)))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for updating revoked key, got %d", rec.Code)
	}
}

func TestHandleListKeys_EmptyArrayNotNull(t *testing.T) {
	server := NewServer(nil, "https://api.autorix.io", newTestHealthHandler(false))
	server.repo = newMockKeyRepo()
	router := server.Routes()

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest("GET", "/keys", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"data":[]`) {
		t.Fatalf("expected JSON empty array data:[], got %s", rec.Body.String())
	}
}
