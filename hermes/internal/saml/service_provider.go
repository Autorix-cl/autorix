package saml

import (
	"bytes"
	"compress/flate"
	"crypto"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/subtle"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"encoding/xml"
	"errors"
	"fmt"
	"hash"
	"io"
	"math/big"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/autorix/hermes/internal/core"
	"github.com/google/uuid"
)

const (
	samlStatusSuccess   = "urn:oasis:names:tc:SAML:2.0:status:Success"
	samlProtocolNS      = "urn:oasis:names:tc:SAML:2.0:protocol"
	samlAssertionNS     = "urn:oasis:names:tc:SAML:2.0:assertion"
	xmldsigNS           = "http://www.w3.org/2000/09/xmldsig#"
	excC14NAlgorithm    = "http://www.w3.org/2001/10/xml-exc-c14n#"
	excC14NWithComments = "http://www.w3.org/2001/10/xml-exc-c14n#WithComments"
	c14NAlgorithm       = "http://www.w3.org/TR/2001/REC-xml-c14n-20010315"
	envelopedSigAlgo    = "http://www.w3.org/2000/09/xmldsig#enveloped-signature"
	sigAlgRSASHA256     = "http://www.w3.org/2001/04/xmldsig-more#rsa-sha256"
	sigAlgRSASHA384     = "http://www.w3.org/2001/04/xmldsig-more#rsa-sha384"
	sigAlgRSASHA512     = "http://www.w3.org/2001/04/xmldsig-more#rsa-sha512"
	sigAlgECDSASHA256   = "http://www.w3.org/2001/04/xmldsig-more#ecdsa-sha256"
	sigAlgECDSASHA384   = "http://www.w3.org/2001/04/xmldsig-more#ecdsa-sha384"
	sigAlgECDSASHA512   = "http://www.w3.org/2001/04/xmldsig-more#ecdsa-sha512"
	digestAlgSHA256     = "http://www.w3.org/2001/04/xmlenc#sha256"
	digestAlgSHA384     = "http://www.w3.org/2001/04/xmldsig-more#sha384"
	digestAlgSHA512     = "http://www.w3.org/2001/04/xmlenc#sha512"
	assertionClockSkew  = 2 * time.Minute
	defaultReplayTTL    = 10 * time.Minute
	maxReplayTTL        = 24 * time.Hour
	maxReplayCacheSize  = 10000
	maxSAMLDecodedBytes = 2 << 20 // 2 MiB
	maxXMLDepth         = 64
	maxXMLElements      = 10000
)

var (
	ErrSignatureInvalid   = errors.New("SAML signature verification failed")
	ErrAssertionReplay    = errors.New("SAML assertion replay detected")
	ErrInvalidSSOURL      = errors.New("invalid SAML IdP SSO URL")
	ErrInvalidCertificate = errors.New("invalid or expired SAML IdP certificate")
	ErrProviderDisabled   = errors.New("SAML provider is disabled")
	ErrIssuerMismatch     = errors.New("SAML issuer mismatch")
	ErrInvalidStatus      = errors.New("SAML response status is not Success")
	ErrMultipleAssertions = errors.New("SAML response must contain exactly one assertion")
	ErrConditionsInvalid  = errors.New("SAML assertion conditions violated")
)

type ServiceProvider struct {
	BaseURL     string
	replayMu    sync.Mutex
	replayCache map[string]time.Time
}

func NewServiceProvider(baseURL string) *ServiceProvider {
	return &ServiceProvider{
		BaseURL:     strings.TrimRight(baseURL, "/"),
		replayCache: make(map[string]time.Time),
	}
}

// ValidateSSOURL ensures the IdP SSO URL uses HTTPS with a valid host and
// rejects unsafe schemes such as javascript:, data:, or plain http:.
func ValidateSSOURL(rawURL string) error {
	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" {
		return fmt.Errorf("%w: URL is empty", ErrInvalidSSOURL)
	}
	u, err := url.Parse(trimmed)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidSSOURL, err)
	}
	if !strings.EqualFold(u.Scheme, "https") {
		return fmt.Errorf("%w: scheme must be https", ErrInvalidSSOURL)
	}
	if u.Host == "" || u.Hostname() == "" {
		return fmt.Errorf("%w: missing host", ErrInvalidSSOURL)
	}
	if u.User != nil {
		return fmt.Errorf("%w: embedded credentials are not allowed", ErrInvalidSSOURL)
	}
	return nil
}

// ValidateCertificatesPEM parses all X.509 certificates in pemStr and rejects
// empty, malformed, or expired certificates.
func ValidateCertificatesPEM(pemStr string) ([]core.CertificateInfo, *time.Time, []string, error) {
	certs, expiresAt, warnings, err := ParseCertificatesPEM(pemStr)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("%w: %v", ErrInvalidCertificate, err)
	}
	for _, c := range certs {
		if c.Expired {
			return nil, nil, nil, fmt.Errorf("%w: certificate expired on %s", ErrInvalidCertificate, c.NotAfter.Format(time.RFC3339))
		}
	}
	return certs, expiresAt, warnings, nil
}

type spEntityDescriptorXML struct {
	XMLName         xml.Name           `xml:"md:EntityDescriptor"`
	XmlnsMD         string             `xml:"xmlns:md,attr"`
	EntityID        string             `xml:"entityID,attr"`
	SPSSODescriptor spSSODescriptorXML `xml:"md:SPSSODescriptor"`
}

type spSSODescriptorXML struct {
	AuthnRequestsSigned        bool                      `xml:"AuthnRequestsSigned,attr"`
	WantAssertionsSigned       bool                      `xml:"WantAssertionsSigned,attr"`
	ProtocolSupportEnumeration string                    `xml:"protocolSupportEnumeration,attr"`
	NameIDFormat               string                    `xml:"md:NameIDFormat"`
	AssertionConsumerService   spAssertionConsumerSvcXML `xml:"md:AssertionConsumerService"`
}

type spAssertionConsumerSvcXML struct {
	Binding   string `xml:"Binding,attr"`
	Location  string `xml:"Location,attr"`
	Index     int    `xml:"index,attr"`
	IsDefault bool   `xml:"isDefault,attr"`
}

// GenerateSPMetadataXML returns standard SAML 2.0 Service Provider Metadata XML
// serialized safely via encoding/xml to prevent XML injection.
func (sp *ServiceProvider) GenerateSPMetadataXML(entityID string) string {
	acsURL := fmt.Sprintf("%s/saml/acs", strings.TrimRight(sp.BaseURL, "/"))
	doc := spEntityDescriptorXML{
		XmlnsMD:  "urn:oasis:names:tc:SAML:2.0:metadata",
		EntityID: entityID,
		SPSSODescriptor: spSSODescriptorXML{
			AuthnRequestsSigned:        false,
			WantAssertionsSigned:       true,
			ProtocolSupportEnumeration: samlProtocolNS,
			NameIDFormat:               "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress",
			AssertionConsumerService: spAssertionConsumerSvcXML{
				Binding:   "urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST",
				Location:  acsURL,
				Index:     1,
				IsDefault: true,
			},
		},
	}

	out, err := xml.MarshalIndent(doc, "", "  ")
	if err != nil {
		return ""
	}
	return xml.Header + string(out)
}

type samlAuthnRequestXML struct {
	XMLName                     xml.Name            `xml:"samlp:AuthnRequest"`
	XmlnsSAMLP                  string              `xml:"xmlns:samlp,attr"`
	XmlnsSAML                   string              `xml:"xmlns:saml,attr"`
	ID                          string              `xml:"ID,attr"`
	Version                     string              `xml:"Version,attr"`
	IssueInstant                string              `xml:"IssueInstant,attr"`
	Destination                 string              `xml:"Destination,attr"`
	AssertionConsumerServiceURL string              `xml:"AssertionConsumerServiceURL,attr"`
	ProtocolBinding             string              `xml:"ProtocolBinding,attr"`
	Issuer                      string              `xml:"saml:Issuer"`
	NameIDPolicy                samlNameIDPolicyXML `xml:"samlp:NameIDPolicy"`
}

type samlNameIDPolicyXML struct {
	Format      string `xml:"Format,attr"`
	AllowCreate bool   `xml:"AllowCreate,attr"`
}

// GenerateAuthnRequestURL creates a SAML 2.0 AuthnRequest using safe XML
// marshaling and RFC 1951 DEFLATE compression for the HTTP-Redirect binding.
func (sp *ServiceProvider) GenerateAuthnRequestURL(p *core.SAMLProvider) (string, error) {
	if p == nil {
		return "", errors.New("SAML provider is required")
	}
	if err := ValidateSSOURL(p.IdPSSOURL); err != nil {
		return "", err
	}

	reqID := fmt.Sprintf("ARX_%s", uuid.New().String())
	issueInstant := time.Now().UTC().Format(time.RFC3339)
	acsURL := fmt.Sprintf("%s/saml/acs", strings.TrimRight(sp.BaseURL, "/"))

	authn := samlAuthnRequestXML{
		XmlnsSAMLP:                  samlProtocolNS,
		XmlnsSAML:                   samlAssertionNS,
		ID:                          reqID,
		Version:                     "2.0",
		IssueInstant:                issueInstant,
		Destination:                 p.IdPSSOURL,
		AssertionConsumerServiceURL: acsURL,
		ProtocolBinding:             "urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST",
		Issuer:                      p.SPEntityID,
		NameIDPolicy: samlNameIDPolicyXML{
			Format:      "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress",
			AllowCreate: true,
		},
	}

	xmlBytes, err := xml.Marshal(authn)
	if err != nil {
		return "", fmt.Errorf("failed to marshal AuthnRequest XML: %w", err)
	}

	var compressed bytes.Buffer
	fw, err := flate.NewWriter(&compressed, flate.DefaultCompression)
	if err != nil {
		return "", fmt.Errorf("failed to create flate writer: %w", err)
	}
	if _, err := fw.Write(xmlBytes); err != nil {
		_ = fw.Close()
		return "", fmt.Errorf("failed to deflate AuthnRequest: %w", err)
	}
	if err := fw.Close(); err != nil {
		return "", fmt.Errorf("failed to finalize deflated AuthnRequest: %w", err)
	}

	b64Request := base64.StdEncoding.EncodeToString(compressed.Bytes())

	u, err := url.Parse(p.IdPSSOURL)
	if err != nil {
		return "", fmt.Errorf("invalid IdP SSO URL: %w", err)
	}

	q := u.Query()
	q.Set("SAMLRequest", b64Request)
	u.RawQuery = q.Encode()

	return u.String(), nil
}

// ExtractIssuer decodes a base64 SAMLResponse and returns the untrusted Issuer
// string so the caller can look up the configured SAMLProvider prior to
// cryptographic verification.
func (sp *ServiceProvider) ExtractIssuer(rawB64 string) (string, error) {
	decoded, err := decodeSAMLBase64(rawB64)
	if err != nil {
		return "", err
	}

	root, err := parseStrictXML(decoded)
	if err != nil {
		return "", fmt.Errorf("failed to parse SAML XML: %w", err)
	}
	if root.local != "Response" {
		return "", errors.New("SAML document root must be Response")
	}

	if issuerEl := root.firstChild("Issuer", samlAssertionNS); issuerEl != nil {
		if issuer := strings.TrimSpace(issuerEl.textContent()); issuer != "" {
			return issuer, nil
		}
	}
	if assertionEl := root.firstChild("Assertion", samlAssertionNS); assertionEl != nil {
		if issuerEl := assertionEl.firstChild("Issuer", samlAssertionNS); issuerEl != nil {
			if issuer := strings.TrimSpace(issuerEl.textContent()); issuer != "" {
				return issuer, nil
			}
		}
	}
	return "", errors.New("SAML Response missing Issuer")
}

// ParseAssertion validates cryptographic XML signatures, SAML protocol status,
// single-assertion structure, issuer, conditions, destination, audience, and
// assertion replay protection before extracting normalized claims.
func (sp *ServiceProvider) ParseAssertion(rawB64 string, p *core.SAMLProvider) (*core.SAMLAssertion, error) {
	if p == nil {
		return nil, fmt.Errorf("%w: SAML provider is required for verification", ErrSignatureInvalid)
	}
	if !p.Enabled {
		return nil, ErrProviderDisabled
	}

	trustedCerts, err := parseActiveX509Certificates(p.IdPCertificatePEM)
	if err != nil {
		return nil, err
	}

	decoded, err := decodeSAMLBase64(rawB64)
	if err != nil {
		return nil, err
	}

	root, err := parseStrictXML(decoded)
	if err != nil {
		return nil, fmt.Errorf("failed to parse SAML XML: %w", err)
	}
	if root.local != "Response" || (root.nsURI != "" && root.nsURI != samlProtocolNS) {
		return nil, errors.New("SAML document root must be Response")
	}

	// 1. Validate Response Destination if present
	expectedACS := strings.TrimRight(sp.BaseURL, "/") + "/saml/acs"
	if dest := strings.TrimSpace(root.attr("Destination")); dest != "" && dest != expectedACS {
		return nil, fmt.Errorf("%w: response destination %q does not match expected ACS %q", ErrConditionsInvalid, dest, expectedACS)
	}

	// 2. Validate SAML StatusCode == Success
	statusEl := root.firstChild("Status", samlProtocolNS)
	if statusEl == nil {
		return nil, fmt.Errorf("%w: missing Status element", ErrInvalidStatus)
	}
	statusCodeEl := statusEl.firstChild("StatusCode", samlProtocolNS)
	if statusCodeEl == nil || strings.TrimSpace(statusCodeEl.attr("Value")) != samlStatusSuccess {
		got := ""
		if statusCodeEl != nil {
			got = statusCodeEl.attr("Value")
		}
		return nil, fmt.Errorf("%w: got %q", ErrInvalidStatus, got)
	}

	// 3. Enforce single-assertion strictness across the ENTIRE document (prevents XSW)
	allAssertions := root.findAll("Assertion", samlAssertionNS)
	if len(allAssertions) != 1 {
		return nil, fmt.Errorf("%w: found %d assertions", ErrMultipleAssertions, len(allAssertions))
	}
	assertionEl := root.firstChild("Assertion", samlAssertionNS)
	if assertionEl == nil || assertionEl != allAssertions[0] {
		return nil, fmt.Errorf("%w: assertion must be a direct child of Response", ErrMultipleAssertions)
	}

	assertionID := strings.TrimSpace(assertionEl.attr("ID"))
	if assertionID == "" {
		return nil, fmt.Errorf("%w: assertion missing required ID attribute", ErrSignatureInvalid)
	}

	// 4. Validate Issuer against configured provider
	respIssuer := ""
	if issuerEl := root.firstChild("Issuer", samlAssertionNS); issuerEl != nil {
		respIssuer = strings.TrimSpace(issuerEl.textContent())
	}
	assertIssuer := ""
	if issuerEl := assertionEl.firstChild("Issuer", samlAssertionNS); issuerEl != nil {
		assertIssuer = strings.TrimSpace(issuerEl.textContent())
	}
	if p.IdPEntityID != "" {
		if respIssuer == "" && assertIssuer == "" {
			return nil, fmt.Errorf("%w: missing Issuer element", ErrIssuerMismatch)
		}
		if respIssuer != "" && respIssuer != p.IdPEntityID {
			return nil, fmt.Errorf("%w: response issuer %q != expected %q", ErrIssuerMismatch, respIssuer, p.IdPEntityID)
		}
		if assertIssuer != "" && assertIssuer != p.IdPEntityID {
			return nil, fmt.Errorf("%w: assertion issuer %q != expected %q", ErrIssuerMismatch, assertIssuer, p.IdPEntityID)
		}
	}

	// 5. Cryptographically verify XML <ds:Signature> on Assertion and/or Response
	respSigs := root.childrenByName("Signature", xmldsigNS)
	assertSigs := assertionEl.childrenByName("Signature", xmldsigNS)
	if len(respSigs) > 1 || len(assertSigs) > 1 {
		return nil, fmt.Errorf("%w: multiple signatures on a single element are not allowed", ErrSignatureInvalid)
	}
	if len(respSigs) == 0 && len(assertSigs) == 0 {
		return nil, fmt.Errorf("%w: unsigned SAML response/assertion rejected", ErrSignatureInvalid)
	}
	for _, sigEl := range respSigs {
		if err := verifyEnvelopedSignature(root, sigEl, trustedCerts); err != nil {
			return nil, fmt.Errorf("%w: response signature: %v", ErrSignatureInvalid, err)
		}
	}
	for _, sigEl := range assertSigs {
		if err := verifyEnvelopedSignature(assertionEl, sigEl, trustedCerts); err != nil {
			return nil, fmt.Errorf("%w: assertion signature: %v", ErrSignatureInvalid, err)
		}
	}

	// 6. Validate Conditions (NotBefore, NotOnOrAfter with 2-min skew, Audience)
	now := time.Now().UTC()
	replayExpiry := now.Add(defaultReplayTTL)
	if condEl := assertionEl.firstChild("Conditions", samlAssertionNS); condEl != nil {
		if nbStr := strings.TrimSpace(condEl.attr("NotBefore")); nbStr != "" {
			nb, err := parseSAMLTime(nbStr)
			if err != nil {
				return nil, fmt.Errorf("%w: invalid NotBefore: %v", ErrConditionsInvalid, err)
			}
			if now.Add(assertionClockSkew).Before(nb) {
				return nil, fmt.Errorf("%w: assertion not yet valid (NotBefore %s)", ErrConditionsInvalid, nbStr)
			}
		}
		if nooaStr := strings.TrimSpace(condEl.attr("NotOnOrAfter")); nooaStr != "" {
			nooa, err := parseSAMLTime(nooaStr)
			if err != nil {
				return nil, fmt.Errorf("%w: invalid NotOnOrAfter: %v", ErrConditionsInvalid, err)
			}
			if !now.Add(-assertionClockSkew).Before(nooa) {
				return nil, fmt.Errorf("%w: assertion expired (NotOnOrAfter %s)", ErrConditionsInvalid, nooaStr)
			}
			candidate := nooa.Add(assertionClockSkew)
			if candidate.After(replayExpiry) {
				replayExpiry = candidate
			}
			if maxExp := now.Add(maxReplayTTL); replayExpiry.After(maxExp) {
				replayExpiry = maxExp
			}
		}
		if p.SPEntityID != "" {
			restrictions := condEl.childrenByName("AudienceRestriction", samlAssertionNS)
			for _, ar := range restrictions {
				audiences := ar.childrenByName("Audience", samlAssertionNS)
				if len(audiences) == 0 {
					continue
				}
				matched := false
				for _, aud := range audiences {
					if strings.TrimSpace(aud.textContent()) == p.SPEntityID {
						matched = true
						break
					}
				}
				if !matched {
					return nil, fmt.Errorf("%w: SP entity ID %q not in AudienceRestriction", ErrConditionsInvalid, p.SPEntityID)
				}
			}
		}
	}

	// Validate SubjectConfirmationData Recipient & NotOnOrAfter when present
	subjectEl := assertionEl.firstChild("Subject", samlAssertionNS)
	if subjectEl == nil {
		return nil, errors.New("SAML Response missing Subject")
	}
	for _, sc := range subjectEl.childrenByName("SubjectConfirmation", samlAssertionNS) {
		if scd := sc.firstChild("SubjectConfirmationData", samlAssertionNS); scd != nil {
			if recipient := strings.TrimSpace(scd.attr("Recipient")); recipient != "" && recipient != expectedACS {
				return nil, fmt.Errorf("%w: recipient %q does not match expected ACS %q", ErrConditionsInvalid, recipient, expectedACS)
			}
			if nooaStr := strings.TrimSpace(scd.attr("NotOnOrAfter")); nooaStr != "" {
				nooa, err := parseSAMLTime(nooaStr)
				if err != nil {
					return nil, fmt.Errorf("%w: invalid SubjectConfirmationData NotOnOrAfter: %v", ErrConditionsInvalid, err)
				}
				if !now.Add(-assertionClockSkew).Before(nooa) {
					return nil, fmt.Errorf("%w: subject confirmation expired (NotOnOrAfter %s)", ErrConditionsInvalid, nooaStr)
				}
			}
		}
	}

	// 7. Replay protection: record assertion ID only after signature & conditions pass
	if err := sp.checkAndRecordAssertionID(assertionID, now, replayExpiry); err != nil {
		return nil, err
	}

	// 8. Extract Subject NameID and Attributes from the verified assertion node
	nameIDEl := subjectEl.firstChild("NameID", samlAssertionNS)
	nameID := ""
	if nameIDEl != nil {
		nameID = strings.TrimSpace(nameIDEl.textContent())
	}
	if nameID == "" {
		return nil, errors.New("SAML Response missing Subject NameID")
	}

	attrMap := make(map[string]string)
	for _, stmt := range assertionEl.childrenByName("AttributeStatement", samlAssertionNS) {
		for _, attr := range stmt.childrenByName("Attribute", samlAssertionNS) {
			attrName := attr.attr("Name")
			vals := attr.childrenByName("AttributeValue", samlAssertionNS)
			if attrName != "" && len(vals) > 0 {
				attrMap[attrName] = vals[0].textContent()
			}
		}
	}

	// Extract standard default attributes (email, first_name, last_name)
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

	// Apply custom AttributeMapping rules if provider has mapping defined
	traits := make(map[string]interface{})
	if len(p.AttributeMapping) > 0 {
		for traitKey, samlAttr := range p.AttributeMapping {
			if val, ok := attrMap[samlAttr]; ok {
				traits[traitKey] = val
				switch traitKey {
				case "email":
					email = val
				case "first_name", "given_name":
					firstName = val
				case "last_name", "surname", "family_name":
					lastName = val
				case "subject", "name_id":
					nameID = val
				}
			}
		}
	}

	return &core.SAMLAssertion{
		Subject:    nameID,
		Email:      email,
		FirstName:  firstName,
		LastName:   lastName,
		Traits:     traits,
		Attributes: attrMap,
	}, nil
}

func (sp *ServiceProvider) checkAndRecordAssertionID(id string, now, expiresAt time.Time) error {
	sp.replayMu.Lock()
	defer sp.replayMu.Unlock()

	if sp.replayCache == nil {
		sp.replayCache = make(map[string]time.Time)
	}

	// Prune expired entries opportunistically
	if len(sp.replayCache) >= maxReplayCacheSize {
		for k, exp := range sp.replayCache {
			if !now.Before(exp) {
				delete(sp.replayCache, k)
			}
		}
		// If still at capacity, evict the earliest-expiring entry
		if len(sp.replayCache) >= maxReplayCacheSize {
			var oldestKey string
			var oldestExp time.Time
			first := true
			for k, exp := range sp.replayCache {
				if first || exp.Before(oldestExp) {
					oldestKey = k
					oldestExp = exp
					first = false
				}
			}
			if !first {
				delete(sp.replayCache, oldestKey)
			}
		}
	}

	if exp, exists := sp.replayCache[id]; exists && now.Before(exp) {
		return fmt.Errorf("%w: assertion ID %q already consumed", ErrAssertionReplay, id)
	}
	sp.replayCache[id] = expiresAt
	return nil
}

// ParseCertificatesPEM decodes all PEM blocks, parses x509 certificates, computes expiration details and warnings.
// Multiple certificates are supported for key rollover.
func ParseCertificatesPEM(pemStr string) ([]core.CertificateInfo, *time.Time, []string, error) {
	clean := strings.TrimSpace(pemStr)
	if clean == "" {
		return nil, nil, nil, errors.New("empty certificate PEM")
	}

	var certs []core.CertificateInfo
	var warnings []string
	var latestExpiry *time.Time

	rest := []byte(clean)
	now := time.Now()

	for {
		var block *pem.Block
		block, rest = pem.Decode(rest)
		if block == nil {
			break
		}
		if block.Type != "CERTIFICATE" {
			continue
		}

		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to parse x509 certificate: %w", err)
		}

		expired := now.After(cert.NotAfter)
		daysUntilExpiry := int(cert.NotAfter.Sub(now).Hours() / 24)
		expiringSoon := !expired && daysUntilExpiry <= 30

		var warning string
		if expired {
			warning = fmt.Sprintf("Certificate '%s' expired on %s", cert.Subject.CommonName, cert.NotAfter.Format(time.RFC3339))
			warnings = append(warnings, warning)
		} else if expiringSoon {
			warning = fmt.Sprintf("Certificate '%s' will expire in %d days (%s)", cert.Subject.CommonName, daysUntilExpiry, cert.NotAfter.Format(time.RFC3339))
			warnings = append(warnings, warning)
		}

		info := core.CertificateInfo{
			Subject:         cert.Subject.String(),
			Issuer:          cert.Issuer.String(),
			SerialNumber:    cert.SerialNumber.String(),
			NotBefore:       cert.NotBefore,
			NotAfter:        cert.NotAfter,
			Expired:         expired,
			ExpiringSoon:    expiringSoon,
			DaysUntilExpiry: daysUntilExpiry,
			Warning:         warning,
		}
		certs = append(certs, info)

		if latestExpiry == nil || cert.NotAfter.After(*latestExpiry) {
			t := cert.NotAfter
			latestExpiry = &t
		}
	}

	if len(certs) == 0 {
		return nil, nil, nil, errors.New("no valid CERTIFICATE PEM blocks found")
	}

	return certs, latestExpiry, warnings, nil
}

func parseActiveX509Certificates(pemStr string) ([]*x509.Certificate, error) {
	clean := strings.TrimSpace(pemStr)
	if clean == "" {
		return nil, fmt.Errorf("%w: empty certificate PEM", ErrInvalidCertificate)
	}
	now := time.Now()
	var active []*x509.Certificate
	var total int
	rest := []byte(clean)
	for {
		var block *pem.Block
		block, rest = pem.Decode(rest)
		if block == nil {
			break
		}
		if block.Type != "CERTIFICATE" {
			continue
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidCertificate, err)
		}
		total++
		if !now.After(cert.NotAfter) && !now.Before(cert.NotBefore.Add(-assertionClockSkew)) {
			active = append(active, cert)
		}
	}
	if total == 0 {
		return nil, fmt.Errorf("%w: no CERTIFICATE PEM blocks found", ErrInvalidCertificate)
	}
	if len(active) == 0 {
		return nil, fmt.Errorf("%w: all configured IdP certificates are expired or not yet valid", ErrInvalidCertificate)
	}
	return active, nil
}

func decodeSAMLBase64(rawB64 string) ([]byte, error) {
	trimmed := strings.TrimSpace(rawB64)
	if len(trimmed) > base64.StdEncoding.EncodedLen(maxSAMLDecodedBytes) {
		return nil, errors.New("SAMLResponse payload exceeds maximum allowed size")
	}
	decoded, err := base64.StdEncoding.DecodeString(trimmed)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64 SAMLResponse: %w", err)
	}
	if len(decoded) > maxSAMLDecodedBytes {
		return nil, errors.New("SAMLResponse XML exceeds maximum allowed size")
	}
	return decoded, nil
}

func parseSAMLTime(v string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339Nano, v); err == nil {
		return t.UTC(), nil
	}
	t, err := time.Parse(time.RFC3339, v)
	if err != nil {
		return time.Time{}, err
	}
	return t.UTC(), nil
}

// --- Strict XML DOM & Exclusive C14N / XMLDSig Verification ---

type xmlAttr struct {
	prefix string
	local  string
	nsURI  string
	value  string
}

type xmlNode struct {
	isText   bool
	text     string
	prefix   string
	local    string
	nsURI    string
	nsScope  map[string]string // prefix -> URI in scope on this element
	attrs    []xmlAttr         // non-xmlns attributes only
	children []*xmlNode
}

func (n *xmlNode) attr(local string) string {
	for _, a := range n.attrs {
		if a.prefix == "" && a.local == local {
			return a.value
		}
	}
	return ""
}

func (n *xmlNode) firstChild(local, expectedNS string) *xmlNode {
	for _, c := range n.children {
		if !c.isText && c.local == local && (c.nsURI == expectedNS || c.nsURI == "") {
			return c
		}
	}
	return nil
}

func (n *xmlNode) childrenByName(local, expectedNS string) []*xmlNode {
	var out []*xmlNode
	for _, c := range n.children {
		if !c.isText && c.local == local && (c.nsURI == expectedNS || c.nsURI == "") {
			out = append(out, c)
		}
	}
	return out
}

func (n *xmlNode) findAll(local, expectedNS string) []*xmlNode {
	var out []*xmlNode
	var walk func(*xmlNode)
	walk = func(cur *xmlNode) {
		if cur.isText {
			return
		}
		if cur.local == local && (cur.nsURI == expectedNS || cur.nsURI == "") {
			out = append(out, cur)
		}
		for _, ch := range cur.children {
			walk(ch)
		}
	}
	walk(n)
	return out
}

func (n *xmlNode) textContent() string {
	var b strings.Builder
	for _, c := range n.children {
		if c.isText {
			b.WriteString(c.text)
		}
	}
	return b.String()
}

func parseStrictXML(data []byte) (*xmlNode, error) {
	dec := xml.NewDecoder(bytes.NewReader(data))
	dec.Strict = true

	var stack []*xmlNode
	var root *xmlNode
	elemCount := 0

	for {
		tok, err := dec.RawToken()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}

		switch t := tok.(type) {
		case xml.Directive:
			// Reject DOCTYPE / ENTITY directives to prevent XXE and Billion Laughs
			return nil, errors.New("XML directives (DOCTYPE/ENTITY) are forbidden")
		case xml.StartElement:
			elemCount++
			if elemCount > maxXMLElements {
				return nil, errors.New("XML document exceeds maximum element count")
			}
			if len(stack) >= maxXMLDepth {
				return nil, errors.New("XML document exceeds maximum nesting depth")
			}

			scope := make(map[string]string)
			if len(stack) > 0 {
				for k, v := range stack[len(stack)-1].nsScope {
					scope[k] = v
				}
			}

			// First pass: collect namespace declarations
			for _, a := range t.Attr {
				if a.Name.Space == "" && a.Name.Local == "xmlns" {
					scope[""] = a.Value
				} else if a.Name.Space == "xmlns" {
					scope[a.Name.Local] = a.Value
				}
			}

			// Second pass: resolve non-xmlns attributes
			var attrs []xmlAttr
			for _, a := range t.Attr {
				if (a.Name.Space == "" && a.Name.Local == "xmlns") || a.Name.Space == "xmlns" {
					continue
				}
				attrNS := ""
				if a.Name.Space != "" {
					if a.Name.Space == "xml" {
						attrNS = "http://www.w3.org/XML/1998/namespace"
					} else {
						attrNS = scope[a.Name.Space]
					}
				}
				attrs = append(attrs, xmlAttr{
					prefix: a.Name.Space,
					local:  a.Name.Local,
					nsURI:  attrNS,
					value:  a.Value,
				})
			}

			node := &xmlNode{
				prefix:  t.Name.Space,
				local:   t.Name.Local,
				nsURI:   scope[t.Name.Space],
				nsScope: scope,
				attrs:   attrs,
			}

			if len(stack) == 0 {
				if root != nil {
					return nil, errors.New("XML document contains multiple root elements")
				}
				root = node
			} else {
				parent := stack[len(stack)-1]
				parent.children = append(parent.children, node)
			}
			stack = append(stack, node)

		case xml.EndElement:
			if len(stack) == 0 {
				return nil, errors.New("unexpected closing XML tag")
			}
			top := stack[len(stack)-1]
			if top.prefix != t.Name.Space || top.local != t.Name.Local {
				return nil, fmt.Errorf("mismatched XML closing tag: expected %s:%s, got %s:%s", top.prefix, top.local, t.Name.Space, t.Name.Local)
			}
			stack = stack[:len(stack)-1]

		case xml.CharData:
			if len(stack) > 0 {
				parent := stack[len(stack)-1]
				parent.children = append(parent.children, &xmlNode{
					isText: true,
					text:   string(t),
				})
			}
		case xml.Comment, xml.ProcInst:
			// Comments and processing instructions are omitted per Exclusive C14N
		}
	}

	if len(stack) != 0 || root == nil {
		return nil, errors.New("incomplete XML document")
	}
	return root, nil
}

// canonicalizeExcC14N serializes node according to RFC 3741 Exclusive XML
// Canonicalization (http://www.w3.org/2001/10/xml-exc-c14n#), skipping exclude.
func canonicalizeExcC14N(node *xmlNode, exclude *xmlNode) []byte {
	var buf bytes.Buffer
	writeExcC14N(&buf, node, exclude, map[string]string{})
	return buf.Bytes()
}

func writeExcC14N(buf *bytes.Buffer, node *xmlNode, exclude *xmlNode, renderedNS map[string]string) {
	if node == nil || node == exclude {
		return
	}
	if node.isText {
		writeEscapedC14NText(buf, node.text)
		return
	}

	nextRendered := make(map[string]string, len(renderedNS))
	for k, v := range renderedNS {
		nextRendered[k] = v
	}

	// Collect visibly utilized namespace prefixes
	utilized := map[string]struct{}{
		node.prefix: {},
	}
	for _, a := range node.attrs {
		if a.prefix != "" && a.prefix != "xml" {
			utilized[a.prefix] = struct{}{}
		}
	}

	type nsDecl struct {
		prefix string
		uri    string
	}
	var decls []nsDecl
	for p := range utilized {
		uri := node.nsScope[p]
		if nextRendered[p] != uri {
			if p == "" && uri == "" && nextRendered[""] == "" {
				continue
			}
			decls = append(decls, nsDecl{prefix: p, uri: uri})
			nextRendered[p] = uri
		}
	}
	sort.Slice(decls, func(i, j int) bool {
		return decls[i].prefix < decls[j].prefix
	})

	attrs := make([]xmlAttr, len(node.attrs))
	copy(attrs, node.attrs)
	sort.Slice(attrs, func(i, j int) bool {
		if attrs[i].nsURI != attrs[j].nsURI {
			return attrs[i].nsURI < attrs[j].nsURI
		}
		return attrs[i].local < attrs[j].local
	})

	buf.WriteByte('<')
	writeQName(buf, node.prefix, node.local)

	for _, d := range decls {
		if d.prefix == "" {
			buf.WriteString(` xmlns="`)
		} else {
			buf.WriteString(` xmlns:`)
			buf.WriteString(d.prefix)
			buf.WriteString(`="`)
		}
		writeEscapedC14NAttr(buf, d.uri)
		buf.WriteByte('"')
	}

	for _, a := range attrs {
		buf.WriteByte(' ')
		writeQName(buf, a.prefix, a.local)
		buf.WriteString(`="`)
		writeEscapedC14NAttr(buf, a.value)
		buf.WriteByte('"')
	}
	buf.WriteByte('>')

	for _, ch := range node.children {
		writeExcC14N(buf, ch, exclude, nextRendered)
	}

	buf.WriteString("</")
	writeQName(buf, node.prefix, node.local)
	buf.WriteByte('>')
}

func writeQName(buf *bytes.Buffer, prefix, local string) {
	if prefix != "" {
		buf.WriteString(prefix)
		buf.WriteByte(':')
	}
	buf.WriteString(local)
}

func writeEscapedC14NAttr(buf *bytes.Buffer, s string) {
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '&':
			buf.WriteString("&amp;")
		case '<':
			buf.WriteString("&lt;")
		case '"':
			buf.WriteString("&quot;")
		case '\t':
			buf.WriteString("&#x9;")
		case '\n':
			buf.WriteString("&#xA;")
		case '\r':
			buf.WriteString("&#xD;")
		default:
			buf.WriteByte(s[i])
		}
	}
}

func writeEscapedC14NText(buf *bytes.Buffer, s string) {
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '&':
			buf.WriteString("&amp;")
		case '<':
			buf.WriteString("&lt;")
		case '>':
			buf.WriteString("&gt;")
		case '\r':
			buf.WriteString("&#xD;")
		default:
			buf.WriteByte(s[i])
		}
	}
}

func verifyEnvelopedSignature(target *xmlNode, sigEl *xmlNode, certs []*x509.Certificate) error {
	signedInfo := sigEl.firstChild("SignedInfo", xmldsigNS)
	if signedInfo == nil {
		return errors.New("missing SignedInfo")
	}
	sigValEl := sigEl.firstChild("SignatureValue", xmldsigNS)
	if sigValEl == nil {
		return errors.New("missing SignatureValue")
	}

	c14nEl := signedInfo.firstChild("CanonicalizationMethod", xmldsigNS)
	if c14nEl == nil {
		return errors.New("missing CanonicalizationMethod")
	}
	c14nAlg := strings.TrimSpace(c14nEl.attr("Algorithm"))
	if c14nAlg != excC14NAlgorithm && c14nAlg != excC14NWithComments && c14nAlg != c14NAlgorithm {
		return fmt.Errorf("unsupported CanonicalizationMethod: %s", c14nAlg)
	}

	sigMethodEl := signedInfo.firstChild("SignatureMethod", xmldsigNS)
	if sigMethodEl == nil {
		return errors.New("missing SignatureMethod")
	}
	sigAlg := strings.TrimSpace(sigMethodEl.attr("Algorithm"))

	refs := signedInfo.childrenByName("Reference", xmldsigNS)
	if len(refs) != 1 {
		return fmt.Errorf("expected exactly 1 Reference in SignedInfo, got %d", len(refs))
	}
	ref := refs[0]

	targetID := strings.TrimSpace(target.attr("ID"))
	refURI := strings.TrimSpace(ref.attr("URI"))
	if refURI != "" {
		if targetID == "" || refURI != "#"+targetID {
			return fmt.Errorf("reference URI %q does not match signed element ID %q", refURI, targetID)
		}
	}

	// Validate transforms allow only enveloped-signature + canonicalization
	if transformsEl := ref.firstChild("Transforms", xmldsigNS); transformsEl != nil {
		for _, tr := range transformsEl.childrenByName("Transform", xmldsigNS) {
			alg := strings.TrimSpace(tr.attr("Algorithm"))
			switch alg {
			case envelopedSigAlgo, excC14NAlgorithm, excC14NWithComments, c14NAlgorithm:
			default:
				return fmt.Errorf("unsupported Transform algorithm: %s", alg)
			}
		}
	}

	digestMethodEl := ref.firstChild("DigestMethod", xmldsigNS)
	if digestMethodEl == nil {
		return errors.New("missing DigestMethod")
	}
	digestHash, err := newDigestHasher(strings.TrimSpace(digestMethodEl.attr("Algorithm")))
	if err != nil {
		return err
	}

	digestValEl := ref.firstChild("DigestValue", xmldsigNS)
	if digestValEl == nil {
		return errors.New("missing DigestValue")
	}
	expectedDigest, err := base64.StdEncoding.DecodeString(stripWhitespace(digestValEl.textContent()))
	if err != nil {
		return fmt.Errorf("invalid base64 DigestValue: %w", err)
	}

	canonTarget := canonicalizeExcC14N(target, sigEl)
	digestHash.Write(canonTarget)
	actualDigest := digestHash.Sum(nil)
	if subtle.ConstantTimeCompare(actualDigest, expectedDigest) != 1 {
		return errors.New("XML digest verification failed")
	}

	sigBytes, err := base64.StdEncoding.DecodeString(stripWhitespace(sigValEl.textContent()))
	if err != nil {
		return fmt.Errorf("invalid base64 SignatureValue: %w", err)
	}

	canonSignedInfo := canonicalizeExcC14N(signedInfo, nil)
	for _, cert := range certs {
		if verifySignatureWithCert(cert, sigAlg, canonSignedInfo, sigBytes) == nil {
			return nil
		}
	}
	return errors.New("cryptographic signature did not verify against any active IdP certificate")
}

func newDigestHasher(alg string) (hash.Hash, error) {
	switch alg {
	case digestAlgSHA256:
		return sha256.New(), nil
	case digestAlgSHA384:
		return sha512.New384(), nil
	case digestAlgSHA512:
		return sha512.New(), nil
	default:
		return nil, fmt.Errorf("unsupported DigestMethod algorithm: %s", alg)
	}
}

func verifySignatureWithCert(cert *x509.Certificate, alg string, data, sig []byte) error {
	switch alg {
	case sigAlgRSASHA256, sigAlgRSASHA384, sigAlgRSASHA512:
		pub, ok := cert.PublicKey.(*rsa.PublicKey)
		if !ok {
			return errors.New("certificate public key is not RSA")
		}
		var h crypto.Hash
		var sum []byte
		switch alg {
		case sigAlgRSASHA256:
			h = crypto.SHA256
			d := sha256.Sum256(data)
			sum = d[:]
		case sigAlgRSASHA384:
			h = crypto.SHA384
			d := sha512.Sum384(data)
			sum = d[:]
		case sigAlgRSASHA512:
			h = crypto.SHA512
			d := sha512.Sum512(data)
			sum = d[:]
		}
		return rsa.VerifyPKCS1v15(pub, h, sum, sig)

	case sigAlgECDSASHA256, sigAlgECDSASHA384, sigAlgECDSASHA512:
		pub, ok := cert.PublicKey.(*ecdsa.PublicKey)
		if !ok {
			return errors.New("certificate public key is not ECDSA")
		}
		var sum []byte
		switch alg {
		case sigAlgECDSASHA256:
			d := sha256.Sum256(data)
			sum = d[:]
		case sigAlgECDSASHA384:
			d := sha512.Sum384(data)
			sum = d[:]
		case sigAlgECDSASHA512:
			d := sha512.Sum512(data)
			sum = d[:]
		}
		byteLen := (pub.Curve.Params().BitSize + 7) / 8
		if len(sig) == 2*byteLen {
			r := new(big.Int).SetBytes(sig[:byteLen])
			s := new(big.Int).SetBytes(sig[byteLen:])
			if ecdsa.Verify(pub, sum, r, s) {
				return nil
			}
			return errors.New("ECDSA signature mismatch")
		}
		if ecdsa.VerifyASN1(pub, sum, sig) {
			return nil
		}
		return errors.New("ECDSA signature mismatch")

	default:
		return fmt.Errorf("unsupported SignatureMethod algorithm: %s", alg)
	}
}

func stripWhitespace(s string) string {
	return strings.Map(func(r rune) rune {
		switch r {
		case ' ', '\t', '\n', '\r':
			return -1
		default:
			return r
		}
	}, s)
}

// SignAssertionXML computes an enveloped XMLDSig RSA-SHA256 signature over the
// single Assertion inside rawResponseXML and returns the signed Response XML.
// Intended for SAML signing utilities and test fixtures.
func SignAssertionXML(rawResponseXML []byte, privKey *rsa.PrivateKey) ([]byte, error) {
	if privKey == nil {
		return nil, errors.New("private key is required")
	}
	root, err := parseStrictXML(rawResponseXML)
	if err != nil {
		return nil, err
	}
	assertionEl := root.firstChild("Assertion", samlAssertionNS)
	if assertionEl == nil {
		return nil, errors.New("response missing Assertion")
	}
	assertionID := strings.TrimSpace(assertionEl.attr("ID"))
	if assertionID == "" {
		return nil, errors.New("assertion missing ID attribute")
	}

	canonAssertion := canonicalizeExcC14N(assertionEl, nil)
	digest := sha256.Sum256(canonAssertion)
	digestB64 := base64.StdEncoding.EncodeToString(digest[:])

	signedInfoXML := fmt.Sprintf(
		`<ds:SignedInfo xmlns:ds="%s"><ds:CanonicalizationMethod Algorithm="%s"></ds:CanonicalizationMethod><ds:SignatureMethod Algorithm="%s"></ds:SignatureMethod><ds:Reference URI="#%s"><ds:Transforms><ds:Transform Algorithm="%s"></ds:Transform><ds:Transform Algorithm="%s"></ds:Transform></ds:Transforms><ds:DigestMethod Algorithm="%s"></ds:DigestMethod><ds:DigestValue>%s</ds:DigestValue></ds:Reference></ds:SignedInfo>`,
		xmldsigNS, excC14NAlgorithm, sigAlgRSASHA256, assertionID, envelopedSigAlgo, excC14NAlgorithm, digestAlgSHA256, digestB64,
	)

	siNode, err := parseStrictXML([]byte(signedInfoXML))
	if err != nil {
		return nil, err
	}
	canonSI := canonicalizeExcC14N(siNode, nil)
	siHash := sha256.Sum256(canonSI)
	sigBytes, err := rsa.SignPKCS1v15(rand.Reader, privKey, crypto.SHA256, siHash[:])
	if err != nil {
		return nil, err
	}
	sigB64 := base64.StdEncoding.EncodeToString(sigBytes)

	sigXML := fmt.Sprintf(`<ds:Signature xmlns:ds="%s">%s<ds:SignatureValue>%s</ds:SignatureValue></ds:Signature>`, xmldsigNS, signedInfoXML, sigB64)
	sigNode, err := parseStrictXML([]byte(sigXML))
	if err != nil {
		return nil, err
	}

	// Attach <ds:Signature> as child of <Assertion> and serialize canonical XML
	assertionEl.children = append(assertionEl.children, sigNode)
	return canonicalizeExcC14N(root, nil), nil
}
