package autorix

import (
	"context"
	"net/http"
	"sync"
	"time"
)

type contextKey string

const (
	userCtxKey         contextKey = "autorix_user"
	requestIDCtxKey    contextKey = "autorix_request_id"
	correlationIDCtxKey contextKey = "autorix_correlation_id"
)

// Config contains comprehensive connection parameters for all Autorix engines.
type Config struct {
	// Endpoint URLs
	BaseURL   string // Global gateway or Aegis PEP proxy (e.g. "http://localhost:4455")
	NexusURL  string // Nexus Zanzibar ReBAC engine (e.g. "http://localhost:8080")
	ThemisURL string // Themis ABAC CEL policy engine (e.g. "http://localhost:4488")
	EgoURL    string // Ego Identity engine (e.g. "http://localhost:4433")
	JanusURL  string // Janus OAuth2/OIDC server (e.g. "http://localhost:4444")
	VulcanURL string // Vulcan API Keys & Macaroon engine (e.g. "http://localhost:4466")
	HermesURL string // Hermes SAML & SCIM bridge (e.g. "http://localhost:4477")
	ArgusURL  string // Argus Control Plane & Governance (e.g. "http://localhost:4400")

	// Authentication Credentials
	APIKey       string // Vulcan API Key ("av_live_..." or "av_test_...")
	SessionToken string // Argus Operator / Ego Session Token ("ast_...")

	// Resilience & Performance
	RetryConfig RetryConfig   // Exponential backoff and full jitter configuration
	EnableCache bool          // In-memory cache for authorization decisions
	CacheTTL    time.Duration // Cache lifespan (default: 10s)
	HTTPClient  *http.Client  // Underlying HTTP client
}

// User represents an authenticated identity propagated from Aegis or Ego.
type User struct {
	ID        string                 `json:"id"`
	Email     string                 `json:"email,omitempty"`
	Roles     []string               `json:"roles,omitempty"`
	Traits    map[string]interface{} `json:"traits,omitempty"`
	IsMachine bool                   `json:"is_machine,omitempty"`
}

// Client is the primary enterprise SDK interface for microservices.
type Client struct {
	config Config
	cache  *cacheStore

	// Sub-clients for domain-specific operations
	Nexus  *NexusClient
	Themis *ThemisClient
	Ego    *EgoClient
	Janus  *JanusClient
	Vulcan *VulcanClient
	Argus  *ArgusClient
}

type cacheStore struct {
	mu    sync.RWMutex
	items map[string]cacheItem
}

type cacheItem struct {
	allowed   bool
	expiresAt time.Time
}

// Option configures the Autorix Client using functional options.
type Option func(*Config)

// WithBaseURL sets the base gateway URL.
func WithBaseURL(url string) Option {
	return func(c *Config) { c.BaseURL = url }
}

// WithAPIKey sets the client Vulcan API key.
func WithAPIKey(key string) Option {
	return func(c *Config) { c.APIKey = key }
}

// WithRetryConfig customizes the retry policy.
func WithRetryConfig(rc RetryConfig) Option {
	return func(c *Config) { c.RetryConfig = rc }
}

// WithCache enables in-memory caching with specified TTL.
func WithCache(ttl time.Duration) Option {
	return func(c *Config) {
		c.EnableCache = true
		c.CacheTTL = ttl
	}
}

// NewClient initializes a new Autorix Go SDK instance with enterprise defaults.
func NewClient(cfg Config, opts ...Option) *Client {
	for _, opt := range opts {
		opt(&cfg)
	}

	if cfg.BaseURL == "" {
		cfg.BaseURL = "http://localhost:4455"
	}
	if cfg.NexusURL == "" {
		cfg.NexusURL = "http://localhost:8080"
	}
	if cfg.ThemisURL == "" {
		cfg.ThemisURL = "http://localhost:4488"
	}
	if cfg.EgoURL == "" {
		cfg.EgoURL = "http://localhost:4433"
	}
	if cfg.JanusURL == "" {
		cfg.JanusURL = "http://localhost:4444"
	}
	if cfg.VulcanURL == "" {
		cfg.VulcanURL = "http://localhost:4466"
	}
	if cfg.HermesURL == "" {
		cfg.HermesURL = "http://localhost:4477"
	}
	if cfg.ArgusURL == "" {
		cfg.ArgusURL = "http://localhost:4400"
	}

	if cfg.CacheTTL == 0 {
		cfg.CacheTTL = 10 * time.Second
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: 10 * time.Second}
	}
	if cfg.RetryConfig.MaxRetries == 0 && cfg.RetryConfig.InitialDelay == 0 {
		cfg.RetryConfig = DefaultRetryConfig()
	}

	c := &Client{
		config: cfg,
		cache: &cacheStore{
			items: make(map[string]cacheItem),
		},
	}

	c.Nexus = newNexusClient(c)
	c.Themis = newThemisClient(c)
	c.Ego = newEgoClient(c)
	c.Janus = newJanusClient(c)
	c.Vulcan = newVulcanClient(c)
	c.Argus = newArgusClient(c)

	return c
}

// UserFromContext extracts the authenticated User from request context.
func UserFromContext(ctx context.Context) (*User, bool) {
	u, ok := ctx.Value(userCtxKey).(*User)
	return u, ok && u != nil
}

// WithUser embeds the User inside context.
func WithUser(ctx context.Context, u *User) context.Context {
	return context.WithValue(ctx, userCtxKey, u)
}

// prepareRequest attaches authentication headers and W3C distributed tracing context.
func (c *Client) prepareRequest(ctx context.Context, req *http.Request) {
	req.Header.Set("User-Agent", "Autorix-Go-SDK/1.0.0")

	// Inject API Key if present
	if c.config.APIKey != "" && req.Header.Get("Authorization") == "" {
		req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	}

	// Propagate Correlation / Request IDs
	if reqID, ok := ctx.Value(requestIDCtxKey).(string); ok && reqID != "" {
		req.Header.Set("X-Request-ID", reqID)
	}
	if corrID, ok := ctx.Value(correlationIDCtxKey).(string); ok && corrID != "" {
		req.Header.Set("X-Correlation-ID", corrID)
	}
}
