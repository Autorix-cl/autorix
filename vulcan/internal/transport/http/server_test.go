package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/autorix/vulcan/internal/core"
	"github.com/autorix/vulcan/internal/macaroon"
)

func TestAttenuateHandler(t *testing.T) {
	server := NewServer(nil, "https://api.autorix.io")
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
