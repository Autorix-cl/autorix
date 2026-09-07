package http

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPublicRoutesRejectEveryAdministrativeEndpoint(t *testing.T) {
	public := NewServer("https://issuer.example", nil, nil, nil, nil).Routes()
	for _, path := range []string{
		"/admin/clients", "/admin/clients/client", "/admin/clients/client/rotate-secret",
		"/admin/oauth2/auth/requests/login/accept", "/admin/oauth2/auth/requests/consent/accept",
		"/admin/grants", "/admin/keys/rotate", "/admin/scopes", "/admin/scopes/read",
	} {
		for _, method := range []string{"GET", "HEAD", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"} {
			t.Run(method+" "+path, func(t *testing.T) {
				rec := httptest.NewRecorder()
				public.ServeHTTP(rec, httptest.NewRequest(method, path, strings.NewReader(`{"subject":"attacker"}`)))
				if rec.Code != http.StatusNotFound {
					t.Fatalf("public administrative endpoint returned %d, want 404", rec.Code)
				}
			})
		}
	}
}

func TestAdminRoutesReachPrivilegedHandlers(t *testing.T) {
	admin := NewServer("https://issuer.example", nil, nil, nil, nil).AdminRoutes()
	for _, tc := range []struct{ method, path string }{
		{"POST", "/admin/clients"}, {"POST", "/admin/scopes"},
		{"PUT", "/admin/oauth2/auth/requests/login/accept"},
		{"PUT", "/admin/oauth2/auth/requests/consent/accept"},
	} {
		t.Run(tc.path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			admin.ServeHTTP(rec, httptest.NewRequest(tc.method, tc.path, strings.NewReader("{")))
			if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "invalid_request") {
				t.Fatalf("private handler did not validate input: %d %s", rec.Code, rec.Body.String())
			}
		})
	}
	for _, path := range []string{"/.well-known/openid-configuration", "/oauth2/auth", "/oauth2/token"} {
		rec := httptest.NewRecorder()
		admin.ServeHTTP(rec, httptest.NewRequest("GET", path, nil))
		if rec.Code != http.StatusNotFound {
			t.Fatalf("admin listener exposes public route %s: %d", path, rec.Code)
		}
	}
}
