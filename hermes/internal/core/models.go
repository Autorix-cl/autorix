package core

import (
	"time"

	"github.com/google/uuid"
)

// SAMLProvider represents an external SAML 2.0 Identity Provider (IdP)
type SAMLProvider struct {
	ID                string            `json:"id"` // e.g. "okta-corporate"
	DisplayName       string            `json:"display_name"`
	IdPEntityID       string            `json:"idp_entity_id"`
	IdPSSOURL         string            `json:"idp_sso_url"`
	IdPCertificatePEM string            `json:"idp_certificate_pem"`
	SPEntityID        string            `json:"sp_entity_id"`
	AttributeMapping  map[string]string `json:"attribute_mapping"`
	Enabled           bool              `json:"enabled"`
	CreatedAt         time.Time         `json:"created_at"`
	UpdatedAt         time.Time         `json:"updated_at"`
}

// SAMLAssertion represents normalized attributes extracted from a SAML response
type SAMLAssertion struct {
	Subject    string            `json:"subject"`
	Email      string            `json:"email"`
	FirstName  string            `json:"first_name"`
	LastName   string            `json:"last_name"`
	Attributes map[string]string `json:"attributes"`
}

// SCIMUser represents an RFC 7643 User Resource
type SCIMUser struct {
	Schemas     []string               `json:"schemas"`
	ID          uuid.UUID              `json:"id"`
	ExternalID  string                 `json:"externalId"`
	UserName    string                 `json:"userName"`
	DisplayName string                 `json:"displayName,omitempty"`
	Emails      []SCIMEmail            `json:"emails"`
	Active      bool                   `json:"active"`
	Meta        SCIMMeta               `json:"meta"`
	Attributes  map[string]interface{} `json:"-"`
}

type SCIMEmail struct {
	Value   string `json:"value"`
	Type    string `json:"type"`
	Primary bool   `json:"primary"`
}

// SCIMGroup represents an RFC 7643 Group Resource
type SCIMGroup struct {
	Schemas     []string     `json:"schemas"`
	ID          uuid.UUID    `json:"id"`
	DisplayName string       `json:"displayName"`
	Members     []SCIMMember `json:"members"`
	Meta        SCIMMeta     `json:"meta"`
}

type SCIMMember struct {
	Value   string `json:"value"` // User UUID
	Display string `json:"display,omitempty"`
}

type SCIMMeta struct {
	ResourceType string    `json:"resourceType"`
	Created      time.Time `json:"created"`
	LastModified time.Time `json:"lastModified"`
	Location     string    `json:"location"`
}

// SCIMListResponse represents RFC 7644 Paginated List Response
type SCIMListResponse struct {
	Schemas      []string      `json:"schemas"`
	TotalResults int           `json:"totalResults"`
	StartIndex   int           `json:"startIndex"`
	ItemsPerPage int           `json:"itemsPerPage"`
	Resources    []interface{} `json:"Resources"`
}
