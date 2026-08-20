package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/autorix/platform/health"
	"github.com/autorix/platform/metrics"
	"github.com/autorix/platform/paging"
	"github.com/autorix/themis/internal/core"
)

type Server struct {
	engine core.PolicyEngine
	health *health.Handler
}

func NewServer(engine core.PolicyEngine, healthHandler *health.Handler) *Server {
	return &Server{engine: engine, health: healthHandler}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	s.health.Register(mux)
	// GET /health is a deprecated alias kept for existing callers; it now
	// delegates to the real liveness check instead of claiming health it
	// never measured. New callers should use /health/alive or /health/ready.
	mux.HandleFunc("GET /health", s.health.Alive)
	mux.Handle("GET /metrics", metrics.Handler())
	mux.Handle("/metrics", metrics.Handler())
	mux.HandleFunc("GET /policies", s.handleListPolicies)
	mux.HandleFunc("POST /policies", s.handleCreatePolicy)
	mux.HandleFunc("GET /policies/{id}", s.handleGetPolicy)
	mux.HandleFunc("PUT /policies/{id}", s.handleUpdatePolicy)
	mux.HandleFunc("DELETE /policies/{id}", s.handleDeletePolicy)
	mux.HandleFunc("POST /policies/evaluate", s.handleEvaluate)

	// P6-S4-T1: Policy Versioning
	mux.HandleFunc("GET /admin/policies/{id}/versions", s.handleListVersions)
	mux.HandleFunc("GET /policies/{id}/versions", s.handleListVersions)
	mux.HandleFunc("POST /admin/policies/{id}/rollback/{version}", s.handleRollback)
	mux.HandleFunc("POST /policies/{id}/rollback/{version}", s.handleRollback)

	// P6-S4-T2: CEL Validation
	mux.HandleFunc("POST /admin/policies/validate", s.handleValidate)
	mux.HandleFunc("POST /policies/validate", s.handleValidate)

	// P6-S4-T3: Dry-Run Evaluation
	mux.HandleFunc("POST /admin/policies/dry-run", s.handleDryRun)
	mux.HandleFunc("POST /policies/dry-run", s.handleDryRun)

	// P6-S4-T7: Test Fixtures per Policy
	mux.HandleFunc("GET /admin/policies/{id}/fixtures", s.handleListFixtures)
	mux.HandleFunc("GET /policies/{id}/fixtures", s.handleListFixtures)
	mux.HandleFunc("POST /admin/policies/{id}/fixtures", s.handleCreateFixture)
	mux.HandleFunc("POST /policies/{id}/fixtures", s.handleCreateFixture)
	mux.HandleFunc("DELETE /admin/policies/{id}/fixtures/{fixture_id}", s.handleDeleteFixture)
	mux.HandleFunc("DELETE /policies/{id}/fixtures/{fixture_id}", s.handleDeleteFixture)
	mux.HandleFunc("POST /admin/policies/{id}/test-suite", s.handleTestSuite)
	mux.HandleFunc("POST /policies/{id}/test-suite", s.handleTestSuite)

	return mux
}

func (s *Server) handleListPolicies(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}

	enabledOnlyStr := r.URL.Query().Get("enabled_only")
	enabledOnly := enabledOnlyStr == "true" || enabledOnlyStr == "1"

	pageReq := paging.ParseRequest(r)
	policies, hasMore, err := s.engine.ListPoliciesPage(r.Context(), core.ListFilter{
		TenantID:    tenantID,
		EnabledOnly: enabledOnly,
	}, pageReq.Limit, pageReq.Cursor)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var nextCursor string
	if hasMore && len(policies) > 0 {
		last := policies[len(policies)-1]
		nextCursor = paging.EncodeCursor(fmt.Sprintf("%d|%s|%s", last.Priority, last.CreatedAt.Format(time.RFC3339Nano), last.ID))
	}
	paging.WriteEnvelope(w, policies, nextCursor, hasMore)
}

func (s *Server) handleGetPolicy(w http.ResponseWriter, r *http.Request) {
	policyID := r.PathValue("id")
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}

	policy, err := s.engine.GetPolicy(r.Context(), tenantID, policyID)
	if err != nil {
		writeError(w, http.StatusNotFound, "Policy not found: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, policy)
}

func (s *Server) handleCreatePolicy(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TenantID    string            `json:"tenant_id"`
		Name        string            `json:"name"`
		Description string            `json:"description"`
		Expression  string            `json:"expression"`
		Priority    int32             `json:"priority"`
		Enabled     *bool             `json:"enabled"`
		Labels      map[string]string `json:"labels"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	if req.TenantID == "" {
		req.TenantID = "default"
	}
	if req.Name == "" || req.Expression == "" {
		writeError(w, http.StatusBadRequest, "name and expression are required")
		return
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	policy := &core.Policy{
		TenantID:    req.TenantID,
		Name:        req.Name,
		Description: req.Description,
		Expression:  req.Expression,
		Priority:    req.Priority,
		Enabled:     enabled,
		Labels:      req.Labels,
	}

	if err := s.engine.CreatePolicy(r.Context(), policy); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, policy)
}

func (s *Server) handleUpdatePolicy(w http.ResponseWriter, r *http.Request) {
	policyID := r.PathValue("id")
	var req struct {
		TenantID    string            `json:"tenant_id"`
		Name        string            `json:"name"`
		Description string            `json:"description"`
		Expression  string            `json:"expression"`
		Priority    int32             `json:"priority"`
		Enabled     *bool             `json:"enabled"`
		Labels      map[string]string `json:"labels"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	if req.TenantID == "" {
		req.TenantID = "default"
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	policy := &core.Policy{
		ID:          policyID,
		TenantID:    req.TenantID,
		Name:        req.Name,
		Description: req.Description,
		Expression:  req.Expression,
		Priority:    req.Priority,
		Enabled:     enabled,
		Labels:      req.Labels,
	}

	if err := s.engine.UpdatePolicy(r.Context(), policy); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, policy)
}

func (s *Server) handleDeletePolicy(w http.ResponseWriter, r *http.Request) {
	policyID := r.PathValue("id")
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}

	if err := s.engine.DeletePolicy(r.Context(), tenantID, policyID); err != nil {
		writeError(w, http.StatusNotFound, "Failed to delete policy: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) handleEvaluate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TenantID    string                 `json:"tenant_id"`
		PolicyID    string                 `json:"policy_id"`
		Payload     map[string]interface{} `json:"payload"`
		LabelFilter map[string]string      `json:"label_filter"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	if req.TenantID == "" {
		req.TenantID = "default"
	}

	evalReq := core.EvaluateRequest{
		TenantID:    req.TenantID,
		PolicyID:    req.PolicyID,
		Payload:     req.Payload,
		LabelFilter: req.LabelFilter,
	}

	resp, err := s.engine.Evaluate(r.Context(), evalReq)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Evaluation failed: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleListVersions(w http.ResponseWriter, r *http.Request) {
	policyID := r.PathValue("id")
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}

	versions, err := s.engine.ListPolicyVersions(r.Context(), tenantID, policyID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, versions)
}

func (s *Server) handleRollback(w http.ResponseWriter, r *http.Request) {
	policyID := r.PathValue("id")
	versionStr := r.PathValue("version")
	v, err := strconv.ParseInt(versionStr, 10, 32)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid version parameter: "+err.Error())
		return
	}

	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}

	policy, err := s.engine.RollbackPolicy(r.Context(), tenantID, policyID, int32(v))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, policy)
}

func (s *Server) handleValidate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Expression string `json:"expression"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	if req.Expression == "" {
		writeError(w, http.StatusBadRequest, "expression is required")
		return
	}

	res, err := s.engine.ValidateExpression(r.Context(), req.Expression)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, res)
}

func (s *Server) handleDryRun(w http.ResponseWriter, r *http.Request) {
	var req core.DryRunRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	if req.Expression == "" {
		writeError(w, http.StatusBadRequest, "expression is required")
		return
	}

	if req.Payload == nil {
		req.Payload = make(map[string]interface{})
	}

	res, err := s.engine.DryRun(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, res)
}

func (s *Server) handleListFixtures(w http.ResponseWriter, r *http.Request) {
	policyID := r.PathValue("id")
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}

	fixtures, err := s.engine.ListFixtures(r.Context(), tenantID, policyID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, fixtures)
}

func (s *Server) handleCreateFixture(w http.ResponseWriter, r *http.Request) {
	policyID := r.PathValue("id")
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}

	var req struct {
		Name           string                 `json:"name"`
		Description    string                 `json:"description"`
		Payload        map[string]interface{} `json:"payload"`
		ExpectedResult *bool                  `json:"expected_result"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	if req.Name == "" || req.ExpectedResult == nil {
		writeError(w, http.StatusBadRequest, "name and expected_result are required")
		return
	}

	if req.Payload == nil {
		req.Payload = make(map[string]interface{})
	}

	fixture := &core.PolicyFixture{
		PolicyID:       policyID,
		TenantID:       tenantID,
		Name:           req.Name,
		Description:    req.Description,
		Payload:        req.Payload,
		ExpectedResult: *req.ExpectedResult,
	}

	if err := s.engine.CreateFixture(r.Context(), fixture); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, fixture)
}

func (s *Server) handleDeleteFixture(w http.ResponseWriter, r *http.Request) {
	policyID := r.PathValue("id")
	fixtureID := r.PathValue("fixture_id")
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}

	if err := s.engine.DeleteFixture(r.Context(), tenantID, policyID, fixtureID); err != nil {
		writeError(w, http.StatusNotFound, "Failed to delete fixture: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) handleTestSuite(w http.ResponseWriter, r *http.Request) {
	policyID := r.PathValue("id")
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}

	res, err := s.engine.RunTestSuite(r.Context(), tenantID, policyID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Test suite execution failed: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, res)
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
