package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/autorix/platform/health"
	"github.com/autorix/platform/pgtest"
	"github.com/autorix/vulcan/internal/core"
	"github.com/autorix/vulcan/internal/macaroon"
	"github.com/autorix/vulcan/internal/storage/postgres"
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

