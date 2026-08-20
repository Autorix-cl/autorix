package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/autorix/platform/health"
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
		return health.Info{Engine: "ego"}
	})
}

func TestHTTP_WhoAmI_Unauthorized(t *testing.T) {
	server := NewServer(nil, nil, nil, newTestHealthHandler(false), true)
	router := server.Routes()

	req := httptest.NewRequest("GET", "/sessions/whoami", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}

	if !strings.Contains(rec.Body.String(), "No active session token provided") {
		t.Errorf("expected error message in body, got: %s", rec.Body.String())
	}
}

func TestHTTP_HealthAlive_Returns200(t *testing.T) {
	server := NewServer(nil, nil, nil, newTestHealthHandler(false), true)
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
	server := NewServer(nil, nil, nil, newTestHealthHandler(true), true)
	router := server.Routes()

	req := httptest.NewRequest("GET", "/health/ready", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
}

// newContractMux builds a real ego Server with its readiness check rigged to
// fail iff checkErr is non-nil, for the shared platform/health.Contract
// suite.
func newContractMux(checkErr error) http.Handler {
	return NewServer(nil, nil, nil, newTestHealthHandler(checkErr != nil), true).Routes()
}

func TestHealthContract(t *testing.T) {
	health.Contract(t, newContractMux)
}

func TestHTTP_AdminIdentities_InvalidUUID(t *testing.T) {
	server := NewServer(nil, nil, nil, newTestHealthHandler(false), true)
	router := server.Routes()

	// 1. GET with invalid UUID
	req := httptest.NewRequest("GET", "/admin/identities/not-a-uuid", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("GET /admin/identities/not-a-uuid: expected 400, got %d", rec.Code)
	}

	// 2. PATCH with invalid UUID
	req = httptest.NewRequest("PATCH", "/admin/identities/not-a-uuid", strings.NewReader(`{"state":"active"}`))
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("PATCH /admin/identities/not-a-uuid: expected 400, got %d", rec.Code)
	}

	// 3. DELETE with invalid UUID
	req = httptest.NewRequest("DELETE", "/admin/identities/not-a-uuid", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("DELETE /admin/identities/not-a-uuid: expected 400, got %d", rec.Code)
	}
}

func TestHTTP_AdminSessions_InvalidUUID(t *testing.T) {
	server := NewServer(nil, nil, nil, newTestHealthHandler(false), true)
	router := server.Routes()

	// 1. DELETE /admin/sessions/{id} with invalid UUID
	req := httptest.NewRequest("DELETE", "/admin/sessions/not-a-uuid", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("DELETE /admin/sessions/not-a-uuid: expected 400, got %d", rec.Code)
	}

	// 2. GET /admin/identities/{id}/sessions with invalid UUID
	req = httptest.NewRequest("GET", "/admin/identities/not-a-uuid/sessions", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("GET /admin/identities/not-a-uuid/sessions: expected 400, got %d", rec.Code)
	}

	// 3. DELETE /admin/identities/{id}/sessions with invalid UUID
	req = httptest.NewRequest("DELETE", "/admin/identities/not-a-uuid/sessions", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("DELETE /admin/identities/not-a-uuid/sessions: expected 400, got %d", rec.Code)
	}
}

func TestHTTP_AdminCredentialsAndMFA_InvalidUUID(t *testing.T) {
	server := NewServer(nil, nil, nil, newTestHealthHandler(false), true)
	router := server.Routes()

	// 1. Reset password
	req := httptest.NewRequest("POST", "/admin/identities/not-a-uuid/credentials/reset-password", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("POST reset-password invalid UUID: expected 400, got %d", rec.Code)
	}

	// 2. Recovery link
	req = httptest.NewRequest("POST", "/admin/identities/not-a-uuid/recovery-link", strings.NewReader(`{}`))
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("POST recovery-link invalid UUID: expected 400, got %d", rec.Code)
	}

	// 3. List credentials
	req = httptest.NewRequest("GET", "/admin/identities/not-a-uuid/credentials", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("GET credentials invalid UUID: expected 400, got %d", rec.Code)
	}

	// 4. Get MFA
	req = httptest.NewRequest("GET", "/admin/identities/not-a-uuid/mfa", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("GET mfa invalid UUID: expected 400, got %d", rec.Code)
	}

	// 5. Delete MFA
	req = httptest.NewRequest("DELETE", "/admin/identities/not-a-uuid/mfa", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("DELETE mfa invalid UUID: expected 400, got %d", rec.Code)
	}
}

func TestHTTP_AdminSchemas_Validation(t *testing.T) {
	server := NewServer(nil, nil, nil, newTestHealthHandler(false), true)
	router := server.Routes()

	// 1. POST schema with invalid JSON
	req := httptest.NewRequest("POST", "/admin/schemas", strings.NewReader(`not-json`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("POST /admin/schemas with invalid JSON: expected 400, got %d", rec.Code)
	}

	// 2. POST schema with missing ID
	req = httptest.NewRequest("POST", "/admin/schemas", strings.NewReader(`{"name":"Schema 1","schema":{}}`))
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("POST /admin/schemas missing ID: expected 400, got %d", rec.Code)
	}

	// 3. PATCH schema with invalid JSON
	req = httptest.NewRequest("PATCH", "/admin/schemas/custom", strings.NewReader(`not-json`))
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("PATCH /admin/schemas/custom with invalid JSON: expected 400, got %d", rec.Code)
	}
}

func TestHTTP_Metrics(t *testing.T) {
	server := NewServer(nil, nil, nil, newTestHealthHandler(false), true)
	router := server.Routes()

	// Trigger a failed login to exercise egoLoginsTotal
	loginReq := httptest.NewRequest("POST", "/self-service/login", strings.NewReader(`invalid-json`))
	loginRec := httptest.NewRecorder()
	router.ServeHTTP(loginRec, loginReq)

	req := httptest.NewRequest("GET", "/metrics", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 from /metrics, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "autorix_ego_active_sessions") {
		t.Errorf("expected body to contain autorix_ego_active_sessions, got: %s", body)
	}
	if !strings.Contains(body, "autorix_ego_logins_total") {
		t.Errorf("expected body to contain autorix_ego_logins_total, got: %s", body)
	}
}




