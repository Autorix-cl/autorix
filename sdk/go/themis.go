package autorix

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// ThemisClient provides ABAC policy evaluation powered by Google CEL.
type ThemisClient struct {
	client *Client
}

func newThemisClient(c *Client) *ThemisClient {
	return &ThemisClient{client: c}
}

// EvaluatePolicyRequest specifies the context for CEL policy evaluation.
type EvaluatePolicyRequest struct {
	TenantID string                 `json:"tenant_id,omitempty"`
	Context  map[string]interface{} `json:"context"`
}

// PolicyEvaluationResult details the decision for an individual policy.
type PolicyEvaluationResult struct {
	PolicyID   string `json:"policy_id"`
	PolicyName string `json:"policy_name"`
	Passed     bool   `json:"passed"`
	Expression string `json:"expression"`
	Error      string `json:"error,omitempty"`
}

// EvaluatePolicyResponse summarizes the multi-policy evaluation outcome.
type EvaluatePolicyResponse struct {
	AllPassed      bool                     `json:"all_passed"`
	Results        []PolicyEvaluationResult `json:"results"`
	TotalEvaluated int                      `json:"total_evaluated"`
}

// Evaluate evaluates all active CEL policies against the provided request context.
func (t *ThemisClient) Evaluate(ctx context.Context, req EvaluatePolicyRequest) (*EvaluatePolicyResponse, error) {
	if req.TenantID == "" {
		req.TenantID = "default"
	}

	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal themis evaluate request: %w", err)
	}

	url := strings.TrimRight(t.client.config.ThemisURL, "/") + "/v1/policies/evaluate"

	httpResp, err := executeWithRetry(ctx, t.client.config.RetryConfig, func(opCtx context.Context) (*http.Response, error) {
		httpReq, err := http.NewRequestWithContext(opCtx, http.MethodPost, url, bytes.NewReader(data))
		if err != nil {
			return nil, err
		}
		httpReq.Header.Set("Content-Type", "application/json")
		t.client.prepareRequest(opCtx, httpReq)
		return t.client.config.HTTPClient.Do(httpReq)
	})

	if err != nil {
		return nil, fmt.Errorf("themis policy evaluation failed: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("themis returned status code %d", httpResp.StatusCode)
	}

	var res EvaluatePolicyResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to decode themis response: %w", err)
	}

	return &res, nil
}
