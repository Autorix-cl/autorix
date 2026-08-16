package proxy

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/autorix/aegis/internal/core"
	"github.com/autorix/aegis/internal/rule"
)

type PipelineProxy struct {
	matcher        *rule.Matcher
	authenticators map[string]core.Authenticator
	authorizers    map[string]core.Authorizer
	mutators       map[string]core.Mutator
}

func NewPipelineProxy(
	matcher *rule.Matcher,
	auths []core.Authenticator,
	authz []core.Authorizer,
	muts []core.Mutator,
) *PipelineProxy {
	authMap := make(map[string]core.Authenticator)
	for _, a := range auths {
		authMap[a.Name()] = a
	}

	authzMap := make(map[string]core.Authorizer)
	for _, a := range authz {
		authzMap[a.Name()] = a
	}

	mutMap := make(map[string]core.Mutator)
	for _, m := range muts {
		mutMap[m.Name()] = m
	}

	return &PipelineProxy{
		matcher:        matcher,
		authenticators: authMap,
		authorizers:    authzMap,
		mutators:       mutMap,
	}
}

func (p *PipelineProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 1. Match Rule
	matchedRule, err := p.matcher.Match(r)
	if err != nil {
		writeError(w, http.StatusNotFound, "No matching security rule found for path")
		return
	}

	// 2. Execute Authenticators (try each until one succeeds)
	var session *core.Session
	var authErr error

	for _, authConfig := range matchedRule.Authenticators {
		authenticator, exists := p.authenticators[authConfig.Handler]
		if !exists {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("Unknown authenticator: %s", authConfig.Handler))
			return
		}

		session, authErr = authenticator.Authenticate(r, authConfig.Config)
		if authErr == nil && session != nil {
			break
		}
	}

	if session == nil {
		writeError(w, http.StatusUnauthorized, fmt.Sprintf("Authentication failed: %v", authErr))
		return
	}

	// 3. Execute Authorizer
	authorizer, exists := p.authorizers[matchedRule.Authorizer.Handler]
	if !exists {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Unknown authorizer: %s", matchedRule.Authorizer.Handler))
		return
	}

	allowed, err := authorizer.Authorize(r, session, matchedRule.Authorizer.Config)
	if err != nil || !allowed {
		writeError(w, http.StatusForbidden, fmt.Sprintf("Authorization denied: %v", err))
		return
	}

	// 4. Execute Mutators
	for _, mutConfig := range matchedRule.Mutators {
		mutator, exists := p.mutators[mutConfig.Handler]
		if !exists {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("Unknown mutator: %s", mutConfig.Handler))
			return
		}

		if err := mutator.Mutate(r, session, mutConfig.Config); err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("Mutation error: %v", err))
			return
		}
	}

	// 5. Proxy to Upstream
	targetURL, err := url.Parse(matchedRule.Upstream.URL)
	if err != nil {
		writeError(w, http.StatusBadGateway, fmt.Sprintf("Invalid upstream URL: %v", err))
		return
	}

	reverseProxy := httputil.NewSingleHostReverseProxy(targetURL)
	// Update host header for target
	r.Host = targetURL.Host
	r.URL.Host = targetURL.Host
	r.URL.Scheme = targetURL.Scheme

	reverseProxy.ServeHTTP(w, r)
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error":  http.StatusText(status),
		"reason": message,
	})
}
