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
