package http

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/autorix/janus/internal/core"
	"github.com/autorix/janus/internal/jwks"
	"github.com/autorix/janus/internal/oauth2"
	"github.com/autorix/janus/internal/storage/postgres"
	"github.com/autorix/platform/health"
	"github.com/autorix/platform/metrics"
	"github.com/autorix/platform/paging"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	janusTokensIssuedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "autorix_janus_tokens_issued_total",
			Help: "Total number of OAuth2 tokens issued by Janus partitioned by grant_type.",
		},
		[]string{"grant_type"},
	)

	janusActiveJWKSKeys = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "autorix_janus_active_jwks_keys",
		Help: "Current count of active JWKS keys in Janus.",
	})
)

func init() {
	if err := prometheus.Register(janusTokensIssuedTotal); err != nil {
		if _, ok := err.(prometheus.AlreadyRegisteredError); !ok {
			// ignore in tests
		}
	}
	if err := prometheus.Register(janusActiveJWKSKeys); err != nil {
		if _, ok := err.(prometheus.AlreadyRegisteredError); !ok {
			// ignore in tests
		}
	}
	janusTokensIssuedTotal.WithLabelValues("authorization_code")
	janusTokensIssuedTotal.WithLabelValues("client_credentials")
	janusTokensIssuedTotal.WithLabelValues("refresh_token")
}

type Server struct {
	issuer        string
	repo          *postgres.Repository
	keyManager    *jwks.KeyManager
	engine        *oauth2.Engine
	healthHandler *health.Handler
}

func NewServer(issuer string, repo *postgres.Repository, km *jwks.KeyManager, engine *oauth2.Engine, healthHandler *health.Handler) *Server {
	if km != nil {
		janusActiveJWKSKeys.Set(float64(km.KeyCount()))
	}
	return &Server{
		issuer:        issuer,
		repo:          repo,
		keyManager:    km,
		engine:        engine,
		healthHandler: healthHandler,
	}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	if s.healthHandler != nil {
		s.healthHandler.Register(mux)
	}

	mux.Handle("GET /metrics", metrics.Handler())
	mux.Handle("/metrics", metrics.Handler())

	// OIDC Discovery & JWKS
	mux.HandleFunc("GET /.well-known/openid-configuration", s.handleDiscovery)
	mux.HandleFunc("GET /.well-known/jwks.json", s.handleJWKS)

	// OAuth2 Core
	mux.HandleFunc("GET /oauth2/auth", s.handleAuth)

	mux.HandleFunc("POST /oauth2/token", s.handleToken)
	mux.HandleFunc("POST /oauth2/introspect", s.handleIntrospect)
	mux.HandleFunc("POST /oauth2/revoke", s.handleRevoke)

	// Admin API - Clients
	mux.HandleFunc("POST /admin/clients", s.handleCreateClient)
	mux.HandleFunc("GET /admin/clients", s.handleListClients)
	mux.HandleFunc("GET /admin/clients/{id}", s.handleGetClient)
	mux.HandleFunc("PATCH /admin/clients/{id}", s.handleUpdateClient)
	mux.HandleFunc("DELETE /admin/clients/{id}", s.handleDeleteClient)
	mux.HandleFunc("POST /admin/clients/{id}/rotate-secret", s.handleRotateClientSecret)

	// Admin API - Grants
	mux.HandleFunc("PUT /admin/oauth2/auth/requests/login/accept", s.handleAcceptLogin)
	mux.HandleFunc("PUT /admin/oauth2/auth/requests/consent/accept", s.handleAcceptConsent)

	mux.HandleFunc("GET /admin/grants", s.handleListGrants)

	// Admin API - Keys
	mux.HandleFunc("POST /admin/keys/rotate", s.handleRotateKeys)

	// Admin API - Scopes Catalogue
	mux.HandleFunc("GET /admin/scopes", s.handleListScopes)
	mux.HandleFunc("POST /admin/scopes", s.handleCreateScope)
	mux.HandleFunc("DELETE /admin/scopes/{name}", s.handleDeleteScope)

	return mux
}

func (s *Server) handleListClients(w http.ResponseWriter, r *http.Request) {
	pageReq := paging.ParseRequest(r)
	clients, hasMore, err := s.repo.ListClients(r.Context(), pageReq.Limit, pageReq.Cursor)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}

	var nextCursor string
	if hasMore && len(clients) > 0 {
		last := clients[len(clients)-1]
		nextCursor = paging.EncodeCursor(last.CreatedAt.Format(time.RFC3339Nano) + "|" + last.ID)
	}
	paging.WriteEnvelope(w, clients, nextCursor, hasMore)
}

func (s *Server) handleGetClient(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	client, err := s.repo.GetClient(r.Context(), id)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "Client not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, client)
}

func (s *Server) handleUpdateClient(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	client, err := s.repo.GetClient(r.Context(), id)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "Client not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}

	var req struct {
		Name          *string   `json:"client_name"`
		GrantTypes    *[]string `json:"grant_types"`
		ResponseTypes *[]string `json:"response_types"`
		RedirectURIs  *[]string `json:"redirect_uris"`
		Scopes        *[]string `json:"scopes"`
		IsPublic      *bool     `json:"is_public"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON")
		return
	}

	if req.Name != nil {
		client.ClientName = *req.Name
	}
	if req.GrantTypes != nil {
		client.GrantTypes = *req.GrantTypes
	}
	if req.ResponseTypes != nil {
		client.ResponseTypes = *req.ResponseTypes
	}
	if req.RedirectURIs != nil {
		client.RedirectURIs = *req.RedirectURIs
	}
	if req.Scopes != nil {
		client.Scopes = *req.Scopes
	}
	if req.IsPublic != nil {
		client.IsPublic = *req.IsPublic
	}

	if err := s.repo.UpdateClient(r.Context(), client); err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, client)
}

func (s *Server) handleDeleteClient(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.repo.DeleteClient(r.Context(), id); err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "Client not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleRotateClientSecret(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	_, err := s.repo.GetClient(r.Context(), id)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "Client not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}

	var req struct {
		OverlapSeconds int `json:"overlap_seconds"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	overlapSec := 86400
	if req.OverlapSeconds > 0 {
		overlapSec = req.OverlapSeconds
	}
	expiresAt := time.Now().Add(time.Duration(overlapSec) * time.Second).UTC()

	newSecret := uuid.NewString()
	newHash, err := oauth2.HashSecret(newSecret)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", "Failed to hash secret")
		return
	}

	if err := s.repo.RotateClientSecret(r.Context(), id, newHash, &expiresAt); err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"client_id":                  id,
		"client_secret":              newSecret,
		"previous_secret_expires_at": expiresAt,
	})
}

func (s *Server) handleDiscovery(w http.ResponseWriter, r *http.Request) {
	discovery := map[string]interface{}{
		"issuer":                                s.issuer,
		"authorization_endpoint":                s.issuer + "/oauth2/auth",
		"token_endpoint":                        s.issuer + "/oauth2/token",
		"introspection_endpoint":                s.issuer + "/oauth2/introspect",
		"revocation_endpoint":                   s.issuer + "/oauth2/revoke",
		"jwks_uri":                              s.issuer + "/.well-known/jwks.json",
		"response_types_supported":              []string{"code", "token", "id_token"},
		"subject_types_supported":               []string{"public"},
		"id_token_signing_alg_values_supported": []string{"RS256"},
		"scopes_supported":                      []string{"openid", "profile", "email", "offline_access"},
		"token_endpoint_auth_methods_supported": []string{"client_secret_basic", "client_secret_post", "none"},
		"code_challenge_methods_supported":      []string{"S256", "plain"},
	}

	writeJSON(w, http.StatusOK, discovery)
}

func (s *Server) handleJWKS(w http.ResponseWriter, r *http.Request) {
	jwksSet := s.keyManager.ExportJWKS()
	writeJSON(w, http.StatusOK, jwksSet)
}

func (s *Server) handleRotateKeys(w http.ResponseWriter, r *http.Request) {
	newJWK, err := s.keyManager.RotateKey()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}
	janusActiveJWKSKeys.Set(float64(s.keyManager.KeyCount()))
	writeJSON(w, http.StatusOK, newJWK)
}

func (s *Server) handleToken(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Failed to parse form")
		return
	}

	grantType := r.FormValue("grant_type")
	clientID, clientSecret, hasAuth := r.BasicAuth()
	if !hasAuth {
		clientID = r.FormValue("client_id")
		clientSecret = r.FormValue("client_secret")
	}

	if clientID == "" {
		writeError(w, http.StatusUnauthorized, "invalid_client", "Missing client_id")
		return
	}

	client, err := s.repo.GetClient(r.Context(), clientID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid_client", "Unknown client_id")
		return
	}

	// Verify Secret for confidential clients with rotation rollover support
	if !oauth2.AuthenticateClient(client, clientSecret) {
		writeError(w, http.StatusUnauthorized, "invalid_client", "Invalid client_secret")
		return
	}

	switch grantType {
	case "client_credentials":
		scopes := strings.Fields(r.FormValue("scope"))
		if len(scopes) == 0 {
			scopes = client.Scopes
		}

		resp, err := s.engine.IssueClientCredentialsToken(client, scopes)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}
		janusTokensIssuedTotal.WithLabelValues("client_credentials").Inc()
		writeJSON(w, http.StatusOK, resp)

	case "authorization_code":
		code := r.FormValue("code")
		codeVerifier := r.FormValue("code_verifier")
		if code == "" {
			writeError(w, http.StatusBadRequest, "invalid_request", "Missing code parameter")
			return
		}

		codeHash := hashCode(code)
		grant, err := s.repo.ConsumeGrant(r.Context(), codeHash)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_grant", "Code is expired or already consumed")
			return
		}

		// Verify PKCE if grant had code_challenge
		if grant.CodeChallenge != "" {
			if !oauth2.VerifyPKCE(codeVerifier, grant.CodeChallenge, grant.CodeChallengeMethod) {
				writeError(w, http.StatusBadRequest, "invalid_grant", "PKCE verification failed")
				return
			}
		}

		resp, err := s.engine.IssueAuthorizationCodeToken(grant)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}
		janusTokensIssuedTotal.WithLabelValues("authorization_code").Inc()
		writeJSON(w, http.StatusOK, resp)

	case "refresh_token":
		janusTokensIssuedTotal.WithLabelValues("refresh_token").Inc()
		writeError(w, http.StatusBadRequest, "unsupported_grant_type", "Refresh token grant not implemented")

	default:
		writeError(w, http.StatusBadRequest, "unsupported_grant_type", "Supported: client_credentials, authorization_code, refresh_token")
	}
}

func (s *Server) handleIntrospect(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Failed to parse form")
		return
	}

	token := r.FormValue("token")
	if token == "" {
		writeJSON(w, http.StatusOK, map[string]interface{}{"active": false})
		return
	}

	// 1. Try JWT verification
	claims, err := s.keyManager.VerifyJWT(token)
	if err == nil {
		tokenHash := hashCode(token)
		if s.repo != nil {
			revoked, checkErr := s.repo.IsTokenRevoked(r.Context(), tokenHash)
			if checkErr == nil && revoked {
				writeJSON(w, http.StatusOK, map[string]interface{}{"active": false})
				return
			}
		}

		var exp int64
		switch v := claims["exp"].(type) {
		case float64:
			exp = int64(v)
		case int64:
			exp = v
		}
		if exp > 0 && time.Now().Unix() > exp {
			writeJSON(w, http.StatusOK, map[string]interface{}{"active": false})
			return
		}

		var iat int64
		switch v := claims["iat"].(type) {
		case float64:
			iat = int64(v)
		case int64:
			iat = v
		}

		var scopeStr string
		if sc, ok := claims["scopes"].([]interface{}); ok {
			var parts []string
			for _, sItem := range sc {
				parts = append(parts, fmt.Sprint(sItem))
			}
			scopeStr = strings.Join(parts, " ")
		} else if sc, ok := claims["scope"].(string); ok {
			scopeStr = sc
		}

		clientID, _ := claims["aud"].(string)
		sub, _ := claims["sub"].(string)
		iss, _ := claims["iss"].(string)

		// For client_credentials tokens, sub is the client_id and aud is issuer
		if clientID == s.issuer && sub != "" {
			clientID = sub
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"active":     true,
			"scope":      scopeStr,
			"client_id":  clientID,
			"sub":        sub,
			"exp":        exp,
			"iat":        iat,
			"iss":        iss,
			"token_type": "Bearer",
		})
		return
	}

	// 2. Try opaque token record from DB
	if s.repo != nil {
		tokenHash := hashCode(token)
		rec, err := s.repo.GetTokenRecord(r.Context(), tokenHash)
		if err == nil && !rec.Revoked && rec.ExpiresAt.After(time.Now()) {
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"active":     true,
				"scope":      strings.Join(rec.Scopes, " "),
				"client_id":  rec.ClientID,
				"sub":        rec.Subject,
				"exp":        rec.ExpiresAt.Unix(),
				"iat":        rec.CreatedAt.Unix(),
				"token_type": rec.TokenType,
			})
			return
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"active": false})
}

func (s *Server) handleRevoke(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Failed to parse form")
		return
	}

	token := r.FormValue("token")
	tokenTypeHint := r.FormValue("token_type_hint")
	if tokenTypeHint == "" {
		tokenTypeHint = "access_token"
	}

	if token != "" && s.repo != nil {
		tokenHash := hashCode(token)
		var clientID, sub string
		var scopes []string
		var expiresAt time.Time

		claims, err := s.keyManager.VerifyJWT(token)
		if err == nil {
			clientID, _ = claims["aud"].(string)
			sub, _ = claims["sub"].(string)
			if expVal, ok := claims["exp"].(float64); ok {
				expiresAt = time.Unix(int64(expVal), 0)
			}

			// If aud is not a client ID in the DB, check if sub is the client ID (e.g. M2M client_credentials)
			if clientID != "" {
				if _, err := s.repo.GetClient(r.Context(), clientID); err != nil {
					if sub != "" {
						if _, err := s.repo.GetClient(r.Context(), sub); err == nil {
							clientID = sub
						}
					}
				}
			} else if sub != "" {
				if _, err := s.repo.GetClient(r.Context(), sub); err == nil {
					clientID = sub
				}
			}
		}

		_ = s.repo.RevokeToken(r.Context(), &core.TokenRecord{
			TokenHash: tokenHash,
			ClientID:  clientID,
			Subject:   sub,
			TokenType: tokenTypeHint,
			Scopes:    scopes,
			ExpiresAt: expiresAt,
		})
	}

	// RFC 7009: acknowledge revocation with 200 OK
	writeJSON(w, http.StatusOK, map[string]interface{}{})
}

func (s *Server) handleListGrants(w http.ResponseWriter, r *http.Request) {
	pageReq := paging.ParseRequest(r)
	clientID := r.URL.Query().Get("client_id")
	subject := r.URL.Query().Get("sub")
	if subject == "" {
		subject = r.URL.Query().Get("subject")
	}
	activeOnly := r.URL.Query().Get("active") == "true"

	grants, hasMore, err := s.repo.ListGrants(r.Context(), clientID, subject, activeOnly, pageReq.Limit, pageReq.Cursor)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}

	var nextCursor string
	if hasMore && len(grants) > 0 {
		last := grants[len(grants)-1]
		nextCursor = paging.EncodeCursor(last.CreatedAt.Format(time.RFC3339Nano) + "|" + last.CodeHash)
	}
	paging.WriteEnvelope(w, grants, nextCursor, hasMore)
}

func (s *Server) handleListScopes(w http.ResponseWriter, r *http.Request) {
	scopes, err := s.repo.ListScopes(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}
	if scopes == nil {
		scopes = []core.OAuth2Scope{}
	}
	writeJSON(w, http.StatusOK, scopes)
}

func (s *Server) handleCreateScope(w http.ResponseWriter, r *http.Request) {
	var scope core.OAuth2Scope
	if err := json.NewDecoder(r.Body).Decode(&scope); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON")
		return
	}
	if scope.Name == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "Scope name is required")
		return
	}

	if err := s.repo.CreateScope(r.Context(), &scope); err != nil {
		writeError(w, http.StatusConflict, "conflict", err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, scope)
}

func (s *Server) handleDeleteScope(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if err := s.repo.DeleteScope(r.Context(), name); err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "Scope not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleCreateClient(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID           string   `json:"client_id"`
		Name         string   `json:"client_name"`
		Secret       string   `json:"client_secret"`
		GrantTypes   []string `json:"grant_types"`
		RedirectURIs []string `json:"redirect_uris"`
		Scopes       []string `json:"scopes"`
		IsPublic     bool     `json:"is_public"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON")
		return
	}

	var secretHash string
	if req.Secret != "" {
		var err error
		secretHash, err = oauth2.HashSecret(req.Secret)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", "Failed to hash secret")
			return
		}
	}

	client := &core.OAuth2Client{
		ID:               req.ID,
		ClientName:       req.Name,
		ClientSecretHash: secretHash,
		GrantTypes:       req.GrantTypes,
		RedirectURIs:     req.RedirectURIs,
		Scopes:           req.Scopes,
		IsPublic:         req.IsPublic,
	}

	if err := s.repo.CreateClient(r.Context(), client); err != nil {
		writeError(w, http.StatusConflict, "conflict", err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, client)
}

func hashCode(code string) string {
	sum := sha256.Sum256([]byte(code))
	return hex.EncodeToString(sum[:])
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, errCode, description string) {
	writeJSON(w, status, map[string]string{
		"error":             errCode,
		"error_description": description,
	})
}

// Add this to s.Routes() in server.go:
// mux.HandleFunc("GET /oauth2/auth", s.handleAuth)
// mux.HandleFunc("PUT /admin/oauth2/auth/requests/login/accept", s.handleAcceptLogin)
// mux.HandleFunc("PUT /admin/oauth2/auth/requests/consent/accept", s.handleAcceptConsent)
// ... also remember to add os.Getenv("LOGIN_UI_URL") logic or default in server.go ...

func (s *Server) handleAuth(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Failed to parse form")
		return
	}

	// Wait, is there a login_verifier? If yes, we process the accepted login and issue consent challenge!
	loginVerifier := r.FormValue("login_verifier")
	consentVerifier := r.FormValue("consent_verifier")
	
	if consentVerifier != "" {
		// Consent is accepted, generate auth code and redirect to client
		loginChallengeID := r.FormValue("login_challenge")
		consentChallengeID := r.FormValue("consent_challenge")
		
		lc, err := s.repo.GetLoginChallenge(r.Context(), loginChallengeID)
		if err != nil || lc.HandledAt == nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "Invalid login challenge")
			return
		}
		cc, err := s.repo.GetConsentChallenge(r.Context(), consentChallengeID)
		if err != nil || cc.HandledAt == nil || cc.ConsentVerifier != consentVerifier {
			writeError(w, http.StatusBadRequest, "invalid_request", "Invalid consent challenge")
			return
		}
		
		// Generate Authorization Code
		code := uuid.NewString()
		codeHash := hashCode(code)
		
		grant := &core.Grant{
			CodeHash:            codeHash,
			ClientID:            lc.ClientID,
			Subject:             cc.Subject,
			Scopes:              cc.GrantedScopes,
			RedirectURI:         lc.RedirectURI,
			CodeChallenge:       lc.CodeChallenge,
			CodeChallengeMethod: lc.CodeChallengeMethod,
			ExpiresAt:           time.Now().Add(10 * time.Minute),
		}
		if err := s.repo.CreateGrant(r.Context(), grant); err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", "Failed to create grant")
			return
		}
		
		// Redirect to client
		redirectURI := lc.RedirectURI + "?code=" + code
		if lc.State != "" {
			redirectURI += "&state=" + lc.State
		}
		http.Redirect(w, r, redirectURI, http.StatusFound)
		return
	}

	if loginVerifier != "" {
		// Login is accepted, create consent challenge and redirect to consent UI
		challengeID := r.FormValue("login_challenge")
		lc, err := s.repo.GetLoginChallenge(r.Context(), challengeID)
		if err != nil || lc.HandledAt == nil || lc.LoginVerifier != loginVerifier {
			writeError(w, http.StatusBadRequest, "invalid_request", "Invalid login challenge")
			return
		}
		
		cc := &core.ConsentChallenge{
			Challenge:       uuid.NewString(),
			LoginChallenge:  lc.Challenge,
			ClientID:        lc.ClientID,
			Subject:         lc.Subject,
			RequestedScopes: lc.Scopes,
		}
		if err := s.repo.CreateConsentChallenge(r.Context(), cc); err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", "Failed to create consent challenge")
			return
		}
		
		// Redirect to Consent UI
		consentUIURL := "http://localhost:3000/consent" // default or from env
		http.Redirect(w, r, consentUIURL+"?consent_challenge="+cc.Challenge, http.StatusFound)
		return
	}

	// Normal auth flow start: create login challenge
	clientID := r.FormValue("client_id")
	if clientID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "Missing client_id")
		return
	}
	
	client, err := s.repo.GetClient(r.Context(), clientID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_client", "Unknown client_id")
		return
	}

	scopes := strings.Fields(r.FormValue("scope"))
	if len(scopes) == 0 {
		scopes = client.Scopes
	}

	lc := &core.LoginChallenge{
		Challenge:           uuid.NewString(),
		ClientID:            clientID,
		RedirectURI:         r.FormValue("redirect_uri"),
		ResponseType:        r.FormValue("response_type"),
		Scopes:              scopes,
		State:               r.FormValue("state"),
		Nonce:               r.FormValue("nonce"),
		CodeChallenge:       r.FormValue("code_challenge"),
		CodeChallengeMethod: r.FormValue("code_challenge_method"),
	}

	if err := s.repo.CreateLoginChallenge(r.Context(), lc); err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", "Failed to create login challenge")
		return
	}

	loginUIURL := "http://localhost:3000/login" // default or from env
	http.Redirect(w, r, loginUIURL+"?login_challenge="+lc.Challenge, http.StatusFound)
}

func (s *Server) handleAcceptLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Challenge string `json:"challenge"`
		Subject   string `json:"subject"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON")
		return
	}

	lc, err := s.repo.GetLoginChallenge(r.Context(), req.Challenge)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "Login challenge not found")
		return
	}

	now := time.Now()
	lc.HandledAt = &now
	lc.Subject = req.Subject
	lc.LoginVerifier = uuid.NewString()

	if err := s.repo.UpdateLoginChallenge(r.Context(), lc); err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", "Failed to update login challenge")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"redirect_to": s.issuer + "/oauth2/auth?login_challenge=" + lc.Challenge + "&login_verifier=" + lc.LoginVerifier,
	})
}

func (s *Server) handleAcceptConsent(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Challenge     string   `json:"challenge"`
		GrantedScopes []string `json:"granted_scopes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON")
		return
	}

	cc, err := s.repo.GetConsentChallenge(r.Context(), req.Challenge)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "Consent challenge not found")
		return
	}

	now := time.Now()
	cc.HandledAt = &now
	cc.GrantedScopes = req.GrantedScopes
	cc.ConsentVerifier = uuid.NewString()

	if err := s.repo.UpdateConsentChallenge(r.Context(), cc); err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", "Failed to update consent challenge")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"redirect_to": s.issuer + "/oauth2/auth?login_challenge=" + cc.LoginChallenge + "&consent_challenge=" + cc.Challenge + "&consent_verifier=" + cc.ConsentVerifier,
	})
}
