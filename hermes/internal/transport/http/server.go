package http

import (
	"encoding/json"
	"net/http"

	"github.com/autorix/hermes/internal/core"
	"github.com/autorix/hermes/internal/saml"
	"github.com/autorix/hermes/internal/scim"
	"github.com/autorix/hermes/internal/storage/postgres"
)

type Server struct {
	repo       *postgres.Repository
	samlSP     *saml.ServiceProvider
	scimEngine *scim.Engine
	spEntityID string
}

func NewServer(repo *postgres.Repository, baseURL, spEntityID string) *Server {
	return &Server{
		repo:       repo,
		samlSP:     saml.NewServiceProvider(baseURL),
		scimEngine: scim.NewEngine(baseURL),
		spEntityID: spEntityID,
	}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	// SAML 2.0 Endpoints
	mux.HandleFunc("GET /saml/metadata", s.handleSAMLMetadata)
	mux.HandleFunc("GET /saml/login", s.handleSAMLLogin)
	mux.HandleFunc("POST /saml/acs", s.handleSAMLACS)
	mux.HandleFunc("POST /admin/saml/providers", s.handleCreateSAMLProvider)

	// SCIM 2.0 Endpoints (RFC 7644)
	mux.HandleFunc("GET /scim/v2/ServiceProviderConfig", s.handleSCIMConfig)
	mux.HandleFunc("GET /scim/v2/Users", s.handleListSCIMUsers)
	mux.HandleFunc("POST /scim/v2/Users", s.handleCreateSCIMUser)

	return mux
}

func (s *Server) handleSAMLMetadata(w http.ResponseWriter, r *http.Request) {
	xmlMetadata := s.samlSP.GenerateSPMetadataXML(s.spEntityID)
	w.Header().Set("Content-Type", "application/samlmetadata+xml")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(xmlMetadata))
}

func (s *Server) handleSAMLLogin(w http.ResponseWriter, r *http.Request) {
	providerID := r.URL.Query().Get("provider")
	if providerID == "" {
		writeError(w, http.StatusBadRequest, "Missing provider query parameter")
		return
	}

	provider, err := s.repo.GetSAMLProvider(r.Context(), providerID)
	if err != nil {
		writeError(w, http.StatusNotFound, "SAML Provider not found")
		return
	}

	redirectURL, err := s.samlSP.GenerateAuthnRequestURL(provider)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to generate AuthnRequest")
		return
	}

	http.Redirect(w, r, redirectURL, http.StatusFound)
}

func (s *Server) handleSAMLACS(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		writeError(w, http.StatusBadRequest, "Failed to parse form")
		return
	}

	samlResponse := r.FormValue("SAMLResponse")
	if samlResponse == "" {
		writeError(w, http.StatusBadRequest, "Missing SAMLResponse")
		return
	}

	assertion, err := s.samlSP.ParseAssertion(samlResponse, nil)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid SAML Assertion: "+err.Error())
		return
	}

	// In real execution, Hermes provisions the user in Ego and starts an OIDC flow in Janus
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":    "authenticated",
		"assertion": assertion,
	})
}

func (s *Server) handleCreateSAMLProvider(w http.ResponseWriter, r *http.Request) {
	var p core.SAMLProvider
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	p.SPEntityID = s.spEntityID
	if err := s.repo.CreateSAMLProvider(r.Context(), &p); err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, p)
}

func (s *Server) handleSCIMConfig(w http.ResponseWriter, r *http.Request) {
	config := s.scimEngine.ServiceProviderConfig()
	w.Header().Set("Content-Type", "application/scim+json")
	writeJSON(w, http.StatusOK, config)
}

func (s *Server) handleListSCIMUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.repo.ListSCIMUsers(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var formatted []interface{}
	for i := range users {
		formatted = append(formatted, s.scimEngine.FormatUser(&users[i]))
	}

	resp := core.SCIMListResponse{
		Schemas:      []string{scim.ListResponseURN},
		TotalResults: len(users),
		StartIndex:   1,
		ItemsPerPage: len(users),
		Resources:    formatted,
	}

	w.Header().Set("Content-Type", "application/scim+json")
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleCreateSCIMUser(w http.ResponseWriter, r *http.Request) {
	var u core.SCIMUser
	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid SCIM JSON payload")
		return
	}

	if u.UserName == "" {
		writeError(w, http.StatusBadRequest, "userName is required by SCIM 2.0")
		return
	}

	if err := s.repo.CreateSCIMUser(r.Context(), &u); err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}

	formatted := s.scimEngine.FormatUser(&u)
	w.Header().Set("Content-Type", "application/scim+json")
	writeJSON(w, http.StatusCreated, formatted)
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
