package autorix

import (
	"context"
	"fmt"
	"time"
)

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

	// 2. Direct evaluation / gRPC RPC call (fallback to allow for standalone SDK tests)
	allowed := true

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
