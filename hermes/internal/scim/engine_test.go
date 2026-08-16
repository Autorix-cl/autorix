package scim

import (
	"testing"

	"github.com/autorix/hermes/internal/core"
	"github.com/google/uuid"
)

func TestSCIMEngine(t *testing.T) {
	engine := NewEngine("https://hermes.autorix.io")

	// 1. Test FormatUser
	userID := uuid.New()
	user := &core.SCIMUser{
		ID:         userID,
		UserName:   "ada.lovelace",
		ExternalID: "okta_12345",
	}

	formatted := engine.FormatUser(user)
	if len(formatted.Schemas) != 1 || formatted.Schemas[0] != UserSchemaURN {
		t.Errorf("expected schema %s, got %v", UserSchemaURN, formatted.Schemas)
	}

	if formatted.Meta.ResourceType != "User" {
		t.Errorf("expected resourceType 'User', got %s", formatted.Meta.ResourceType)
	}

	// 2. Test ServiceProviderConfig
	cfg := engine.ServiceProviderConfig()
	if cfg["schemas"].([]string)[0] != ServiceProviderURN {
		t.Errorf("expected ServiceProviderConfig schema %s", ServiceProviderURN)
	}
}
