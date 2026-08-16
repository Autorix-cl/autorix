package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/autorix/ego/internal/core"
	"github.com/autorix/ego/internal/credential"
	"github.com/autorix/ego/internal/session"
	"github.com/autorix/ego/internal/storage/postgres"
)

const SessionCookieName = "autorix_session_token"

type Server struct {
	repo           *postgres.Repository
	hasher         *credential.Hasher
	sessionManager *session.Manager
}

func NewServer(repo *postgres.Repository, hasher *credential.Hasher, sm *session.Manager) *Server {
	return &Server{
		repo:           repo,
		hasher:         hasher,
		sessionManager: sm,
	}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /self-service/registration", s.handleRegistration)
	mux.HandleFunc("POST /self-service/login", s.handleLogin)
	mux.HandleFunc("GET /sessions/whoami", s.handleWhoAmI)
	mux.HandleFunc("POST /self-service/logout", s.handleLogout)

	return mux
}

func (s *Server) handleRegistration(w http.ResponseWriter, r *http.Request) {
	var payload core.RegistrationPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	if len(payload.Password) < 8 {
		writeError(w, http.StatusBadRequest, "Password must be at least 8 characters long")
		return
	}

	// 1. Hash password with Argon2id
	hash, err := s.hasher.GenerateHash(payload.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to hash password")
		return
	}

	// 2. Persist Identity
	identity, err := s.repo.CreateIdentityWithPassword(r.Context(), payload.Traits, hash)
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}

	// 3. Create Session immediately upon registration
	sess, err := s.sessionManager.GenerateSession(identity.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create session")
		return
	}

	if err := s.repo.CreateSession(r.Context(), sess); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to persist session")
		return
	}

	// Set HTTP-only Cookie
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    sess.Token,
		Path:     "/",
		Expires:  sess.ExpiresAt,
		HttpOnly: true,
		Secure:   false, // Set to true in TLS production
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
		writeError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	// 1. Find Identity
	identity, err := s.repo.FindIdentityByIdentifier(r.Context(), payload.Identifier)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "Invalid identifier or password")
		return
	}

	// 2. Fetch Password Hash
	hash, err := s.repo.GetPasswordCredential(r.Context(), identity.ID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "Invalid identifier or password")
		return
	}

	// 3. Verify Argon2id Hash
	valid, err := s.hasher.ComparePasswordAndHash(payload.Password, hash)
	if err != nil || !valid {
		writeError(w, http.StatusUnauthorized, "Invalid identifier or password")
		return
	}

	// 4. Create Session
	sess, err := s.sessionManager.GenerateSession(identity.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create session")
		return
	}

	if err := s.repo.CreateSession(r.Context(), sess); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to persist session")
		return
	}

	// Set HTTP-only Cookie
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    sess.Token,
		Path:     "/",
		Expires:  sess.ExpiresAt,
		HttpOnly: true,
		Secure:   false,
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
	}

	// Clear Cookie
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
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
