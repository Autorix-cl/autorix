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
	"strings"
	"time"

	"github.com/autorix/argus/internal/core"
	"github.com/autorix/argus/internal/credential"
	"github.com/autorix/platform/health"
	"github.com/autorix/platform/metrics"
	"github.com/autorix/platform/paging"
	"github.com/autorix/platform/tracing"
	"github.com/google/uuid"
)

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
		mux.HandleFunc("GET /v1/instances", s.handleListInstances)
		mux.HandleFunc("GET /admin/instances", s.handleListInstances)
		mux.HandleFunc("GET /v1/instances/{id}", s.handleGetInstance)
		mux.HandleFunc("GET /admin/instances/{id}", s.handleGetInstance)
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
		mux.HandleFunc("GET /v1/stream", s.handleStream)

		// Console Identity & Auth (P3-S1, P3-S2)
		mux.HandleFunc("GET /v1/auth/status", s.handleAuthStatus)
		mux.HandleFunc("POST /v1/auth/bootstrap", s.handleBootstrap)
		mux.HandleFunc("POST /v1/auth/login", s.handleLogin)
		mux.HandleFunc("GET /v1/auth/session", s.handleValidateSession)
		mux.HandleFunc("DELETE /v1/auth/session", s.handleLogout)
		mux.HandleFunc("GET /v1/operators", s.handleListOperators)
	}
	return metrics.HTTPMiddleware("argus")(tracing.HTTPMiddleware()(mux))
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
	EngineType        string `json:"engine_type"`
	Environment       string `json:"environment"`
	UsesAllowed       int    `json:"uses_allowed"`
	ExpiresInSeconds  int    `json:"expires_in_seconds"`
	CreatedBy         string `json:"created_by"`
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
		data, err := json.Marshal(instances)
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
		failed := op.FailedAttempts + 1
		var lockUntil *time.Time
		if failed >= 5 {
			t := now.Add(15 * time.Minute)
			lockUntil = &t
		}
		_ = s.repo.UpdateOperatorFailedAttempts(r.Context(), op.ID, failed, lockUntil)
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

func writeJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, statusCode int, message string) {
	writeJSON(w, statusCode, map[string]string{"error": message})
}
