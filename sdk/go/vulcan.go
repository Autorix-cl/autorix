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

// MacaroonCaveat represents a first-party caveat on a Vulcan Macaroon.
type MacaroonCaveat struct {
	Predicate string `json:"predicate"`
}

// Macaroon represents a structured capability token issued by Vulcan.
type Macaroon struct {
	Location  string           `json:"location"`
	KeyID     string           `json:"key_id"`
	Caveats   []MacaroonCaveat `json:"caveats"`
	Signature string           `json:"signature"`
}

// APIKeyInfo describes API key metadata returned by Vulcan on verification.
type APIKeyInfo struct {
	ID        string   `json:"id"`
	KeyPrefix string   `json:"key_prefix,omitempty"`
	KeyHint   string   `json:"key_hint,omitempty"`
	Name      string   `json:"name"`
	OwnerID   string   `json:"owner_id,omitempty"`
	Scopes    []string `json:"scopes,omitempty"`
	State     string   `json:"state,omitempty"`
}

// VerifyKeyRequest specifies the macaroon or key token and calling context for validation.
type VerifyKeyRequest struct {
	Macaroon *Macaroon              `json:"macaroon,omitempty"`
	Token    string                 `json:"token,omitempty"`
	Context  map[string]interface{} `json:"context,omitempty"`
}

// VerifyKeyResponse returns whether the API key or Macaroon is valid.
type VerifyKeyResponse struct {
	Valid       bool        `json:"valid"`
	APIKey      *APIKeyInfo `json:"api_key,omitempty"`
	KeyID       string      `json:"key_id,omitempty"`
	Name        string      `json:"name,omitempty"`
	Scopes      []string    `json:"scopes,omitempty"`
	Environment string      `json:"environment,omitempty"`
	Error       string      `json:"error,omitempty"`
}

// AttenuateKeyRequest defines caveats to append to a Macaroon.
type AttenuateKeyRequest struct {
	Token   string   `json:"token,omitempty"`
	Caveats []string `json:"caveats,omitempty"`
}

// AttenuateKeyResponse returns the new attenuated token.
type AttenuateKeyResponse struct {
	AttenuatedToken string   `json:"attenuated_token"`
	CaveatsApplied  []string `json:"caveats_applied"`
}

// VerifyMacaroon validates a structured Macaroon against Vulcan's POST /keys/verify contract.
func (v *VulcanClient) VerifyMacaroon(ctx context.Context, m *Macaroon, evalCtx map[string]interface{}) (*VerifyKeyResponse, error) {
	if m == nil {
		return &VerifyKeyResponse{Valid: false, Error: "macaroon is required"}, nil
	}
	return v.doVerify(ctx, VerifyKeyRequest{
		Macaroon: m,
		Context:  evalCtx,
	})
}

// Verify validates a JSON-encoded Macaroon (preferred) or legacy token string against Vulcan.
func (v *VulcanClient) Verify(ctx context.Context, token string, evalCtx map[string]interface{}) (*VerifyKeyResponse, error) {
	trimmed := strings.TrimSpace(token)
	if trimmed == "" {
		return &VerifyKeyResponse{Valid: false, Error: "empty token"}, nil
	}

	if strings.HasPrefix(trimmed, "{") {
		var m Macaroon
		if err := json.Unmarshal([]byte(trimmed), &m); err == nil && m.KeyID != "" {
			return v.VerifyMacaroon(ctx, &m, evalCtx)
		}
	}

	return v.doVerify(ctx, VerifyKeyRequest{
		Token:   token,
		Context: evalCtx,
	})
}

func (v *VulcanClient) doVerify(ctx context.Context, payload VerifyKeyRequest) (*VerifyKeyResponse, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	endpoint := strings.TrimRight(v.client.config.VulcanURL, "/") + "/keys/verify"

	httpResp, err := executeWithRetry(ctx, v.client.config.RetryConfig, func(opCtx context.Context) (*http.Response, error) {
		httpReq, err := http.NewRequestWithContext(opCtx, http.MethodPost, endpoint, bytes.NewReader(data))
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

	if httpResp.StatusCode == http.StatusBadRequest || httpResp.StatusCode == http.StatusUnauthorized || httpResp.StatusCode == http.StatusForbidden {
		var errBody struct {
			Error string `json:"error"`
		}
		_ = json.NewDecoder(httpResp.Body).Decode(&errBody)
		msg := errBody.Error
		if msg == "" {
			msg = fmt.Sprintf("vulcan rejected credential (status %d)", httpResp.StatusCode)
		}
		return &VerifyKeyResponse{Valid: false, Error: msg}, nil
	}
	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("vulcan verify returned unexpected status: %d", httpResp.StatusCode)
	}

	var res VerifyKeyResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&res); err != nil {
		return nil, err
	}
	if res.APIKey != nil {
		if res.KeyID == "" {
			res.KeyID = res.APIKey.ID
		}
		if res.Name == "" {
			res.Name = res.APIKey.Name
		}
		if len(res.Scopes) == 0 && len(res.APIKey.Scopes) > 0 {
			res.Scopes = res.APIKey.Scopes
		}
	}
	return &res, nil
}

// AttenuateMacaroon appends a single first-party caveat via Vulcan's POST /keys/attenuate contract.
func (v *VulcanClient) AttenuateMacaroon(ctx context.Context, m *Macaroon, caveat string) (*Macaroon, error) {
	if m == nil || strings.TrimSpace(caveat) == "" {
		return nil, fmt.Errorf("macaroon and caveat are required")
	}

	data, err := json.Marshal(struct {
		Macaroon *Macaroon `json:"macaroon"`
		Caveat   string    `json:"caveat"`
	}{
		Macaroon: m,
		Caveat:   caveat,
	})
	if err != nil {
		return nil, err
	}

	endpoint := strings.TrimRight(v.client.config.VulcanURL, "/") + "/keys/attenuate"

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	v.client.prepareRequest(ctx, httpReq)

	resp, err := v.client.config.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("vulcan attenuate returned status: %d", resp.StatusCode)
	}

	var attenuated Macaroon
	if err := json.NewDecoder(resp.Body).Decode(&attenuated); err != nil {
		return nil, err
	}
	return &attenuated, nil
}

// Attenuate adds first-party caveats to a JSON-encoded Macaroon (sequentially via
// POST /keys/attenuate) or legacy token string.
func (v *VulcanClient) Attenuate(ctx context.Context, token string, caveats []string) (string, error) {
	trimmed := strings.TrimSpace(token)
	if trimmed == "" || len(caveats) == 0 {
		return "", fmt.Errorf("token and at least one caveat are required")
	}

	if strings.HasPrefix(trimmed, "{") {
		var current Macaroon
		if err := json.Unmarshal([]byte(trimmed), &current); err == nil && current.KeyID != "" {
			curPtr := &current
			for _, c := range caveats {
				next, err := v.AttenuateMacaroon(ctx, curPtr, c)
				if err != nil {
					return "", err
				}
				curPtr = next
			}
			encoded, err := json.Marshal(curPtr)
			if err != nil {
				return "", err
			}
			return string(encoded), nil
		}
	}

	payload := AttenuateKeyRequest{
		Token:   token,
		Caveats: caveats,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	endpoint := strings.TrimRight(v.client.config.VulcanURL, "/") + "/keys/attenuate"

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(data))
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
