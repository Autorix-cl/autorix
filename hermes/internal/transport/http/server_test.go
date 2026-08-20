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

	"github.com/autorix/hermes/internal/core"
	"github.com/autorix/hermes/internal/storage/postgres"
	"github.com/autorix/platform/health"
	"github.com/autorix/platform/pgtest"
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
		return health.Info{Engine: "hermes"}
	})
}

func newTestServer(t *testing.T) (*Server, http.Handler) {
	t.Helper()
	pool := pgtest.StartPostgres(t, "../../../migrations")
	repo := postgres.NewRepository(pool)
	server := NewServer(repo, "http://localhost:4477", "https://hermes.autorix.io/sp", newTestHealthHandler(false))
	return server, server.Routes()
}

func TestHTTP_HealthAlive_Returns200(t *testing.T) {
	server := NewServer(nil, "http://localhost:4477", "https://hermes.autorix.io/sp", newTestHealthHandler(false))
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
	server := NewServer(nil, "http://localhost:4477", "https://hermes.autorix.io/sp", newTestHealthHandler(true))
	router := server.Routes()

	req := httptest.NewRequest("GET", "/health/ready", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
}

func TestHermesEndpoints(t *testing.T) {
	server := NewServer(nil, "http://localhost:4477", "https://hermes.autorix.io/sp", newTestHealthHandler(false))
	router := server.Routes()

	// 1. Test SAML Metadata XML
	reqMeta := httptest.NewRequest("GET", "/saml/metadata", nil)
	recMeta := httptest.NewRecorder()
	router.ServeHTTP(recMeta, reqMeta)

	if recMeta.Code != http.StatusOK {
		t.Errorf("expected 200 OK for metadata, got %d", recMeta.Code)
	}

	if !strings.Contains(recMeta.Body.String(), "EntityDescriptor") {
		t.Errorf("expected XML EntityDescriptor in response")
	}

	// 2. Test SCIM ServiceProviderConfig
	reqSCIM := httptest.NewRequest("GET", "/scim/v2/ServiceProviderConfig", nil)
	recSCIM := httptest.NewRecorder()
	router.ServeHTTP(recSCIM, reqSCIM)

	if recSCIM.Code != http.StatusOK {
		t.Errorf("expected 200 OK for SCIM config, got %d", recSCIM.Code)
	}

	var scimConfig map[string]interface{}
	if err := json.Unmarshal(recSCIM.Body.Bytes(), &scimConfig); err != nil {
		t.Fatalf("failed to parse SCIM config: %v", err)
	}

	if scimConfig["documentationUri"] != "https://docs.autorix.io/scim" {
		t.Errorf("unexpected documentationUri: %v", scimConfig["documentationUri"])
	}
}

func TestProviderLifecycleEndpoints(t *testing.T) {
	_, router := newTestServer(t)

	// 1. Create provider via POST /admin/providers
	createBody := `{
		"id": "azure-ad",
		"display_name": "Azure Active Directory",
		"idp_entity_id": "https://sts.windows.net/tenant-id/",
		"idp_sso_url": "https://login.microsoftonline.com/sso",
		"attribute_mapping": {
			"email": "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress"
		},
		"enabled": true
	}`
	req := httptest.NewRequest("POST", "/admin/providers", bytes.NewBufferString(createBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", rec.Code, rec.Body.String())
	}

	// 2. GET /admin/providers/{id}
	req = httptest.NewRequest("GET", "/admin/providers/azure-ad", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}
	var provider core.SAMLProvider
	if err := json.Unmarshal(rec.Body.Bytes(), &provider); err != nil {
		t.Fatalf("unmarshal provider: %v", err)
	}
	if provider.DisplayName != "Azure Active Directory" {
		t.Fatalf("unexpected provider display name: %s", provider.DisplayName)
	}

	// 3. PATCH /admin/providers/{id}
	patchBody := `{"display_name": "Microsoft Entra ID"}`
	req = httptest.NewRequest("PATCH", "/admin/providers/azure-ad", bytes.NewBufferString(patchBody))
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}

	// 4. POST /admin/providers/{id}/disable
	req = httptest.NewRequest("POST", "/admin/providers/azure-ad/disable", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for disable, got %d: %s", rec.Code, rec.Body.String())
	}

	// 5. POST /admin/providers/{id}/enable
	req = httptest.NewRequest("POST", "/admin/providers/azure-ad/enable", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for enable, got %d: %s", rec.Code, rec.Body.String())
	}

	// 6. DELETE /admin/providers/{id}
	req = httptest.NewRequest("DELETE", "/admin/providers/azure-ad", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204 No Content for delete, got %d: %s", rec.Code, rec.Body.String())
	}

	// 7. Verify 404 after deletion
	req = httptest.NewRequest("GET", "/admin/providers/azure-ad", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 Not Found, got %d", rec.Code)
	}
}

func TestSCIMGroupsEndpoints(t *testing.T) {
	_, router := newTestServer(t)

	// 1. POST /scim/v2/Groups
	groupBody := `{
		"displayName": "Platform Engineers",
		"members": [
			{"value": "11111111-1111-1111-1111-111111111111", "display": "Linus Torvalds"}
		]
	}`
	req := httptest.NewRequest("POST", "/scim/v2/Groups", bytes.NewBufferString(groupBody))
	req.Header.Set("Content-Type", "application/scim+json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", rec.Code, rec.Body.String())
	}

	var created core.SCIMGroup
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("unmarshal created group: %v", err)
	}
	if created.ID == uuid.Nil || created.DisplayName != "Platform Engineers" {
		t.Fatalf("unexpected group: %+v", created)
	}

	// 2. GET /scim/v2/Groups
	req = httptest.NewRequest("GET", "/scim/v2/Groups", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}

	// 3. GET /scim/v2/Groups/{id}
	req = httptest.NewRequest("GET", "/scim/v2/Groups/"+created.ID.String(), nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}

	// 4. PATCH /scim/v2/Groups/{id}
	updateBody := `{"displayName": "Core Infrastructure"}`
	req = httptest.NewRequest("PATCH", "/scim/v2/Groups/"+created.ID.String(), bytes.NewBufferString(updateBody))
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for patch, got %d", rec.Code)
	}

	// 5. DELETE /scim/v2/Groups/{id}
	req = httptest.NewRequest("DELETE", "/scim/v2/Groups/"+created.ID.String(), nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204 No Content, got %d", rec.Code)
	}
}

func TestSCIMSyncHistoryEndpoints(t *testing.T) {
	_, router := newTestServer(t)

	// 1. POST /admin/scim/sync-history
	syncBody := `{
		"provider_id": "okta-corp",
		"resource_type": "Users",
		"status": "success",
		"total_records": 100,
		"created_count": 80,
		"updated_count": 20,
		"deleted_count": 0,
		"error_count": 0,
		"errors": []
	}`
	req := httptest.NewRequest("POST", "/admin/scim/sync-history", bytes.NewBufferString(syncBody))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", rec.Code, rec.Body.String())
	}

	// 2. GET /admin/scim/sync-history
	req = httptest.NewRequest("GET", "/admin/scim/sync-history?limit=10", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}

	var history []core.SCIMSyncHistory
	if err := json.Unmarshal(rec.Body.Bytes(), &history); err != nil {
		t.Fatalf("unmarshal sync history: %v", err)
	}
	if len(history) != 1 || history[0].TotalRecords != 100 {
		t.Fatalf("unexpected sync history: %+v", history)
	}
}

// newContractMux builds a real hermes Server with its readiness check rigged
// to fail iff checkErr is non-nil, for the shared platform/health.Contract
// suite.
func newContractMux(checkErr error) http.Handler {
	server := NewServer(nil, "http://localhost:4477", "https://hermes.autorix.io/sp", newTestHealthHandler(checkErr != nil))
	return server.Routes()
}

func TestHealthContract(t *testing.T) {
	health.Contract(t, newContractMux)
}

func TestHTTP_Metrics(t *testing.T) {
	server := NewServer(nil, "http://localhost:4477", "https://hermes.autorix.io/sp", newTestHealthHandler(false))
	router := server.Routes()

	// Trigger failed SCIM sync record to exercise error metric
	syncReq := httptest.NewRequest("POST", "/admin/scim/sync-history", strings.NewReader(`invalid-json`))
	syncRec := httptest.NewRecorder()
	router.ServeHTTP(syncRec, syncReq)

	req := httptest.NewRequest("GET", "/metrics", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 from /metrics, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "autorix_hermes_saml_logins_total") {
		t.Errorf("expected body to contain autorix_hermes_saml_logins_total, got: %s", body)
	}
	if !strings.Contains(body, "autorix_hermes_scim_sync_total") {
		t.Errorf("expected body to contain autorix_hermes_scim_sync_total, got: %s", body)
	}
}

