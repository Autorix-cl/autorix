package cache

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// ErrNotFound is returned when a requested key does not exist or has expired.
var ErrNotFound = errors.New("cache: key not found")

const defaultMemoryCacheMaxEntries = 10000

// Cache defines the contract for key-value caching and distributed invalidation in Autorix engines.
type Cache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, keys ...string) error
	PublishInvalidation(ctx context.Context, channel, key string) error
	SubscribeInvalidations(ctx context.Context, channel string, handler func(key string)) error
	Close() error
}

// MemoryCache is a bounded, thread-safe in-memory cache with per-key TTL.
type MemoryCache struct {
	mu         sync.RWMutex
	items      map[string]memItem
	maxEntries int
	seq        uint64
}

type memItem struct {
	data      []byte
	expiresAt time.Time
	seq       uint64
}

// NewMemoryCache creates a new thread-safe memory cache with the default maximum capacity.
func NewMemoryCache() *MemoryCache {
	return NewMemoryCacheWithLimit(defaultMemoryCacheMaxEntries)
}

// NewMemoryCacheWithLimit creates a new thread-safe memory cache bounded to maxEntries items.
func NewMemoryCacheWithLimit(maxEntries int) *MemoryCache {
	if maxEntries <= 0 {
		maxEntries = defaultMemoryCacheMaxEntries
	}
	return &MemoryCache{
		items:      make(map[string]memItem),
		maxEntries: maxEntries,
	}
}

func (m *MemoryCache) Get(_ context.Context, key string) ([]byte, error) {
	now := time.Now()

	m.mu.RLock()
	item, ok := m.items[key]
	if !ok {
		m.mu.RUnlock()
		return nil, ErrNotFound
	}
	if item.expiresAt.IsZero() || !now.After(item.expiresAt) {
		out := make([]byte, len(item.data))
		copy(out, item.data)
		m.mu.RUnlock()
		return out, nil
	}
	m.mu.RUnlock()

	// Re-verify expiration under write lock before deleting to avoid racing with a concurrent Set.
	m.mu.Lock()
	current, exists := m.items[key]
	if !exists {
		m.mu.Unlock()
		return nil, ErrNotFound
	}
	if !current.expiresAt.IsZero() && time.Now().After(current.expiresAt) {
		delete(m.items, key)
		m.mu.Unlock()
		return nil, ErrNotFound
	}
	out := make([]byte, len(current.data))
	copy(out, current.data)
	m.mu.Unlock()
	return out, nil
}

func (m *MemoryCache) Set(_ context.Context, key string, value []byte, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	var expiresAt time.Time
	if ttl > 0 {
		expiresAt = now.Add(ttl)
	}

	if _, exists := m.items[key]; !exists && m.maxEntries > 0 && len(m.items) >= m.maxEntries {
		// Evict expired entries first.
		for k, it := range m.items {
			if !it.expiresAt.IsZero() && now.After(it.expiresAt) {
				delete(m.items, k)
			}
		}
		// If still at capacity, evict the oldest entry.
		if len(m.items) >= m.maxEntries {
			var oldestKey string
			var oldestSeq uint64
			first := true
			for k, it := range m.items {
				if first || it.seq < oldestSeq {
					oldestKey = k
					oldestSeq = it.seq
					first = false
				}
			}
			if !first {
				delete(m.items, oldestKey)
			}
		}
	}

	stored := make([]byte, len(value))
	copy(stored, value)

	m.seq++
	m.items[key] = memItem{
		data:      stored,
		expiresAt: expiresAt,
		seq:       m.seq,
	}
	return nil
}

func (m *MemoryCache) Delete(_ context.Context, keys ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, k := range keys {
		delete(m.items, k)
	}
	return nil
}

func (m *MemoryCache) PublishInvalidation(ctx context.Context, channel, key string) error {
	return m.Delete(ctx, key)
}

func (m *MemoryCache) SubscribeInvalidations(_ context.Context, _ string, _ func(key string)) error {
	// Standalone in-memory cache does not require network pub/sub
	return nil
}

func (m *MemoryCache) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items = make(map[string]memItem)
	return nil
}

// RedisCache wraps a Redis client with resilient error handling.
type RedisCache struct {
	client *redis.Client
}

// NewRedisCache creates a new RedisCache from an existing go-redis client.
func NewRedisCache(client *redis.Client) *RedisCache {
	return &RedisCache{client: client}
}

// Client returns the underlying go-redis client for metrics/pool inspection.
func (r *RedisCache) Client() *redis.Client {
	return r.client
}

func (r *RedisCache) Get(ctx context.Context, key string) ([]byte, error) {
	val, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("redis get: %w", err)
	}
	return val, nil
}

func (r *RedisCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return r.client.Set(ctx, key, value, ttl).Err()
}

func (r *RedisCache) Delete(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	return r.client.Del(ctx, keys...).Err()
}

func (r *RedisCache) PublishInvalidation(ctx context.Context, channel, key string) error {
	return r.client.Publish(ctx, channel, key).Err()
}

func (r *RedisCache) SubscribeInvalidations(ctx context.Context, channel string, handler func(key string)) error {
	pubsub := r.client.Subscribe(ctx, channel)
	go func() {
		defer pubsub.Close()
		ch := pubsub.Channel()
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-ch:
				if !ok {
					return
				}
				if msg != nil && msg.Payload != "" {
					handler(msg.Payload)
				}
			}
		}
	}()
	return nil
}

func (r *RedisCache) Close() error {
	return r.client.Close()
}

func redactRedisURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return "<redacted>"
	}
	if u.User != nil {
		u.User = url.User("redacted")
	}
	if u.RawQuery != "" {
		q := u.Query()
		for k := range q {
			lower := strings.ToLower(k)
			if strings.Contains(lower, "pass") || strings.Contains(lower, "secret") || strings.Contains(lower, "token") || strings.Contains(lower, "auth") {
				q.Set(k, "redacted")
			}
		}
		u.RawQuery = q.Encode()
	}
	return u.String()
}

// New creates a Cache instance. If redisURL is empty, it returns an in-memory cache for standalone/test environments.
// If redisURL is provided, it connects to Redis, hardens the connection pool topology, and validates connectivity.
// If Redis is unreachable, it returns an explicit error to enforce fail-closed security and prevent split-brain clusters.
func New(ctx context.Context, redisURL string) (Cache, error) {
	if redisURL == "" {
		return NewMemoryCache(), nil
	}

	redactedURL := redactRedisURL(redisURL)

	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		var urlErr *url.Error
		if errors.As(err, &urlErr) {
			return nil, fmt.Errorf("cache: invalid redis URL %q: %w", redactedURL, urlErr.Err)
		}
		return nil, fmt.Errorf("cache: invalid redis URL %q", redactedURL)
	}

	// Network Timeouts
	opt.DialTimeout = 2 * time.Second
	opt.ReadTimeout = 1 * time.Second
	opt.WriteTimeout = 1 * time.Second

	// Connection Pool Hardening
	opt.PoolSize = 50
	opt.MinIdleConns = 10
	opt.ConnMaxLifetime = 30 * time.Minute
	opt.PoolTimeout = 4 * time.Second
	opt.ConnMaxIdleTime = 5 * time.Minute

	client := redis.NewClient(opt)

	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	if err := client.Ping(pingCtx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("cache: failed to connect to configured redis at %s: %w", redactedURL, err)
	}

	return NewRedisCache(client), nil
}
