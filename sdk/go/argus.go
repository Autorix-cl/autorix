package autorix

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// ArgusClient provides control plane, audit verification, and compliance operations.
type ArgusClient struct {
	client *Client
}

func newArgusClient(c *Client) *ArgusClient {
	return &ArgusClient{client: c}
}

// AuditVerificationResult summarizes the Merkle hash chain verification.
type AuditVerificationResult struct {
	Verified    bool   `json:"verified"`
	ChainLength int    `json:"chain_length"`
	HeadHash    string `json:"head_hash"`
	VerifiedAt  string `json:"verified_at"`
	Algorithm   string `json:"algorithm"`
}

// VerifyAuditTrail queries Argus to verify that the SHA-256 hash chain is intact.
func (a *ArgusClient) VerifyAuditTrail(ctx context.Context) (*AuditVerificationResult, error) {
	url := strings.TrimRight(a.client.config.ArgusURL, "/") + "/v1/audit/verify"

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	a.client.prepareRequest(ctx, httpReq)

	resp, err := a.client.config.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("argus verify returned status: %d", resp.StatusCode)
	}

	var res AuditVerificationResult
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}
	return &res, nil
}
