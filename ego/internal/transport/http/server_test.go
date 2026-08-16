package http

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTTP_WhoAmI_Unauthorized(t *testing.T) {
	server := NewServer(nil, nil, nil)
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
