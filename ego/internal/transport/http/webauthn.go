package http

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/autorix/ego/internal/core"
	"github.com/autorix/ego/internal/session"
	"github.com/autorix/ego/internal/storage/postgres"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
)

const (
	webAuthnChallengeTTL      = 5 * time.Minute
	maxWebAuthnChallengeCount = 10_000
)

type webAuthnStore interface {
	GetSessionByTokenHash(ctx context.Context, tokenHash string) (*core.Session, error)
	GetIdentityByID(ctx context.Context, id uuid.UUID) (*core.Identity, error)
	FindIdentityByIdentifier(ctx context.Context, identifier string) (*core.Identity, error)
	GetWebAuthnCredentials(ctx context.Context, identityID uuid.UUID) ([]webauthn.Credential, error)
	SaveWebAuthnCredentials(ctx context.Context, identityID uuid.UUID, creds []webauthn.Credential) error
	CreateSession(ctx context.Context, s *core.Session) error
}

type webAuthnUser struct {
	identity    *core.Identity
	credentials []webauthn.Credential
}

func (u *webAuthnUser) WebAuthnID() []byte {
	return u.identity.ID[:]
}

func (u *webAuthnUser) WebAuthnName() string {
	if email, ok := u.identity.Traits["email"].(string); ok && strings.TrimSpace(email) != "" {
		return strings.TrimSpace(email)
	}
	return u.identity.ID.String()
}

func (u *webAuthnUser) WebAuthnDisplayName() string {
	if nameMap, ok := u.identity.Traits["name"].(map[string]interface{}); ok {
		first, _ := nameMap["first"].(string)
		last, _ := nameMap["last"].(string)
		if full := strings.TrimSpace(first + " " + last); full != "" {
			return full
		}
	}
	if name, ok := u.identity.Traits["name"].(string); ok && strings.TrimSpace(name) != "" {
		return strings.TrimSpace(name)
	}
	return u.WebAuthnName()
}

func (u *webAuthnUser) WebAuthnCredentials() []webauthn.Credential {
	return u.credentials
}

type webAuthnChallengeEntry struct {
	kind       string // "registration" or "login"
	identityID uuid.UUID
	session    *webauthn.SessionData
	expiresAt  time.Time
}

type webAuthnChallengeStore struct {
	mu      sync.Mutex
	entries map[string]webAuthnChallengeEntry
	now     func() time.Time
}

func newWebAuthnChallengeStore() *webAuthnChallengeStore {
	return &webAuthnChallengeStore{
		entries: make(map[string]webAuthnChallengeEntry),
		now:     time.Now,
	}
}

func (c *webAuthnChallengeStore) Put(kind string, identityID uuid.UUID, data *webauthn.SessionData) string {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := c.now()
	for k, v := range c.entries {
		if now.After(v.expiresAt) {
			delete(c.entries, k)
		}
	}
	if len(c.entries) >= maxWebAuthnChallengeCount {
		// Evict oldest entry when bounded capacity is reached.
		var oldestKey string
		var oldestTime time.Time
		for k, v := range c.entries {
			if oldestKey == "" || v.expiresAt.Before(oldestTime) {
				oldestKey = k
				oldestTime = v.expiresAt
			}
		}
		if oldestKey != "" {
			delete(c.entries, oldestKey)
		}
	}

	id := uuid.NewString()
	c.entries[id] = webAuthnChallengeEntry{
		kind:       kind,
		identityID: identityID,
		session:    data,
		expiresAt:  now.Add(webAuthnChallengeTTL),
	}
	return id
}

// Consume pops and returns a challenge entry once (preventing replay attacks).
func (c *webAuthnChallengeStore) Consume(id, expectedKind string) (webAuthnChallengeEntry, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.entries[id]
	if !ok {
		return webAuthnChallengeEntry{}, false
	}
	delete(c.entries, id)

	if entry.kind != expectedKind || c.now().After(entry.expiresAt) {
		return webAuthnChallengeEntry{}, false
	}
	return entry, true
}

func newWebAuthnFromEnv() (*webauthn.WebAuthn, error) {
	rpName := strings.TrimSpace(os.Getenv("WEBAUTHN_RP_DISPLAY_NAME"))
	if rpName == "" {
		rpName = "Autorix"
	}
	rpID := strings.TrimSpace(os.Getenv("WEBAUTHN_RP_ID"))
	if rpID == "" {
		rpID = "localhost"
	}
	rawOrigins := strings.TrimSpace(os.Getenv("WEBAUTHN_RP_ORIGINS"))
	var origins []string
	if rawOrigins != "" {
		for _, o := range strings.Split(rawOrigins, ",") {
			if trimmed := strings.TrimSpace(o); trimmed != "" {
				origins = append(origins, trimmed)
			}
		}
	}
	if len(origins) == 0 {
		origins = []string{"http://localhost:3000", "http://localhost:4433"}
	}

	return webauthn.New(&webauthn.Config{
		RPDisplayName: rpName,
		RPID:          rpID,
		RPOrigins:     origins,
	})
}

func (s *Server) getWebAuthnStore() webAuthnStore {
	if s.waStore != nil {
		return s.waStore
	}
	if s.repo != nil {
		return s.repo
	}
	return nil
}

func (s *Server) authenticatedIdentity(r *http.Request) (*core.Identity, int, string) {
	store := s.getWebAuthnStore()
	if store == nil {
		return nil, http.StatusServiceUnavailable, "identity store is not configured"
	}
	token := extractToken(r)
	if token == "" {
		return nil, http.StatusUnauthorized, "No active session token provided"
	}
	sess, err := store.GetSessionByTokenHash(r.Context(), session.HashToken(token))
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, http.StatusUnauthorized, "Session expired or invalid"
		}
		return nil, http.StatusInternalServerError, "Failed to query session"
	}
	if !sess.IsActive() {
		return nil, http.StatusUnauthorized, "Session expired or invalid"
	}
	identity := sess.Identity
	if identity == nil {
		identity, err = store.GetIdentityByID(r.Context(), sess.IdentityID)
		if err != nil {
			return nil, http.StatusUnauthorized, "Identity not found"
		}
	}
	if identity.State != core.StateActive {
		return nil, http.StatusForbidden, "Identity is not active"
	}
	return identity, http.StatusOK, ""
}

func (s *Server) handleWebAuthnRegistrationStart(w http.ResponseWriter, r *http.Request) {
	if s.wa == nil {
		writeError(w, http.StatusServiceUnavailable, "webauthn is not configured")
		return
	}
	identity, code, msg := s.authenticatedIdentity(r)
	if identity == nil {
		writeError(w, code, msg)
		return
	}
	store := s.getWebAuthnStore()
	creds, err := store.GetWebAuthnCredentials(r.Context(), identity.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load existing passkeys")
		return
	}

	user := &webAuthnUser{identity: identity, credentials: creds}
	excludeList := make([]protocol.CredentialDescriptor, 0, len(creds))
	for _, c := range creds {
		excludeList = append(excludeList, c.Descriptor())
	}

	creation, sessData, err := s.wa.BeginRegistration(user, webauthn.WithExclusions(excludeList))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to start WebAuthn registration")
		return
	}

	challengeID := s.waChallenges.Put("registration", identity.ID, sessData)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"challenge_id": challengeID,
		"options":      creation,
	})
}

type webAuthnFinishEnvelope struct {
	ChallengeID string          `json:"challenge_id"`
	Credential  json.RawMessage `json:"credential"`
}

func decodeWebAuthnFinishRequest(r *http.Request) (challengeID string, credentialBytes []byte, err error) {
	var raw bytes.Buffer
	if _, err := raw.ReadFrom(r.Body); err != nil {
		return "", nil, err
	}
	var env webAuthnFinishEnvelope
	if err := json.Unmarshal(raw.Bytes(), &env); err == nil && len(env.Credential) > 0 {
		cid := strings.TrimSpace(env.ChallengeID)
		if cid == "" {
			cid = strings.TrimSpace(r.Header.Get("X-WebAuthn-Challenge-ID"))
		}
		return cid, env.Credential, nil
	}
	cid := strings.TrimSpace(r.Header.Get("X-WebAuthn-Challenge-ID"))
	if cid == "" {
		cid = strings.TrimSpace(r.URL.Query().Get("challenge_id"))
	}
	return cid, raw.Bytes(), nil
}

func (s *Server) handleWebAuthnRegistrationFinish(w http.ResponseWriter, r *http.Request) {
	if s.wa == nil {
		writeError(w, http.StatusServiceUnavailable, "webauthn is not configured")
		return
	}
	identity, code, msg := s.authenticatedIdentity(r)
	if identity == nil {
		writeError(w, code, msg)
		return
	}

	challengeID, credBytes, err := decodeWebAuthnFinishRequest(r)
	if err != nil || challengeID == "" || len(credBytes) == 0 {
		writeError(w, http.StatusBadRequest, "challenge_id and WebAuthn credential payload are required")
		return
	}

	entry, ok := s.waChallenges.Consume(challengeID, "registration")
	if !ok {
		writeError(w, http.StatusUnauthorized, "Invalid, expired, or already used WebAuthn challenge")
		return
	}
	if entry.identityID != identity.ID {
		writeError(w, http.StatusForbidden, "WebAuthn challenge does not belong to authenticated identity")
		return
	}

	parsed, err := protocol.ParseCredentialCreationResponseBytes(credBytes)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid WebAuthn attestation response")
		return
	}

	store := s.getWebAuthnStore()
	existing, err := store.GetWebAuthnCredentials(r.Context(), identity.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load existing passkeys")
		return
	}

	user := &webAuthnUser{identity: identity, credentials: existing}
	cred, err := s.wa.CreateCredential(user, *entry.session, parsed)
	if err != nil {
		writeError(w, http.StatusBadRequest, "WebAuthn attestation verification failed")
		return
	}

	updated := append(existing, *cred)
	if err := store.SaveWebAuthnCredentials(r.Context(), identity.ID, updated); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to persist WebAuthn credential")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"status":        "registered",
		"credential_id": base64.RawURLEncoding.EncodeToString(cred.ID),
	})
}

type webAuthnLoginStartRequest struct {
	Identifier string `json:"identifier"`
	Email      string `json:"email"`
	IdentityID string `json:"identity_id"`
}

func (s *Server) handleWebAuthnLoginStart(w http.ResponseWriter, r *http.Request) {
	if s.wa == nil {
		writeError(w, http.StatusServiceUnavailable, "webauthn is not configured")
		return
	}
	store := s.getWebAuthnStore()
	if store == nil {
		writeError(w, http.StatusServiceUnavailable, "identity store is not configured")
		return
	}

	var req webAuthnLoginStartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	var identity *core.Identity
	var err error
	if idStr := strings.TrimSpace(req.IdentityID); idStr != "" {
		id, parseErr := uuid.Parse(idStr)
		if parseErr != nil {
			writeError(w, http.StatusBadRequest, "Invalid identity_id")
			return
		}
		identity, err = store.GetIdentityByID(r.Context(), id)
	} else {
		identifier := strings.TrimSpace(req.Identifier)
		if identifier == "" {
			identifier = strings.TrimSpace(req.Email)
		}
		if identifier == "" {
			writeError(w, http.StatusBadRequest, "identifier is required")
			return
		}
		identity, err = store.FindIdentityByIdentifier(r.Context(), identifier)
	}

	if err != nil || identity == nil || identity.State != core.StateActive {
		writeError(w, http.StatusUnauthorized, "Invalid identifier or no passkeys registered")
		return
	}

	creds, err := store.GetWebAuthnCredentials(r.Context(), identity.ID)
	if err != nil || len(creds) == 0 {
		writeError(w, http.StatusUnauthorized, "Invalid identifier or no passkeys registered")
		return
	}

	user := &webAuthnUser{identity: identity, credentials: creds}
	assertion, sessData, err := s.wa.BeginLogin(user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to start WebAuthn login")
		return
	}

	challengeID := s.waChallenges.Put("login", identity.ID, sessData)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"challenge_id": challengeID,
		"options":      assertion,
	})
}

func (s *Server) handleWebAuthnLoginFinish(w http.ResponseWriter, r *http.Request) {
	if s.wa == nil {
		writeError(w, http.StatusServiceUnavailable, "webauthn is not configured")
		return
	}
	store := s.getWebAuthnStore()
	if store == nil {
		writeError(w, http.StatusServiceUnavailable, "identity store is not configured")
		return
	}

	challengeID, credBytes, err := decodeWebAuthnFinishRequest(r)
	if err != nil || challengeID == "" || len(credBytes) == 0 {
		egoLoginsTotal.WithLabelValues("failure").Inc()
		writeError(w, http.StatusBadRequest, "challenge_id and WebAuthn assertion payload are required")
		return
	}

	entry, ok := s.waChallenges.Consume(challengeID, "login")
	if !ok {
		egoLoginsTotal.WithLabelValues("failure").Inc()
		writeError(w, http.StatusUnauthorized, "Invalid, expired, or already used WebAuthn challenge")
		return
	}

	identity, err := store.GetIdentityByID(r.Context(), entry.identityID)
	if err != nil || identity == nil || identity.State != core.StateActive {
		egoLoginsTotal.WithLabelValues("failure").Inc()
		writeError(w, http.StatusUnauthorized, "Identity is inactive or unavailable")
		return
	}

	creds, err := store.GetWebAuthnCredentials(r.Context(), identity.ID)
	if err != nil || len(creds) == 0 {
		egoLoginsTotal.WithLabelValues("failure").Inc()
		writeError(w, http.StatusUnauthorized, "No passkeys registered for identity")
		return
	}

	parsed, err := protocol.ParseCredentialRequestResponseBytes(credBytes)
	if err != nil {
		egoLoginsTotal.WithLabelValues("failure").Inc()
		writeError(w, http.StatusBadRequest, "Invalid WebAuthn assertion response")
		return
	}

	user := &webAuthnUser{identity: identity, credentials: creds}
	matched, err := s.wa.ValidateLogin(user, *entry.session, parsed)
	if err != nil {
		egoLoginsTotal.WithLabelValues("failure").Inc()
		writeError(w, http.StatusUnauthorized, "WebAuthn assertion verification failed")
		return
	}
	if matched.Authenticator.CloneWarning {
		egoLoginsTotal.WithLabelValues("failure").Inc()
		writeError(w, http.StatusForbidden, "Authenticator clone warning detected")
		return
	}

	// Persist updated sign count for anti-cloning protection.
	for i := range creds {
		if bytes.Equal(creds[i].ID, matched.ID) {
			creds[i].Authenticator.SignCount = matched.Authenticator.SignCount
			creds[i].Flags = matched.Flags
			break
		}
	}
	if err := store.SaveWebAuthnCredentials(r.Context(), identity.ID, creds); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update passkey sign count")
		return
	}

	sess, err := s.sessionManager.GenerateSession(identity.ID)
	if err != nil {
		egoLoginsTotal.WithLabelValues("failure").Inc()
		writeError(w, http.StatusInternalServerError, "Failed to create session")
		return
	}
	if err := store.CreateSession(r.Context(), sess); err != nil {
		egoLoginsTotal.WithLabelValues("failure").Inc()
		writeError(w, http.StatusInternalServerError, "Failed to persist session")
		return
	}

	egoLoginsTotal.WithLabelValues("success").Inc()
	egoActiveSessions.Inc()

	http.SetCookie(w, &http.Cookie{ // #nosec G124 -- Secure is set from s.secureCookies (config-driven, true by default)
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
