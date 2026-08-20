package proxy

import (
	"fmt"
	"net/http"

	"github.com/autorix/aegis/internal/core"
)

// Catalogue returns metadata and JSON schemas for all registered authenticators, authorizers, and mutators.
func (p *PipelineProxy) Catalogue() core.HandlerCatalogue {
	cat := core.HandlerCatalogue{
		Authenticators: make([]core.HandlerInfo, 0, len(p.authenticators)),
		Authorizers:    make([]core.HandlerInfo, 0, len(p.authorizers)),
		Mutators:       make([]core.HandlerInfo, 0, len(p.mutators)),
	}

	for name := range p.authenticators {
		var desc string
		var schema map[string]interface{}
		switch name {
		case "jwt":
			desc = "Validates JSON Web Tokens using asymmetric public keys or remote JWKS"
			schema = map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"jwks_url": map[string]interface{}{
						"type":        "string",
						"description": "JWKS endpoint URL for key rotation",
					},
					"allowed_algorithms": map[string]interface{}{
						"type": "array",
						"items": map[string]interface{}{
							"type": "string",
						},
					},
				},
			}
		case "anonymous":
			desc = "Provides an anonymous identity session for public endpoints"
			schema = map[string]interface{}{
				"type":        "object",
				"description": "No configuration required",
			}
		case "noop":
			desc = "Pass-through authenticator without credentials verification"
			schema = map[string]interface{}{
				"type":        "object",
				"description": "No configuration required",
			}
		default:
			desc = fmt.Sprintf("Authenticator handler %s", name)
			schema = map[string]interface{}{"type": "object"}
		}

		cat.Authenticators = append(cat.Authenticators, core.HandlerInfo{
			Name:         name,
			Description:  desc,
			ConfigSchema: schema,
		})
	}

	for name := range p.authorizers {
		var desc string
		var schema map[string]interface{}
		switch name {
		case "allow":
			desc = "Explicitly authorizes all requests"
			schema = map[string]interface{}{
				"type":        "object",
				"description": "No configuration required",
			}
		case "deny":
			desc = "Explicitly rejects all requests"
			schema = map[string]interface{}{
				"type":        "object",
				"description": "No configuration required",
			}
		case "nexus_rebac":
			desc = "Evaluates relationship-based access control (ReBAC) policies via Autorix Nexus"
			schema = map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"namespace": map[string]interface{}{
						"type":        "string",
						"description": "Nexus namespace (e.g. document)",
					},
					"relation": map[string]interface{}{
						"type":        "string",
						"description": "Relation to evaluate (e.g. viewer, editor, owner)",
					},
					"object": map[string]interface{}{
						"type":        "string",
						"description": "Object identifier or template",
					},
				},
			}
		default:
			desc = fmt.Sprintf("Authorizer handler %s", name)
			schema = map[string]interface{}{"type": "object"}
		}

		cat.Authorizers = append(cat.Authorizers, core.HandlerInfo{
			Name:         name,
			Description:  desc,
			ConfigSchema: schema,
		})
	}

	for name := range p.mutators {
		var desc string
		var schema map[string]interface{}
		switch name {
		case "header":
			desc = "Injects and transforms outgoing HTTP headers using Go text/template"
			schema = map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"headers": map[string]interface{}{
						"type": "object",
						"additionalProperties": map[string]interface{}{
							"type": "string",
						},
						"description": "Map of header name to Go template string",
					},
				},
			}
		case "noop":
			desc = "Pass-through mutator leaving headers unchanged"
			schema = map[string]interface{}{
				"type":        "object",
				"description": "No configuration required",
			}
		default:
			desc = fmt.Sprintf("Mutator handler %s", name)
			schema = map[string]interface{}{"type": "object"}
		}

		cat.Mutators = append(cat.Mutators, core.HandlerInfo{
			Name:         name,
			Description:  desc,
			ConfigSchema: schema,
		})
	}

	return cat
}

// DryRun executes a full simulation of the pipeline for r without contacting the upstream server.
func (p *PipelineProxy) DryRun(r *http.Request) (*core.PipelineTrace, error) {
	trace := &core.PipelineTrace{
		Steps: make([]core.PipelineTraceStep, 0),
	}

	// Step 1: Matching
	matchedRule, err := p.matcher.Match(r)
	if err != nil {
		trace.Steps = append(trace.Steps, core.PipelineTraceStep{
			Stage:   "match",
			Status:  "failure",
			Details: "No matching security rule found for path",
			Error:   err.Error(),
		})
		trace.FinalVerdict = "error"
		trace.Error = err.Error()
		return trace, nil
	}

	trace.MatchedRuleID = matchedRule.ID
	trace.Steps = append(trace.Steps, core.PipelineTraceStep{
		Stage:   "match",
		Status:  "success",
		Details: fmt.Sprintf("Matched rule %s (pattern: %s)", matchedRule.ID, matchedRule.Match.URL),
	})

	// Step 2: Authenticators
	var session *core.Session
	var authErr error

	for _, authConfig := range matchedRule.Authenticators {
		authenticator, exists := p.authenticators[authConfig.Handler]
		if !exists {
			trace.Steps = append(trace.Steps, core.PipelineTraceStep{
				Stage:   "authenticator",
				Handler: authConfig.Handler,
				Status:  "failure",
				Error:   fmt.Sprintf("Unknown authenticator: %s", authConfig.Handler),
			})
			continue
		}

		session, authErr = authenticator.Authenticate(r, authConfig.Config)
		if authErr != nil || session == nil {
			errStr := ""
			if authErr != nil {
				errStr = authErr.Error()
			}
			trace.Steps = append(trace.Steps, core.PipelineTraceStep{
				Stage:   "authenticator",
				Handler: authConfig.Handler,
				Status:  "failure",
				Error:   errStr,
			})
		} else {
			trace.Steps = append(trace.Steps, core.PipelineTraceStep{
				Stage:   "authenticator",
				Handler: authConfig.Handler,
				Status:  "success",
				Session: session,
			})
			break
		}
	}

	if session == nil {
		trace.FinalVerdict = "unauthorized"
		trace.Error = fmt.Sprintf("Authentication failed: %v", authErr)
		return trace, nil
	}

	// Step 3: Authorizer
	authorizer, exists := p.authorizers[matchedRule.Authorizer.Handler]
	if !exists {
		trace.Steps = append(trace.Steps, core.PipelineTraceStep{
			Stage:   "authorizer",
			Handler: matchedRule.Authorizer.Handler,
			Status:  "failure",
			Error:   fmt.Sprintf("Unknown authorizer: %s", matchedRule.Authorizer.Handler),
		})
		trace.FinalVerdict = "error"
		trace.Error = fmt.Sprintf("Unknown authorizer: %s", matchedRule.Authorizer.Handler)
		return trace, nil
	}

	allowed, err := authorizer.Authorize(r, session, matchedRule.Authorizer.Config)
	if err != nil || !allowed {
		errStr := ""
		if err != nil {
			errStr = err.Error()
		}
		trace.Steps = append(trace.Steps, core.PipelineTraceStep{
			Stage:   "authorizer",
			Handler: matchedRule.Authorizer.Handler,
			Status:  "failure",
			Allowed: &allowed,
			Error:   errStr,
		})
		trace.FinalVerdict = "deny"
		trace.Error = fmt.Sprintf("Authorization denied: %v", err)
		return trace, nil
	}

	trace.Steps = append(trace.Steps, core.PipelineTraceStep{
		Stage:   "authorizer",
		Handler: matchedRule.Authorizer.Handler,
		Status:  "success",
		Allowed: &allowed,
	})

	// Step 4: Mutators
	mutatedReq := r.Clone(r.Context())
	for _, mutConfig := range matchedRule.Mutators {
		mutator, exists := p.mutators[mutConfig.Handler]
		if !exists {
			trace.Steps = append(trace.Steps, core.PipelineTraceStep{
				Stage:   "mutator",
				Handler: mutConfig.Handler,
				Status:  "failure",
				Error:   fmt.Sprintf("Unknown mutator: %s", mutConfig.Handler),
			})
			trace.FinalVerdict = "error"
			trace.Error = fmt.Sprintf("Unknown mutator: %s", mutConfig.Handler)
			return trace, nil
		}

		if err := mutator.Mutate(mutatedReq, session, mutConfig.Config); err != nil {
			trace.Steps = append(trace.Steps, core.PipelineTraceStep{
				Stage:   "mutator",
				Handler: mutConfig.Handler,
				Status:  "failure",
				Error:   err.Error(),
			})
			trace.FinalVerdict = "error"
			trace.Error = fmt.Sprintf("Mutation error: %v", err)
			return trace, nil
		}

		trace.Steps = append(trace.Steps, core.PipelineTraceStep{
			Stage:          "mutator",
			Handler:        mutConfig.Handler,
			Status:         "success",
			MutatedHeaders: mutatedReq.Header.Clone(),
		})
	}

	// Step 5: Upstream
	targetURL, err := BuildUpstreamURL(r.URL, matchedRule.Match.URL, matchedRule.Upstream)
	if err != nil {
		trace.Steps = append(trace.Steps, core.PipelineTraceStep{
			Stage:   "upstream",
			Status:  "failure",
			Error:   fmt.Sprintf("Invalid upstream URL: %v", err),
		})
		trace.FinalVerdict = "error"
		trace.Error = fmt.Sprintf("Invalid upstream URL: %v", err)
		return trace, nil
	}

	trace.Steps = append(trace.Steps, core.PipelineTraceStep{
		Stage:     "upstream",
		Status:    "success",
		TargetURL: targetURL.String(),
	})

	trace.FinalVerdict = "allow"
	return trace, nil
}
