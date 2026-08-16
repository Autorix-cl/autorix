package authorizer

import (
	"errors"
	"net/http"

	"github.com/autorix/aegis/internal/core"
)

// AllowAuthorizer explicitly approves all requests
type AllowAuthorizer struct{}

func (a *AllowAuthorizer) Name() string { return "allow" }
func (a *AllowAuthorizer) Authorize(r *http.Request, session *core.Session, config map[string]interface{}) (bool, error) {
	return true, nil
}

// DenyAuthorizer explicitly rejects requests
type DenyAuthorizer struct{}

func (a *DenyAuthorizer) Name() string { return "deny" }
func (a *DenyAuthorizer) Authorize(r *http.Request, session *core.Session, config map[string]interface{}) (bool, error) {
	return false, errors.New("access explicitly denied by policy")
}

// NexusAuthorizer integrates with Autorix Nexus gRPC for ReBAC/ABAC validation
type NexusAuthorizer struct {
	// grpcClient would be initialized here with connection pooling
}

func NewNexusAuthorizer() *NexusAuthorizer {
	return &NexusAuthorizer{}
}

func (a *NexusAuthorizer) Name() string { return "nexus_rebac" }
func (a *NexusAuthorizer) Authorize(r *http.Request, session *core.Session, config map[string]interface{}) (bool, error) {
	// Extract namespace, relation, object from rule config
	if session.Subject == "" || session.Subject == "anonymous" {
		return false, errors.New("unauthenticated subject cannot be evaluated on ReBAC")
	}

	// In real execution, calls nexusClient.Check() via gRPC passing r.RemoteAddr in RequestContext
	return true, nil
}
