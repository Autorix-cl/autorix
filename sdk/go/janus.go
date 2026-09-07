package autorix

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// JanusClient provides OAuth2 and OpenID Connect token verification and JWKS caching.
type JanusClient struct {
	client    *Client
	jwksMu    sync.RWMutex
	jwksCache map[string]interface{}
	jwksExp   time.Time
}

func newJanusClient(c *Client) *JanusClient {
	return &JanusClient{client: c}
}

// TokenIntrospectionResponse describes the token status under RFC 7662.
type TokenIntrospectionResponse struct {
	Active    bool     `json:"active"`
	Scope     string   `json:"scope,omitempty"`
	ClientID  string   `json:"client_id,omitempty"`
	Subject   string   `json:"sub,omitempty"`
	ExpiresAt int64    `json:"exp,omitempty"`
	IssuedAt  int64    `json:"iat,omitempty"`
	Issuer    string   `json:"iss,omitempty"`
	Audience  []string `json:"aud,omitempty"`
}

// DiscoveryDocument is the subset of OIDC discovery metadata needed by
// applications to configure an OAuth client without hard-coding endpoints.
type DiscoveryDocument struct {
	Issuer                        string   `json:"issuer"`
	AuthorizationEndpoint         string   `json:"authorization_endpoint"`
	TokenEndpoint                 string   `json:"token_endpoint"`
	IntrospectionEndpoint         string   `json:"introspection_endpoint"`
	RevocationEndpoint            string   `json:"revocation_endpoint"`
	JWKSURI                       string   `json:"jwks_uri"`
	CodeChallengeMethodsSupported []string `json:"code_challenge_methods_supported"`
}

// TokenRequest represents a supported OAuth token exchange. ClientSecret must
// only be supplied by confidential (server-side) clients.
type TokenRequest struct {
	GrantType    string
	ClientID     string
	ClientSecret string
	Code         string
	CodeVerifier string
	RedirectURI  string
	RefreshToken string
	Scope        string
	Resource     string
}

// TokenResponse is returned by Janus after a successful token exchange.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	IDToken      string `json:"id_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
}

// AuthorizationURL builds an authorization-code request. PKCE values are
// caller-owned: the SDK never stores a verifier or browser tokens.
func (j *JanusClient) AuthorizationURL(clientID, redirectURI, state, scope, codeChallenge string) (string, error) {
	if clientID == "" || redirectURI == "" {
		return "", fmt.Errorf("client_id and redirect_uri are required")
	}
	u, err := url.Parse(strings.TrimRight(j.client.config.JanusURL, "/") + "/oauth2/auth")
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("response_type", "code")
	q.Set("client_id", clientID)
	q.Set("redirect_uri", redirectURI)
	if state != "" {
		q.Set("state", state)
	}
	if scope != "" {
		q.Set("scope", scope)
	}
	if codeChallenge != "" {
		q.Set("code_challenge", codeChallenge)
		q.Set("code_challenge_method", "S256")
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// Discovery retrieves OIDC discovery metadata. It is safe to retry because it
// is an idempotent GET operation.
func (j *JanusClient) Discovery(ctx context.Context) (*DiscoveryDocument, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(j.client.config.JanusURL, "/")+"/.well-known/openid-configuration", nil)
	if err != nil {
		return nil, err
	}
	j.client.prepareRequest(ctx, req)
	resp, err := executeWithRetry(ctx, j.client.config.RetryConfig, func(context.Context) (*http.Response, error) { return j.client.config.HTTPClient.Do(req) })
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("janus discovery returned status: %d", resp.StatusCode)
	}
	var result DiscoveryDocument
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ExchangeToken performs a token exchange. It deliberately does not retry:
// authorization codes and refresh tokens are single-use credentials.
func (j *JanusClient) ExchangeToken(ctx context.Context, input TokenRequest) (*TokenResponse, error) {
	if input.GrantType == "" || input.ClientID == "" {
		return nil, fmt.Errorf("grant_type and client_id are required")
	}
	form := url.Values{"grant_type": {input.GrantType}, "client_id": {input.ClientID}}
	for key, value := range map[string]string{"code": input.Code, "code_verifier": input.CodeVerifier, "redirect_uri": input.RedirectURI, "refresh_token": input.RefreshToken, "scope": input.Scope, "resource": input.Resource} {
		if value != "" {
			form.Set(key, value)
		}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(j.client.config.JanusURL, "/")+"/oauth2/token", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if input.ClientSecret != "" {
		req.SetBasicAuth(input.ClientID, input.ClientSecret)
	}
	j.client.prepareRequest(ctx, req)
	resp, err := j.client.config.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("janus token exchange returned status: %d", resp.StatusCode)
	}
	var result TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Revoke revokes a token using confidential-client authentication. It does not
// retry because a transport failure leaves the caller unable to distinguish a
// completed revocation from an uncompleted one.
func (j *JanusClient) Revoke(ctx context.Context, clientID, clientSecret, token, hint string) error {
	if clientID == "" || clientSecret == "" || token == "" {
		return fmt.Errorf("client_id, client_secret, and token are required")
	}
	form := url.Values{"token": {token}}
	if hint != "" {
		form.Set("token_type_hint", hint)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(j.client.config.JanusURL, "/")+"/oauth2/revoke", strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(clientID, clientSecret)
	j.client.prepareRequest(ctx, req)
	resp, err := j.client.config.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("janus revoke returned status: %d", resp.StatusCode)
	}
	return nil
}

// Introspect validates an OAuth2 token using RFC 7662 token introspection.
func (j *JanusClient) Introspect(ctx context.Context, token string) (*TokenIntrospectionResponse, error) {
	formData := url.Values{}
	formData.Set("token", token)

	endpoint := strings.TrimRight(j.client.config.JanusURL, "/") + "/oauth2/introspect"

	httpResp, err := executeWithRetry(ctx, j.client.config.RetryConfig, func(opCtx context.Context) (*http.Response, error) {
		httpReq, err := http.NewRequestWithContext(opCtx, http.MethodPost, endpoint, strings.NewReader(formData.Encode()))
		if err != nil {
			return nil, err
		}
		httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		j.client.prepareRequest(opCtx, httpReq)
		return j.client.config.HTTPClient.Do(httpReq)
	})

	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("janus introspect returned status: %d", httpResp.StatusCode)
	}

	var res TokenIntrospectionResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&res); err != nil {
		return nil, err
	}
	return &res, nil
}

// GetJWKS retrieves the public JWKS keys with Stale-While-Revalidate caching.
func (j *JanusClient) GetJWKS(ctx context.Context) (map[string]interface{}, error) {
	j.jwksMu.RLock()
	if j.jwksCache != nil && time.Now().Before(j.jwksExp) {
		defer j.jwksMu.RUnlock()
		return j.jwksCache, nil
	}
	j.jwksMu.RUnlock()

	endpoint := strings.TrimRight(j.client.config.JanusURL, "/") + "/.well-known/jwks.json"

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	j.client.prepareRequest(ctx, httpReq)

	resp, err := j.client.config.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var jwks map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return nil, err
	}

	j.jwksMu.Lock()
	j.jwksCache = jwks
	j.jwksExp = time.Now().Add(5 * time.Minute) // 5 min cache
	j.jwksMu.Unlock()

	return jwks, nil
}
