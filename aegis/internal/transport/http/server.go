// Package http exposes the Aegis admin REST API used by the console to
// manage proxy routing rules. It runs on a separate port from the proxy
// pipeline itself (ServeMux in internal/proxy is a catch-all reverse proxy
// and can't share a mux with named admin routes).
package http

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/autorix/aegis/internal/core"
	"github.com/autorix/aegis/internal/rule"
	"github.com/autorix/platform/health"
	"github.com/autorix/platform/metrics"
	"github.com/autorix/platform/paging"
	"gopkg.in/yaml.v3"
)

// RuleStore is the interface needed by the admin API for security rule operations.
type RuleStore interface {
	List() []core.Rule
	ListPage(limit int, cursor string) ([]core.Rule, bool, error)
	Get(id string) (core.Rule, error)
	Create(r core.Rule) (core.Rule, error)
	Update(id string, r core.Rule) (core.Rule, error)
	Delete(id string) error
	TestMatch(method, path string) (*core.Rule, error)
	Reorder(ids []string) error
	Rollback(version int) error
	GetVersions() ([]core.RuleVersion, error)
	Import(rules []core.Rule) error
	Export() []core.Rule
}

// PipelineDryRunner provides dry-run simulation and handler metadata.
type PipelineDryRunner interface {
	DryRun(r *http.Request) (*core.PipelineTrace, error)
	Catalogue() core.HandlerCatalogue
}

type Server struct {
	store     RuleStore
	health    *health.Handler
	dryRunner PipelineDryRunner
}

func NewServer(store RuleStore, healthHandler *health.Handler, dryRunners ...PipelineDryRunner) *Server {
	var dr PipelineDryRunner
	if len(dryRunners) > 0 {
		dr = dryRunners[0]
	}
	return &Server{store: store, health: healthHandler, dryRunner: dr}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	if s.health != nil {
		s.health.Register(mux)
		mux.HandleFunc("GET /health", s.health.Alive)
	}

	mux.Handle("GET /metrics", metrics.Handler())
	mux.Handle("/metrics", metrics.Handler())

	// Standard endpoints
	mux.HandleFunc("GET /rules", s.handleListRules)
	mux.HandleFunc("POST /rules", s.handleCreateRule)
	mux.HandleFunc("GET /rules/{id}", s.handleGetRule)
	mux.HandleFunc("PUT /rules/{id}", s.handleUpdateRule)
	mux.HandleFunc("DELETE /rules/{id}", s.handleDeleteRule)
	mux.HandleFunc("PUT /rules/reorder", s.handleReorderRules)
	mux.HandleFunc("POST /rules/import", s.handleImportRules)
	mux.HandleFunc("GET /rules/export", s.handleExportRules)
	mux.HandleFunc("GET /rules/versions", s.handleGetVersions)
	mux.HandleFunc("POST /rules/rollback/{version}", s.handleRollback)
	mux.HandleFunc("GET /handlers", s.handleCatalogue)
	mux.HandleFunc("POST /rules/test-match", s.handleTestMatch)

	// Admin-prefixed aliases for full URL compatibility
	mux.HandleFunc("GET /admin/rules", s.handleListRules)
	mux.HandleFunc("POST /admin/rules", s.handleCreateRule)
	mux.HandleFunc("GET /admin/rules/{id}", s.handleGetRule)
	mux.HandleFunc("PUT /admin/rules/{id}", s.handleUpdateRule)
	mux.HandleFunc("DELETE /admin/rules/{id}", s.handleDeleteRule)
	mux.HandleFunc("PUT /admin/rules/reorder", s.handleReorderRules)
	mux.HandleFunc("POST /admin/rules/import", s.handleImportRules)
	mux.HandleFunc("GET /admin/rules/export", s.handleExportRules)
	mux.HandleFunc("GET /admin/rules/versions", s.handleGetVersions)
	mux.HandleFunc("POST /admin/rules/rollback/{version}", s.handleRollback)
	mux.HandleFunc("GET /admin/handlers", s.handleCatalogue)
	mux.HandleFunc("POST /admin/rules/test-match", s.handleTestMatch)

	return mux
}

func (s *Server) handleListRules(w http.ResponseWriter, r *http.Request) {
	pageReq := paging.ParseRequest(r)
	rules, hasMore, err := s.store.ListPage(pageReq.Limit, pageReq.Cursor)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	var nextCursor string
	if hasMore && len(rules) > 0 {
		nextCursor = paging.EncodeCursor(rules[len(rules)-1].ID)
	}
	paging.WriteEnvelope(w, rules, nextCursor, hasMore)
}

func (s *Server) handleGetRule(w http.ResponseWriter, r *http.Request) {
	rule, err := s.store.Get(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, rule)
}

func (s *Server) handleCreateRule(w http.ResponseWriter, r *http.Request) {
	var req core.Rule
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	if err := validateRule(req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	created, err := s.store.Create(req)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, rule.ErrDuplicateID) {
			status = http.StatusConflict
		}
		writeError(w, status, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) handleUpdateRule(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var req core.Rule
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	if err := validateRule(req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	updated, err := s.store.Update(id, req)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, rule.ErrRuleNotFound) {
			status = http.StatusNotFound
		}
		writeError(w, status, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) handleDeleteRule(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if err := s.store.Delete(id); err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, rule.ErrRuleNotFound) {
			status = http.StatusNotFound
		}
		writeError(w, status, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) handleReorderRules(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Failed to read body")
		return
	}

	var ids []string
	var objPayload struct {
		RuleIDs []string `json:"rule_ids"`
		IDs     []string `json:"ids"`
	}

	if err := json.Unmarshal(body, &objPayload); err == nil && (len(objPayload.RuleIDs) > 0 || len(objPayload.IDs) > 0) {
		if len(objPayload.RuleIDs) > 0 {
			ids = objPayload.RuleIDs
		} else {
			ids = objPayload.IDs
		}
	} else if err := json.Unmarshal(body, &ids); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON payload; expected {rule_ids: []} or string array")
		return
	}

	if err := s.store.Reorder(ids); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "reordered"})
}

func (s *Server) handleImportRules(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Failed to read body")
		return
	}

	var rules []core.Rule
	if err := json.Unmarshal(body, &rules); err != nil {
		if yamlErr := yaml.Unmarshal(body, &rules); yamlErr != nil {
			writeError(w, http.StatusBadRequest, "Invalid rules payload: must be valid JSON or YAML array of rules")
			return
		}
	}

	for _, r := range rules {
		if err := validateRule(r); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid rule in import: "+err.Error())
			return
		}
	}

	if err := s.store.Import(rules); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status": "imported",
		"count":  len(rules),
	})
}

func (s *Server) handleExportRules(w http.ResponseWriter, r *http.Request) {
	rules := s.store.Export()
	yamlBytes, err := yaml.Marshal(rules)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to marshal rules YAML: "+err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/yaml")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(yamlBytes)
}

func (s *Server) handleGetVersions(w http.ResponseWriter, r *http.Request) {
	versions, err := s.store.GetVersions()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if versions == nil {
		versions = []core.RuleVersion{}
	}
	writeJSON(w, http.StatusOK, versions)
}

func (s *Server) handleRollback(w http.ResponseWriter, r *http.Request) {
	versionStr := r.PathValue("version")
	version, err := strconv.Atoi(versionStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid version integer")
		return
	}

	if err := s.store.Rollback(version); err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, rule.ErrRuleNotFound) || strings.Contains(err.Error(), "not found") {
			status = http.StatusNotFound
		}
		writeError(w, status, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "rolled_back",
		"version": version,
	})
}

func (s *Server) handleCatalogue(w http.ResponseWriter, r *http.Request) {
	if s.dryRunner != nil {
		writeJSON(w, http.StatusOK, s.dryRunner.Catalogue())
		return
	}
	writeJSON(w, http.StatusOK, core.HandlerCatalogue{
		Authenticators: []core.HandlerInfo{},
		Authorizers:    []core.HandlerInfo{},
		Mutators:       []core.HandlerInfo{},
	})
}

// handleTestMatch backs the console's proxy-rules simulator with full dry-run trace support.
func (s *Server) handleTestMatch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Method  string            `json:"method"`
		Path    string            `json:"path"`
		Headers map[string]string `json:"headers,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}
	if req.Method == "" || req.Path == "" {
		writeError(w, http.StatusBadRequest, "method and path are required")
		return
	}

	if s.dryRunner != nil {
		simReq, err := http.NewRequest(strings.ToUpper(req.Method), req.Path, nil)
		if err != nil {
			writeError(w, http.StatusBadRequest, "Invalid request parameters: "+err.Error())
			return
		}
		for k, v := range req.Headers {
			simReq.Header.Set(k, v)
		}

		trace, err := s.dryRunner.DryRun(simReq)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		if trace.MatchedRuleID == "" {
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"matched": false,
				"trace":   trace,
			})
			return
		}

		matchedRule, _ := s.store.Get(trace.MatchedRuleID)
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"matched": true,
			"rule":    matchedRule,
			"trace":   trace,
		})
		return
	}

	matched, err := s.store.TestMatch(req.Method, req.Path)
	if err != nil || matched == nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{"matched": false})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"matched": true, "rule": matched})
}

func validateRule(r core.Rule) error {
	if r.Match.URL == "" {
		return errors.New("match.url is required")
	}
	if len(r.Match.Methods) == 0 {
		return errors.New("match.methods must have at least one entry")
	}
	if r.Upstream.URL == "" {
		return errors.New("upstream.url is required")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

