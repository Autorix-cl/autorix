package cache

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// ErrNotFound is returned when a requested key does not exist or has expired.
var ErrNotFound = errors.New("cache: key not found")

// Cache defines the contract for key-value caching and distributed invalidation in Autorix engines.
type Cache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, keys ...string) error
	PublishInvalidation(ctx context.Context, channel, key string) error
	SubscribeInvalidations(ctx context.Context, channel string, handler func(key string)) error
	Close() error
}

// MemoryCache is a high-performance in-memory cache with per-key TTL.
type MemoryCache struct {
	mu    sync.RWMutex
	items map[string]memItem
}

type memItem struct {
	data      []byte
	expiresAt time.Time
}

// NewMemoryCache creates a new thread-safe memory cache.
func NewMemoryCache() *MemoryCache {
	return &MemoryCache{
		items: make(map[string]memItem),
	}
}

func (m *MemoryCache) Get(_ context.Context, key string) ([]byte, error) {
	m.mu.RLock()
	item, ok := m.items[key]
	m.mu.RUnlock()

	if !ok {
		return nil, ErrNotFound
	}

	if !item.expiresAt.IsZero() && time.Now().After(item.expiresAt) {
		m.mu.Lock()
		delete(m.items, key)
		m.mu.Unlock()
		return nil, ErrNotFound
	}

	return item.data, nil
}

func (m *MemoryCache) Set(_ context.Context, key string, value []byte, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var expiresAt time.Time
	if ttl > 0 {
		expiresAt = time.Now().Add(ttl)
	}

	m.items[key] = memItem{
		data:      value,
		expiresAt: expiresAt,
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

// New creates a Cache instance. If redisURL is empty, it returns an in-memory cache for standalone/test environments.
// If redisURL is provided, it connects to Redis, hardens the connection pool topology, and validates connectivity.
// If Redis is unreachable, it returns an explicit error to enforce fail-closed security and prevent split-brain clusters.
func New(ctx context.Context, redisURL string) (Cache, error) {
	if redisURL == "" {
		return NewMemoryCache(), nil
	}

	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("cache: invalid redis URL %q: %w", redisURL, err)
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
		return nil, fmt.Errorf("cache: failed to connect to configured redis at %s: %w", redisURL, err)
	}

	return NewRedisCache(client), nil
}
