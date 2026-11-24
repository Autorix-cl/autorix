// Package http is Argus's REST admin surface: enrollment token minting,
// fleet listing, topology and events (P2-S3, P2-S6). Handlers land with
// their owning specs; the fleet-query surface here (P2-S6-T1) is a thin
// wrapper over the same core.Repository the gRPC hot path uses — no SQL is
// duplicated between the two transports.
package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/autorix/argus/internal/core"
	"github.com/autorix/argus/internal/credential"
	"github.com/autorix/platform/health"
	"github.com/autorix/platform/httpx"
	"github.com/autorix/platform/metrics"
	"github.com/autorix/platform/paging"
	"github.com/autorix/platform/tracing"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

const maxRequestBodyBytes int64 = 1 << 20 // 1 MiB

// enrollmentTokenPrefix marks a minted enrollment token as recognizably
// Argus's, the same way vulcan's API keys carry "aet_" — a reviewer
// scanning logs or a secrets scanner can tell what kind of credential they
// are looking at without decoding anything.
const enrollmentTokenPrefix = "aet_"

// defaultTokenTTL is used when a mint request omits expires_in_seconds.
const defaultTokenTTL = 24 * time.Hour

type Server struct {
	health *health.Handler
	repo   core.Repository
}

// NewServer builds a Server backed by healthHandler and repo. repo may be
// nil in tests that only exercise the health contract.
func NewServer(healthHandler *health.Handler, repo core.Repository) *Server {
	return &Server{health: healthHandler, repo: repo}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /metrics", metrics.Handler())
	if s.health != nil {
		s.health.Register(mux)
	}
	if s.repo != nil {
		mux.HandleFunc("GET /v1/engines", s.handleListEngines)
		mux.HandleFunc("GET /admin/engines", s.handleListEngines)
		mux.HandleFunc("GET /v1/events", s.handleListEvents)
		mux.HandleFunc("GET /admin/events", s.handleListEvents)
		mux.HandleFunc("GET /v1/instances", s.handleListInstances)
		mux.HandleFunc("GET /admin/instances", s.handleListInstances)
		mux.HandleFunc("GET /v1/instances/{id}", s.handleGetInstance)
		mux.HandleFunc("GET /admin/instances/{id}", s.handleGetInstance)
		mux.HandleFunc("DELETE /v1/instances/{id}", s.handleForceRemove)
		mux.HandleFunc("DELETE /admin/instances/{id}", s.handleForceRemove)
		mux.HandleFunc("POST /v1/instances/manual", s.handleManualRegister)
		mux.HandleFunc("POST /admin/instances/manual", s.handleManualRegister)
		mux.HandleFunc("POST /v1/instances/{id}/force-remove", s.handleForceRemove)
		mux.HandleFunc("POST /admin/instances/{id}/force-remove", s.handleForceRemove)
		mux.HandleFunc("POST /v1/enrollment-tokens", s.handleMintEnrollmentToken)
		mux.HandleFunc("POST /admin/enrollment-tokens", s.handleMintEnrollmentToken)
		mux.HandleFunc("GET /v1/enrollment-tokens", s.handleListEnrollmentTokens)
		mux.HandleFunc("GET /admin/enrollment-tokens", s.handleListEnrollmentTokens)
		mux.HandleFunc("DELETE /v1/enrollment-tokens/{id}", s.handleRevokeEnrollmentToken)
		mux.HandleFunc("DELETE /admin/enrollment-tokens/{id}", s.handleRevokeEnrollmentToken)
		mux.HandleFunc("GET /v1/enrollment-audit", s.handleListEnrollmentAudit)
		mux.HandleFunc("GET /admin/enrollment-audit", s.handleListEnrollmentAudit)
		mux.HandleFunc("GET /v1/topology", s.handleGetTopology)
		mux.HandleFunc("GET /admin/topology", s.handleGetTopology)
		mux.HandleFunc("GET /v1/metrics/summary", s.handleGetMetricsSummary)
		mux.HandleFunc("GET /admin/metrics/summary", s.handleGetMetricsSummary)
		mux.HandleFunc("GET /v1/stream", s.handleStream)
		mux.HandleFunc("GET /admin/stream", s.handleStream)

		// Console Identity & Auth (P3-S1, P3-S2)
		mux.HandleFunc("GET /v1/auth/status", s.handleAuthStatus)
		mux.HandleFunc("POST /v1/auth/bootstrap", s.handleBootstrap)
		mux.HandleFunc("POST /v1/auth/login", s.handleLogin)
		mux.HandleFunc("GET /v1/auth/session", s.handleValidateSession)
		mux.HandleFunc("DELETE /v1/auth/session", s.handleLogout)
		mux.HandleFunc("GET /v1/operators", s.handleListOperators)
		mux.HandleFunc("POST /v1/operators", s.handleCreateOperator)
		mux.HandleFunc("PATCH /v1/operators/{id}", s.handleUpdateOperatorStatus)
		mux.HandleFunc("DELETE /v1/operators/{id}", s.handleDeleteOperator)

		// Audit & Governance (P8-S1, P8-S4)
		mux.HandleFunc("GET /v1/audit", s.handleListAudit)
		mux.HandleFunc("GET /admin/audit", s.handleListAudit)
		mux.HandleFunc("POST /v1/audit", s.handleRecordAudit)
		mux.HandleFunc("POST /admin/audit", s.handleRecordAudit)
		mux.HandleFunc("GET /v1/audit/verify", s.handleVerifyAudit)
		mux.HandleFunc("GET /admin/audit/verify", s.handleVerifyAudit)
		mux.HandleFunc("GET /v1/audit/export", s.handleExportAudit)
		mux.HandleFunc("GET /admin/audit/export", s.handleExportAudit)
		mux.HandleFunc("GET /v1/compliance/evidence", s.handleGetComplianceEvidence)
		mux.HandleFunc("GET /admin/compliance/evidence", s.handleGetComplianceEvidence)
		mux.HandleFunc("POST /v1/compliance/evidence", s.handleGenerateComplianceEvidence)
		mux.HandleFunc("POST /admin/compliance/evidence", s.handleGenerateComplianceEvidence)
		mux.HandleFunc("GET /v1/environments", s.handleListEnvironments)
		mux.HandleFunc("GET /admin/environments", s.handleListEnvironments)
		mux.HandleFunc("POST /v1/environments", s.handleCreateEnvironment)
		mux.HandleFunc("POST /admin/environments", s.handleCreateEnvironment)
		mux.HandleFunc("GET /v1/governance/orgs", s.handleListOrganisations)
		mux.HandleFunc("GET /admin/governance/orgs", s.handleListOrganisations)
		mux.HandleFunc("POST /v1/governance/orgs", s.handleCreateOrganisation)
		mux.HandleFunc("POST /admin/governance/orgs", s.handleCreateOrganisation)
		mux.HandleFunc("GET /v1/organisations", s.handleListOrganisations)
		mux.HandleFunc("GET /admin/organisations", s.handleListOrganisations)
		mux.HandleFunc("POST /v1/organisations", s.handleCreateOrganisation)
		mux.HandleFunc("POST /admin/organisations", s.handleCreateOrganisation)
		mux.HandleFunc("GET /v1/governance/projects", s.handleListProjects)
		mux.HandleFunc("GET /admin/governance/projects", s.handleListProjects)
		mux.HandleFunc("POST /v1/governance/projects", s.handleCreateProject)
		mux.HandleFunc("POST /admin/governance/projects", s.handleCreateProject)
		mux.HandleFunc("GET /v1/projects", s.handleListProjects)
		mux.HandleFunc("GET /admin/projects", s.handleListProjects)
		mux.HandleFunc("POST /v1/projects", s.handleCreateProject)
		mux.HandleFunc("POST /admin/projects", s.handleCreateProject)
	}
	return httpx.MaxBodyBytes(maxRequestBodyBytes)(metrics.HTTPMiddleware("argus")(tracing.HTTPMiddleware()(mux)))
}

// actorAndOrigin resolves the caller identity and network origin recorded
// against every enrollment-token audit entry (P2-S3-T6). Actor prefers an
// explicit X-Actor header (set by the console/CLI on behalf of the signed-in
// operator) over an anonymous fallback — Argus has no operator auth of its
// own yet, so this is best-effort attribution, not a security boundary.
func actorAndOrigin(r *http.Request) (actor string, origin string) {
	actor = r.Header.Get("X-Actor")
	if actor == "" {
		actor = "unknown"
	}
	origin = r.Header.Get("X-Forwarded-For")
	if origin == "" {
		origin = r.RemoteAddr
	}
	return actor, origin
}

// handleListInstances lists instances with the same filters ListInstances
// (gRPC) accepts, using platform/paging's cursor/filter convention:
// ?filter.environment=<slug>&filter.engine_type=<slug>&filter.status=<status>&cursor=<token>&limit=<n>.
func (s *Server) handleListInstances(w http.ResponseWriter, r *http.Request) {
	preq := paging.ParseRequest(r)

	filter := core.InstanceFilter{Limit: preq.Limit}
	if v := firstFilter(preq, "engine_type"); v != "" {
		filter.EngineType = v
	}
	if v := firstFilter(preq, "status"); v != "" {
		filter.Status = core.InstanceStatus(v)
	}
	if slug := firstFilter(preq, "environment"); slug != "" {
		env, err := s.repo.GetEnvironmentBySlug(r.Context(), slug)
		if err != nil {
			if errors.Is(err, core.ErrNotFound) {
				writeError(w, http.StatusBadRequest, "unknown environment "+slug)
				return
			}
			writeError(w, http.StatusInternalServerError, "looking up environment")
			return
		}
		filter.EnvironmentID = env.ID
	}
	if preq.Cursor != "" {
		raw, err := paging.DecodeCursor(preq.Cursor)
		if err != nil {
			writeError(w, http.StatusBadRequest, "cursor is not valid")
			return
		}
		filter.Cursor = raw
	}

	instances, nextCursor, hasMore, err := s.repo.ListInstances(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	envs, err := s.repo.ListEnvironments(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	envSlugByID := make(map[uuid.UUID]string, len(envs))
	for _, e := range envs {
		envSlugByID[e.ID] = e.Slug
	}

	out := make([]apiInstance, 0, len(instances))
	for _, inst := range instances {
		out = append(out, toAPIInstance(inst, envSlugByID[inst.EnvironmentID]))
	}

	encodedCursor := ""
	if hasMore {
		encodedCursor = paging.EncodeCursor(nextCursor)
	}
	paging.WriteEnvelope(w, out, encodedCursor, hasMore)
}

// firstFilter returns the first value of a paging.Request filter, or "" if
// unset — every filter argus's list endpoint accepts is single-valued.
func firstFilter(preq paging.Request, name string) string {
	values := preq.Filter[name]
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

// handleGetInstance returns one instance's current state, its most recent
// timeline events and its dependency edges — the REST mirror of the gRPC
// GetInstance RPC, over the same repository.
func (s *Server) handleGetInstance(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id is not a valid uuid")
		return
	}

	inst, err := s.repo.GetInstance(r.Context(), id)
	if err != nil {
		if errors.Is(err, core.ErrNotFound) {
			writeError(w, http.StatusNotFound, "instance not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	envSlug := ""
	if env, err := s.repo.GetEnvironmentByID(r.Context(), inst.EnvironmentID); err == nil {
		envSlug = env.Slug
	} else if !errors.Is(err, core.ErrNotFound) {
		writeError(w, http.StatusInternalServerError, "looking up environment")
		return
	}

	events, err := s.repo.ListEvents(r.Context(), &id, 20)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	deps, err := s.repo.ListDependencies(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"instance":     toAPIInstance(inst, envSlug),
		"events":       toAPIEvents(events),
		"dependencies": toAPIDependencies(deps),
	})
}

// ---------------------------------------------------------------------
// Enrollment tokens (P2-S3-T1)
// ---------------------------------------------------------------------

type mintEnrollmentTokenRequest struct {
	EngineType       string `json:"engine_type"`
	Environment      string `json:"environment"`
	UsesAllowed      int    `json:"uses_allowed"`
	ExpiresInSeconds int    `json:"expires_in_seconds"`
	CreatedBy        string `json:"created_by"`
}

// handleMintEnrollmentToken mints a fresh, high-entropy enrollment token
// (P2-S3-T1), persisting only its hash — the plaintext, prefixed "aet_" so
// it is recognizable in logs/secret scanners, is returned exactly once in
// this response, mirroring how vulcan reveals its "av_live_..." API keys.
func (s *Server) handleMintEnrollmentToken(w http.ResponseWriter, r *http.Request) {
	var req mintEnrollmentTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.EngineType == "" || req.Environment == "" {
		writeError(w, http.StatusBadRequest, "engine_type and environment are required")
		return
	}
	if req.UsesAllowed <= 0 {
		req.UsesAllowed = 1
	}
	ttl := defaultTokenTTL
	if req.ExpiresInSeconds > 0 {
		ttl = time.Duration(req.ExpiresInSeconds) * time.Second
	}

	if _, err := s.repo.GetEngineType(r.Context(), req.EngineType); err != nil {
		if errors.Is(err, core.ErrNotFound) {
			writeError(w, http.StatusBadRequest, "unknown engine_type "+req.EngineType)
			return
		}
		writeError(w, http.StatusInternalServerError, "looking up engine type")
		return
	}
	env, err := s.repo.GetEnvironmentBySlug(r.Context(), req.Environment)
	if err != nil {
		if errors.Is(err, core.ErrNotFound) {
			writeError(w, http.StatusBadRequest, "unknown environment "+req.Environment)
			return
		}
		writeError(w, http.StatusInternalServerError, "looking up environment")
		return
	}

	actor, origin := actorAndOrigin(r)
	createdBy := req.CreatedBy
	if createdBy == "" {
		createdBy = actor
	}

	rawToken, tokenHash, err := credential.GenerateSecret(enrollmentTokenPrefix)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "generating enrollment token")
		return
	}

	created, err := s.repo.CreateEnrollmentToken(r.Context(), core.EnrollmentToken{
		TokenHash:     tokenHash,
		EngineType:    req.EngineType,
		EnvironmentID: env.ID,
		UsesAllowed:   req.UsesAllowed,
		ExpiresAt:     time.Now().UTC().Add(ttl),
		CreatedBy:     createdBy,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "creating enrollment token")
		return
	}

	_ = s.repo.AppendEnrollmentAudit(r.Context(), core.EnrollmentAuditEntry{
		TokenID: &created.ID, EngineType: req.EngineType, EnvironmentID: env.ID,
		Actor: actor, Origin: origin, Action: core.AuditActionMint,
	})

	writeJSON(w, http.StatusCreated, toAPIEnrollmentToken(created, req.Environment, rawToken))
}

// handleListEnrollmentTokens lists enrollment tokens (never their secret —
// only the mint response ever reveals it), optionally filtered by
// ?engine_type= and ?environment=.
func (s *Server) handleListEnrollmentTokens(w http.ResponseWriter, r *http.Request) {
	engineType := r.URL.Query().Get("engine_type")
	var envID uuid.UUID
	envSlug := r.URL.Query().Get("environment")
	if envSlug != "" {
		env, err := s.repo.GetEnvironmentBySlug(r.Context(), envSlug)
		if err != nil {
			if errors.Is(err, core.ErrNotFound) {
				writeError(w, http.StatusBadRequest, "unknown environment "+envSlug)
				return
			}
			writeError(w, http.StatusInternalServerError, "looking up environment")
			return
		}
		envID = env.ID
	}

	tokens, err := s.repo.ListEnrollmentTokens(r.Context(), engineType, envID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	out := make([]apiEnrollmentToken, 0, len(tokens))
	for _, t := range tokens {
		out = append(out, toAPIEnrollmentToken(t, envSlug, ""))
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"data": out})
}

// handleRevokeEnrollmentToken revokes an enrollment token so it can never
// be consumed again, even if it has remaining uses left.
func (s *Server) handleRevokeEnrollmentToken(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id is not a valid uuid")
		return
	}
	if err := s.repo.RevokeEnrollmentToken(r.Context(), id); err != nil {
		if errors.Is(err, core.ErrNotFound) {
			writeError(w, http.StatusNotFound, "enrollment token not found or already revoked")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	actor, origin := actorAndOrigin(r)
	_ = s.repo.AppendEnrollmentAudit(r.Context(), core.EnrollmentAuditEntry{
		TokenID: &id, Actor: actor, Origin: origin, Action: core.AuditActionRevoke,
	})

	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------------------------------------------------
// Enrollment audit trail (P2-S3-T6)
// ---------------------------------------------------------------------

// handleListEnrollmentAudit returns the enrollment audit trail — every
// mint, use, failed attempt and revocation — paginated with the platform
// paging convention, optionally filtered by ?token_id= and ?action=.
func (s *Server) handleListEnrollmentAudit(w http.ResponseWriter, r *http.Request) {
	preq := paging.ParseRequest(r)

	filter := core.EnrollmentAuditFilter{Limit: preq.Limit}
	if v := firstFilter(preq, "action"); v != "" {
		filter.Action = core.EnrollmentAuditAction(v)
	}
	if v := firstFilter(preq, "token_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "filter.token_id is not a valid uuid")
			return
		}
		filter.TokenID = id
	}
	if preq.Cursor != "" {
		raw, err := paging.DecodeCursor(preq.Cursor)
		if err != nil {
			writeError(w, http.StatusBadRequest, "cursor is not valid")
			return
		}
		filter.Cursor = raw
	}

	entries, nextCursor, hasMore, err := s.repo.ListEnrollmentAudit(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	out := make([]apiEnrollmentAuditEntry, 0, len(entries))
	for _, e := range entries {
		out = append(out, toAPIEnrollmentAuditEntry(e))
	}

	encodedCursor := ""
	if hasMore {
		encodedCursor = paging.EncodeCursor(nextCursor)
	}
	paging.WriteEnvelope(w, out, encodedCursor, hasMore)
}

// ---------------------------------------------------------------------
// Manual registration (P2-S3-T7)
// ---------------------------------------------------------------------

type manualRegisterRequest struct {
	EngineType    string            `json:"engine_type"`
	InstanceID    string            `json:"instance_id"`
	Environment   string            `json:"environment"`
	Version       string            `json:"version"`
	BuildSHA      string            `json:"build_sha"`
	SchemaVersion string            `json:"schema_version"`
	Capabilities  []string          `json:"capabilities"`
	Endpoints     core.Endpoints    `json:"endpoints"`
	Labels        map[string]string `json:"labels"`
}

// handleManualRegister lets an operator register an endpoint by hand when
// it cannot receive an enrollment token itself (P2-S3-T7). The resulting
// instance is always Unverified — it never went through the enrollment
// token trust path — so the console and any policy consuming this fleet
// data can keep that trust difference visible instead of it being silently
// indistinguishable from a normal registration.
func (s *Server) handleManualRegister(w http.ResponseWriter, r *http.Request) {
	var req manualRegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.EngineType == "" || req.InstanceID == "" || req.Environment == "" {
		writeError(w, http.StatusBadRequest, "engine_type, instance_id and environment are required")
		return
	}

	if _, err := s.repo.GetEngineType(r.Context(), req.EngineType); err != nil {
		if errors.Is(err, core.ErrNotFound) {
			writeError(w, http.StatusBadRequest, "unknown engine_type "+req.EngineType)
			return
		}
		writeError(w, http.StatusInternalServerError, "looking up engine type")
		return
	}
	env, err := s.repo.GetEnvironmentBySlug(r.Context(), req.Environment)
	if err != nil {
		if errors.Is(err, core.ErrNotFound) {
			writeError(w, http.StatusBadRequest, "unknown environment "+req.Environment)
			return
		}
		writeError(w, http.StatusInternalServerError, "looking up environment")
		return
	}

	actor, _ := actorAndOrigin(r)
	inst, err := s.repo.UpsertInstance(r.Context(), core.RegistrationRequest{
		EngineType:    req.EngineType,
		InstanceID:    req.InstanceID,
		EnvironmentID: env.ID,
		Version:       req.Version,
		BuildSHA:      req.BuildSHA,
		SchemaVersion: req.SchemaVersion,
		Capabilities:  req.Capabilities,
		Endpoints:     req.Endpoints,
	}, "manual:"+actor, true)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "registering instance")
		return
	}
	_ = s.repo.AppendEvent(r.Context(), inst.ID, core.EventRegistered, map[string]interface{}{
		"manual": true, "actor": actor,
	})

	writeJSON(w, http.StatusCreated, toAPIInstance(inst, req.Environment))
}

// ---------------------------------------------------------------------
// Force-remove (P2-S5-T5)
// ---------------------------------------------------------------------

type forceRemoveRequest struct {
	Reason string `json:"reason"`
}

// handleForceRemove is the administrative, immediate exit: the instance is
// evicted at once and its credential revoked, so a zombie process cannot
// keep heartbeating or silently re-register under the same identity.
func (s *Server) handleForceRemove(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id is not a valid uuid")
		return
	}
	var req forceRemoveRequest
	if r.ContentLength != 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
	}
	if req.Reason == "" {
		req.Reason = "operator requested"
	}

	if err := core.ForceRemoveInstance(r.Context(), s.repo, id, req.Reason); err != nil {
		if errors.Is(err, core.ErrNotFound) {
			writeError(w, http.StatusNotFound, "instance not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------------------------------------------------
// Topology & SSE Stream (P2-S6-T3, P2-S6-T5)
// ---------------------------------------------------------------------

func (s *Server) handleGetTopology(w http.ResponseWriter, r *http.Request) {
	var envID *uuid.UUID
	envSlug := r.URL.Query().Get("environment")
	if envSlug != "" {
		env, err := s.repo.GetEnvironmentBySlug(r.Context(), envSlug)
		if err != nil {
			if errors.Is(err, core.ErrNotFound) {
				writeError(w, http.StatusBadRequest, "unknown environment "+envSlug)
				return
			}
			writeError(w, http.StatusInternalServerError, "looking up environment")
			return
		}
		envID = &env.ID
	}
	graph, err := s.repo.GetTopology(r.Context(), envID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "building topology graph")
		return
	}
	writeJSON(w, http.StatusOK, graph)
}

func (s *Server) handleGetMetricsSummary(w http.ResponseWriter, r *http.Request) {
	instances, _, _, err := s.repo.ListInstances(r.Context(), core.InstanceFilter{Limit: 1000})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "listing instances for metrics summary")
		return
	}
	summary := core.AggregateFleetMetrics(instances)
	writeJSON(w, http.StatusOK, summary)
}

func (s *Server) handleStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	sendSnapshot := func() error {
		instances, _, _, err := s.repo.ListInstances(r.Context(), core.InstanceFilter{Limit: 100})
		if err != nil {
			return err
		}
		envs, err := s.repo.ListEnvironments(r.Context())
		if err != nil {
			return err
		}
		envSlugByID := make(map[uuid.UUID]string, len(envs))
		for _, e := range envs {
			envSlugByID[e.ID] = e.Slug
		}
		out := make([]apiInstance, 0, len(instances))
		for _, inst := range instances {
			out = append(out, toAPIInstance(inst, envSlugByID[inst.EnvironmentID]))
		}
		data, err := json.Marshal(out)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(w, "event: instances\ndata: %s\n\n", data)
		if err != nil {
			return err
		}
		flusher.Flush()
		return nil
	}

	if err := sendSnapshot(); err != nil {
		return
	}

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			if err := sendSnapshot(); err != nil {
				return
			}
		}
	}
}

// ---------------------------------------------------------------------
// Console Identity & Auth Handlers (P3-S1, P3-S2)
// ---------------------------------------------------------------------

type bootstrapRequest struct {
	BootstrapToken string `json:"bootstrap_token"`
	Email          string `json:"email"`
	Name           string `json:"name"`
	Password       string `json:"password"`
}

func (s *Server) handleAuthStatus(w http.ResponseWriter, r *http.Request) {
	count, err := s.repo.CountOperators(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "checking auth status")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"bootstrapped":    count > 0,
		"operators_count": count,
	})
}

func (s *Server) handleBootstrap(w http.ResponseWriter, r *http.Request) {
	var req bootstrapRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON request")
		return
	}
	if req.BootstrapToken == "" || req.Email == "" || req.Name == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "bootstrap_token, email, name, and password are required")
		return
	}
	if len(req.Password) < 8 {
		writeError(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}

	tokenHash := credential.HashSecret(req.BootstrapToken)
	valid, err := s.repo.ConsumeBootstrapToken(r.Context(), tokenHash)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "validating bootstrap token")
		return
	}
	if !valid {
		writeError(w, http.StatusUnauthorized, "invalid, expired, or already consumed bootstrap token")
		return
	}

	passHash, err := credential.HashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "hashing password")
		return
	}

	op := core.Operator{
		Email:    req.Email,
		Name:     req.Name,
		Role:     core.RoleOwner,
		IsLocal:  true,
		IsActive: true,
	}
	createdOp, err := s.repo.CreateOperatorWithPassword(r.Context(), op, passHash)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "creating initial owner: "+err.Error())
		return
	}

	rawToken, sessionHash, err := credential.GenerateSecret("ast_")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "generating session token")
		return
	}

	now := time.Now().UTC()
	session, err := s.repo.CreateSession(r.Context(), core.OperatorSession{
		OperatorID: createdOp.ID,
		TokenHash:  sessionHash,
		UserAgent:  r.UserAgent(),
		IPAddress:  r.RemoteAddr,
		ExpiresAt:  now.Add(core.DefaultSessionAbsoluteTTL),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "creating session")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"operator":      createdOp,
		"session_token": rawToken,
		"session_id":    session.ID,
		"expires_at":    session.ExpiresAt,
	})
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON request")
		return
	}
	if req.Email == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "email and password are required")
		return
	}

	op, err := s.repo.GetOperatorByEmail(r.Context(), req.Email)
	if err != nil {
		if errors.Is(err, core.ErrNotFound) {
			writeError(w, http.StatusUnauthorized, "invalid email or password")
			return
		}
		writeError(w, http.StatusInternalServerError, "looking up operator")
		return
	}

	if !op.IsActive {
		writeError(w, http.StatusForbidden, "operator account is deactivated")
		return
	}

	now := time.Now().UTC()
	if op.LockedUntil != nil && now.Before(*op.LockedUntil) {
		writeError(w, http.StatusTooManyRequests, "account is temporarily locked due to repeated failed attempts")
		return
	}

	valid, err := credential.VerifyPassword(req.Password, op.PasswordHash)
	if err != nil || !valid {
		_, _, _ = s.repo.RecordOperatorFailedLogin(r.Context(), op.ID, 5, 15*time.Minute)
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	_ = s.repo.ResetOperatorFailedAttempts(r.Context(), op.ID)

	rawToken, sessionHash, err := credential.GenerateSecret("ast_")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "generating session token")
		return
	}

	session, err := s.repo.CreateSession(r.Context(), core.OperatorSession{
		OperatorID: op.ID,
		TokenHash:  sessionHash,
		UserAgent:  r.UserAgent(),
		IPAddress:  r.RemoteAddr,
		ExpiresAt:  now.Add(core.DefaultSessionAbsoluteTTL),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "creating session")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"operator":      op.Operator,
		"session_token": rawToken,
		"session_id":    session.ID,
		"expires_at":    session.ExpiresAt,
	})
}

func extractSessionToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	if tok := r.Header.Get("X-Session-Token"); tok != "" {
		return tok
	}
	return ""
}

func (s *Server) handleValidateSession(w http.ResponseWriter, r *http.Request) {
	rawToken := extractSessionToken(r)
	if rawToken == "" {
		writeError(w, http.StatusUnauthorized, "missing session token")
		return
	}

	tokenHash := credential.HashSecret(rawToken)
	session, op, err := s.repo.GetSessionByTokenHash(r.Context(), tokenHash)
	if err != nil {
		if errors.Is(err, core.ErrNotFound) {
			writeError(w, http.StatusUnauthorized, "session is invalid or expired")
			return
		}
		writeError(w, http.StatusInternalServerError, "validating session")
		return
	}

	now := time.Now().UTC()
	newExpiry := now.Add(core.DefaultSessionIdleTTL)
	_ = s.repo.TouchSession(r.Context(), session.ID, now, newExpiry)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"valid":    true,
		"operator": op,
		"session":  session,
	})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	rawToken := extractSessionToken(r)
	if rawToken != "" {
		tokenHash := credential.HashSecret(rawToken)
		session, _, err := s.repo.GetSessionByTokenHash(r.Context(), tokenHash)
		if err == nil {
			_ = s.repo.RevokeSession(r.Context(), session.ID)
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleListOperators(w http.ResponseWriter, r *http.Request) {
	operators, err := s.repo.ListOperators(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "listing operators")
		return
	}
	writeJSON(w, http.StatusOK, operators)
}

type createOperatorRequest struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Role     string `json:"role"`
	Password string `json:"password"`
}

func (s *Server) handleCreateOperator(w http.ResponseWriter, r *http.Request) {
	var req createOperatorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON request")
		return
	}
	email := strings.TrimSpace(strings.ToLower(req.Email))
	name := strings.TrimSpace(req.Name)
	role := strings.TrimSpace(strings.ToLower(req.Role))

	if email == "" || name == "" || role == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "email, name, role and password are required")
		return
	}
	if len(req.Password) < 8 {
		writeError(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}

	validRoles := map[string]bool{
		"owner": true, "admin": true, "operator": true, "auditor": true,
	}
	if !validRoles[role] {
		writeError(w, http.StatusBadRequest, "invalid role: must be owner, admin, operator, or auditor")
		return
	}

	passHash, err := credential.HashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "hashing password")
		return
	}

	op := core.Operator{
		Email:    email,
		Name:     name,
		Role:     core.OperatorRole(role),
		IsLocal:  true,
		IsActive: true,
	}
	created, err := s.repo.CreateOperatorWithPassword(r.Context(), op, passHash)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "creating operator: "+err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

type updateOperatorStatusRequest struct {
	IsActive *bool `json:"is_active"`
}

func (s *Server) handleUpdateOperatorStatus(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid operator id format")
		return
	}

	var req updateOperatorStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}
	if req.IsActive == nil {
		writeError(w, http.StatusBadRequest, "is_active boolean field is required")
		return
	}

	err = s.repo.UpdateOperatorStatus(r.Context(), id, *req.IsActive)
	if err != nil {
		if errors.Is(err, core.ErrNotFound) {
			writeError(w, http.StatusNotFound, "operator not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "updating operator status: "+err.Error())
		return
	}

	updated, err := s.repo.GetOperatorByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "fetching updated operator: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) handleDeleteOperator(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid operator id format")
		return
	}

	err = s.repo.DeleteOperator(r.Context(), id)
	if err != nil {
		if errors.Is(err, core.ErrNotFound) {
			writeError(w, http.StatusNotFound, "operator not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "deleting operator: "+err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleListAudit(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 200 {
			limit = parsed
		}
	}
	filter := core.AuditRecordFilter{
		ActorID:      r.URL.Query().Get("actor_id"),
		Action:       r.URL.Query().Get("action"),
		ResourceType: r.URL.Query().Get("resource_type"),
		ResourceID:   r.URL.Query().Get("resource_id"),
		Environment:  r.URL.Query().Get("environment"),
		Outcome:      r.URL.Query().Get("outcome"),
		Cursor:       r.URL.Query().Get("cursor"),
		Limit:        limit,
	}
	records, nextCursor, hasMore, err := s.repo.ListAuditRecords(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data":        records,
		"next_cursor": nextCursor,
		"has_more":    hasMore,
	})
}

func (s *Server) handleRecordAudit(w http.ResponseWriter, r *http.Request) {
	var record core.AuditRecord
	if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	created, err := s.repo.RecordAudit(r.Context(), record)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) handleVerifyAudit(w http.ResponseWriter, r *http.Request) {
	res, err := s.repo.VerifyAuditChainDetailed(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) handleExportAudit(w http.ResponseWriter, r *http.Request) {
	format := r.URL.Query().Get("format")
	if format == "csv" {
		data, err := s.repo.ExportAuditRecords(r.Context(), "csv")
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", "attachment; filename=audit-records.csv")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
		return
	}
	data, err := s.repo.ExportAuditRecords(r.Context(), "json")
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=audit-records.json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func (s *Server) handleListEngines(w http.ResponseWriter, r *http.Request) {
	filter := core.InstanceFilter{Limit: 1000}

	envParam := strings.TrimSpace(r.URL.Query().Get("environment"))
	if envParam == "" {
		envParam = strings.TrimSpace(r.URL.Query().Get("filter.environment"))
	}
	if envParam != "" {
		env, err := s.repo.GetEnvironmentBySlug(r.Context(), envParam)
		if err != nil && errors.Is(err, core.ErrNotFound) {
			if parsedID, parseErr := uuid.Parse(envParam); parseErr == nil {
				env, err = s.repo.GetEnvironmentByID(r.Context(), parsedID)
			}
		}
		if err != nil {
			if errors.Is(err, core.ErrNotFound) {
				writeError(w, http.StatusBadRequest, "unknown environment "+envParam)
				return
			}
			writeError(w, http.StatusInternalServerError, "looking up environment")
			return
		}
		filter.EnvironmentID = env.ID
	}

	instances, _, _, err := s.repo.ListInstances(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "listing instances for engine rollup")
		return
	}

	envs, err := s.repo.ListEnvironments(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "listing environments")
		return
	}
	envSlugByID := make(map[uuid.UUID]string, len(envs))
	for _, e := range envs {
		envSlugByID[e.ID] = e.Slug
	}

	rolledUp := core.RollupInstances(instances)
	out := make([]apiLogicalEngine, 0, len(rolledUp))
	for _, le := range rolledUp {
		out = append(out, toAPILogicalEngine(le, envSlugByID[le.EnvironmentID]))
	}

	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleListEvents(w http.ResponseWriter, r *http.Request) {
	var instanceID *uuid.UUID
	instParam := strings.TrimSpace(r.URL.Query().Get("instance_id"))
	if instParam == "" {
		instParam = strings.TrimSpace(r.URL.Query().Get("filter.instance_id"))
	}
	if instParam != "" {
		parsed, err := uuid.Parse(instParam)
		if err != nil {
			writeError(w, http.StatusBadRequest, "instance_id is not a valid uuid")
			return
		}
		instanceID = &parsed
	}

	limit := 100
	if l := strings.TrimSpace(r.URL.Query().Get("limit")); l != "" {
		parsed, err := strconv.Atoi(l)
		if err != nil || parsed <= 0 {
			writeError(w, http.StatusBadRequest, "limit must be a positive integer")
			return
		}
		if parsed > 500 {
			parsed = 500
		}
		limit = parsed
	}

	events, err := s.repo.ListEvents(r.Context(), instanceID, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "listing fleet events")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data": toAPIEvents(events),
	})
}

func (s *Server) handleGetComplianceEvidence(w http.ResponseWriter, r *http.Request) {
	report, err := s.repo.GetComplianceEvidence(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, report)
}

func (s *Server) handleGenerateComplianceEvidence(w http.ResponseWriter, r *http.Request) {
	report, err := s.repo.GetComplianceEvidence(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, report)
}

func (s *Server) handleListEnvironments(w http.ResponseWriter, r *http.Request) {
	envs, err := s.repo.ListEnvironments(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if envs == nil {
		envs = []core.Environment{}
	}
	writeJSON(w, http.StatusOK, envs)
}

type createEnvironmentRequest struct {
	Name         string `json:"name"`
	Slug         string `json:"slug"`
	Description  string `json:"description"`
	Type         string `json:"type"`
	IsProduction bool   `json:"is_production"`
}

func (s *Server) handleCreateEnvironment(w http.ResponseWriter, r *http.Request) {
	var req createEnvironmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	name := strings.TrimSpace(req.Name)
	slug := normalizeSlug(req.Slug)
	if slug == "" {
		slug = normalizeSlug(name)
	}
	if name == "" || slug == "" {
		writeError(w, http.StatusBadRequest, "name and slug are required")
		return
	}
	isProduction := req.IsProduction || strings.EqualFold(strings.TrimSpace(req.Type), "production")

	env, err := s.repo.CreateEnvironment(r.Context(), name, slug, strings.TrimSpace(req.Description), isProduction)
	if err != nil {
		if isPGUniqueViolation(err) {
			writeError(w, http.StatusConflict, "environment with that slug already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "creating environment")
		return
	}
	writeJSON(w, http.StatusCreated, env)
}

func (s *Server) handleListOrganisations(w http.ResponseWriter, r *http.Request) {
	orgs, err := s.repo.ListOrganisations(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if orgs == nil {
		orgs = []core.Organisation{}
	}
	writeJSON(w, http.StatusOK, orgs)
}

type createOrganisationRequest struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

func (s *Server) handleCreateOrganisation(w http.ResponseWriter, r *http.Request) {
	var req createOrganisationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	name := strings.TrimSpace(req.Name)
	slug := normalizeSlug(req.Slug)
	if slug == "" {
		slug = normalizeSlug(name)
	}
	if name == "" || slug == "" {
		writeError(w, http.StatusBadRequest, "name and slug are required")
		return
	}

	org, err := s.repo.CreateOrganisation(r.Context(), core.Organisation{
		Name: name,
		Slug: slug,
	})
	if err != nil {
		if isPGUniqueViolation(err) {
			writeError(w, http.StatusConflict, "organisation with that slug already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "creating organisation")
		return
	}
	writeJSON(w, http.StatusCreated, org)
}

func (s *Server) handleListProjects(w http.ResponseWriter, r *http.Request) {
	var orgID *uuid.UUID
	if o := strings.TrimSpace(r.URL.Query().Get("org_id")); o != "" {
		parsed, err := uuid.Parse(o)
		if err != nil {
			writeError(w, http.StatusBadRequest, "org_id is not a valid uuid")
			return
		}
		orgID = &parsed
	}
	projects, err := s.repo.ListProjects(r.Context(), orgID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if projects == nil {
		projects = []core.Project{}
	}
	writeJSON(w, http.StatusOK, projects)
}

type createProjectRequest struct {
	OrgID          string `json:"org_id"`
	OrganisationID string `json:"organisation_id"`
	Name           string `json:"name"`
	Slug           string `json:"slug"`
}

func (s *Server) handleCreateProject(w http.ResponseWriter, r *http.Request) {
	var req createProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	rawOrgID := strings.TrimSpace(req.OrgID)
	if rawOrgID == "" {
		rawOrgID = strings.TrimSpace(req.OrganisationID)
	}
	orgID, err := uuid.Parse(rawOrgID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "org_id must be a valid uuid")
		return
	}
	name := strings.TrimSpace(req.Name)
	slug := normalizeSlug(req.Slug)
	if slug == "" {
		slug = normalizeSlug(name)
	}
	if name == "" || slug == "" {
		writeError(w, http.StatusBadRequest, "name and slug are required")
		return
	}

	proj, err := s.repo.CreateProject(r.Context(), core.Project{
		OrgID: orgID,
		Name:  name,
		Slug:  slug,
	})
	if err != nil {
		if isPGUniqueViolation(err) {
			writeError(w, http.StatusConflict, "project with that slug already exists in organisation")
			return
		}
		if isPGForeignKeyViolation(err) {
			writeError(w, http.StatusBadRequest, "referenced organisation does not exist")
			return
		}
		writeError(w, http.StatusInternalServerError, "creating project")
		return
	}
	writeJSON(w, http.StatusCreated, proj)
}

func normalizeSlug(raw string) string {
	raw = strings.ToLower(strings.TrimSpace(raw))
	if raw == "" {
		return ""
	}
	var b strings.Builder
	prevDash := false
	for i := 0; i < len(raw); i++ {
		c := raw[i]
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') {
			b.WriteByte(c)
			prevDash = false
		} else if c == '-' || c == '_' || c == ' ' {
			if !prevDash && b.Len() > 0 {
				b.WriteByte('-')
				prevDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

func isPGUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func isPGForeignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}

func writeJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, statusCode int, message string) {
	writeJSON(w, statusCode, map[string]string{"error": message})
}
