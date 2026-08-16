package saml

import (
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/autorix/hermes/internal/core"
	"github.com/google/uuid"
)

type ServiceProvider struct {
	BaseURL string
}

func NewServiceProvider(baseURL string) *ServiceProvider {
	return &ServiceProvider{BaseURL: baseURL}
}

// GenerateSPMetadataXML returns standard SAML 2.0 Service Provider Metadata XML
func (sp *ServiceProvider) GenerateSPMetadataXML(entityID string) string {
	acsURL := fmt.Sprintf("%s/saml/acs", sp.BaseURL)
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<md:EntityDescriptor xmlns:md="urn:oasis:names:tc:SAML:2.0:metadata" entityID="%s">
  <md:SPSSODescriptor AuthnRequestsSigned="false" WantAssertionsSigned="true" protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol">
    <md:NameIDFormat>urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress</md:NameIDFormat>
    <md:AssertionConsumerService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST" Location="%s" index="1" isDefault="true"/>
  </md:SPSSODescriptor>
</md:EntityDescriptor>`, entityID, acsURL)
}

// GenerateAuthnRequestURL creates the SAML 2.0 AuthnRequest and returns the IdP redirect URL
func (sp *ServiceProvider) GenerateAuthnRequestURL(p *core.SAMLProvider) (string, error) {
	reqID := fmt.Sprintf("ARX_%s", uuid.New().String())
	issueInstant := time.Now().UTC().Format(time.RFC3339)
	acsURL := fmt.Sprintf("%s/saml/acs", sp.BaseURL)

	authnXML := fmt.Sprintf(`<samlp:AuthnRequest xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol" xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion" ID="%s" Version="2.0" IssueInstant="%s" Destination="%s" AssertionConsumerServiceURL="%s" ProtocolBinding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST">
  <saml:Issuer>%s</saml:Issuer>
  <samlp:NameIDPolicy Format="urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress" AllowCreate="true"/>
</samlp:AuthnRequest>`, reqID, issueInstant, p.IdPSSOURL, acsURL, p.SPEntityID)

	b64Request := base64.StdEncoding.EncodeToString([]byte(authnXML))

	u, err := url.Parse(p.IdPSSOURL)
	if err != nil {
		return "", fmt.Errorf("invalid IdP SSO URL: %w", err)
	}

	q := u.Query()
	q.Set("SAMLRequest", b64Request)
	u.RawQuery = q.Encode()

	return u.String(), nil
}

// SAML Response XML Structures for parsing
type samlResponseXML struct {
	XMLName   xml.Name        `xml:"Response"`
	Assertion samlAssertionXML `xml:"Assertion"`
}

type samlAssertionXML struct {
	Subject struct {
		NameID string `xml:"NameID"`
	} `xml:"Subject"`
	AttributeStatement struct {
		Attributes []struct {
			Name       string   `xml:"Name,attr"`
			AttributeValues []string `xml:"AttributeValue"`
		} `xml:"Attribute"`
	} `xml:"AttributeStatement"`
}

// ParseAssertion extracts normalized claims from a Base64-encoded SAMLResponse
func (sp *ServiceProvider) ParseAssertion(rawB64 string, p *core.SAMLProvider) (*core.SAMLAssertion, error) {
	decoded, err := base64.StdEncoding.DecodeString(rawB64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64 SAMLResponse: %w", err)
	}

	var respXML samlResponseXML
	if err := xml.Unmarshal(decoded, &respXML); err != nil {
		return nil, fmt.Errorf("failed to unmarshal SAML XML: %w", err)
	}

	nameID := strings.TrimSpace(respXML.Assertion.Subject.NameID)
	if nameID == "" {
		return nil, errors.New("SAML Response missing Subject NameID")
	}

	attrMap := make(map[string]string)
	for _, attr := range respXML.Assertion.AttributeStatement.Attributes {
		if len(attr.AttributeValues) > 0 {
			attrMap[attr.Name] = attr.AttributeValues[0]
		}
	}

	// Extract standard attributes (email, first_name, last_name)
	email := nameID
	if em, ok := attrMap["email"]; ok {
		email = em
	} else if em, ok := attrMap["http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress"]; ok {
		email = em
	}

	firstName := attrMap["first_name"]
	if firstName == "" {
		firstName = attrMap["http://schemas.xmlsoap.org/ws/2005/05/identity/claims/givenname"]
	}

	lastName := attrMap["last_name"]
	if lastName == "" {
		lastName = attrMap["http://schemas.xmlsoap.org/ws/2005/05/identity/claims/surname"]
	}

	return &core.SAMLAssertion{
		Subject:    nameID,
		Email:      email,
		FirstName:  firstName,
		LastName:   lastName,
		Attributes: attrMap,
	}, nil
}
