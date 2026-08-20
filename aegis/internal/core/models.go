package core

import (
	"net/http"
	"time"
)

// MatchConfig defines rule triggering criteria
type MatchConfig struct {
	URL     string   `yaml:"url" json:"url"`
	Methods []string `yaml:"methods" json:"methods"`
}

// HandlerConfig defines a generic handler invocation with arbitrary configuration
type HandlerConfig struct {
	Handler string                 `yaml:"handler" json:"handler"`
	Config  map[string]interface{} `yaml:"config,omitempty" json:"config,omitempty"`
}

// UpstreamConfig defines the target backend service
type UpstreamConfig struct {
	URL         string `yaml:"url" json:"url"`
	StripPrefix string `yaml:"strip_prefix,omitempty" json:"strip_prefix,omitempty"`
	Rewrite     string `yaml:"rewrite,omitempty" json:"rewrite,omitempty"`
}

// Rule defines an Oathkeeper-style declarative routing & security policy
type Rule struct {
	ID             string          `yaml:"id" json:"id"`
	Description    string          `yaml:"description,omitempty" json:"description,omitempty"`
	OrderIdx       int             `yaml:"order_idx,omitempty" json:"order_idx,omitempty"`
	Match          MatchConfig     `yaml:"match" json:"match"`
	Authenticators []HandlerConfig `yaml:"authenticators" json:"authenticators"`
	Authorizer     HandlerConfig   `yaml:"authorizer" json:"authorizer"`
	Mutators       []HandlerConfig `yaml:"mutators" json:"mutators"`
	Upstream       UpstreamConfig  `yaml:"upstream" json:"upstream"`
}

// RuleVersion represents a versioned snapshot of all security rules.
type RuleVersion struct {
	Version     int       `json:"version"`
	Description string    `json:"description,omitempty"`
	Rules       []Rule    `json:"rules"`
	CreatedAt   time.Time `json:"created_at"`
}

// Session represents the authenticated identity context traversing the pipeline
type Session struct {
	Subject string                 `json:"subject"`
	Scopes  []string               `json:"scopes"`
	Extra   map[string]interface{} `json:"extra"`
	Headers http.Header            `json:"headers"`
}

// Authenticator interface validates incoming requests and extracts an identity Session
type Authenticator interface {
	Name() string
	Authenticate(r *http.Request, config map[string]interface{}) (*Session, error)
}

// Authorizer interface checks permissions for the Session
type Authorizer interface {
	Name() string
	Authorize(r *http.Request, session *Session, config map[string]interface{}) (bool, error)
}

// Mutator interface alters outgoing headers before forwarding to upstream
type Mutator interface {
	Name() string
	Mutate(r *http.Request, session *Session, config map[string]interface{}) error
}

// PipelineTraceStep records one step in a dry-run pipeline trace.
type PipelineTraceStep struct {
	Stage          string      `json:"stage"` // "match", "authenticator", "authorizer", "mutator", "upstream"
	Handler        string      `json:"handler,omitempty"`
	Status         string      `json:"status"` // "success", "failure", "skipped"
	Details        string      `json:"details,omitempty"`
	Allowed        *bool       `json:"allowed,omitempty"`
	Session        *Session    `json:"session,omitempty"`
	MutatedHeaders http.Header `json:"mutated_headers,omitempty"`
	TargetURL      string      `json:"target_url,omitempty"`
	Error          string      `json:"error,omitempty"`
}

// PipelineTrace contains the step-by-step trace of dry-run execution.
type PipelineTrace struct {
	MatchedRuleID string              `json:"matched_rule_id,omitempty"`
	Steps         []PipelineTraceStep `json:"steps"`
	FinalVerdict  string              `json:"final_verdict"` // "allow", "deny", "unauthorized", "error"
	Error         string              `json:"error,omitempty"`
}

// HandlerInfo describes an authenticator, authorizer, or mutator handler schema.
type HandlerInfo struct {
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	ConfigSchema map[string]interface{} `json:"config_schema"`
}

// HandlerCatalogue provides schema and capability metadata for available handlers.
type HandlerCatalogue struct {
	Authenticators []HandlerInfo `json:"authenticators"`
	Authorizers    []HandlerInfo `json:"authorizers"`
	Mutators       []HandlerInfo `json:"mutators"`
}
