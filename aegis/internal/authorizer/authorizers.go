package authorizer

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

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

// NexusAuthorizer integrates with Autorix Nexus for ReBAC/ABAC validation.
// It fails closed whenever no Nexus endpoint is configured or on any check error.
type NexusAuthorizer struct {
	endpoint   string
	httpClient *http.Client
}

// NewNexusAuthorizer creates a NexusAuthorizer with no configured endpoint (fail-closed).
func NewNexusAuthorizer() *NexusAuthorizer {
	return NewNexusAuthorizerWithEndpoint("", nil)
}

// NewNexusAuthorizerWithEndpoint creates a NexusAuthorizer targeting a Nexus HTTP endpoint.
func NewNexusAuthorizerWithEndpoint(endpoint string, httpClient *http.Client) *NexusAuthorizer {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 5 * time.Second}
	}
	return &NexusAuthorizer{
		endpoint:   strings.TrimRight(strings.TrimSpace(endpoint), "/"),
		httpClient: httpClient,
	}
}

func (a *NexusAuthorizer) Name() string { return "nexus_rebac" }

func (a *NexusAuthorizer) Authorize(r *http.Request, session *core.Session, config map[string]interface{}) (bool, error) {
	if session == nil {
		return false, errors.New("unauthenticated subject cannot be evaluated on ReBAC")
	}
	subj := strings.TrimSpace(session.Subject)
	if subj == "" || strings.EqualFold(subj, "anonymous") || strings.EqualFold(subj, "noop") {
		return false, errors.New("unauthenticated or synthetic subject cannot be evaluated on ReBAC")
	}

	if a.endpoint == "" {
		return false, errors.New("nexus authorizer is not configured")
	}

	namespace, _ := config["namespace"].(string)
	relation, _ := config["relation"].(string)
	object, _ := config["object"].(string)
	subjectNamespace, _ := config["subject_namespace"].(string)
	if subjectNamespace == "" {
		subjectNamespace = "user"
	}
	if namespace == "" || relation == "" || object == "" {
		return false, errors.New("nexus authorizer requires namespace, relation, and object configuration")
	}

	payload := map[string]interface{}{
		"namespace":         namespace,
		"object":            object,
		"relation":          relation,
		"subject_namespace": subjectNamespace,
		"subject_id":        subj,
		"request_context": map[string]interface{}{
			"remote_addr": r.RemoteAddr,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return false, fmt.Errorf("marshal nexus check request: %w", err)
	}

	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, a.endpoint+"/check", bytes.NewReader(body))
	if err != nil {
		return false, fmt.Errorf("build nexus check request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("nexus check request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("nexus check returned unexpected status %d", resp.StatusCode)
	}

	var checkResp struct {
		Allowed bool   `json:"allowed"`
		Reason  string `json:"reason"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&checkResp); err != nil {
		return false, fmt.Errorf("decode nexus check response: %w", err)
	}

	if !checkResp.Allowed {
		if checkResp.Reason != "" {
			return false, errors.New(checkResp.Reason)
		}
		return false, errors.New("access denied by nexus")
	}
	return true, nil
}
