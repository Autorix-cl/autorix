package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/autorix/vulcan/internal/core"
	"github.com/autorix/vulcan/internal/keygen"
	"github.com/autorix/vulcan/internal/macaroon"
	"github.com/autorix/vulcan/internal/storage/postgres"
	"github.com/google/uuid"
)

type Server struct {
	repo     *postgres.Repository
	location string
}

func NewServer(repo *postgres.Repository, location string) *Server {
	return &Server{
		repo:     repo,
		location: location,
	}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /keys", s.handleCreateKey)
	mux.HandleFunc("POST /keys/attenuate", s.handleAttenuate)
	mux.HandleFunc("POST /keys/verify", s.handleVerify)
	mux.HandleFunc("DELETE /keys/{id}", s.handleRevoke)

	return mux
}

func (s *Server) handleCreateKey(w http.ResponseWriter, r *http.Request) {
	var req core.CreateKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	if req.Name == "" || req.OwnerID == "" {
		writeError(w, http.StatusBadRequest, "name and owner_id are required")
		return
	}

	// 1. Generate key and root signature key
	gen, err := keygen.GenerateAPIKey(req.IsLive)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to generate key")
		return
	}

	keyID := uuid.New()
	apiKey := &core.APIKey{
		ID:               keyID,
		KeyPrefix:        gen.Prefix,
		KeyHint:          gen.Hint,
		KeyHash:          gen.KeyHash,
		RootSignatureKey: gen.RootKey,
		Name:             req.Name,
		OwnerID:          req.OwnerID,
		Scopes:           req.Scopes,
		ExpiresAt:        req.ExpiresAt,
		State:            "active",
	}

	// 2. Persist to DB
	if err := s.repo.CreateKey(r.Context(), apiKey); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// 3. Issue root Macaroon
	rootMacaroon := macaroon.New(s.location, keyID.String(), gen.RootKey)

	writeJSON(w, http.StatusCreated, core.CreateKeyResponse{
		APIKey:   apiKey,
		RawToken: gen.RawToken,
		Macaroon: rootMacaroon,
	})
}

func (s *Server) handleAttenuate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Macaroon *core.Macaroon `json:"macaroon"`
		Caveat   string         `json:"caveat"` // e.g. "time_before = 2026-08-17T00:00:00Z"
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Macaroon == nil || req.Caveat == "" {
		writeError(w, http.StatusBadRequest, "macaroon and caveat are required")
		return
	}

	attenuated, err := macaroon.Attenuate(req.Macaroon, req.Caveat)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, attenuated)
}

func (s *Server) handleVerify(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Macaroon *core.Macaroon            `json:"macaroon"`
		Context  *core.VerificationContext `json:"context,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Macaroon == nil {
		writeError(w, http.StatusBadRequest, "macaroon is required")
		return
	}

	keyUUID, err := uuid.Parse(req.Macaroon.KeyID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid macaroon key_id")
		return
	}

	// 1. Fetch Root Signature Key from DB
	apiKey, err := s.repo.GetKeyByID(r.Context(), keyUUID)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			writeError(w, http.StatusUnauthorized, "API key not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "Database error")
		return
	}

	if apiKey.State != "active" {
		writeError(w, http.StatusForbidden, "API key is revoked or expired")
		return
	}

	// 2. Set default time context if not provided
	if req.Context == nil {
		req.Context = &core.VerificationContext{Now: time.Now()}
	} else if req.Context.Now.IsZero() {
		req.Context.Now = time.Now()
	}

	// 3. Verify Chained Signature and Caveats
	valid, err := macaroon.Verify(req.Macaroon, apiKey.RootSignatureKey, req.Context)
	if err != nil || !valid {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	// Update last used asynchronously
	go s.repo.UpdateLastUsed(context.Background(), apiKey.ID)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"valid":   true,
		"api_key": apiKey,
	})
}

func (s *Server) handleRevoke(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	keyUUID, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid UUID")
		return
	}

	if err := s.repo.RevokeKey(r.Context(), keyUUID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "revoked"})
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
