package scim

import (
	"fmt"

	"github.com/autorix/hermes/internal/core"
)

const (
	UserSchemaURN      = "urn:ietf:params:scim:schemas:core:2.0:User"
	GroupSchemaURN     = "urn:ietf:params:scim:schemas:core:2.0:Group"
	ListResponseURN    = "urn:ietf:params:scim:api:messages:2.0:ListResponse"
	ServiceProviderURN = "urn:ietf:params:scim:schemas:core:2.0:ServiceProviderConfig"
)

type Engine struct {
	BaseURL string
}

func NewEngine(baseURL string) *Engine {
	return &Engine{BaseURL: baseURL}
}

// FormatUser ensures standard RFC 7643 structure and meta links
func (e *Engine) FormatUser(u *core.SCIMUser) *core.SCIMUser {
	u.Schemas = []string{UserSchemaURN}
	u.Meta.ResourceType = "User"
	u.Meta.Location = fmt.Sprintf("%s/scim/v2/Users/%s", e.BaseURL, u.ID)
	return u
}

// FormatGroup ensures standard RFC 7643 structure and meta links
func (e *Engine) FormatGroup(g *core.SCIMGroup) *core.SCIMGroup {
	g.Schemas = []string{GroupSchemaURN}
	g.Meta.ResourceType = "Group"
	g.Meta.Location = fmt.Sprintf("%s/scim/v2/Groups/%s", e.BaseURL, g.ID)
	return g
}

// ServiceProviderConfig returns RFC 7643 metadata for IdP discovery
func (e *Engine) ServiceProviderConfig() map[string]interface{} {
	return map[string]interface{}{
		"schemas":          []string{ServiceProviderURN},
		"documentationUri": "https://docs.autorix.io/scim",
		"patch": map[string]bool{
			"supported": false,
		},
		"bulk": map[string]interface{}{
			"supported":      false,
			"maxOperations":  1000,
			"maxPayloadSize": 1048576,
		},
		"filter": map[string]interface{}{
			"supported":  true,
			"maxResults": 200,
		},
		"changePassword": map[string]bool{
			"supported": false,
		},
		"sort": map[string]bool{
			"supported": false,
		},
		"etag": map[string]bool{
			"supported": false,
		},
		"authenticationSchemes": []map[string]interface{}{
			{
				"name":        "OAuth Bearer Token",
				"description": "Authentication using OAuth 2.0 Bearer Token (Janus)",
				"specUri":     "http://www.rfc-editor.org/info/rfc6750",
				"type":        "oauthbearertoken",
				"primary":     true,
			},
		},
	}
}
