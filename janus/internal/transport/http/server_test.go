package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/autorix/janus/internal/core"
	"github.com/autorix/janus/internal/jwks"
	"github.com/autorix/janus/internal/oauth2"
	"github.com/autorix/janus/internal/storage/postgres"
	"github.com/autorix/platform/health"
	"github.com/autorix/platform/pgtest"
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
		return health.Info{Engine: "janus"}
	})
}

func TestHTTP_HealthAlive_Returns200(t *testing.T) {
	km, err := jwks.NewKeyManager()
	if err != nil {
		t.Fatalf("failed to create KeyManager: %v", err)
	}
	engine := oauth2.NewEngine("http://localhost:4444", km)
	server := NewServer("http://localhost:4444", nil, km, engine, newTestHealthHandler(false))
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
	km, err := jwks.NewKeyManager()
	if err != nil {
		t.Fatalf("failed to create KeyManager: %v", err)
	}
	engine := oauth2.NewEngine("http://localhost:4444", km)
	server := NewServer("http://localhost:4444", nil, km, engine, newTestHealthHandler(true))
	router := server.Routes()

	req := httptest.NewRequest("GET", "/health/ready", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
}

func TestDiscoveryAndJWKS(t *testing.T) {
	km, err := jwks.NewKeyManager()
	if err != nil {
		t.Fatalf("failed to create KeyManager: %v", err)
	}

	engine := oauth2.NewEngine("http://localhost:4444", km)
	server := NewServer("http://localhost:4444", nil, km, engine, newTestHealthHandler(false))
	router := server.Routes()

	// 1. Test OpenID Discovery
	req := httptest.NewRequest("GET", "/.well-known/openid-configuration", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 OK from discovery, got %d", rec.Code)
	}

	var discovery map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &discovery); err != nil {
		t.Fatalf("failed to parse discovery JSON: %v", err)
	}

	if discovery["issuer"] != "http://localhost:4444" {
		t.Errorf("expected issuer 'http://localhost:4444', got %v", discovery["issuer"])
	}

	// 2. Test JWKS endpoint
	reqJWKS := httptest.NewRequest("GET", "/.well-known/jwks.json", nil)
	recJWKS := httptest.NewRecorder()
	router.ServeHTTP(recJWKS, reqJWKS)

	if recJWKS.Code != http.StatusOK {
		t.Errorf("expected 200 OK from JWKS, got %d", recJWKS.Code)
	}

	var jwksResp map[string][]interface{}
	if err := json.Unmarshal(recJWKS.Body.Bytes(), &jwksResp); err != nil {
		t.Fatalf("failed to parse JWKS JSON: %v", err)
	}

	if len(jwksResp["keys"]) != 1 {
		t.Errorf("expected 1 key in JWKS, got %d", len(jwksResp["keys"]))
	}
}

// newContractMux builds a real janus Server with its readiness check rigged
// to fail iff checkErr is non-nil, for the shared platform/health.Contract
// suite.
func newContractMux(checkErr error) http.Handler {
	km, err := jwks.NewKeyManager()
	if err != nil {
		panic(err)
	}
	engine := oauth2.NewEngine("http://localhost:4444", km)
	server := NewServer("http://localhost:4444", nil, km, engine, newTestHealthHandler(checkErr != nil))
	return server.Routes()
}

func TestHealthContract(t *testing.T) {
	health.Contract(t, newContractMux)
}

func TestAdminClientsLifecycleAndSecretRotation(t *testing.T) {
	km, err := jwks.NewKeyManager()
	if err != nil {
		t.Fatalf("failed to create KeyManager: %v", err)
	}
	engine := oauth2.NewEngine("http://localhost:4444", km)

	// In -short mode, we can test HTTP routing and handlers with a mocked or pgtest repo.
	// We'll use pgtest when available, or skip if -short.
	if testing.Short() {
		t.Skip("skipping postgres-backed HTTP test in -short mode")
	}

	pool := pgtest.StartPostgres(t, "../../../migrations")
	repo := postgres.NewRepository(pool)
	server := NewServer("http://localhost:4444", repo, km, engine, newTestHealthHandler(false))
	router := server.Routes()

	// 1. POST /admin/clients - Create client
	createBody := `{
		"client_id": "app-client-1",
		"client_name": "App Client 1",
		"client_secret": "initial-secret-123",
		"grant_types": ["client_credentials"],
		"scopes": ["read", "write"],
		"is_public": false
	}`
	req := httptest.NewRequest("POST", "/admin/clients", strings.NewReader(createBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", rec.Code, rec.Body.String())
	}

	// 2. GET /admin/clients/{id} - Get client
	reqGet := httptest.NewRequest("GET", "/admin/clients/app-client-1", nil)
	recGet := httptest.NewRecorder()
	router.ServeHTTP(recGet, reqGet)

	if recGet.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from GET /admin/clients/{id}, got %d: %s", recGet.Code, recGet.Body.String())
	}
	var clientResp map[string]interface{}
	if err := json.Unmarshal(recGet.Body.Bytes(), &clientResp); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if clientResp["client_name"] != "App Client 1" {
		t.Errorf("expected client_name 'App Client 1', got %v", clientResp["client_name"])
	}

	// 3. Authenticate with initial secret via /oauth2/token
	reqTok := httptest.NewRequest("POST", "/oauth2/token", strings.NewReader("grant_type=client_credentials&client_id=app-client-1&client_secret=initial-secret-123"))
	reqTok.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recTok := httptest.NewRecorder()
	router.ServeHTTP(recTok, reqTok)

	if recTok.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from token endpoint with initial secret, got %d: %s", recTok.Code, recTok.Body.String())
	}

	// 4. POST /admin/clients/{id}/rotate-secret - Rotate Secret with overlap
	rotateReq := httptest.NewRequest("POST", "/admin/clients/app-client-1/rotate-secret", strings.NewReader(`{"overlap_seconds": 3600}`))
	rotateReq.Header.Set("Content-Type", "application/json")
	rotateRec := httptest.NewRecorder()
	router.ServeHTTP(rotateRec, rotateReq)

	if rotateRec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from rotate-secret, got %d: %s", rotateRec.Code, rotateRec.Body.String())
	}
	var rotateResp map[string]interface{}
	_ = json.Unmarshal(rotateRec.Body.Bytes(), &rotateResp)
	newSecret, ok := rotateResp["client_secret"].(string)
	if !ok || newSecret == "" {
		t.Fatalf("expected new client_secret in rotate response, got %v", rotateResp)
	}

	// 5. Authenticate with new secret - must succeed
	reqTokNew := httptest.NewRequest("POST", "/oauth2/token", strings.NewReader("grant_type=client_credentials&client_id=app-client-1&client_secret="+newSecret))
	reqTokNew.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recTokNew := httptest.NewRecorder()
	router.ServeHTTP(recTokNew, reqTokNew)

	if recTokNew.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from token endpoint with new secret, got %d: %s", recTokNew.Code, recTokNew.Body.String())
	}

	// 6. Authenticate with OLD secret during rollover window - must STILL succeed
	reqTokOld := httptest.NewRequest("POST", "/oauth2/token", strings.NewReader("grant_type=client_credentials&client_id=app-client-1&client_secret=initial-secret-123"))
	reqTokOld.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recTokOld := httptest.NewRecorder()
	router.ServeHTTP(recTokOld, reqTokOld)

	if recTokOld.Code != http.StatusOK {
		t.Fatalf("expected 200 OK with old secret during rollover window, got %d: %s", recTokOld.Code, recTokOld.Body.String())
	}

	// 7. PATCH /admin/clients/{id} - Update client
	patchBody := `{"client_name": "Patched App Name"}`
	reqPatch := httptest.NewRequest("PATCH", "/admin/clients/app-client-1", strings.NewReader(patchBody))
	reqPatch.Header.Set("Content-Type", "application/json")
	recPatch := httptest.NewRecorder()
	router.ServeHTTP(recPatch, reqPatch)

	if recPatch.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from PATCH /admin/clients/{id}, got %d: %s", recPatch.Code, recPatch.Body.String())
	}

	// 8. DELETE /admin/clients/{id} - Delete client
	reqDel := httptest.NewRequest("DELETE", "/admin/clients/app-client-1", nil)
	recDel := httptest.NewRecorder()
	router.ServeHTTP(recDel, reqDel)

	if recDel.Code != http.StatusNoContent {
		t.Fatalf("expected 204 No Content from DELETE /admin/clients/{id}, got %d", recDel.Code)
	}

	// Verify 404 after deletion
	reqGetDeleted := httptest.NewRequest("GET", "/admin/clients/app-client-1", nil)
	recGetDeleted := httptest.NewRecorder()
	router.ServeHTTP(recGetDeleted, reqGetDeleted)
	if recGetDeleted.Code != http.StatusNotFound {
		t.Fatalf("expected 404 Not Found for deleted client, got %d", recGetDeleted.Code)
	}
}

func TestTokenIntrospectionRevocationAndGrants(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping postgres-backed HTTP test in -short mode")
	}

	km, err := jwks.NewKeyManager()
	if err != nil {
		t.Fatalf("failed to create KeyManager: %v", err)
	}
	engine := oauth2.NewEngine("http://localhost:4444", km)
	pool := pgtest.StartPostgres(t, "../../../migrations")
	repo := postgres.NewRepository(pool)
	server := NewServer("http://localhost:4444", repo, km, engine, newTestHealthHandler(false))
	router := server.Routes()

	// 1. Issue a valid token
	client := &core.OAuth2Client{
		ID:         "client-intro",
		ClientName: "Intro Client",
		IsPublic:   true,
		Scopes:     []string{"openid", "email"},
	}
	if err := repo.CreateClient(context.Background(), client); err != nil {
		t.Fatalf("CreateClient error: %v", err)
	}

	tokenResp, err := engine.IssueClientCredentialsToken(client, []string{"openid", "email"})
	if err != nil {
		t.Fatalf("IssueClientCredentialsToken error: %v", err)
	}

	// 2. POST /oauth2/introspect - Token is active
	reqIntro := httptest.NewRequest("POST", "/oauth2/introspect", strings.NewReader("token="+tokenResp.AccessToken))
	reqIntro.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recIntro := httptest.NewRecorder()
	router.ServeHTTP(recIntro, reqIntro)

	if recIntro.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from /oauth2/introspect, got %d: %s", recIntro.Code, recIntro.Body.String())
	}
	var introResp map[string]interface{}
	_ = json.Unmarshal(recIntro.Body.Bytes(), &introResp)
	if introResp["active"] != true {
		t.Errorf("expected active=true, got %v", introResp["active"])
	}
	if introResp["sub"] != "client-intro" {
		t.Errorf("expected sub 'client-intro', got %v", introResp["sub"])
	}

	// 3. POST /oauth2/revoke - Revoke token
	reqRevoke := httptest.NewRequest("POST", "/oauth2/revoke", strings.NewReader("token="+tokenResp.AccessToken))
	reqRevoke.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recRevoke := httptest.NewRecorder()
	router.ServeHTTP(recRevoke, reqRevoke)

	if recRevoke.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from /oauth2/revoke, got %d", recRevoke.Code)
	}

	// 4. POST /oauth2/introspect - Token is now inactive
	reqIntro2 := httptest.NewRequest("POST", "/oauth2/introspect", strings.NewReader("token="+tokenResp.AccessToken))
	reqIntro2.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recIntro2 := httptest.NewRecorder()
	router.ServeHTTP(recIntro2, reqIntro2)

	if recIntro2.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from /oauth2/introspect after revocation, got %d", recIntro2.Code)
	}
	var introResp2 map[string]interface{}
	_ = json.Unmarshal(recIntro2.Body.Bytes(), &introResp2)
	if introResp2["active"] != false {
		t.Errorf("expected active=false after revocation, got %v", introResp2["active"])
	}

	// 5. GET /admin/grants
	_ = repo.CreateGrant(context.Background(), &core.Grant{
		CodeHash:    "grant-code-1",
		ClientID:    client.ID,
		Subject:     "user-x",
		Scopes:      []string{"openid"},
		RedirectURI: "https://example.com/cb",
		ExpiresAt:   time.Now().Add(5 * time.Minute).UTC(),
	})

	reqGrants := httptest.NewRequest("GET", "/admin/grants?client_id=client-intro", nil)
	recGrants := httptest.NewRecorder()
	router.ServeHTTP(recGrants, reqGrants)

	if recGrants.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from /admin/grants, got %d: %s", recGrants.Code, recGrants.Body.String())
	}
	var grantsEnvelope map[string]interface{}
	_ = json.Unmarshal(recGrants.Body.Bytes(), &grantsEnvelope)
	items, ok := grantsEnvelope["data"].([]interface{})
	if !ok || len(items) != 1 {
		t.Errorf("expected 1 grant in /admin/grants, got %v", items)
	}
}

func TestKeyRotationEndpoint(t *testing.T) {
	km, err := jwks.NewKeyManager()
	if err != nil {
		t.Fatalf("failed to create KeyManager: %v", err)
	}
	engine := oauth2.NewEngine("http://localhost:4444", km)
	server := NewServer("http://localhost:4444", nil, km, engine, newTestHealthHandler(false))
	router := server.Routes()

	// Initial JWKS has 1 key
	reqJWKS1 := httptest.NewRequest("GET", "/.well-known/jwks.json", nil)
	recJWKS1 := httptest.NewRecorder()
	router.ServeHTTP(recJWKS1, reqJWKS1)
	var jwks1 map[string][]interface{}
	_ = json.Unmarshal(recJWKS1.Body.Bytes(), &jwks1)
	if len(jwks1["keys"]) != 1 {
		t.Fatalf("expected 1 key initially, got %d", len(jwks1["keys"]))
	}

	// POST /admin/keys/rotate
	reqRotate := httptest.NewRequest("POST", "/admin/keys/rotate", nil)
	recRotate := httptest.NewRecorder()
	router.ServeHTTP(recRotate, reqRotate)

	if recRotate.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from /admin/keys/rotate, got %d", recRotate.Code)
	}
	var rotatedJWK map[string]interface{}
	_ = json.Unmarshal(recRotate.Body.Bytes(), &rotatedJWK)
	if rotatedJWK["kid"] == "" {
		t.Error("expected kid in rotated JWK response")
	}

	// JWKS now exposes 2 keys (current + retiring)
	reqJWKS2 := httptest.NewRequest("GET", "/.well-known/jwks.json", nil)
	recJWKS2 := httptest.NewRecorder()
	router.ServeHTTP(recJWKS2, reqJWKS2)
	var jwks2 map[string][]interface{}
	_ = json.Unmarshal(recJWKS2.Body.Bytes(), &jwks2)
	if len(jwks2["keys"]) != 2 {
		t.Fatalf("expected 2 keys in JWKS after rotation, got %d", len(jwks2["keys"]))
	}
}

func TestScopeCatalogueEndpoints(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping postgres-backed HTTP test in -short mode")
	}

	km, err := jwks.NewKeyManager()
	if err != nil {
		t.Fatalf("failed to create KeyManager: %v", err)
	}
	engine := oauth2.NewEngine("http://localhost:4444", km)
	pool := pgtest.StartPostgres(t, "../../../migrations")
	repo := postgres.NewRepository(pool)
	server := NewServer("http://localhost:4444", repo, km, engine, newTestHealthHandler(false))
	router := server.Routes()

	// 1. POST /admin/scopes - Create scope
	createBody := `{
		"name": "billing:read",
		"description": "Read billing invoices",
		"claims": ["org_id", "invoice_access"]
	}`
	reqCreate := httptest.NewRequest("POST", "/admin/scopes", strings.NewReader(createBody))
	reqCreate.Header.Set("Content-Type", "application/json")
	recCreate := httptest.NewRecorder()
	router.ServeHTTP(recCreate, reqCreate)

	if recCreate.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created from POST /admin/scopes, got %d: %s", recCreate.Code, recCreate.Body.String())
	}

	// 2. GET /admin/scopes - List scopes
	reqList := httptest.NewRequest("GET", "/admin/scopes", nil)
	recList := httptest.NewRecorder()
	router.ServeHTTP(recList, reqList)

	if recList.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from GET /admin/scopes, got %d", recList.Code)
	}
	var scopes []map[string]interface{}
	if err := json.Unmarshal(recList.Body.Bytes(), &scopes); err != nil {
		t.Fatalf("failed to unmarshal scopes: %v", err)
	}
	if len(scopes) != 1 || scopes[0]["name"] != "billing:read" {
		t.Errorf("expected 1 scope with name 'billing:read', got %v", scopes)
	}

	// 3. DELETE /admin/scopes/{name} - Delete scope
	reqDel := httptest.NewRequest("DELETE", "/admin/scopes/billing:read", nil)
	recDel := httptest.NewRecorder()
	router.ServeHTTP(recDel, reqDel)

	if recDel.Code != http.StatusNoContent {
		t.Fatalf("expected 204 No Content from DELETE /admin/scopes/{name}, got %d", recDel.Code)
	}

	// 4. DELETE /admin/scopes/{name} again - 404 Not Found
	reqDel2 := httptest.NewRequest("DELETE", "/admin/scopes/billing:read", nil)
	recDel2 := httptest.NewRecorder()
	router.ServeHTTP(recDel2, reqDel2)

	if recDel2.Code != http.StatusNotFound {
		t.Fatalf("expected 404 Not Found from deleting non-existent scope, got %d", recDel2.Code)
	}
}

func TestHTTP_Metrics(t *testing.T) {
	km, err := jwks.NewKeyManager()
	if err != nil {
		t.Fatalf("failed to create KeyManager: %v", err)
	}
	engine := oauth2.NewEngine("http://localhost:4444", km)
	server := NewServer("http://localhost:4444", nil, km, engine, newTestHealthHandler(false))
	router := server.Routes()

	// Trigger token endpoint error to exercise token metrics
	tokenReq := httptest.NewRequest("POST", "/oauth2/token", strings.NewReader("grant_type=refresh_token"))
	tokenReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	tokenRec := httptest.NewRecorder()
	router.ServeHTTP(tokenRec, tokenReq)

	req := httptest.NewRequest("GET", "/metrics", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 from /metrics, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "autorix_janus_active_jwks_keys") {
		t.Errorf("expected body to contain autorix_janus_active_jwks_keys, got: %s", body)
	}
	if !strings.Contains(body, "autorix_janus_tokens_issued_total") {
		t.Errorf("expected body to contain autorix_janus_tokens_issued_total, got: %s", body)
	}
}

