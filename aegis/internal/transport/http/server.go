// Package http exposes the Aegis admin REST API used by the console to
// manage proxy routing rules. It runs on a separate port from the proxy
// pipeline itself (ServeMux in internal/proxy is a catch-all reverse proxy
// and can't share a mux with named admin routes).
package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/autorix/aegis/internal/core"
	"github.com/autorix/aegis/internal/rule"
)

// RuleStore is the subset of rule.Store needed by the admin API.
type RuleStore interface {
	List() []core.Rule
	Get(id string) (core.Rule, error)
	Create(r core.Rule) (core.Rule, error)
	Update(id string, r core.Rule) (core.Rule, error)
	Delete(id string) error
	TestMatch(method, path string) (*core.Rule, error)
}

type Server struct {
	store RuleStore
}

func NewServer(store RuleStore) *Server {
	return &Server{store: store}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("GET /rules", s.handleListRules)
	mux.HandleFunc("POST /rules", s.handleCreateRule)
	mux.HandleFunc("GET /rules/{id}", s.handleGetRule)
	mux.HandleFunc("PUT /rules/{id}", s.handleUpdateRule)
	mux.HandleFunc("DELETE /rules/{id}", s.handleDeleteRule)
	mux.HandleFunc("POST /rules/test-match", s.handleTestMatch)

	return mux
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "healthy",
		"service": "aegis",
		"engine":  "Zero Trust Access Proxy",
	})
}

func (s *Server) handleListRules(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.store.List())
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

// handleTestMatch backs the console's proxy-rules simulator: given a method
// and path, it returns whichever rule the live pipeline would route to.
func (s *Server) handleTestMatch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Method string `json:"method"`
		Path   string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}
	if req.Method == "" || req.Path == "" {
		writeError(w, http.StatusBadRequest, "method and path are required")
		return
	}

	matched, err := s.store.TestMatch(req.Method, req.Path)
	if err != nil {
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
