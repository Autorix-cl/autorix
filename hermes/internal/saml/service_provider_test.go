package saml

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/autorix/hermes/internal/core"
)

func generateTestCertPEM(cn string, notBefore, notAfter time.Time) (string, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", err
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject: pkix.Name{
			CommonName:   cn,
			Organization: []string{"Autorix Test Org"},
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return "", err
	}

	pemBlock := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: derBytes,
	})

	return string(pemBlock), nil
}

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
		ID:         "okta-corp",
		IdPSSOURL:  "https://okta.corp.com/app/sso/saml",
		SPEntityID: "https://hermes.autorix.io/sp",
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

func TestServiceProvider_AttributeMapping(t *testing.T) {
	sp := NewServiceProvider("https://hermes.autorix.io")

	mockXML := `<Response xmlns="urn:oasis:names:tc:SAML:2.0:protocol">
  <Assertion xmlns="urn:oasis:names:tc:SAML:2.0:assertion">
    <Subject>
      <NameID>adalo@example.com</NameID>
    </Subject>
    <AttributeStatement>
      <Attribute Name="http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress">
        <AttributeValue>ada.lovelace@example.com</AttributeValue>
      </Attribute>
      <Attribute Name="urn:oid:2.5.4.42">
        <AttributeValue>Ada</AttributeValue>
      </Attribute>
      <Attribute Name="urn:oid:2.5.4.4">
        <AttributeValue>Lovelace</AttributeValue>
      </Attribute>
      <Attribute Name="custom_role">
        <AttributeValue>Software Architect</AttributeValue>
      </Attribute>
    </AttributeStatement>
  </Assertion>
</Response>`

	b64Response := base64.StdEncoding.EncodeToString([]byte(mockXML))

	provider := &core.SAMLProvider{
		ID: "okta-saml",
		AttributeMapping: map[string]string{
			"email":      "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress",
			"first_name": "urn:oid:2.5.4.42",
			"last_name":  "urn:oid:2.5.4.4",
			"role":       "custom_role",
		},
	}

	assertion, err := sp.ParseAssertion(b64Response, provider)
	if err != nil {
		t.Fatalf("ParseAssertion failed: %v", err)
	}

	if assertion.Email != "ada.lovelace@example.com" {
		t.Errorf("expected mapped email 'ada.lovelace@example.com', got %s", assertion.Email)
	}
	if assertion.FirstName != "Ada" || assertion.LastName != "Lovelace" {
		t.Errorf("expected Ada Lovelace, got %s %s", assertion.FirstName, assertion.LastName)
	}
	if assertion.Traits == nil || assertion.Traits["role"] != "Software Architect" {
		t.Errorf("expected trait 'role' = 'Software Architect', got %v", assertion.Traits)
	}
}

func TestCertificateManagement_ParseAndValidate(t *testing.T) {
	// Generate valid cert for testing
	now := time.Now()
	validCertPEM, err := generateTestCertPEM("Valid Cert", now.Add(-1*time.Hour), now.Add(365*24*time.Hour))
	if err != nil {
		t.Fatalf("failed to generate valid cert: %v", err)
	}

	expiringCertPEM, err := generateTestCertPEM("Expiring Cert", now.Add(-100*24*time.Hour), now.Add(10*24*time.Hour))
	if err != nil {
		t.Fatalf("failed to generate expiring cert: %v", err)
	}

	expiredCertPEM, err := generateTestCertPEM("Expired Cert", now.Add(-400*24*time.Hour), now.Add(-10*24*time.Hour))
	if err != nil {
		t.Fatalf("failed to generate expired cert: %v", err)
	}

	// 1. Single valid cert
	certs, expiresAt, warnings, err := ParseCertificatesPEM(validCertPEM)
	if err != nil {
		t.Fatalf("ParseCertificatesPEM valid failed: %v", err)
	}
	if len(certs) != 1 {
		t.Fatalf("expected 1 cert, got %d", len(certs))
	}
	if certs[0].Expired || certs[0].ExpiringSoon {
		t.Errorf("expected valid cert not expired or expiring soon, got %+v", certs[0])
	}
	if expiresAt == nil || expiresAt.IsZero() {
		t.Errorf("expected expiresAt to be populated")
	}
	if len(warnings) != 0 {
		t.Errorf("expected 0 warnings for valid cert, got %v", warnings)
	}

	// 2. Expiring soon cert (10 days remaining)
	certs, _, warnings, err = ParseCertificatesPEM(expiringCertPEM)
	if err != nil {
		t.Fatalf("ParseCertificatesPEM expiring failed: %v", err)
	}
	if len(certs) != 1 || !certs[0].ExpiringSoon || certs[0].Expired {
		t.Errorf("expected expiring soon cert: %+v", certs[0])
	}
	if len(warnings) == 0 {
		t.Errorf("expected warning for expiring soon cert")
	}

	// 3. Expired cert
	certs, _, warnings, err = ParseCertificatesPEM(expiredCertPEM)
	if err != nil {
		t.Fatalf("ParseCertificatesPEM expired failed: %v", err)
	}
	if len(certs) != 1 || !certs[0].Expired {
		t.Errorf("expected expired cert: %+v", certs[0])
	}
	if len(warnings) == 0 {
		t.Errorf("expected warning for expired cert")
	}

	// 4. Multiple certs for rollover (Valid + Expiring)
	multiPEM := validCertPEM + "\n" + expiringCertPEM
	certs, _, _, err = ParseCertificatesPEM(multiPEM)
	if err != nil {
		t.Fatalf("ParseCertificatesPEM multi failed: %v", err)
	}
	if len(certs) != 2 {
		t.Fatalf("expected 2 certs for rollover, got %d", len(certs))
	}
}
