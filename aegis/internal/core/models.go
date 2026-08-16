package core

import (
	"net/http"
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
	URL string `yaml:"url" json:"url"`
}

// Rule defines an Oathkeeper-style declarative routing & security policy
type Rule struct {
	ID             string          `yaml:"id" json:"id"`
	Description    string          `yaml:"description,omitempty" json:"description,omitempty"`
	Match          MatchConfig     `yaml:"match" json:"match"`
	Authenticators []HandlerConfig `yaml:"authenticators" json:"authenticators"`
	Authorizer     HandlerConfig   `yaml:"authorizer" json:"authorizer"`
	Mutators       []HandlerConfig `yaml:"mutators" json:"mutators"`
	Upstream       UpstreamConfig  `yaml:"upstream" json:"upstream"`
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
