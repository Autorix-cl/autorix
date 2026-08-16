package http

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/autorix/janus/internal/core"
	"github.com/autorix/janus/internal/jwks"
	"github.com/autorix/janus/internal/oauth2"
	"github.com/autorix/janus/internal/storage/postgres"
)

type Server struct {
	issuer     string
	repo       *postgres.Repository
	keyManager *jwks.KeyManager
	engine     *oauth2.Engine
}

func NewServer(issuer string, repo *postgres.Repository, km *jwks.KeyManager, engine *oauth2.Engine) *Server {
	return &Server{
		issuer:     issuer,
		repo:       repo,
		keyManager: km,
		engine:     engine,
	}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	// OIDC Discovery & JWKS
	mux.HandleFunc("GET /.well-known/openid-configuration", s.handleDiscovery)
	mux.HandleFunc("GET /.well-known/jwks.json", s.handleJWKS)

	// OAuth2 Core
	mux.HandleFunc("POST /oauth2/token", s.handleToken)

	// Admin API
	mux.HandleFunc("POST /admin/clients", s.handleCreateClient)

	return mux
}

func (s *Server) handleDiscovery(w http.ResponseWriter, r *http.Request) {
	discovery := map[string]interface{}{
		"issuer":                                s.issuer,
		"authorization_endpoint":                s.issuer + "/oauth2/auth",
		"token_endpoint":                        s.issuer + "/oauth2/token",
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

	// Verify Secret for confidential clients
	if !client.IsPublic && client.ClientSecretHash != "" {
		if !oauth2.VerifySecret(clientSecret, client.ClientSecretHash) {
			writeError(w, http.StatusUnauthorized, "invalid_client", "Invalid client_secret")
			return
		}
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
		writeJSON(w, http.StatusOK, resp)

	default:
		writeError(w, http.StatusBadRequest, "unsupported_grant_type", "Supported: client_credentials, authorization_code")
	}
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
