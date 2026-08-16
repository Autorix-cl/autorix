package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHermesEndpoints(t *testing.T) {
	server := NewServer(nil, "http://localhost:4477", "https://hermes.autorix.io/sp")
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
