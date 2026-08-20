package main

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

type ProviderConfig struct {
	Endpoint string
	Token    string
	Timeout  time.Duration
}

type Client struct {
	config     ProviderConfig
	httpClient *http.Client
}

func NewClient(cfg ProviderConfig) *Client {
	if cfg.Timeout == 0 {
		cfg.Timeout = 10 * time.Second
	}
	return &Client{
		config: cfg,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

// Resource: Environment
type EnvironmentResource struct {
	ID        string `json:"id"`
	ProjectID string `json:"project_id"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	Type      string `json:"type"`
}

// Resource: Policy
type PolicyResource struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Expression string `json:"expression"`
	Priority   int    `json:"priority"`
	Action     string `json:"action"`
}

// Resource: OAuth2Client
type OAuth2ClientResource struct {
	ClientID     string   `json:"client_id"`
	ClientName   string   `json:"client_name"`
	GrantTypes   []string `json:"grant_types"`
	RedirectURIs []string `json:"redirect_uris"`
	Scopes       []string `json:"scopes"`
}

func (c *Client) Ping(ctx context.Context) error {
	if c.config.Endpoint == "" {
		return fmt.Errorf("provider endpoint is required")
	}
	return nil
}
