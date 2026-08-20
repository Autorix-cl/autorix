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

type checkPayload struct {
	Namespace string                 `json:"namespace"`
	Object    string                 `json:"object"`
	Relation  string                 `json:"relation"`
	Subject   string                 `json:"subject"`
	Context   map[string]interface{} `json:"context,omitempty"`
}

type checkResponse struct {
	Allowed bool   `json:"allowed"`
	Reason  string `json:"reason,omitempty"`
}

// Check evaluates whether a subject has a relation on a namespace:object
func (c *Client) Check(
	ctx context.Context,
	namespace string,
	object string,
	relation string,
	subject string,
	requestContext map[string]interface{},
) (bool, error) {
	if subject == "" || namespace == "" || object == "" || relation == "" {
		return false, fmt.Errorf("invalid check arguments: missing subject, namespace, object, or relation")
	}

	cacheKey := fmt.Sprintf("%s:%s#%s@%s", namespace, object, relation, subject)

	// 1. Local Cache Lookup
	if c.config.EnableCache {
		c.cache.mu.RLock()
		if item, found := c.cache.items[cacheKey]; found {
			if time.Now().Before(item.expiresAt) {
				c.cache.mu.RUnlock()
				return item.allowed, nil
			}
		}
		c.cache.mu.RUnlock()
	}

	// 2. Direct evaluation against Nexus Engine (Fail-Closed)
	allowed := false

	if c.config.NexusURL != "" {
		url := strings.TrimRight(c.config.NexusURL, "/") + "/check"
		payload := checkPayload{
			Namespace: namespace,
			Object:    object,
			Relation:  relation,
			Subject:   subject,
			Context:   requestContext,
		}
		data, err := json.Marshal(payload)
		if err != nil {
			return false, fmt.Errorf("failed to marshal check payload: %w", err)
		}

		req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
		if err != nil {
			return false, fmt.Errorf("failed to create check request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.config.HTTPClient.Do(req)
		if err != nil {
			// Fail-closed on network error
			return false, fmt.Errorf("failed to reach nexus authorization engine: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return false, fmt.Errorf("nexus returned non-200 status code: %d", resp.StatusCode)
		}

		var checkRes checkResponse
		if err := json.NewDecoder(resp.Body).Decode(&checkRes); err != nil {
			return false, fmt.Errorf("failed to decode nexus check response: %w", err)
		}
		allowed = checkRes.Allowed
	} else {
		// Standalone test/mock mode without active nexus endpoint
		allowed = true
	}

	// 3. Cache Storage
	if c.config.EnableCache {
		c.cache.mu.Lock()
		c.cache.items[cacheKey] = cacheItem{
			allowed:   allowed,
			expiresAt: time.Now().Add(c.config.CacheTTL),
		}
		c.cache.mu.Unlock()
	}

	return allowed, nil
}
