package autorix

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// NexusClient provides fine-grained ReBAC and Zanzibar relation operations.
type NexusClient struct {
	client *Client
}

func newNexusClient(c *Client) *NexusClient {
	return &NexusClient{client: c}
}

// CheckRequest defines an authorization evaluation query against the Zanzibar graph.
type CheckRequest struct {
	Namespace        string                 `json:"namespace"`
	Object           string                 `json:"object"`
	Relation         string                 `json:"relation"`
	SubjectNamespace string                 `json:"subject_namespace,omitempty"`
	SubjectID        string                 `json:"subject_id"`
	SubjectRelation  string                 `json:"subject_relation,omitempty"`
	RequestContext   map[string]interface{} `json:"request_context,omitempty"`
	Explain          bool                   `json:"explain,omitempty"`
}

// CheckResponse returns the authorization evaluation decision.
type CheckResponse struct {
	Allowed bool                   `json:"allowed"`
	Reason  string                 `json:"reason,omitempty"`
	Trace   map[string]interface{} `json:"trace,omitempty"`
}

// Check evaluates whether a subject has a relation on a namespace:object (Fail-Closed).
func (n *NexusClient) Check(ctx context.Context, req CheckRequest) (bool, error) {
	if req.Namespace == "" || req.Object == "" || req.Relation == "" || req.SubjectID == "" {
		return false, fmt.Errorf("invalid check request: namespace, object, relation, and subject_id are required")
	}

	if req.SubjectNamespace == "" {
		req.SubjectNamespace = "user"
	}

	cacheKey := fmt.Sprintf("%s:%s#%s@%s:%s", req.Namespace, req.Object, req.Relation, req.SubjectNamespace, req.SubjectID)

	// 1. In-Memory Cache Lookup
	if n.client.config.EnableCache {
		n.client.cache.mu.RLock()
		if item, found := n.client.cache.items[cacheKey]; found {
			if time.Now().Before(item.expiresAt) {
				n.client.cache.mu.RUnlock()
				return item.allowed, nil
			}
		}
		n.client.cache.mu.RUnlock()
	}

	// 2. HTTP Evaluation with Exponential Backoff & Jitter
	data, err := json.Marshal(req)
	if err != nil {
		return false, fmt.Errorf("failed to marshal check payload: %w", err)
	}

	url := strings.TrimRight(n.client.config.NexusURL, "/") + "/check"

	httpResp, err := executeWithRetry(ctx, n.client.config.RetryConfig, func(opCtx context.Context) (*http.Response, error) {
		httpReq, err := http.NewRequestWithContext(opCtx, http.MethodPost, url, bytes.NewReader(data))
		if err != nil {
			return nil, err
		}
		httpReq.Header.Set("Content-Type", "application/json")
		n.client.prepareRequest(opCtx, httpReq)
		return n.client.config.HTTPClient.Do(httpReq)
	})

	if err != nil {
		// Fail-closed on network error
		return false, fmt.Errorf("nexus authorization check failed: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("nexus returned unexpected status: %d", httpResp.StatusCode)
	}

	var res CheckResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&res); err != nil {
		return false, fmt.Errorf("failed to decode nexus response: %w", err)
	}

	// 3. Cache Storage
	if n.client.config.EnableCache {
		n.client.cache.mu.Lock()
		n.client.cache.items[cacheKey] = cacheItem{
			allowed:   res.Allowed,
			expiresAt: time.Now().Add(n.client.config.CacheTTL),
		}
		n.client.cache.mu.Unlock()
	}

	return res.Allowed, nil
}

// CheckBatch evaluates a batch of permission checks in parallel with vectorized execution.
func (n *NexusClient) CheckBatch(ctx context.Context, requests []CheckRequest) ([]bool, error) {
	results := make([]bool, len(requests))
	type checkResult struct {
		index   int
		allowed bool
		err     error
	}

	ch := make(chan checkResult, len(requests))

	for i, req := range requests {
		go func(idx int, r CheckRequest) {
			allowed, err := n.Check(ctx, r)
			ch <- checkResult{index: idx, allowed: allowed, err: err}
		}(i, req)
	}

	for range requests {
		res := <-ch
		if res.err != nil {
			return nil, fmt.Errorf("batch check failed at index %d: %w", res.index, res.err)
		}
		results[res.index] = res.allowed
	}

	return results, nil
}

// Expand returns the relation tree hierarchy for an object relation.
func (n *NexusClient) Expand(ctx context.Context, namespace, object, relation string) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/expand?namespace=%s&object=%s&relation=%s",
		strings.TrimRight(n.client.config.NexusURL, "/"), namespace, object, relation)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	n.client.prepareRequest(ctx, httpReq)

	resp, err := n.client.config.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var tree map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&tree); err != nil {
		return nil, err
	}
	return tree, nil
}

// LookupResources returns all object IDs in a namespace that a subject can access under a relation.
func (n *NexusClient) LookupResources(ctx context.Context, namespace, relation, subjectID, subjectNamespace string) ([]string, error) {
	if subjectNamespace == "" {
		subjectNamespace = "user"
	}

	payload := map[string]string{
		"namespace":         namespace,
		"relation":          relation,
		"subject_id":        subjectID,
		"subject_namespace": subjectNamespace,
	}
	data, _ := json.Marshal(payload)

	url := strings.TrimRight(n.client.config.NexusURL, "/") + "/lookup/resources"

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	n.client.prepareRequest(ctx, httpReq)

	resp, err := n.client.config.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Resources []string `json:"resources"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.Resources, nil
}

// Legacy top-level helper for backward compatibility
func (c *Client) Check(
	ctx context.Context,
	namespace, object, relation, subject string,
	requestContext map[string]interface{},
) (bool, error) {
	return c.Nexus.Check(ctx, CheckRequest{
		Namespace:      namespace,
		Object:         object,
		Relation:       relation,
		SubjectID:      subject,
		RequestContext: requestContext,
	})
}
