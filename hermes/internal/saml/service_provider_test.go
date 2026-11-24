package saml

import (
	"bytes"
	"compress/flate"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/autorix/hermes/internal/core"
)

func generateTestKeyAndCertPEM(cn string, notBefore, notAfter time.Time) (*rsa.PrivateKey, string, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, "", err
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
		return nil, "", err
	}

	pemBlock := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: derBytes,
	})

	return priv, string(pemBlock), nil
}

func generateTestCertPEM(cn string, notBefore, notAfter time.Time) (string, error) {
	_, pemStr, err := generateTestKeyAndCertPEM(cn, notBefore, notAfter)
	return pemStr, err
}

func buildSignedSAMLResponse(t *testing.T, priv *rsa.PrivateKey, assertionID, issuer, destination, audience, statusCode, nameID, attrsXML string, notBefore, notOnOrAfter time.Time) string {
	t.Helper()
	rawXML := fmt.Sprintf(`<samlp:Response xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol" xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion" ID="_resp_%s" Version="2.0" Destination="%s">
  <saml:Issuer>%s</saml:Issuer>
  <samlp:Status>
    <samlp:StatusCode Value="%s"/>
  </samlp:Status>
  <saml:Assertion ID="%s" Version="2.0">
    <saml:Issuer>%s</saml:Issuer>
    <saml:Subject>
      <saml:NameID>%s</saml:NameID>
    </saml:Subject>
    <saml:Conditions NotBefore="%s" NotOnOrAfter="%s">
      <saml:AudienceRestriction>
        <saml:Audience>%s</saml:Audience>
      </saml:AudienceRestriction>
    </saml:Conditions>
    <saml:AttributeStatement>
      %s
    </saml:AttributeStatement>
  </saml:Assertion>
</samlp:Response>`,
		assertionID, destination, issuer, statusCode, assertionID, issuer, nameID,
		notBefore.UTC().Format(time.RFC3339), notOnOrAfter.UTC().Format(time.RFC3339), audience, attrsXML,
	)

	signed, err := SignAssertionXML([]byte(rawXML), priv)
	if err != nil {
		t.Fatalf("SignAssertionXML failed: %v", err)
	}
	return base64.StdEncoding.EncodeToString(signed)
}

func TestServiceProvider_MetadataAndAuthnRequest(t *testing.T) {
	sp := NewServiceProvider("https://hermes.autorix.io")

	// 1. Test Metadata XML & XML escaping against injection
	metadataXML := sp.GenerateSPMetadataXML("https://hermes.autorix.io/sp")
	if !strings.Contains(metadataXML, `entityID="https://hermes.autorix.io/sp"`) {
		t.Errorf("metadata missing entityID")
	}
	if !strings.Contains(metadataXML, `Location="https://hermes.autorix.io/saml/acs"`) {
		t.Errorf("metadata missing ACS Location")
	}

	injectedMeta := sp.GenerateSPMetadataXML(`https://sp.test"><Evil/>`)
	if strings.Contains(injectedMeta, `<Evil/>`) {
		t.Errorf("metadata failed to XML-escape entityID: %s", injectedMeta)
	}

	// 2. Test AuthnRequest URL uses RFC 1951 DEFLATE + Base64
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

	parsed, err := url.Parse(redirectURL)
	if err != nil {
		t.Fatalf("failed to parse redirect URL: %v", err)
	}
	compressed, err := base64.StdEncoding.DecodeString(parsed.Query().Get("SAMLRequest"))
	if err != nil {
		t.Fatalf("failed to base64 decode SAMLRequest: %v", err)
	}
	inflated, err := io.ReadAll(flate.NewReader(bytes.NewReader(compressed)))
	if err != nil {
		t.Fatalf("failed to inflate RFC 1951 deflated SAMLRequest: %v", err)
	}
	if !strings.Contains(string(inflated), "AuthnRequest") || !strings.Contains(string(inflated), "https://hermes.autorix.io/sp") {
		t.Errorf("unexpected inflated AuthnRequest XML: %s", string(inflated))
	}
}

func TestValidateSSOURL(t *testing.T) {
	if err := ValidateSSOURL("https://idp.example.com/sso"); err != nil {
		t.Fatalf("expected valid https URL to pass, got %v", err)
	}
	for _, bad := range []string{"", "javascript:alert(1)", "data:text/html,hi", "http://idp.example.com/sso", "https://user:pass@idp.example.com/sso"} {
		if err := ValidateSSOURL(bad); err == nil {
			t.Errorf("expected ValidateSSOURL(%q) to fail", bad)
		}
	}
}

func TestServiceProvider_ParseAssertion(t *testing.T) {
	sp := NewServiceProvider("https://hermes.autorix.io")
	now := time.Now()
	priv, certPEM, err := generateTestKeyAndCertPEM("Okta Test IdP", now.Add(-time.Hour), now.Add(24*time.Hour))
	if err != nil {
		t.Fatalf("generateTestKeyAndCertPEM: %v", err)
	}

	provider := &core.SAMLProvider{
		ID:                "okta-corp",
		IdPEntityID:       "https://idp.oktacorp.com",
		IdPSSOURL:         "https://idp.oktacorp.com/sso",
		IdPCertificatePEM: certPEM,
		SPEntityID:        "https://hermes.autorix.io/sp",
		Enabled:           true,
	}

	attrs := `<saml:Attribute Name="email"><saml:AttributeValue>developer@oktacorp.com</saml:AttributeValue></saml:Attribute>
      <saml:Attribute Name="first_name"><saml:AttributeValue>Grace</saml:AttributeValue></saml:Attribute>
      <saml:Attribute Name="last_name"><saml:AttributeValue>Hopper</saml:AttributeValue></saml:Attribute>`

	b64Response := buildSignedSAMLResponse(t, priv, "_assert_1", provider.IdPEntityID, "https://hermes.autorix.io/saml/acs", provider.SPEntityID, samlStatusSuccess, "developer@oktacorp.com", attrs, now.Add(-time.Minute), now.Add(5*time.Minute))

	assertion, err := sp.ParseAssertion(b64Response, provider)
	if err != nil {
		t.Fatalf("ParseAssertion failed: %v", err)
	}

	if assertion.Subject != "developer@oktacorp.com" {
		t.Errorf("expected subject 'developer@oktacorp.com', got %s", assertion.Subject)
	}
	if assertion.FirstName != "Grace" || assertion.LastName != "Hopper" {
		t.Errorf("expected Grace Hopper, got %s %s", assertion.FirstName, assertion.LastName)
	}

	// Replay of the exact same signed assertion must fail
	if _, err := sp.ParseAssertion(b64Response, provider); !errors.Is(err, ErrAssertionReplay) {
		t.Fatalf("expected ErrAssertionReplay on replayed assertion, got %v", err)
	}
}

func TestServiceProvider_AttributeMapping(t *testing.T) {
	sp := NewServiceProvider("https://hermes.autorix.io")
	now := time.Now()
	priv, certPEM, err := generateTestKeyAndCertPEM("Okta SAML IdP", now.Add(-time.Hour), now.Add(24*time.Hour))
	if err != nil {
		t.Fatalf("generateTestKeyAndCertPEM: %v", err)
	}

	provider := &core.SAMLProvider{
		ID:                "okta-saml",
		IdPEntityID:       "https://idp.example.com",
		IdPSSOURL:         "https://idp.example.com/sso",
		IdPCertificatePEM: certPEM,
		SPEntityID:        "https://hermes.autorix.io/sp",
		Enabled:           true,
		AttributeMapping: map[string]string{
			"email":      "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress",
			"first_name": "urn:oid:2.5.4.42",
			"last_name":  "urn:oid:2.5.4.4",
			"role":       "custom_role",
		},
	}

	attrs := `<saml:Attribute Name="http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress"><saml:AttributeValue>ada.lovelace@example.com</saml:AttributeValue></saml:Attribute>
      <saml:Attribute Name="urn:oid:2.5.4.42"><saml:AttributeValue>Ada</saml:AttributeValue></saml:Attribute>
      <saml:Attribute Name="urn:oid:2.5.4.4"><saml:AttributeValue>Lovelace</saml:AttributeValue></saml:Attribute>
      <saml:Attribute Name="custom_role"><saml:AttributeValue>Software Architect</saml:AttributeValue></saml:Attribute>`

	b64Response := buildSignedSAMLResponse(t, priv, "_assert_map_1", provider.IdPEntityID, "https://hermes.autorix.io/saml/acs", provider.SPEntityID, samlStatusSuccess, "adalo@example.com", attrs, now.Add(-time.Minute), now.Add(5*time.Minute))

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

func TestServiceProvider_SecurityRejections(t *testing.T) {
	sp := NewServiceProvider("https://hermes.autorix.io")
	now := time.Now()
	priv, certPEM, err := generateTestKeyAndCertPEM("Trusted IdP", now.Add(-time.Hour), now.Add(24*time.Hour))
	if err != nil {
		t.Fatalf("generateTestKeyAndCertPEM: %v", err)
	}
	attackerPriv, _, err := generateTestKeyAndCertPEM("Attacker IdP", now.Add(-time.Hour), now.Add(24*time.Hour))
	if err != nil {
		t.Fatalf("generateTestKeyAndCertPEM attacker: %v", err)
	}

	provider := &core.SAMLProvider{
		ID:                "okta-sec",
		IdPEntityID:       "https://idp.trusted.com",
		IdPSSOURL:         "https://idp.trusted.com/sso",
		IdPCertificatePEM: certPEM,
		SPEntityID:        "https://hermes.autorix.io/sp",
		Enabled:           true,
	}

	t.Run("unsigned assertion rejected", func(t *testing.T) {
		unsigned := fmt.Sprintf(`<samlp:Response xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol" xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion" ID="_r1">
  <saml:Issuer>%s</saml:Issuer>
  <samlp:Status><samlp:StatusCode Value="%s"/></samlp:Status>
  <saml:Assertion ID="_a1"><saml:Issuer>%s</saml:Issuer><saml:Subject><saml:NameID>admin@corp.com</saml:NameID></saml:Subject></saml:Assertion>
</samlp:Response>`, provider.IdPEntityID, samlStatusSuccess, provider.IdPEntityID)
		b64 := base64.StdEncoding.EncodeToString([]byte(unsigned))
		if _, err := sp.ParseAssertion(b64, provider); !errors.Is(err, ErrSignatureInvalid) {
			t.Fatalf("expected ErrSignatureInvalid for unsigned assertion, got %v", err)
		}
	})

	t.Run("untrusted signer rejected", func(t *testing.T) {
		b64 := buildSignedSAMLResponse(t, attackerPriv, "_a_attacker", provider.IdPEntityID, "https://hermes.autorix.io/saml/acs", provider.SPEntityID, samlStatusSuccess, "admin@corp.com", "", now.Add(-time.Minute), now.Add(5*time.Minute))
		if _, err := sp.ParseAssertion(b64, provider); !errors.Is(err, ErrSignatureInvalid) {
			t.Fatalf("expected ErrSignatureInvalid for untrusted signer, got %v", err)
		}
	})

	t.Run("tampered assertion after signing rejected", func(t *testing.T) {
		b64 := buildSignedSAMLResponse(t, priv, "_a_tamper", provider.IdPEntityID, "https://hermes.autorix.io/saml/acs", provider.SPEntityID, samlStatusSuccess, "user@corp.com", "", now.Add(-time.Minute), now.Add(5*time.Minute))
		raw, _ := base64.StdEncoding.DecodeString(b64)
		tampered := strings.Replace(string(raw), "user@corp.com", "admin@corp.com", 1)
		tamperedB64 := base64.StdEncoding.EncodeToString([]byte(tampered))
		if _, err := sp.ParseAssertion(tamperedB64, provider); !errors.Is(err, ErrSignatureInvalid) {
			t.Fatalf("expected ErrSignatureInvalid for tampered assertion, got %v", err)
		}
	})

	t.Run("multiple assertions XSW rejected", func(t *testing.T) {
		b64 := buildSignedSAMLResponse(t, priv, "_a_xsw", provider.IdPEntityID, "https://hermes.autorix.io/saml/acs", provider.SPEntityID, samlStatusSuccess, "user@corp.com", "", now.Add(-time.Minute), now.Add(5*time.Minute))
		raw, _ := base64.StdEncoding.DecodeString(b64)
		injected := strings.Replace(string(raw), "</samlp:Response>", `<saml:Assertion xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion" ID="_evil"><saml:Subject><saml:NameID>root@corp.com</saml:NameID></saml:Subject></saml:Assertion></samlp:Response>`, 1)
		if _, err := sp.ParseAssertion(base64.StdEncoding.EncodeToString([]byte(injected)), provider); !errors.Is(err, ErrMultipleAssertions) {
			t.Fatalf("expected ErrMultipleAssertions for XSW double assertion, got %v", err)
		}
	})

	t.Run("non-success status rejected", func(t *testing.T) {
		b64 := buildSignedSAMLResponse(t, priv, "_a_status", provider.IdPEntityID, "https://hermes.autorix.io/saml/acs", provider.SPEntityID, "urn:oasis:names:tc:SAML:2.0:status:AuthnFailed", "user@corp.com", "", now.Add(-time.Minute), now.Add(5*time.Minute))
		if _, err := sp.ParseAssertion(b64, provider); !errors.Is(err, ErrInvalidStatus) {
			t.Fatalf("expected ErrInvalidStatus, got %v", err)
		}
	})

	t.Run("expired conditions rejected", func(t *testing.T) {
		b64 := buildSignedSAMLResponse(t, priv, "_a_exp", provider.IdPEntityID, "https://hermes.autorix.io/saml/acs", provider.SPEntityID, samlStatusSuccess, "user@corp.com", "", now.Add(-20*time.Minute), now.Add(-5*time.Minute))
		if _, err := sp.ParseAssertion(b64, provider); !errors.Is(err, ErrConditionsInvalid) {
			t.Fatalf("expected ErrConditionsInvalid for expired assertion, got %v", err)
		}
	})

	t.Run("wrong audience rejected", func(t *testing.T) {
		b64 := buildSignedSAMLResponse(t, priv, "_a_aud", provider.IdPEntityID, "https://hermes.autorix.io/saml/acs", "https://other-sp.example.com", samlStatusSuccess, "user@corp.com", "", now.Add(-time.Minute), now.Add(5*time.Minute))
		if _, err := sp.ParseAssertion(b64, provider); !errors.Is(err, ErrConditionsInvalid) {
			t.Fatalf("expected ErrConditionsInvalid for audience mismatch, got %v", err)
		}
	})

	t.Run("wrong destination rejected", func(t *testing.T) {
		b64 := buildSignedSAMLResponse(t, priv, "_a_dest", provider.IdPEntityID, "https://evil.example.com/acs", provider.SPEntityID, samlStatusSuccess, "user@corp.com", "", now.Add(-time.Minute), now.Add(5*time.Minute))
		if _, err := sp.ParseAssertion(b64, provider); !errors.Is(err, ErrConditionsInvalid) {
			t.Fatalf("expected ErrConditionsInvalid for destination mismatch, got %v", err)
		}
	})

	t.Run("XXE DOCTYPE rejected", func(t *testing.T) {
		xxe := `<?xml version="1.0"?><!DOCTYPE foo [<!ENTITY xxe SYSTEM "file:///etc/passwd">]><Response/>`
		if _, err := sp.ParseAssertion(base64.StdEncoding.EncodeToString([]byte(xxe)), provider); err == nil {
			t.Fatalf("expected error for XML with DOCTYPE directive")
		}
	})
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
	if _, _, _, err := ValidateCertificatesPEM(expiredCertPEM); !errors.Is(err, ErrInvalidCertificate) {
		t.Errorf("expected ValidateCertificatesPEM to reject expired cert, got %v", err)
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
