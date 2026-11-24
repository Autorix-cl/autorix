package http

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"time"

	"github.com/autorix/platform/cache"
	"github.com/autorix/platform/health"
	"github.com/autorix/platform/metrics"
	"github.com/autorix/platform/paging"
	"github.com/autorix/vulcan/internal/core"
	"github.com/autorix/vulcan/internal/keygen"
	"github.com/autorix/vulcan/internal/macaroon"
	"github.com/autorix/vulcan/internal/storage/postgres"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	maxRequestBodyBytes = 1 << 20 // 1 MiB
	maxGracePeriod      = 168 * time.Hour
	maxSourceIPLength   = 64
	usageRecordTimeout  = 2 * time.Second
)

type keyRepository interface {
	CreateKey(ctx context.Context, k *core.APIKey) error
	GetKeyByID(ctx context.Context, id uuid.UUID) (*core.APIKey, error)
	UpdateKey(ctx context.Context, id uuid.UUID, req core.UpdateKeyRequest) (*core.APIKey, error)
	RotateKey(ctx context.Context, id uuid.UUID, newHash, newHint, newRootKey string, gracePeriodExpiresAt *time.Time) (*core.APIKey, error)
	RevokeKey(ctx context.Context, id uuid.UUID) error
	RecordUsage(ctx context.Context, id uuid.UUID, sourceIP string) error
	ListKeys(ctx context.Context, limit int, cursor string) ([]core.APIKey, bool, error)
	ListScopes(ctx context.Context) ([]core.Scope, error)
	CreateScope(ctx context.Context, s *core.Scope) error
	DeleteScope(ctx context.Context, name string) error
}

// cachedAPIKey is an internal cache DTO that preserves RootSignatureKey and
// PrevRootSignatureKey across JSON serialization while core.APIKey keeps
// json:"-" on secret fields so they are never exposed in HTTP responses.
type cachedAPIKey struct {
	APIKey               core.APIKey `json:"api_key"`
	RootSignatureKey     string      `json:"root_signature_key"`
	PrevRootSignatureKey string      `json:"prev_root_signature_key,omitempty"`
}

func toCachedAPIKey(k *core.APIKey) cachedAPIKey {
	return cachedAPIKey{
		APIKey:               *k,
		RootSignatureKey:     k.RootSignatureKey,
		PrevRootSignatureKey: k.PrevRootSignatureKey,
	}
}

func (c *cachedAPIKey) toAPIKey() *core.APIKey {
	k := c.APIKey
	k.RootSignatureKey = c.RootSignatureKey
	k.PrevRootSignatureKey = c.PrevRootSignatureKey
	return &k
}

var (
	vulcanKeysVerifiedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "autorix_vulcan_keys_verified_total",
			Help: "Total number of API keys/macaroons verification requests in Vulcan partitioned by status (valid, invalid, expired).",
		},
		[]string{"status"},
	)
)

func init() {
	if err := prometheus.Register(vulcanKeysVerifiedTotal); err != nil {
		if _, ok := err.(prometheus.AlreadyRegisteredError); !ok {
			// ignore in tests
		}
	}
}

type Server struct {
	repo          keyRepository
	location      string
	healthHandler *health.Handler
	cache         cache.Cache
}

func NewServer(repo *postgres.Repository, location string, healthHandler *health.Handler) *Server {
	var r keyRepository
	if repo != nil {
		r = repo
	}
	return &Server{
		repo:          r,
		location:      location,
		healthHandler: healthHandler,
	}
}

func (s *Server) SetCache(c cache.Cache) {
	s.cache = c
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	if s.healthHandler != nil {
		s.healthHandler.Register(mux)
	}

	mux.Handle("GET /metrics", metrics.Handler())
	mux.Handle("/metrics", metrics.Handler())

	mux.HandleFunc("POST /keys", s.handleCreateKey)
	mux.HandleFunc("GET /keys", s.handleListKeys)
	mux.HandleFunc("POST /keys/attenuate", s.handleAttenuate)
	mux.HandleFunc("POST /keys/verify", s.handleVerify)
	mux.HandleFunc("DELETE /keys/{id}", s.handleRevoke)

	// Admin Key Lifecycle & Rotation API (P6-S6-T1)
	mux.HandleFunc("GET /admin/keys/{id}", s.handleAdminGetKey)
	mux.HandleFunc("PATCH /admin/keys/{id}", s.handleAdminUpdateKey)
	mux.HandleFunc("POST /admin/keys/{id}/rotate", s.handleAdminRotateKey)

	// Admin Scope Catalogue API (P6-S6-T3)
	mux.HandleFunc("GET /admin/scopes", s.handleAdminListScopes)
	mux.HandleFunc("POST /admin/scopes", s.handleAdminCreateScope)
	mux.HandleFunc("DELETE /admin/scopes/{name}", s.handleAdminDeleteScope)

	return mux
}

func (s *Server) handleListKeys(w http.ResponseWriter, r *http.Request) {
	pageReq := paging.ParseRequest(r)
	keys, hasMore, err := s.repo.ListKeys(r.Context(), pageReq.Limit, pageReq.Cursor)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if keys == nil {
		keys = make([]core.APIKey, 0)
	}

	var nextCursor string
	if hasMore && len(keys) > 0 {
		last := keys[len(keys)-1]
		nextCursor = paging.EncodeCursor(last.CreatedAt.Format(time.RFC3339Nano) + "|" + last.ID.String())
	}
	paging.WriteEnvelope(w, keys, nextCursor, hasMore)
}

func (s *Server) handleCreateKey(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
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
		Description:      req.Description,
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
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
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
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	var req struct {
		Macaroon *core.Macaroon            `json:"macaroon"`
		Context  *core.VerificationContext `json:"context,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Macaroon == nil {
		vulcanKeysVerifiedTotal.WithLabelValues("invalid").Inc()
		writeError(w, http.StatusBadRequest, "macaroon is required")
		return
	}

	keyUUID, err := uuid.Parse(req.Macaroon.KeyID)
	if err != nil {
		vulcanKeysVerifiedTotal.WithLabelValues("invalid").Inc()
		writeError(w, http.StatusBadRequest, "Invalid macaroon key_id")
		return
	}

	// 1. Fetch Root Signature Key from Cache or DB
	var apiKey *core.APIKey
	cacheKey := "vulcan:key:" + keyUUID.String()

	if s.cache != nil {
		if cachedBytes, err := s.cache.Get(r.Context(), cacheKey); err == nil {
			var cached cachedAPIKey
			if json.Unmarshal(cachedBytes, &cached) == nil && cached.RootSignatureKey != "" {
				apiKey = cached.toAPIKey()
			}
		}
	}

	if apiKey == nil {
		var err error
		apiKey, err = s.repo.GetKeyByID(r.Context(), keyUUID)
		if err != nil {
			vulcanKeysVerifiedTotal.WithLabelValues("invalid").Inc()
			if errors.Is(err, postgres.ErrNotFound) {
				writeError(w, http.StatusUnauthorized, "API key not found")
				return
			}
			writeError(w, http.StatusInternalServerError, "Database error")
			return
		}
		if s.cache != nil && apiKey.State == "active" && apiKey.RootSignatureKey != "" {
			if data, err := json.Marshal(toCachedAPIKey(apiKey)); err == nil {
				_ = s.cache.Set(r.Context(), cacheKey, data, 2*time.Minute)
			}
		}
	}

	now := time.Now()
	if apiKey.State == "expired" || (apiKey.ExpiresAt != nil && now.After(*apiKey.ExpiresAt)) {
		vulcanKeysVerifiedTotal.WithLabelValues("expired").Inc()
		writeError(w, http.StatusForbidden, "API key is expired")
		return
	}

	if apiKey.State != "active" {
		vulcanKeysVerifiedTotal.WithLabelValues("invalid").Inc()
		writeError(w, http.StatusForbidden, "API key is revoked or expired")
		return
	}

	// 2. Always enforce server clock for time_before caveats; ignore client-supplied context.now.
	verifyCtx := &core.VerificationContext{Now: now}
	if req.Context != nil {
		verifyCtx.IPAddress = req.Context.IPAddress
		verifyCtx.Method = req.Context.Method
		verifyCtx.Path = req.Context.Path
	}

	// 3. Verify Chained Signature and Caveats; fall back to previous root key only on signature mismatch.
	valid, err := macaroon.Verify(req.Macaroon, apiKey.RootSignatureKey, verifyCtx)
	if errors.Is(err, macaroon.ErrInvalidSignature) &&
		apiKey.PrevRootSignatureKey != "" &&
		apiKey.GracePeriodExpiresAt != nil &&
		now.Before(*apiKey.GracePeriodExpiresAt) {
		valid, err = macaroon.Verify(req.Macaroon, apiKey.PrevRootSignatureKey, verifyCtx)
	}

	if err != nil || !valid {
		if err != nil && (strings.Contains(err.Error(), "expired") || strings.Contains(err.Error(), "time_before")) {
			vulcanKeysVerifiedTotal.WithLabelValues("expired").Inc()
		} else {
			vulcanKeysVerifiedTotal.WithLabelValues("invalid").Inc()
		}
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	vulcanKeysVerifiedTotal.WithLabelValues("valid").Inc()

	// Usage tracking (P6-S6-T2): record last used timestamp, call count, and validated source IP
	sourceIP := extractSourceIP(verifyCtx.IPAddress, r)
	if s.repo != nil {
		go func(keyID uuid.UUID, ip string) {
			ctx, cancel := context.WithTimeout(context.Background(), usageRecordTimeout)
			defer cancel()
			_ = s.repo.RecordUsage(ctx, keyID, ip)
		}(apiKey.ID, sourceIP)
	}

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
		if errors.Is(err, postgres.ErrNotFound) {
			writeError(w, http.StatusNotFound, "API key not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "Database error")
		return
	}

	if s.cache != nil {
		_ = s.cache.Delete(r.Context(), "vulcan:key:"+keyUUID.String())
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "revoked"})
}

// Admin Key Lifecycle Handlers (P6-S6-T1)

func (s *Server) handleAdminGetKey(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	keyUUID, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid UUID")
		return
	}

	apiKey, err := s.repo.GetKeyByID(r.Context(), keyUUID)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			writeError(w, http.StatusNotFound, "API key not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, apiKey)
}

func (s *Server) handleAdminUpdateKey(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	idStr := r.PathValue("id")
	keyUUID, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid UUID")
		return
	}

	var req core.UpdateKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	existing, err := s.repo.GetKeyByID(r.Context(), keyUUID)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			writeError(w, http.StatusNotFound, "API key not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if existing.State == "revoked" {
		writeError(w, http.StatusBadRequest, "Cannot update revoked key")
		return
	}

	apiKey, err := s.repo.UpdateKey(r.Context(), keyUUID, req)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			writeError(w, http.StatusNotFound, "API key not found")
			return
		}
		if errors.Is(err, postgres.ErrKeyRevoked) {
			writeError(w, http.StatusBadRequest, "Cannot update revoked key")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if s.cache != nil {
		_ = s.cache.Delete(r.Context(), "vulcan:key:"+keyUUID.String())
	}

	writeJSON(w, http.StatusOK, apiKey)
}

func (s *Server) handleAdminRotateKey(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	idStr := r.PathValue("id")
	keyUUID, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid UUID")
		return
	}

	var req core.RotateKeyRequest
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
			writeError(w, http.StatusBadRequest, "Invalid JSON payload")
			return
		}
	}

	graceDuration := 24 * time.Hour
	if req.GracePeriod != "" {
		parsed, err := time.ParseDuration(req.GracePeriod)
		if err != nil || parsed < 0 || parsed > maxGracePeriod {
			writeError(w, http.StatusBadRequest, "Invalid grace_period duration (must be between 0s and 168h)")
			return
		}
		graceDuration = parsed
	}

	existing, err := s.repo.GetKeyByID(r.Context(), keyUUID)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			writeError(w, http.StatusNotFound, "API key not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if existing.State != "active" || (existing.ExpiresAt != nil && time.Now().After(*existing.ExpiresAt)) {
		writeError(w, http.StatusBadRequest, "Cannot rotate inactive or expired key")
		return
	}

	isLive := existing.KeyPrefix == "av_live"
	gen, err := keygen.GenerateAPIKey(isLive)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to generate key")
		return
	}

	gracePeriodExpiresAt := time.Now().Add(graceDuration)
	rotatedKey, err := s.repo.RotateKey(r.Context(), keyUUID, gen.KeyHash, gen.Hint, gen.RootKey, &gracePeriodExpiresAt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if s.cache != nil {
		_ = s.cache.Delete(r.Context(), "vulcan:key:"+keyUUID.String())
	}

	newMacaroon := macaroon.New(s.location, keyUUID.String(), gen.RootKey)

	writeJSON(w, http.StatusOK, core.RotateKeyResponse{
		APIKey:   rotatedKey,
		RawToken: gen.RawToken,
		Macaroon: newMacaroon,
	})
}

// Admin Scope Catalogue Handlers (P6-S6-T3)

func (s *Server) handleAdminListScopes(w http.ResponseWriter, r *http.Request) {
	scopes, err := s.repo.ListScopes(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if scopes == nil {
		scopes = make([]core.Scope, 0)
	}

	writeJSON(w, http.StatusOK, scopes)
}

func (s *Server) handleAdminCreateScope(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	var req core.CreateScopeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	scope := &core.Scope{
		Name:        req.Name,
		Description: req.Description,
	}

	if err := s.repo.CreateScope(r.Context(), scope); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, scope)
}

func (s *Server) handleAdminDeleteScope(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	if err := s.repo.DeleteScope(r.Context(), name); err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			writeError(w, http.StatusNotFound, "Scope not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// extractSourceIP selects and validates a client IP from the verification
// context, X-Forwarded-For, or RemoteAddr, returning a canonical IP string
// capped at maxSourceIPLength (or "" if no valid IP is present).
func extractSourceIP(ctxIP string, r *http.Request) string {
	candidates := []string{ctxIP}
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		first := strings.TrimSpace(strings.Split(xff, ",")[0])
		candidates = append(candidates, first)
	}
	candidates = append(candidates, r.RemoteAddr)

	for _, raw := range candidates {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		host := raw
		if h, _, err := net.SplitHostPort(raw); err == nil {
			host = h
		}
		host = strings.Trim(host, "[]")
		if addr, err := netip.ParseAddr(host); err == nil {
			clean := addr.WithZone("").String()
			if len(clean) > maxSourceIPLength {
				return clean[:maxSourceIPLength]
			}
			return clean
		}
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
