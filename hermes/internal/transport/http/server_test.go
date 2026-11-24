package http

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/autorix/hermes/internal/core"
	"github.com/autorix/hermes/internal/saml"
	"github.com/autorix/hermes/internal/storage/postgres"
	"github.com/autorix/platform/health"
	"github.com/autorix/platform/pgtest"
	"github.com/google/uuid"
)

func generateHTTPTestKeyAndCert(t *testing.T, notBefore, notAfter time.Time) (*rsa.PrivateKey, string) {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey: %v", err)
	}
	tmpl := x509.Certificate{
		SerialNumber:          big.NewInt(time.Now().UnixNano()),
		Subject:               pkix.Name{CommonName: "Hermes Test IdP"},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &priv.PublicKey, priv)
	if err != nil {
		t.Fatalf("x509.CreateCertificate: %v", err)
	}
	return priv, string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
}

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
	now := time.Now()
	_, certPEM := generateHTTPTestKeyAndCert(t, now.Add(-time.Hour), now.Add(24*time.Hour))

	// 1. Create provider via POST /admin/providers
	createPayload, _ := json.Marshal(map[string]interface{}{
		"id":                  "azure-ad",
		"display_name":        "Azure Active Directory",
		"idp_entity_id":       "https://sts.windows.net/tenant-id/",
		"idp_sso_url":         "https://login.microsoftonline.com/sso",
		"idp_certificate_pem": certPEM,
		"attribute_mapping": map[string]string{
			"email": "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress",
		},
		"enabled": true,
	})
	req := httptest.NewRequest("POST", "/admin/providers", bytes.NewReader(createPayload))
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

type mockHermesRepo struct {
	providers      map[string]*core.SAMLProvider
	lastSyncLimit  int
	createErr      error
	createUserErr  error
	createGroupErr error
}

func newMockHermesRepo() *mockHermesRepo {
	return &mockHermesRepo{providers: make(map[string]*core.SAMLProvider)}
}

func (m *mockHermesRepo) CreateSAMLProvider(_ context.Context, p *core.SAMLProvider) error {
	if m.createErr != nil {
		return m.createErr
	}
	if _, exists := m.providers[p.ID]; exists {
		return postgres.ErrConflict
	}
	cp := *p
	m.providers[p.ID] = &cp
	return nil
}

func (m *mockHermesRepo) UpdateSAMLProvider(_ context.Context, p *core.SAMLProvider) error {
	if _, exists := m.providers[p.ID]; !exists {
		return postgres.ErrNotFound
	}
	cp := *p
	m.providers[p.ID] = &cp
	return nil
}

func (m *mockHermesRepo) SetSAMLProviderEnabled(_ context.Context, id string, enabled bool) error {
	p, ok := m.providers[id]
	if !ok {
		return postgres.ErrNotFound
	}
	p.Enabled = enabled
	return nil
}

func (m *mockHermesRepo) DeleteSAMLProvider(_ context.Context, id string) error {
	delete(m.providers, id)
	return nil
}

func (m *mockHermesRepo) ListSAMLProviders(_ context.Context, _ int, _ string) ([]core.SAMLProvider, bool, error) {
	return make([]core.SAMLProvider, 0), false, nil
}

func (m *mockHermesRepo) GetSAMLProvider(_ context.Context, id string) (*core.SAMLProvider, error) {
	p, ok := m.providers[id]
	if !ok {
		return nil, postgres.ErrNotFound
	}
	cp := *p
	return &cp, nil
}

func (m *mockHermesRepo) GetSAMLProviderByEntityID(_ context.Context, entityID string) (*core.SAMLProvider, error) {
	for _, p := range m.providers {
		if p.IdPEntityID == entityID {
			cp := *p
			return &cp, nil
		}
	}
	return nil, postgres.ErrNotFound
}

func (m *mockHermesRepo) CreateSCIMUser(_ context.Context, _ *core.SCIMUser) error {
	return m.createUserErr
}
func (m *mockHermesRepo) ListSCIMUsers(_ context.Context) ([]core.SCIMUser, error) {
	return make([]core.SCIMUser, 0), nil
}
func (m *mockHermesRepo) CreateSCIMGroup(_ context.Context, _ *core.SCIMGroup) error {
	return m.createGroupErr
}
func (m *mockHermesRepo) GetSCIMGroup(_ context.Context, _ uuid.UUID) (*core.SCIMGroup, error) {
	return nil, postgres.ErrNotFound
}
func (m *mockHermesRepo) ListSCIMGroups(_ context.Context) ([]core.SCIMGroup, error) {
	return make([]core.SCIMGroup, 0), nil
}
func (m *mockHermesRepo) UpdateSCIMGroup(_ context.Context, _ *core.SCIMGroup) error { return nil }
func (m *mockHermesRepo) DeleteSCIMGroup(_ context.Context, _ uuid.UUID) error       { return nil }
func (m *mockHermesRepo) RecordSCIMSync(_ context.Context, _ *core.SCIMSyncHistory) error {
	return nil
}
func (m *mockHermesRepo) ListSCIMSyncHistory(_ context.Context, limit int) ([]core.SCIMSyncHistory, error) {
	m.lastSyncLimit = limit
	return make([]core.SCIMSyncHistory, 0), nil
}

func TestSAMLHTTP_SecurityAndValidation(t *testing.T) {
	repo := newMockHermesRepo()
	now := time.Now()
	priv, validCertPEM := generateHTTPTestKeyAndCert(t, now.Add(-time.Hour), now.Add(24*time.Hour))
	_, expiredCertPEM := generateHTTPTestKeyAndCert(t, now.Add(-48*time.Hour), now.Add(-time.Hour))

	server := NewServer(nil, "https://hermes.autorix.io", "https://hermes.autorix.io/sp", newTestHealthHandler(false))
	server.repo = repo
	router := server.Routes()

	// 1. Reject creating provider with empty/placeholder/invalid/expired certificate
	for _, tc := range []struct {
		name string
		pem  string
		sso  string
	}{
		{"empty cert", "", "https://idp.example.com/sso"},
		{"invalid pem", "not-a-cert", "https://idp.example.com/sso"},
		{"expired cert", expiredCertPEM, "https://idp.example.com/sso"},
		{"javascript sso url", validCertPEM, "javascript:alert(1)"},
	} {
		t.Run("create_"+tc.name, func(t *testing.T) {
			body, _ := json.Marshal(map[string]interface{}{
				"id":                  "bad-idp",
				"display_name":        "Bad IdP",
				"idp_entity_id":       "https://bad.example.com",
				"idp_sso_url":         tc.sso,
				"idp_certificate_pem": tc.pem,
				"enabled":             true,
			})
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, httptest.NewRequest("POST", "/admin/providers", bytes.NewReader(body)))
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected 400 for %s, got %d: %s", tc.name, rec.Code, rec.Body.String())
			}
		})
	}

	// 2. Register valid enabled provider + disabled provider + malicious SSO provider directly in repo
	repo.providers["okta"] = &core.SAMLProvider{
		ID:                "okta",
		DisplayName:       "Okta",
		IdPEntityID:       "https://idp.okta.test",
		IdPSSOURL:         "https://idp.okta.test/sso",
		IdPCertificatePEM: validCertPEM,
		SPEntityID:        "https://hermes.autorix.io/sp",
		Enabled:           true,
	}
	repo.providers["disabled-idp"] = &core.SAMLProvider{
		ID:                "disabled-idp",
		IdPEntityID:       "https://idp.disabled.test",
		IdPSSOURL:         "https://idp.disabled.test/sso",
		IdPCertificatePEM: validCertPEM,
		SPEntityID:        "https://hermes.autorix.io/sp",
		Enabled:           false,
	}
	repo.providers["xss-idp"] = &core.SAMLProvider{
		ID:                "xss-idp",
		IdPEntityID:       "https://idp.xss.test",
		IdPSSOURL:         "javascript:alert(1)",
		IdPCertificatePEM: validCertPEM,
		SPEntityID:        "https://hermes.autorix.io/sp",
		Enabled:           true,
	}

	// 3. handleSAMLLogin rejects disabled provider with 403 and javascript: URL with 400
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest("GET", "/saml/login?provider=disabled-idp", nil))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for disabled provider login, got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest("GET", "/saml/login?provider=xss-idp", nil))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for javascript: SSO URL login, got %d", rec.Code)
	}

	// 4. handleSAMLACS rejects unsigned assertion with 401
	unsignedXML := `<samlp:Response xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol" xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion" ID="_r_u" Destination="https://hermes.autorix.io/saml/acs">
  <saml:Issuer>https://idp.okta.test</saml:Issuer>
  <samlp:Status><samlp:StatusCode Value="urn:oasis:names:tc:SAML:2.0:status:Success"/></samlp:Status>
  <saml:Assertion ID="_a_u"><saml:Issuer>https://idp.okta.test</saml:Issuer><saml:Subject><saml:NameID>admin@test.com</saml:NameID></saml:Subject></saml:Assertion>
</samlp:Response>`
	form := url.Values{}
	form.Set("SAMLResponse", base64.StdEncoding.EncodeToString([]byte(unsignedXML)))
	req := httptest.NewRequest("POST", "/saml/acs", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unsigned SAML assertion at ACS, got %d: %s", rec.Code, rec.Body.String())
	}

	// 5. handleSAMLACS accepts validly signed assertion & rejects replay
	signedRaw, err := saml.SignAssertionXML([]byte(fmt.Sprintf(`<samlp:Response xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol" xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion" ID="_r_ok" Destination="https://hermes.autorix.io/saml/acs">
  <saml:Issuer>https://idp.okta.test</saml:Issuer>
  <samlp:Status><samlp:StatusCode Value="urn:oasis:names:tc:SAML:2.0:status:Success"/></samlp:Status>
  <saml:Assertion ID="_a_ok">
    <saml:Issuer>https://idp.okta.test</saml:Issuer>
    <saml:Subject><saml:NameID>alice@test.com</saml:NameID></saml:Subject>
    <saml:Conditions NotBefore="%s" NotOnOrAfter="%s"><saml:AudienceRestriction><saml:Audience>https://hermes.autorix.io/sp</saml:Audience></saml:AudienceRestriction></saml:Conditions>
  </saml:Assertion>
</samlp:Response>`, now.Add(-time.Minute).UTC().Format(time.RFC3339), now.Add(5*time.Minute).UTC().Format(time.RFC3339))), priv)
	if err != nil {
		t.Fatalf("SignAssertionXML: %v", err)
	}
	form = url.Values{}
	form.Set("SAMLResponse", base64.StdEncoding.EncodeToString(signedRaw))

	req = httptest.NewRequest("POST", "/saml/acs", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for validly signed SAML assertion, got %d: %s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Fatalf("expected Content-Type application/json from writeJSON, got %q", ct)
	}

	// Replay attempt must fail with 401
	reqReplay := httptest.NewRequest("POST", "/saml/acs", strings.NewReader(form.Encode()))
	reqReplay.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recReplay := httptest.NewRecorder()
	router.ServeHTTP(recReplay, reqReplay)
	if recReplay.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for replayed SAML assertion, got %d", recReplay.Code)
	}

	// 6. handleListSCIMSyncHistory caps limit at 200 and returns [] not null
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest("GET", "/admin/scim/sync-history?limit=9999", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if repo.lastSyncLimit != 200 {
		t.Fatalf("expected limit capped to 200, got %d", repo.lastSyncLimit)
	}
	if strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Fatalf("expected empty JSON array [], got %q", rec.Body.String())
	}

	// 7. Arbitrary DB error on CreateSCIMUser returns 500, not 409
	repo.createUserErr = errors.New("connection reset by peer")
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest("POST", "/scim/v2/Users", strings.NewReader(`{"userName":"bob"}`)))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 for arbitrary DB error (not 409), got %d", rec.Code)
	}
}
