package proxy

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/autorix/aegis/internal/core"
)

// RuleMatcher is satisfied by both *rule.Matcher (static, YAML-only) and
// *rule.Store (hot-reloadable, backs the admin API), so the proxy pipeline
// doesn't care which one is wired up in main.go.
type RuleMatcher interface {
	Match(r *http.Request) (*core.Rule, error)
}

type PipelineProxy struct {
	matcher        RuleMatcher
	authenticators map[string]core.Authenticator
	authorizers    map[string]core.Authorizer
	mutators       map[string]core.Mutator
}

func NewPipelineProxy(
	matcher RuleMatcher,
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

// BuildUpstreamURL calculates the target upstream URL applying strip_prefix and rewrite options.
func BuildUpstreamURL(reqURL *url.URL, matchURL string, upstream core.UpstreamConfig) (*url.URL, error) {
	targetURL, err := url.Parse(upstream.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid upstream URL %q: %w", upstream.URL, err)
	}

	path := reqURL.Path
	if upstream.StripPrefix != "" {
		if strings.HasPrefix(path, upstream.StripPrefix) {
			path = strings.TrimPrefix(path, upstream.StripPrefix)
			if path == "" || !strings.HasPrefix(path, "/") {
				path = "/" + strings.TrimPrefix(path, "/")
			}
		}
	}

	if upstream.Rewrite != "" {
		pattern := matchURL
		if !strings.HasPrefix(pattern, "^") {
			pattern = "^" + pattern
		}
		if !strings.HasSuffix(pattern, "$") {
			pattern = pattern + "$"
		}
		pattern = strings.ReplaceAll(pattern, "<", "(")
		pattern = strings.ReplaceAll(pattern, ">", ")")

		reg, err := regexp.Compile(pattern)
		if err == nil {
			path = reg.ReplaceAllString(reqURL.Path, upstream.Rewrite)
		}
	}

	basePath := strings.TrimSuffix(targetURL.Path, "/")
	if basePath != "" {
		if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}
		path = basePath + path
	}

	result := *targetURL
	result.Path = path
	result.RawQuery = reqURL.RawQuery
	return &result, nil
}

func (p *PipelineProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 1. Match Rule
	startMatch := time.Now()
	matchedRule, err := p.matcher.Match(r)
	pipelineStageDuration.WithLabelValues("match").Observe(time.Since(startMatch).Seconds())
	if err != nil {
		aegisRequestsTotal.WithLabelValues(strconv.Itoa(http.StatusNotFound)).Inc()
		writeError(w, http.StatusNotFound, "No matching security rule found for path")
		return
	}

	// 2. Execute Authenticators (try each until one succeeds)
	startAuthn := time.Now()
	var session *core.Session
	var authErr error

	for _, authConfig := range matchedRule.Authenticators {
		authenticator, exists := p.authenticators[authConfig.Handler]
		if !exists {
			pipelineStageDuration.WithLabelValues("authn").Observe(time.Since(startAuthn).Seconds())
			aegisRequestsTotal.WithLabelValues(strconv.Itoa(http.StatusInternalServerError)).Inc()
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("Unknown authenticator: %s", authConfig.Handler))
			return
		}

		session, authErr = authenticator.Authenticate(r, authConfig.Config)
		if authErr == nil && session != nil {
			break
		}
	}
	pipelineStageDuration.WithLabelValues("authn").Observe(time.Since(startAuthn).Seconds())

	if session == nil {
		aegisRequestsTotal.WithLabelValues(strconv.Itoa(http.StatusUnauthorized)).Inc()
		writeError(w, http.StatusUnauthorized, fmt.Sprintf("Authentication failed: %v", authErr))
		return
	}

	// 3. Execute Authorizer
	startAuthz := time.Now()
	authorizer, exists := p.authorizers[matchedRule.Authorizer.Handler]
	if !exists {
		pipelineStageDuration.WithLabelValues("authz").Observe(time.Since(startAuthz).Seconds())
		aegisRequestsTotal.WithLabelValues(strconv.Itoa(http.StatusInternalServerError)).Inc()
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Unknown authorizer: %s", matchedRule.Authorizer.Handler))
		return
	}

	allowed, err := authorizer.Authorize(r, session, matchedRule.Authorizer.Config)
	pipelineStageDuration.WithLabelValues("authz").Observe(time.Since(startAuthz).Seconds())
	if err != nil || !allowed {
		aegisRequestsTotal.WithLabelValues(strconv.Itoa(http.StatusForbidden)).Inc()
		writeError(w, http.StatusForbidden, fmt.Sprintf("Authorization denied: %v", err))
		return
	}

	// 4. Execute Mutators
	startMutate := time.Now()
	for _, mutConfig := range matchedRule.Mutators {
		mutator, exists := p.mutators[mutConfig.Handler]
		if !exists {
			pipelineStageDuration.WithLabelValues("mutate").Observe(time.Since(startMutate).Seconds())
			aegisRequestsTotal.WithLabelValues(strconv.Itoa(http.StatusInternalServerError)).Inc()
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("Unknown mutator: %s", mutConfig.Handler))
			return
		}

		if err := mutator.Mutate(r, session, mutConfig.Config); err != nil {
			pipelineStageDuration.WithLabelValues("mutate").Observe(time.Since(startMutate).Seconds())
			aegisRequestsTotal.WithLabelValues(strconv.Itoa(http.StatusInternalServerError)).Inc()
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("Mutation error: %v", err))
			return
		}
	}
	pipelineStageDuration.WithLabelValues("mutate").Observe(time.Since(startMutate).Seconds())

	// 5. Proxy to Upstream
	targetURL, err := BuildUpstreamURL(r.URL, matchedRule.Match.URL, matchedRule.Upstream)
	if err != nil {
		aegisRequestsTotal.WithLabelValues(strconv.Itoa(http.StatusBadGateway)).Inc()
		writeError(w, http.StatusBadGateway, fmt.Sprintf("Invalid upstream URL: %v", err))
		return
	}

	startUpstream := time.Now()
	rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
	reverseProxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.Host = targetURL.Host
			req.URL.Scheme = targetURL.Scheme
			req.URL.Host = targetURL.Host
			req.URL.Path = targetURL.Path
			req.URL.RawQuery = targetURL.RawQuery
		},
	}

	reverseProxy.ServeHTTP(rec, r)
	pipelineStageDuration.WithLabelValues("upstream").Observe(time.Since(startUpstream).Seconds())
	aegisRequestsTotal.WithLabelValues(strconv.Itoa(rec.status)).Inc()
}

type statusRecorder struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (s *statusRecorder) WriteHeader(code int) {
	if !s.wroteHeader {
		s.status = code
		s.wroteHeader = true
	}
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusRecorder) Write(b []byte) (int, error) {
	if !s.wroteHeader {
		s.status = http.StatusOK
		s.wroteHeader = true
	}
	return s.ResponseWriter.Write(b)
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error":  http.StatusText(status),
		"reason": message,
	})
}
