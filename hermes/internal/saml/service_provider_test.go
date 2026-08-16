package saml

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/autorix/hermes/internal/core"
)

func TestServiceProvider_MetadataAndAuthnRequest(t *testing.T) {
	sp := NewServiceProvider("https://hermes.autorix.io")

	// 1. Test Metadata XML
	metadataXML := sp.GenerateSPMetadataXML("https://hermes.autorix.io/sp")
	if !strings.Contains(metadataXML, `entityID="https://hermes.autorix.io/sp"`) {
		t.Errorf("metadata missing entityID")
	}
	if !strings.Contains(metadataXML, `Location="https://hermes.autorix.io/saml/acs"`) {
		t.Errorf("metadata missing ACS Location")
	}

	// 2. Test AuthnRequest URL
	provider := &core.SAMLProvider{
		ID:          "okta-corp",
		IdPSSOURL:   "https://okta.corp.com/app/sso/saml",
		SPEntityID:  "https://hermes.autorix.io/sp",
	}

	redirectURL, err := sp.GenerateAuthnRequestURL(provider)
	if err != nil {
		t.Fatalf("GenerateAuthnRequestURL failed: %v", err)
	}

	if !strings.Contains(redirectURL, "https://okta.corp.com/app/sso/saml?SAMLRequest=") {
		t.Errorf("unexpected redirect URL: %s", redirectURL)
	}
}

func TestServiceProvider_ParseAssertion(t *testing.T) {
	sp := NewServiceProvider("https://hermes.autorix.io")

	mockXML := `<Response xmlns="urn:oasis:names:tc:SAML:2.0:protocol">
  <Assertion xmlns="urn:oasis:names:tc:SAML:2.0:assertion">
    <Subject>
      <NameID>developer@oktacorp.com</NameID>
    </Subject>
    <AttributeStatement>
      <Attribute Name="email">
        <AttributeValue>developer@oktacorp.com</AttributeValue>
      </Attribute>
      <Attribute Name="first_name">
        <AttributeValue>Grace</AttributeValue>
      </Attribute>
      <Attribute Name="last_name">
        <AttributeValue>Hopper</AttributeValue>
      </Attribute>
    </AttributeStatement>
  </Assertion>
</Response>`

	b64Response := base64.StdEncoding.EncodeToString([]byte(mockXML))

	assertion, err := sp.ParseAssertion(b64Response, nil)
	if err != nil {
		t.Fatalf("ParseAssertion failed: %v", err)
	}

	if assertion.Subject != "developer@oktacorp.com" {
		t.Errorf("expected subject 'developer@oktacorp.com', got %s", assertion.Subject)
	}

	if assertion.FirstName != "Grace" || assertion.LastName != "Hopper" {
		t.Errorf("expected Grace Hopper, got %s %s", assertion.FirstName, assertion.LastName)
	}
}
