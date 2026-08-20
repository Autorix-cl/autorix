package autorix

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// VulcanClient provides API Key issuance, verification, and Macaroon caveat attenuation.
type VulcanClient struct {
	client *Client
}

func newVulcanClient(c *Client) *VulcanClient {
	return &VulcanClient{client: c}
}

// VerifyKeyRequest specifies the key token and calling context for validation.
type VerifyKeyRequest struct {
	Token   string                 `json:"token"`
	Context map[string]interface{} `json:"context,omitempty"`
}

// VerifyKeyResponse returns whether the API key or Macaroon is valid.
type VerifyKeyResponse struct {
	Valid       bool     `json:"valid"`
	KeyID       string   `json:"key_id,omitempty"`
	Name        string   `json:"name,omitempty"`
	Scopes      []string `json:"scopes,omitempty"`
	Environment string   `json:"environment,omitempty"`
	Error       string   `json:"error,omitempty"`
}

// AttenuateKeyRequest defines caveats to append to a Macaroon.
type AttenuateKeyRequest struct {
	Token   string   `json:"token"`
	Caveats []string `json:"caveats"`
}

// AttenuateKeyResponse returns the new attenuated token.
type AttenuateKeyResponse struct {
	AttenuatedToken string   `json:"attenuated_token"`
	CaveatsApplied  []string `json:"caveats_applied"`
}

// Verify validates an API Key or Macaroon against Vulcan.
func (v *VulcanClient) Verify(ctx context.Context, token string, evalCtx map[string]interface{}) (*VerifyKeyResponse, error) {
	if token == "" {
		return &VerifyKeyResponse{Valid: false, Error: "empty token"}, nil
	}

	payload := VerifyKeyRequest{
		Token:   token,
		Context: evalCtx,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	url := strings.TrimRight(v.client.config.VulcanURL, "/") + "/keys/verify"

	httpResp, err := executeWithRetry(ctx, v.client.config.RetryConfig, func(opCtx context.Context) (*http.Response, error) {
		httpReq, err := http.NewRequestWithContext(opCtx, http.MethodPost, url, bytes.NewReader(data))
		if err != nil {
			return nil, err
		}
		httpReq.Header.Set("Content-Type", "application/json")
		v.client.prepareRequest(opCtx, httpReq)
		return v.client.config.HTTPClient.Do(httpReq)
	})

	if err != nil {
		return nil, fmt.Errorf("vulcan key verification failed: %w", err)
	}
	defer httpResp.Body.Close()

	var res VerifyKeyResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&res); err != nil {
		return nil, err
	}
	return &res, nil
}

// Attenuate adds first-party caveats to a root or child Macaroon token.
func (v *VulcanClient) Attenuate(ctx context.Context, token string, caveats []string) (string, error) {
	payload := AttenuateKeyRequest{
		Token:   token,
		Caveats: caveats,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	url := strings.TrimRight(v.client.config.VulcanURL, "/") + "/keys/attenuate"

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	v.client.prepareRequest(ctx, httpReq)

	resp, err := v.client.config.HTTPClient.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("vulcan returned status: %d", resp.StatusCode)
	}

	var res AttenuateKeyResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}
	return res.AttenuatedToken, nil
}
