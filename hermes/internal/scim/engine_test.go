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

	// 2. Test FormatGroup
	groupID := uuid.New()
	group := &core.SCIMGroup{
		ID:          groupID,
		DisplayName: "Security Engineers",
		Members: []core.SCIMMember{
			{Value: userID.String(), Display: "ada.lovelace"},
		},
	}

	formattedGroup := engine.FormatGroup(group)
	if len(formattedGroup.Schemas) != 1 || formattedGroup.Schemas[0] != GroupSchemaURN {
		t.Errorf("expected schema %s, got %v", GroupSchemaURN, formattedGroup.Schemas)
	}
	if formattedGroup.Meta.ResourceType != "Group" {
		t.Errorf("expected resourceType 'Group', got %s", formattedGroup.Meta.ResourceType)
	}
	if formattedGroup.Meta.Location != "https://hermes.autorix.io/scim/v2/Groups/"+groupID.String() {
		t.Errorf("unexpected location: %s", formattedGroup.Meta.Location)
	}

	// 3. Test ServiceProviderConfig
	cfg := engine.ServiceProviderConfig()
	if cfg["schemas"].([]string)[0] != ServiceProviderURN {
		t.Errorf("expected ServiceProviderConfig schema %s", ServiceProviderURN)
	}
}

