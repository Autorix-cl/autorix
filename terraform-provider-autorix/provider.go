package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const defaultHTTPTimeout = 15 * time.Second

// Config defines connection settings for the Autorix Terraform provider.
type Config struct {
	Endpoint       string
	ArgusEndpoint  string
	ThemisEndpoint string
	JanusEndpoint  string
	APIToken       string
	HTTPClient     *http.Client
}

// APIError represents a non-2xx HTTP response returned by an Autorix control-plane API.
type APIError struct {
	StatusCode int
	Method     string
	URL        string
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("autorix api %s %s returned HTTP %d: %s", e.Method, e.URL, e.StatusCode, strings.TrimSpace(e.Body))
}

// Provider manages Autorix infrastructure resources via control-plane REST APIs.
type Provider struct {
	argusURL  string
	themisURL string
	janusURL  string
	apiToken  string
	client    *http.Client
}

// NewProvider validates configuration and constructs an Autorix provider client.
func NewProvider(cfg Config) (*Provider, error) {
	base := strings.TrimRight(strings.TrimSpace(cfg.Endpoint), "/")
	if base == "" {
		base = strings.TrimRight(strings.TrimSpace(os.Getenv("AUTORIX_ENDPOINT")), "/")
	}

	argus := strings.TrimRight(strings.TrimSpace(cfg.ArgusEndpoint), "/")
	if argus == "" {
		argus = strings.TrimRight(strings.TrimSpace(os.Getenv("AUTORIX_ARGUS_URL")), "/")
	}
	if argus == "" {
		argus = base
	}

	themis := strings.TrimRight(strings.TrimSpace(cfg.ThemisEndpoint), "/")
	if themis == "" {
		themis = strings.TrimRight(strings.TrimSpace(os.Getenv("AUTORIX_THEMIS_URL")), "/")
	}
	if themis == "" {
		themis = base
	}

	janus := strings.TrimRight(strings.TrimSpace(cfg.JanusEndpoint), "/")
	if janus == "" {
		janus = strings.TrimRight(strings.TrimSpace(os.Getenv("AUTORIX_JANUS_ADMIN_URL")), "/")
	}
	if janus == "" {
		janus = base
	}

	if argus == "" && themis == "" && janus == "" {
		return nil, errors.New("at least one Autorix endpoint (Endpoint, ArgusEndpoint, ThemisEndpoint, or JanusEndpoint) is required")
	}

	token := strings.TrimSpace(cfg.APIToken)
	if token == "" {
		token = strings.TrimSpace(os.Getenv("AUTORIX_API_TOKEN"))
	}
	if token == "" {
		token = strings.TrimSpace(os.Getenv("AUTORIX_INTERNAL_AUTH_SECRET"))
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultHTTPTimeout}
	}

	return &Provider{
		argusURL:  argus,
		themisURL: themis,
		janusURL:  janus,
		apiToken:  token,
		client:    httpClient,
	}, nil
}

// Environment represents an Autorix deployment environment managed by Argus.
type Environment struct {
	ID          string `json:"id,omitempty"`
	Name        string `json:"name"`
	Slug        string `json:"slug,omitempty"`
	Description string `json:"description,omitempty"`
	Tier        string `json:"tier,omitempty"`
}

// Policy represents a Themis CEL authorization policy resource.
type Policy struct {
	ID          string `json:"id,omitempty"`
	TenantID    string `json:"tenant_id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Expression  string `json:"expression"`
	Active      bool   `json:"active"`
}

// OAuth2Client represents a Janus OAuth2/OIDC client resource.
type OAuth2Client struct {
	ClientID     string   `json:"client_id"`
	ClientName   string   `json:"client_name"`
	ClientSecret string   `json:"client_secret,omitempty"`
	RedirectURIs []string `json:"redirect_uris,omitempty"`
	GrantTypes   []string `json:"grant_types,omitempty"`
	Scopes       []string `json:"scopes,omitempty"`
	IsPublic     bool     `json:"is_public"`
}

// Schema returns the declarative Terraform resource schema metadata supported by this provider.
func (p *Provider) Schema() map[string]interface{} {
	return map[string]interface{}{
		"provider": "autorix",
		"resources": map[string]interface{}{
			"autorix_environment": map[string]string{
				"id":          "string (computed)",
				"name":        "string (required)",
				"slug":        "string (optional)",
				"description": "string (optional)",
				"tier":        "string (optional)",
			},
			"autorix_policy": map[string]string{
				"id":          "string (computed)",
				"tenant_id":   "string (required)",
				"name":        "string (required)",
				"description": "string (optional)",
				"expression":  "string (required)",
				"active":      "bool (optional)",
			},
			"autorix_oauth2_client": map[string]string{
				"client_id":     "string (required)",
				"client_name":   "string (required)",
				"client_secret": "string (computed, sensitive)",
				"redirect_uris": "list(string) (optional)",
				"grant_types":   "list(string) (optional)",
				"scopes":        "list(string) (optional)",
				"is_public":     "bool (optional)",
			},
		},
	}
}

func (p *Provider) doJSON(ctx context.Context, method, fullURL string, reqBody, respOut interface{}) error {
	var bodyReader io.Reader
	if reqBody != nil {
		raw, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(raw)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		return fmt.Errorf("build request %s %s: %w", method, fullURL, err)
	}
	req.Header.Set("Accept", "application/json")
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if p.apiToken != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiToken)
		req.Header.Set("X-Autorix-Internal-Token", p.apiToken)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("execute %s %s: %w", method, fullURL, err)
	}
	defer func() { _ = resp.Body.Close() }()

	data, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return fmt.Errorf("read response %s %s: %w", method, fullURL, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &APIError{
			StatusCode: resp.StatusCode,
			Method:     method,
			URL:        fullURL,
			Body:       string(data),
		}
	}
	if respOut != nil && len(bytes.TrimSpace(data)) > 0 {
		if err := json.Unmarshal(data, respOut); err != nil {
			return fmt.Errorf("decode response %s %s: %w", method, fullURL, err)
		}
	}
	return nil
}

// Ping verifies connectivity to the configured Argus control-plane readiness endpoint.
func (p *Provider) Ping(ctx context.Context) error {
	if p.argusURL == "" {
		return errors.New("argus endpoint is not configured")
	}
	return p.doJSON(ctx, http.MethodGet, p.argusURL+"/health/ready", nil, nil)
}

// CreateEnvironment creates an environment in Argus.
func (p *Provider) CreateEnvironment(ctx context.Context, env Environment) (*Environment, error) {
	if strings.TrimSpace(env.Name) == "" {
		return nil, errors.New("environment name is required")
	}
	var out Environment
	if err := p.doJSON(ctx, http.MethodPost, p.argusURL+"/v1/environments", env, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetEnvironment retrieves an environment by ID from Argus.
func (p *Provider) GetEnvironment(ctx context.Context, id string) (*Environment, error) {
	var out Environment
	endpoint := p.argusURL + "/v1/environments/" + url.PathEscape(id)
	if err := p.doJSON(ctx, http.MethodGet, endpoint, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteEnvironment deletes an environment by ID from Argus.
func (p *Provider) DeleteEnvironment(ctx context.Context, id string) error {
	endpoint := p.argusURL + "/v1/environments/" + url.PathEscape(id)
	return p.doJSON(ctx, http.MethodDelete, endpoint, nil, nil)
}

// CreatePolicy creates a CEL policy in Themis.
func (p *Provider) CreatePolicy(ctx context.Context, pol Policy) (*Policy, error) {
	if strings.TrimSpace(pol.Name) == "" || strings.TrimSpace(pol.Expression) == "" {
		return nil, errors.New("policy name and expression are required")
	}
	var out Policy
	if err := p.doJSON(ctx, http.MethodPost, p.themisURL+"/policies", pol, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetPolicy retrieves a policy by ID from Themis.
func (p *Provider) GetPolicy(ctx context.Context, id string) (*Policy, error) {
	var out Policy
	endpoint := p.themisURL + "/policies/" + url.PathEscape(id)
	if err := p.doJSON(ctx, http.MethodGet, endpoint, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdatePolicy updates an existing policy in Themis.
func (p *Provider) UpdatePolicy(ctx context.Context, id string, pol Policy) (*Policy, error) {
	var out Policy
	endpoint := p.themisURL + "/policies/" + url.PathEscape(id)
	if err := p.doJSON(ctx, http.MethodPut, endpoint, pol, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeletePolicy deletes a policy by ID from Themis.
func (p *Provider) DeletePolicy(ctx context.Context, id string) error {
	endpoint := p.themisURL + "/policies/" + url.PathEscape(id)
	return p.doJSON(ctx, http.MethodDelete, endpoint, nil, nil)
}

// ApplyPolicy idempotently creates or updates a Themis policy.
func (p *Provider) ApplyPolicy(ctx context.Context, desired Policy) (*Policy, error) {
	if desired.ID != "" {
		_, err := p.GetPolicy(ctx, desired.ID)
		if err == nil {
			return p.UpdatePolicy(ctx, desired.ID, desired)
		}
		var apiErr *APIError
		if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusNotFound {
			return nil, err
		}
	}
	return p.CreatePolicy(ctx, desired)
}

// CreateOAuth2Client registers a new OAuth2 client in Janus admin API.
func (p *Provider) CreateOAuth2Client(ctx context.Context, client OAuth2Client) (*OAuth2Client, error) {
	if strings.TrimSpace(client.ClientID) == "" || strings.TrimSpace(client.ClientName) == "" {
		return nil, errors.New("client_id and client_name are required")
	}
	var out OAuth2Client
	if err := p.doJSON(ctx, http.MethodPost, p.janusURL+"/admin/clients", client, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetOAuth2Client retrieves an OAuth2 client by client_id from Janus admin API.
func (p *Provider) GetOAuth2Client(ctx context.Context, clientID string) (*OAuth2Client, error) {
	var out OAuth2Client
	endpoint := p.janusURL + "/admin/clients/" + url.PathEscape(clientID)
	if err := p.doJSON(ctx, http.MethodGet, endpoint, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteOAuth2Client removes an OAuth2 client by client_id from Janus admin API.
func (p *Provider) DeleteOAuth2Client(ctx context.Context, clientID string) error {
	endpoint := p.janusURL + "/admin/clients/" + url.PathEscape(clientID)
	return p.doJSON(ctx, http.MethodDelete, endpoint, nil, nil)
}
