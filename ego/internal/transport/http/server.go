package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/autorix/ego/internal/core"
	"github.com/autorix/ego/internal/credential"
	"github.com/autorix/ego/internal/session"
	"github.com/autorix/ego/internal/storage/postgres"
	"github.com/autorix/platform/health"
	"github.com/autorix/platform/metrics"
	"github.com/autorix/platform/paging"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/go-webauthn/webauthn/webauthn"
)

var (
	egoActiveSessions = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "autorix_ego_active_sessions",
		Help: "Current number of active sessions in Ego.",
	})

	egoLoginsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "autorix_ego_logins_total",
			Help: "Total number of login attempts in Ego partitioned by status.",
		},
		[]string{"status"},
	)
)

func init() {
	if err := prometheus.Register(egoActiveSessions); err != nil {
		if _, ok := err.(prometheus.AlreadyRegisteredError); !ok {
			// ignore already registered in tests
		}
	}
	if err := prometheus.Register(egoLoginsTotal); err != nil {
		if _, ok := err.(prometheus.AlreadyRegisteredError); !ok {
			// ignore already registered in tests
		}
	}
}

const SessionCookieName = "autorix_session_token"

type Server struct {
	repo           *postgres.Repository
	hasher         *credential.Hasher
	sessionManager *session.Manager
	healthHandler  *health.Handler
	// secureCookies sets the session cookie's Secure attribute. It must be
	// true whenever the engine is reachable over TLS (the default) — false
	// is only for plain-HTTP local development, and must be set explicitly
	// by the caller, never silently assumed.
	secureCookies bool
	webAuthn      *webauthn.WebAuthn
}

func NewServer(repo *postgres.Repository, hasher *credential.Hasher, sm *session.Manager, healthHandler *health.Handler, secureCookies bool) *Server {
	wConfig := &webauthn.Config{
		RPDisplayName: "Autorix",
		RPID:          "localhost",
		RPOrigins:     []string{"http://localhost:3000"},
	}
	wa, _ := webauthn.New(wConfig)
	return &Server{
		repo:           repo,
		hasher:         hasher,
		sessionManager: sm,
		healthHandler:  healthHandler,
		secureCookies:  secureCookies,
		webAuthn:       wa,
	}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	if s.healthHandler != nil {
		s.healthHandler.Register(mux)
	}

	mux.Handle("GET /metrics", metrics.Handler())
	mux.Handle("/metrics", metrics.Handler())

	mux.HandleFunc("GET /self-service/registration/browser", s.handleInitRegistrationFlow)
	mux.HandleFunc("GET /self-service/registration/flows", s.handleFetchRegistrationFlow)
	mux.HandleFunc("POST /self-service/registration", s.handleRegistration)
	mux.HandleFunc("POST /self-service/webauthn/registration/start", s.handleWebAuthnRegistrationStart)
	mux.HandleFunc("POST /self-service/webauthn/registration/finish", s.handleWebAuthnRegistrationFinish)

	mux.HandleFunc("POST /self-service/login", s.handleLogin)
	mux.HandleFunc("GET /sessions/whoami", s.handleWhoAmI)
	mux.HandleFunc("POST /self-service/logout", s.handleLogout)
	mux.HandleFunc("GET /admin/identities", s.handleListIdentities)
	mux.HandleFunc("GET /admin/identities/{id}", s.handleGetIdentity)
	mux.HandleFunc("PATCH /admin/identities/{id}", s.handleUpdateIdentity)
	mux.HandleFunc("DELETE /admin/identities/{id}", s.handleDeleteIdentity)

	mux.HandleFunc("GET /admin/sessions", s.handleListSessions)
	mux.HandleFunc("DELETE /admin/sessions/{id}", s.handleDeleteSession)
	mux.HandleFunc("GET /admin/identities/{id}/sessions", s.handleListIdentitySessions)
	mux.HandleFunc("DELETE /admin/identities/{id}/sessions", s.handleDeleteIdentitySessions)

	mux.HandleFunc("POST /admin/identities/{id}/credentials/reset-password", s.handleAdminResetPassword)
	mux.HandleFunc("POST /admin/identities/{id}/recovery-link", s.handleAdminRecoveryLink)
	mux.HandleFunc("GET /admin/identities/{id}/credentials", s.handleAdminListCredentials)
	mux.HandleFunc("GET /admin/identities/{id}/mfa", s.handleAdminGetMFA)
	mux.HandleFunc("DELETE /admin/identities/{id}/mfa", s.handleAdminDeleteMFA)

	mux.HandleFunc("GET /admin/schemas", s.handleListSchemas)
	mux.HandleFunc("POST /admin/schemas", s.handleCreateSchema)
	mux.HandleFunc("GET /admin/schemas/{id}", s.handleGetSchema)
	mux.HandleFunc("PATCH /admin/schemas/{id}", s.handleUpdateSchema)
	mux.HandleFunc("DELETE /admin/schemas/{id}", s.handleDeleteSchema)

	return mux
}

func (s *Server) handleListIdentities(w http.ResponseWriter, r *http.Request) {
	pageReq := paging.ParseRequest(r)

	filter := postgres.ListIdentitiesFilter{
		Limit:  pageReq.Limit,
		Cursor: pageReq.Cursor,
		Query:  r.URL.Query().Get("q"),
	}
	if filter.Query == "" {
		filter.Query = r.URL.Query().Get("query")
	}

	if states, ok := pageReq.Filter["state"]; ok && len(states) > 0 {
		filter.State = states[0]
	} else if state := r.URL.Query().Get("state"); state != "" {
		filter.State = state
	}

	if schemas, ok := pageReq.Filter["schema_id"]; ok && len(schemas) > 0 {
		filter.SchemaID = schemas[0]
	} else if schemaID := r.URL.Query().Get("schema_id"); schemaID != "" {
		filter.SchemaID = schemaID
	}

	traitFilters := make(map[string]interface{})
	for k, vals := range pageReq.Filter {
		if len(vals) == 0 {
			continue
		}
		if strings.HasPrefix(k, "traits.") {
			traitKey := strings.TrimPrefix(k, "traits.")
			traitFilters[traitKey] = vals[0]
		}
	}
	if len(traitFilters) > 0 {
		filter.Traits = traitFilters
	}

	identities, hasMore, err := s.repo.ListIdentities(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var nextCursor string
	if hasMore && len(identities) > 0 {
		last := identities[len(identities)-1]
		nextCursor = paging.EncodeCursor(last.CreatedAt.Format(time.RFC3339Nano) + "|" + last.ID.String())
	}
	paging.WriteEnvelope(w, identities, nextCursor, hasMore)
}

func (s *Server) handleGetIdentity(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid identity UUID")
		return
	}

	identity, err := s.repo.GetIdentityByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			writeError(w, http.StatusNotFound, "Identity not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, identity)
}

func (s *Server) handleUpdateIdentity(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid identity UUID")
		return
	}

	var payload core.UpdateIdentityPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	identity, err := s.repo.UpdateIdentity(r.Context(), id, payload.Traits, payload.State, payload.SchemaID)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			writeError(w, http.StatusNotFound, "Identity not found")
			return
		}
		if strings.Contains(err.Error(), "invalid identity state") {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, identity)
}

func (s *Server) handleDeleteIdentity(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid identity UUID")
		return
	}

	err = s.repo.DeleteIdentity(r.Context(), id)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			writeError(w, http.StatusNotFound, "Identity not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleRegistration(w http.ResponseWriter, r *http.Request) {
	var payload core.FlowSubmitPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	flowIDStr := r.URL.Query().Get("flow")
	if flowIDStr != "" {
		id, err := uuid.Parse(flowIDStr)
		if err == nil {
			flow, err := s.repo.GetFlow(r.Context(), id)
			if err == nil {
				if flow.CSRFToken != payload.CSRFToken {
					writeError(w, http.StatusForbidden, "Invalid CSRF token")
					return
				}
				if time.Now().After(flow.ExpiresAt) {
					writeError(w, http.StatusBadRequest, "Flow expired")
					return
				}
				_ = s.repo.UpdateFlowState(r.Context(), id, "success")
			}
		}
	}

	if len(payload.Password) > 0 && len(payload.Password) < 8 {
		writeError(w, http.StatusBadRequest, "Password must be at least 8 characters long")
		return
	}

	hash, err := s.hasher.GenerateHash(payload.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to hash password")
		return
	}

	identity, err := s.repo.CreateIdentityWithPassword(r.Context(), payload.Traits, hash)
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}

	sess, err := s.sessionManager.GenerateSession(identity.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create session")
		return
	}

	if err := s.repo.CreateSession(r.Context(), sess); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to persist session")
		return
	}

	egoActiveSessions.Inc()
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    sess.Token,
		Path:     "/",
		Expires:  sess.ExpiresAt,
		HttpOnly: true,
		Secure:   s.secureCookies,
		SameSite: http.SameSiteLaxMode,
	})

	sess.Identity = identity
	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"session":  sess,
		"identity": identity,
	})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var payload core.LoginPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		egoLoginsTotal.WithLabelValues("failure").Inc()
		writeError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	// 1. Find Identity
	identity, err := s.repo.FindIdentityByIdentifier(r.Context(), payload.Identifier)
	if err != nil {
		egoLoginsTotal.WithLabelValues("failure").Inc()
		writeError(w, http.StatusUnauthorized, "Invalid identifier or password")
		return
	}

	// 2. Fetch Password Hash
	hash, err := s.repo.GetPasswordCredential(r.Context(), identity.ID)
	if err != nil {
		egoLoginsTotal.WithLabelValues("failure").Inc()
		writeError(w, http.StatusUnauthorized, "Invalid identifier or password")
		return
	}

	// 3. Verify Argon2id Hash
	valid, err := s.hasher.ComparePasswordAndHash(payload.Password, hash)
	if err != nil || !valid {
		egoLoginsTotal.WithLabelValues("failure").Inc()
		writeError(w, http.StatusUnauthorized, "Invalid identifier or password")
		return
	}

	// 4. Create Session
	sess, err := s.sessionManager.GenerateSession(identity.ID)
	if err != nil {
		egoLoginsTotal.WithLabelValues("failure").Inc()
		writeError(w, http.StatusInternalServerError, "Failed to create session")
		return
	}

	if err := s.repo.CreateSession(r.Context(), sess); err != nil {
		egoLoginsTotal.WithLabelValues("failure").Inc()
		writeError(w, http.StatusInternalServerError, "Failed to persist session")
		return
	}

	egoLoginsTotal.WithLabelValues("success").Inc()
	egoActiveSessions.Inc()

	// Set HTTP-only Cookie
	http.SetCookie(w, &http.Cookie{ // #nosec G124 -- Secure is set from s.secureCookies (config-driven, true by default); gosec cannot verify a non-literal value
		Name:     SessionCookieName,
		Value:    sess.Token,
		Path:     "/",
		Expires:  sess.ExpiresAt,
		HttpOnly: true,
		Secure:   s.secureCookies,
		SameSite: http.SameSiteLaxMode,
	})

	sess.Identity = identity
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"session":  sess,
		"identity": identity,
	})
}

func (s *Server) handleWhoAmI(w http.ResponseWriter, r *http.Request) {
	token := extractToken(r)
	if token == "" {
		writeError(w, http.StatusUnauthorized, "No active session token provided")
		return
	}

	tokenHash := session.HashToken(token)
	sess, err := s.repo.GetSessionByTokenHash(r.Context(), tokenHash)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			writeError(w, http.StatusUnauthorized, "Session expired or invalid")
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to query session")
		return
	}

	writeJSON(w, http.StatusOK, sess)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	token := extractToken(r)
	if token != "" {
		tokenHash := session.HashToken(token)
		_ = s.repo.DeleteSessionByTokenHash(r.Context(), tokenHash)
		egoActiveSessions.Dec()
	}

	// Clear Cookie — attributes must match the cookie being cleared, or
	// some browsers won't recognize it as the same cookie to delete.
	http.SetCookie(w, &http.Cookie{ // #nosec G124 -- Secure is set from s.secureCookies (config-driven, true by default); gosec cannot verify a non-literal value
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   s.secureCookies,
		SameSite: http.SameSiteLaxMode,
	})

	writeJSON(w, http.StatusOK, map[string]string{"status": "logged_out"})
}

func extractToken(r *http.Request) string {
	// 1. Check Authorization Bearer Header
	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}

	// 2. Check Cookie
	if cookie, err := r.Cookie(SessionCookieName); err == nil {
		return cookie.Value
	}

	return ""
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func (s *Server) handleListSessions(w http.ResponseWriter, r *http.Request) {
	pageReq := paging.ParseRequest(r)
	sessions, hasMore, err := s.repo.ListActiveSessions(r.Context(), pageReq.Limit, pageReq.Cursor)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var nextCursor string
	if hasMore && len(sessions) > 0 {
		last := sessions[len(sessions)-1]
		nextCursor = paging.EncodeCursor(last.AuthenticatedAt.Format(time.RFC3339Nano) + "|" + last.ID.String())
	}
	paging.WriteEnvelope(w, sessions, nextCursor, hasMore)
}

func (s *Server) handleDeleteSession(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid session UUID")
		return
	}

	err = s.repo.DeleteSessionByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			writeError(w, http.StatusNotFound, "Session not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	egoActiveSessions.Dec()
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleListIdentitySessions(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid identity UUID")
		return
	}

	sessions, err := s.repo.ListActiveSessionsByIdentity(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if sessions == nil {
		sessions = []core.Session{}
	}

	writeJSON(w, http.StatusOK, sessions)
}

func (s *Server) handleDeleteIdentitySessions(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid identity UUID")
		return
	}

	err = s.repo.DeleteSessionsByIdentityID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	egoActiveSessions.Dec()
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAdminResetPassword(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid identity UUID")
		return
	}

	var payload core.ResetPasswordPayload
	if r.Body != nil && r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid JSON payload")
			return
		}
	}

	var rawPassword string
	var tempPassword *string
	if payload.Password != nil && *payload.Password != "" {
		if len(*payload.Password) < 8 {
			writeError(w, http.StatusBadRequest, "Password must be at least 8 characters long")
			return
		}
		rawPassword = *payload.Password
	} else {
		generated, err := credential.GenerateSecureRandomString(16)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to generate temporary password")
			return
		}
		rawPassword = generated
		tempPassword = &generated
	}

	hash, err := s.hasher.GenerateHash(rawPassword)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to hash password")
		return
	}

	if err := s.repo.SetPasswordCredential(r.Context(), id, hash, payload.ForceRotation); err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			writeError(w, http.StatusNotFound, "Identity not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, core.ResetPasswordResult{
		Status:            "password_reset",
		TemporaryPassword: tempPassword,
		ForceRotation:     payload.ForceRotation,
	})
}

func (s *Server) handleAdminRecoveryLink(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid identity UUID")
		return
	}

	var payload core.RecoveryLinkPayload
	if r.Body != nil && r.ContentLength > 0 {
		_ = json.NewDecoder(r.Body).Decode(&payload)
	}

	duration := 1 * time.Hour
	if payload.ExpiresIn != "" {
		if d, err := time.ParseDuration(payload.ExpiresIn); err == nil && d > 0 {
			duration = d
		}
	}

	rawToken, tokenHash, err := credential.GenerateRecoveryToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to generate recovery token")
		return
	}

	expiresAt := time.Now().Add(duration)
	if err := s.repo.SaveRecoveryToken(r.Context(), id, tokenHash, expiresAt); err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			writeError(w, http.StatusNotFound, "Identity not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, core.RecoveryLinkResult{
		RecoveryLink: "/self-service/recovery?token=" + rawToken,
		Token:        rawToken,
		ExpiresAt:    expiresAt,
	})
}

func (s *Server) handleAdminListCredentials(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid identity UUID")
		return
	}

	creds, err := s.repo.ListCredentialsByIdentity(r.Context(), id)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			writeError(w, http.StatusNotFound, "Identity not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, creds)
}

func (s *Server) handleAdminGetMFA(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid identity UUID")
		return
	}

	_, err = s.repo.GetIdentityByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			writeError(w, http.StatusNotFound, "Identity not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	_, backupCodes, confirmed, err := s.repo.GetTOTPCredential(r.Context(), id)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"totp_enabled": false,
			})
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"totp_enabled":           true,
		"confirmed":              confirmed,
		"backup_codes_remaining": len(backupCodes),
	})
}

func (s *Server) handleAdminDeleteMFA(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid identity UUID")
		return
	}

	err = s.repo.DeleteTOTPCredential(r.Context(), id)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			writeError(w, http.StatusNotFound, "MFA credential not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleListSchemas(w http.ResponseWriter, r *http.Request) {
	schemas, err := s.repo.ListSchemas(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, schemas)
}

func (s *Server) handleCreateSchema(w http.ResponseWriter, r *http.Request) {
	var payload core.CreateSchemaPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}
	if payload.ID == "" {
		writeError(w, http.StatusBadRequest, "Schema ID is required")
		return
	}
	if payload.Name == "" {
		writeError(w, http.StatusBadRequest, "Schema Name is required")
		return
	}
	if payload.Schema == nil {
		writeError(w, http.StatusBadRequest, "Schema definition is required")
		return
	}

	schema := &core.IdentitySchema{
		ID:     payload.ID,
		Name:   payload.Name,
		Schema: payload.Schema,
	}

	created, err := s.repo.CreateSchema(r.Context(), schema)
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) handleGetSchema(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "Schema ID is required")
		return
	}

	schema, err := s.repo.GetSchemaByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			writeError(w, http.StatusNotFound, "Schema not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, schema)
}

func (s *Server) handleUpdateSchema(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "Schema ID is required")
		return
	}

	var payload core.UpdateSchemaPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	updated, err := s.repo.UpdateSchema(r.Context(), id, payload.Name, payload.Schema)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			writeError(w, http.StatusNotFound, "Schema not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) handleDeleteSchema(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "Schema ID is required")
		return
	}

	err := s.repo.DeleteSchema(r.Context(), id)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			writeError(w, http.StatusNotFound, "Schema not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}




func (s *Server) handleInitRegistrationFlow(w http.ResponseWriter, r *http.Request) {
	flow := &core.IdentityFlow{
		ID:        uuid.New(),
		FlowType:  "registration",
		State:     "choose_method",
		CSRFToken: uuid.New().String(),
		ExpiresAt: time.Now().Add(15 * time.Minute),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UINodes: []core.FlowUINode{
			{Type: "input", Group: "password", Attributes: map[string]interface{}{"name": "traits.email", "type": "email"}},
			{Type: "input", Group: "password", Attributes: map[string]interface{}{"name": "password", "type": "password"}},
			{Type: "input", Group: "webauthn", Attributes: map[string]interface{}{"name": "webauthn_register", "type": "button"}},
		},
	}
	if err := s.repo.CreateFlow(r.Context(), flow); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to init flow")
		return
	}
	writeJSON(w, http.StatusOK, flow)
}

func (s *Server) handleFetchRegistrationFlow(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid flow ID")
		return
	}
	flow, err := s.repo.GetFlow(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "Flow not found")
		return
	}
	writeJSON(w, http.StatusOK, flow)
}

func (s *Server) handleWebAuthnRegistrationStart(w http.ResponseWriter, r *http.Request) {
	user := dummyWebAuthnUser{id: []byte(uuid.New().String()), name: "user@example.com"}
	options, sessionData, err := s.webAuthn.BeginRegistration(user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to begin webauthn")
		return
	}
	_ = sessionData 
	writeJSON(w, http.StatusOK, options)
}

func (s *Server) handleWebAuthnRegistrationFinish(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "webauthn_registered"})
}

type dummyWebAuthnUser struct {
	id []byte
	name string
}
func (u dummyWebAuthnUser) WebAuthnID() []byte { return u.id }
func (u dummyWebAuthnUser) WebAuthnName() string { return u.name }
func (u dummyWebAuthnUser) WebAuthnDisplayName() string { return u.name }
func (u dummyWebAuthnUser) WebAuthnIcon() string { return "" }
func (u dummyWebAuthnUser) WebAuthnCredentials() []webauthn.Credential { return nil }
