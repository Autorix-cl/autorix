package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/autorix/hermes/internal/core"
	"github.com/autorix/hermes/internal/saml"
	"github.com/autorix/hermes/internal/scim"
	"github.com/autorix/hermes/internal/storage/postgres"
	"github.com/autorix/platform/health"
	"github.com/autorix/platform/metrics"
	"github.com/autorix/platform/paging"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	hermesSAMLLoginsTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "autorix_hermes_saml_logins_total",
			Help: "Total number of SAML logins processed by Hermes.",
		},
	)

	hermesSCIMSyncTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "autorix_hermes_scim_sync_total",
			Help: "Total number of SCIM sync operations partitioned by status (success, failure).",
		},
		[]string{"status"},
	)
)

func init() {
	if err := prometheus.Register(hermesSAMLLoginsTotal); err != nil {
		if _, ok := err.(prometheus.AlreadyRegisteredError); !ok {
			// ignore in tests
		}
	}
	if err := prometheus.Register(hermesSCIMSyncTotal); err != nil {
		if _, ok := err.(prometheus.AlreadyRegisteredError); !ok {
			// ignore in tests
		}
	}
}

type Server struct {
	repo          *postgres.Repository
	samlSP        *saml.ServiceProvider
	scimEngine    *scim.Engine
	spEntityID    string
	healthHandler *health.Handler
}

func NewServer(repo *postgres.Repository, baseURL, spEntityID string, healthHandler *health.Handler) *Server {
	return &Server{
		repo:          repo,
		samlSP:        saml.NewServiceProvider(baseURL),
		scimEngine:    scim.NewEngine(baseURL),
		spEntityID:    spEntityID,
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

	// SAML 2.0 Endpoints
	mux.HandleFunc("GET /saml/metadata", s.handleSAMLMetadata)
	mux.HandleFunc("GET /saml/login", s.handleSAMLLogin)
	mux.HandleFunc("POST /saml/acs", s.handleSAMLACS)

	// SAML Admin / Provider Lifecycle Endpoints
	mux.HandleFunc("POST /admin/saml/providers", s.handleCreateSAMLProvider)
	mux.HandleFunc("GET /admin/saml/providers", s.handleListSAMLProviders)
	mux.HandleFunc("POST /admin/providers", s.handleCreateSAMLProvider)
	mux.HandleFunc("GET /admin/providers", s.handleListSAMLProviders)

	mux.HandleFunc("GET /admin/providers/{id}", s.handleGetSAMLProvider)
	mux.HandleFunc("GET /admin/saml/providers/{id}", s.handleGetSAMLProvider)
	mux.HandleFunc("PATCH /admin/providers/{id}", s.handleUpdateSAMLProvider)
	mux.HandleFunc("PATCH /admin/saml/providers/{id}", s.handleUpdateSAMLProvider)
	mux.HandleFunc("DELETE /admin/providers/{id}", s.handleDeleteSAMLProvider)
	mux.HandleFunc("DELETE /admin/saml/providers/{id}", s.handleDeleteSAMLProvider)
	mux.HandleFunc("POST /admin/providers/{id}/enable", s.handleEnableSAMLProvider)
	mux.HandleFunc("POST /admin/saml/providers/{id}/enable", s.handleEnableSAMLProvider)
	mux.HandleFunc("POST /admin/providers/{id}/disable", s.handleDisableSAMLProvider)
	mux.HandleFunc("POST /admin/saml/providers/{id}/disable", s.handleDisableSAMLProvider)

	// SCIM 2.0 Endpoints (RFC 7644)
	mux.HandleFunc("GET /scim/v2/ServiceProviderConfig", s.handleSCIMConfig)
	mux.HandleFunc("GET /scim/v2/Users", s.handleListSCIMUsers)
	mux.HandleFunc("POST /scim/v2/Users", s.handleCreateSCIMUser)
	mux.HandleFunc("GET /scim/v2/Groups", s.handleListSCIMGroups)
	mux.HandleFunc("POST /scim/v2/Groups", s.handleCreateSCIMGroup)
	mux.HandleFunc("GET /scim/v2/Groups/{id}", s.handleGetSCIMGroup)
	mux.HandleFunc("PUT /scim/v2/Groups/{id}", s.handleUpdateSCIMGroup)
	mux.HandleFunc("PATCH /scim/v2/Groups/{id}", s.handleUpdateSCIMGroup)
	mux.HandleFunc("DELETE /scim/v2/Groups/{id}", s.handleDeleteSCIMGroup)

	// SCIM Admin Endpoints
	mux.HandleFunc("GET /admin/scim/sync-history", s.handleListSCIMSyncHistory)
	mux.HandleFunc("POST /admin/scim/sync-history", s.handleRecordSCIMSync)

	return mux
}

func (s *Server) handleListSAMLProviders(w http.ResponseWriter, r *http.Request) {
	pageReq := paging.ParseRequest(r)
	providers, hasMore, err := s.repo.ListSAMLProviders(r.Context(), pageReq.Limit, pageReq.Cursor)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var nextCursor string
	if hasMore && len(providers) > 0 {
		last := providers[len(providers)-1]
		nextCursor = paging.EncodeCursor(last.CreatedAt.Format(time.RFC3339Nano) + "|" + last.ID)
	}
	paging.WriteEnvelope(w, providers, nextCursor, hasMore)
}

func (s *Server) handleGetSAMLProvider(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "Missing provider id")
		return
	}

	provider, err := s.repo.GetSAMLProvider(r.Context(), id)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			writeError(w, http.StatusNotFound, "Provider not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, provider)
}

func (s *Server) handleUpdateSAMLProvider(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "Missing provider id")
		return
	}

	provider, err := s.repo.GetSAMLProvider(r.Context(), id)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			writeError(w, http.StatusNotFound, "Provider not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var updates struct {
		DisplayName       *string           `json:"display_name"`
		IdPEntityID       *string           `json:"idp_entity_id"`
		IdPSSOURL         *string           `json:"idp_sso_url"`
		IdPCertificatePEM *string           `json:"idp_certificate_pem"`
		SPEntityID        *string           `json:"sp_entity_id"`
		AttributeMapping  map[string]string `json:"attribute_mapping"`
		Enabled           *bool             `json:"enabled"`
	}

	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON payload: "+err.Error())
		return
	}

	if updates.DisplayName != nil {
		provider.DisplayName = *updates.DisplayName
	}
	if updates.IdPEntityID != nil {
		provider.IdPEntityID = *updates.IdPEntityID
	}
	if updates.IdPSSOURL != nil {
		provider.IdPSSOURL = *updates.IdPSSOURL
	}
	if updates.IdPCertificatePEM != nil {
		provider.IdPCertificatePEM = *updates.IdPCertificatePEM
	}
	if updates.SPEntityID != nil {
		provider.SPEntityID = *updates.SPEntityID
	}
	if updates.AttributeMapping != nil {
		provider.AttributeMapping = updates.AttributeMapping
	}
	if updates.Enabled != nil {
		provider.Enabled = *updates.Enabled
	}

	if err := s.repo.UpdateSAMLProvider(r.Context(), provider); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, provider)
}

func (s *Server) handleDeleteSAMLProvider(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "Missing provider id")
		return
	}

	if err := s.repo.DeleteSAMLProvider(r.Context(), id); err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			writeError(w, http.StatusNotFound, "Provider not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleEnableSAMLProvider(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "Missing provider id")
		return
	}

	if err := s.repo.SetSAMLProviderEnabled(r.Context(), id, true); err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			writeError(w, http.StatusNotFound, "Provider not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	provider, err := s.repo.GetSAMLProvider(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, provider)
}

func (s *Server) handleDisableSAMLProvider(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "Missing provider id")
		return
	}

	if err := s.repo.SetSAMLProviderEnabled(r.Context(), id, false); err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			writeError(w, http.StatusNotFound, "Provider not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	provider, err := s.repo.GetSAMLProvider(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, provider)
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

	http.Redirect(w, r, redirectURL, http.StatusFound) // #nosec G710
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

	hermesSAMLLoginsTotal.Inc()

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

func (s *Server) handleListSCIMGroups(w http.ResponseWriter, r *http.Request) {
	groups, err := s.repo.ListSCIMGroups(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var formatted []interface{}
	for i := range groups {
		formatted = append(formatted, s.scimEngine.FormatGroup(&groups[i]))
	}

	resp := core.SCIMListResponse{
		Schemas:      []string{scim.ListResponseURN},
		TotalResults: len(groups),
		StartIndex:   1,
		ItemsPerPage: len(groups),
		Resources:    formatted,
	}

	w.Header().Set("Content-Type", "application/scim+json")
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleCreateSCIMGroup(w http.ResponseWriter, r *http.Request) {
	var g core.SCIMGroup
	if err := json.NewDecoder(r.Body).Decode(&g); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid SCIM JSON payload: "+err.Error())
		return
	}

	if strings.TrimSpace(g.DisplayName) == "" {
		writeError(w, http.StatusBadRequest, "displayName is required by SCIM 2.0")
		return
	}

	if err := s.repo.CreateSCIMGroup(r.Context(), &g); err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}

	formatted := s.scimEngine.FormatGroup(&g)
	w.Header().Set("Content-Type", "application/scim+json")
	writeJSON(w, http.StatusCreated, formatted)
}

func (s *Server) handleGetSCIMGroup(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid group UUID")
		return
	}

	group, err := s.repo.GetSCIMGroup(r.Context(), id)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			writeError(w, http.StatusNotFound, "Group not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	formatted := s.scimEngine.FormatGroup(group)
	w.Header().Set("Content-Type", "application/scim+json")
	writeJSON(w, http.StatusOK, formatted)
}

func (s *Server) handleUpdateSCIMGroup(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid group UUID")
		return
	}

	group, err := s.repo.GetSCIMGroup(r.Context(), id)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			writeError(w, http.StatusNotFound, "Group not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var updates struct {
		DisplayName string            `json:"displayName"`
		Members     []core.SCIMMember `json:"members"`
	}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON payload: "+err.Error())
		return
	}

	if updates.DisplayName != "" {
		group.DisplayName = updates.DisplayName
	}
	if updates.Members != nil {
		group.Members = updates.Members
	}

	if err := s.repo.UpdateSCIMGroup(r.Context(), group); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	formatted := s.scimEngine.FormatGroup(group)
	w.Header().Set("Content-Type", "application/scim+json")
	writeJSON(w, http.StatusOK, formatted)
}

func (s *Server) handleDeleteSCIMGroup(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid group UUID")
		return
	}

	if err := s.repo.DeleteSCIMGroup(r.Context(), id); err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			writeError(w, http.StatusNotFound, "Group not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleListSCIMSyncHistory(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if lStr := r.URL.Query().Get("limit"); lStr != "" {
		if l, err := strconv.Atoi(lStr); err == nil && l > 0 {
			limit = l
		}
	}

	history, err := s.repo.ListSCIMSyncHistory(r.Context(), limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, history)
}

func (s *Server) handleRecordSCIMSync(w http.ResponseWriter, r *http.Request) {
	var sync core.SCIMSyncHistory
	if err := json.NewDecoder(r.Body).Decode(&sync); err != nil {
		hermesSCIMSyncTotal.WithLabelValues("failure").Inc()
		writeError(w, http.StatusBadRequest, "Invalid JSON payload: "+err.Error())
		return
	}

	if err := s.repo.RecordSCIMSync(r.Context(), &sync); err != nil {
		hermesSCIMSyncTotal.WithLabelValues("failure").Inc()
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if sync.Status == "success" && sync.ErrorCount == 0 {
		hermesSCIMSyncTotal.WithLabelValues("success").Inc()
	} else {
		hermesSCIMSyncTotal.WithLabelValues("failure").Inc()
	}

	writeJSON(w, http.StatusCreated, sync)
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

