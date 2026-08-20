package autorix

import (
	"context"
	"net/http"
	"sync"
	"time"
)

type contextKey string

const userCtxKey contextKey = "autorix_user"

// Config contains connection parameters for Autorix engines
type Config struct {
	NexusAddr   string        // e.g. "localhost:50051" (gRPC)
	NexusURL    string        // e.g. "http://localhost:8080" (REST HTTP)
	EgoURL      string        // e.g. "http://localhost:4433"
	JanusURL    string        // e.g. "http://localhost:4444"
	EnableCache bool          // Enable local in-memory cache for ReBAC checks
	CacheTTL    time.Duration // Cache validity duration (default: 10s)
	HTTPClient  *http.Client
}

// User represents an authenticated identity propagated from Aegis
type User struct {
	ID        string                 `json:"id"`
	Email     string                 `json:"email,omitempty"`
	Roles     []string               `json:"roles,omitempty"`
	Traits    map[string]interface{} `json:"traits,omitempty"`
	IsMachine bool                   `json:"is_machine,omitempty"`
}

// Client is the primary SDK interface for microservices
type Client struct {
	config Config
	cache  *cacheStore
}

type cacheStore struct {
	mu    sync.RWMutex
	items map[string]cacheItem
}

type cacheItem struct {
	allowed   bool
	expiresAt time.Time
}

// NewClient initializes a new Autorix Go SDK instance
func NewClient(cfg Config) *Client {
	if cfg.CacheTTL == 0 {
		cfg.CacheTTL = 10 * time.Second
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: 5 * time.Second}
	}

	return &Client{
		config: cfg,
		cache: &cacheStore{
			items: make(map[string]cacheItem),
		},
	}
}

// UserFromContext extracts the authenticated User from request context
func UserFromContext(ctx context.Context) (*User, bool) {
	u, ok := ctx.Value(userCtxKey).(*User)
	return u, ok && u != nil
}

// WithUser embeds the User inside context
func WithUser(ctx context.Context, u *User) context.Context {
	return context.WithValue(ctx, userCtxKey, u)
}
