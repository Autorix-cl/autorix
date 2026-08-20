package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/autorix/nexus/internal/core"
	"github.com/autorix/platform/health"
	"github.com/autorix/platform/metrics"
	"github.com/autorix/platform/paging"
)

var (
	errNoTuples     = errors.New("at least one tuple is required")
	errInvalidTuple = errors.New("each tuple requires namespace, object, relation and subject_id")
)

// Repository is the subset of tuple persistence needed by the admin API.
type Repository interface {
	ListTuples(ctx context.Context, namespace string, limit int, cursor string) ([]core.Tuple, bool, error)
	WriteTuples(ctx context.Context, tuples []core.Tuple) error
	DeleteTuples(ctx context.Context, tuples []core.Tuple) error

	WriteCaveat(ctx context.Context, caveat core.CaveatDefinition) error
	GetCaveat(ctx context.Context, name string) (*core.CaveatDefinition, error)
	ListCaveats(ctx context.Context) ([]core.CaveatDefinition, error)
	DeleteCaveat(ctx context.Context, name string) error

	WriteNamespace(ctx context.Context, schema core.NamespaceSchema) error
	GetNamespace(ctx context.Context, name string) (*core.NamespaceSchema, error)
	ListNamespaces(ctx context.Context) ([]core.NamespaceSchema, error)
	DeleteNamespace(ctx context.Context, name string) error
}

type Option func(*Server)

func WithCaveatEvaluator(ce core.CaveatEvaluator) Option {
	return func(s *Server) {
		s.caveatEvaluator = ce
	}
}

type Server struct {
	engine          core.GraphEngine
	repo            Repository
	health          *health.Handler
	caveatEvaluator core.CaveatEvaluator
}

func NewServer(engine core.GraphEngine, repo Repository, healthHandler *health.Handler, opts ...Option) *Server {
	s := &Server{engine: engine, repo: repo, health: healthHandler}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	s.health.Register(mux)
	// GET /health is a deprecated alias kept for existing callers; it now
	// delegates to the real liveness check instead of claiming health it
	// never measured. New callers should use /health/alive or /health/ready.
	mux.HandleFunc("GET /health", s.health.Alive)
	mux.Handle("GET /metrics", metrics.Handler())
	mux.Handle("/metrics", metrics.Handler())
	mux.HandleFunc("POST /check", s.handleCheck)
	mux.HandleFunc("GET /tuples", s.handleListTuples)
	mux.HandleFunc("POST /tuples", s.handleWriteTuples)
	mux.HandleFunc("DELETE /tuples", s.handleDeleteTuples)

	// Namespaces Admin Endpoints (P6-S3-T2)
	mux.HandleFunc("GET /admin/namespaces", s.handleListNamespaces)
	mux.HandleFunc("POST /admin/namespaces", s.handleWriteNamespace)
	mux.HandleFunc("GET /admin/namespaces/{name}", s.handleGetNamespace)
	mux.HandleFunc("DELETE /admin/namespaces/{name}", s.handleDeleteNamespace)

	// Caveats Admin Endpoints (P6-S3-T5)
	mux.HandleFunc("GET /admin/caveats", s.handleListCaveats)
	mux.HandleFunc("POST /admin/caveats", s.handleWriteCaveat)
	mux.HandleFunc("GET /admin/caveats/{name}", s.handleGetCaveat)
	mux.HandleFunc("DELETE /admin/caveats/{name}", s.handleDeleteCaveat)

	// Expansion & Reverse Index Endpoints (P6-S3-T3)
	mux.HandleFunc("POST /expand", s.handleExpand)
	mux.HandleFunc("GET /expand", s.handleExpand)
	mux.HandleFunc("POST /lookup/subjects", s.handleLookupSubjects)
	mux.HandleFunc("GET /lookup/subjects", s.handleLookupSubjects)
	mux.HandleFunc("POST /lookup/resources", s.handleLookupResources)
	mux.HandleFunc("GET /lookup/resources", s.handleLookupResources)

	return mux
}

// handleCheck evaluates whether a subject has a relation on a namespace:object
func (s *Server) handleCheck(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Namespace        string                 `json:"namespace"`
		Object           string                 `json:"object"`
		Relation         string                 `json:"relation"`
		SubjectNamespace string                 `json:"subject_namespace"`
		SubjectID        string                 `json:"subject_id"`
		SubjectRelation  string                 `json:"subject_relation"`
		RequestContext   map[string]interface{} `json:"request_context"`
		Explain          bool                   `json:"explain"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	if req.Namespace == "" || req.Object == "" || req.Relation == "" {
		writeError(w, http.StatusBadRequest, "namespace, object and relation are required")
		return
	}

	if req.SubjectNamespace == "" {
		req.SubjectNamespace = "user"
	}

	domainReq := core.CheckRequest{
		Namespace: req.Namespace,
		Object:    req.Object,
		Relation:  req.Relation,
		Subject: core.Tuple{
			Namespace: req.SubjectNamespace,
			Object:    req.SubjectID,
			Relation:  req.SubjectRelation,
		},
		RequestContext: req.RequestContext,
		Explain:        req.Explain,
	}

	res, err := s.engine.Check(r.Context(), domainReq)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Check failed: "+err.Error())
		return
	}

	respMap := map[string]interface{}{
		"allowed": res.Allowed,
		"reason":  res.Reason,
	}
	if req.Explain && res.Trace != nil {
		respMap["trace"] = res.Trace
	}

	writeJSON(w, http.StatusOK, respMap)
}

// handleListTuples lists relation tuples for the console table view
func (s *Server) handleListTuples(w http.ResponseWriter, r *http.Request) {
	namespace := r.URL.Query().Get("namespace")
	pageReq := paging.ParseRequest(r)

	tuples, hasMore, err := s.repo.ListTuples(r.Context(), namespace, pageReq.Limit, pageReq.Cursor)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var nextCursor string
	if hasMore && len(tuples) > 0 {
		nextCursor = encodeTupleCursor(tuples[len(tuples)-1])
	}
	paging.WriteEnvelope(w, toAPITuples(tuples), nextCursor, hasMore)
}

func encodeTupleCursor(t core.Tuple) string {
	payload, err := json.Marshal(struct {
		CommitTime       time.Time `json:"t"`
		Namespace        string    `json:"ns"`
		Object           string    `json:"o"`
		Relation         string    `json:"r"`
		SubjectNamespace string    `json:"sns"`
		SubjectObject    string    `json:"so"`
		SubjectRelation  string    `json:"sr"`
	}{t.CommitTime, t.Namespace, t.Object, t.Relation, t.SubjectNamespace, t.SubjectObject, t.SubjectRelation})
	if err != nil {
		return ""
	}
	return paging.EncodeCursor(string(payload))
}

// handleWriteTuples creates one or more relation tuples
func (s *Server) handleWriteTuples(w http.ResponseWriter, r *http.Request) {
	tuples, err := decodeAPITuples(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := s.repo.WriteTuples(r.Context(), tuples); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, toAPITuples(tuples))
}

// handleDeleteTuples removes one or more relation tuples
func (s *Server) handleDeleteTuples(w http.ResponseWriter, r *http.Request) {
	tuples, err := decodeAPITuples(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := s.repo.DeleteTuples(r.Context(), tuples); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// Namespaces Handlers (P6-S3-T2)

func (s *Server) handleListNamespaces(w http.ResponseWriter, r *http.Request) {
	list, err := s.repo.ListNamespaces(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if list == nil {
		list = []core.NamespaceSchema{}
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleWriteNamespace(w http.ResponseWriter, r *http.Request) {
	var schema core.NamespaceSchema
	if err := json.NewDecoder(r.Body).Decode(&schema); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON payload: "+err.Error())
		return
	}

	if strings.TrimSpace(schema.Name) == "" {
		writeError(w, http.StatusBadRequest, "namespace name is required")
		return
	}

	if err := s.repo.WriteNamespace(r.Context(), schema); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, schema)
}

func (s *Server) handleGetNamespace(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		writeError(w, http.StatusBadRequest, "namespace name is required")
		return
	}

	schema, err := s.repo.GetNamespace(r.Context(), name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if schema == nil {
		writeError(w, http.StatusNotFound, "namespace not found")
		return
	}

	writeJSON(w, http.StatusOK, schema)
}

func (s *Server) handleDeleteNamespace(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		writeError(w, http.StatusBadRequest, "namespace name is required")
		return
	}

	if err := s.repo.DeleteNamespace(r.Context(), name); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// Caveats Handlers (P6-S3-T5)

func (s *Server) handleListCaveats(w http.ResponseWriter, r *http.Request) {
	list, err := s.repo.ListCaveats(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if list == nil {
		list = []core.CaveatDefinition{}
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleWriteCaveat(w http.ResponseWriter, r *http.Request) {
	var caveat core.CaveatDefinition
	if err := json.NewDecoder(r.Body).Decode(&caveat); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON payload: "+err.Error())
		return
	}

	if strings.TrimSpace(caveat.Name) == "" {
		writeError(w, http.StatusBadRequest, "caveat name is required")
		return
	}

	// Validate CEL expression compilation
	if s.caveatEvaluator != nil {
		if err := s.caveatEvaluator.Validate(caveat.CELExpression); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid CEL expression: "+err.Error())
			return
		}
	}

	if err := s.repo.WriteCaveat(r.Context(), caveat); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, caveat)
}

func (s *Server) handleGetCaveat(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		writeError(w, http.StatusBadRequest, "caveat name is required")
		return
	}

	caveat, err := s.repo.GetCaveat(r.Context(), name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if caveat == nil {
		writeError(w, http.StatusNotFound, "caveat not found")
		return
	}

	writeJSON(w, http.StatusOK, caveat)
}

func (s *Server) handleDeleteCaveat(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		writeError(w, http.StatusBadRequest, "caveat name is required")
		return
	}

	if err := s.repo.DeleteCaveat(r.Context(), name); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// Expand & Lookup Handlers (P6-S3-T3)

func (s *Server) handleExpand(w http.ResponseWriter, r *http.Request) {
	var req core.ExpandRequest
	if r.Method == http.MethodPost {
		_ = json.NewDecoder(r.Body).Decode(&req)
	} else {
		req.Namespace = r.URL.Query().Get("namespace")
		req.Object = r.URL.Query().Get("object")
		req.Relation = r.URL.Query().Get("relation")
	}

	if req.Namespace == "" || req.Object == "" || req.Relation == "" {
		writeError(w, http.StatusBadRequest, "namespace, object and relation are required")
		return
	}

	tree, err := s.engine.Expand(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Expand failed: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"tree": tree})
}

func (s *Server) handleLookupSubjects(w http.ResponseWriter, r *http.Request) {
	var req core.LookupSubjectsRequest
	if r.Method == http.MethodPost {
		_ = json.NewDecoder(r.Body).Decode(&req)
	} else {
		req.Namespace = r.URL.Query().Get("namespace")
		req.Object = r.URL.Query().Get("object")
		req.Relation = r.URL.Query().Get("relation")
	}

	if req.Namespace == "" || req.Object == "" || req.Relation == "" {
		writeError(w, http.StatusBadRequest, "namespace, object and relation are required")
		return
	}

	subjects, err := s.engine.LookupSubjects(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "LookupSubjects failed: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"subjects": subjects})
}

func (s *Server) handleLookupResources(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Namespace        string `json:"namespace"`
		Relation         string `json:"relation"`
		SubjectNamespace string `json:"subject_namespace"`
		SubjectID        string `json:"subject_id"`
		SubjectRelation  string `json:"subject_relation"`
	}

	if r.Method == http.MethodPost {
		_ = json.NewDecoder(r.Body).Decode(&req)
	} else {
		req.Namespace = r.URL.Query().Get("namespace")
		req.Relation = r.URL.Query().Get("relation")
		req.SubjectNamespace = r.URL.Query().Get("subject_namespace")
		req.SubjectID = r.URL.Query().Get("subject_id")
		req.SubjectRelation = r.URL.Query().Get("subject_relation")
	}

	if req.Namespace == "" || req.Relation == "" || req.SubjectID == "" {
		writeError(w, http.StatusBadRequest, "namespace, relation and subject_id are required")
		return
	}

	if req.SubjectNamespace == "" {
		req.SubjectNamespace = "user"
	}

	resources, err := s.engine.LookupResources(r.Context(), core.LookupResourcesRequest{
		Namespace:        req.Namespace,
		Relation:         req.Relation,
		SubjectNamespace: req.SubjectNamespace,
		SubjectObject:    req.SubjectID,
		SubjectRelation:  req.SubjectRelation,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "LookupResources failed: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"resources": resources})
}

// apiTuple is the console-facing JSON shape for a Zanzibar relation tuple
type apiTuple struct {
	Namespace        string                 `json:"namespace"`
	Object           string                 `json:"object"`
	Relation         string                 `json:"relation"`
	SubjectNamespace string                 `json:"subject_namespace"`
	SubjectID        string                 `json:"subject_id"`
	SubjectRelation  string                 `json:"subject_relation,omitempty"`
	CaveatName       string                 `json:"caveat_name,omitempty"`
	CaveatContext    map[string]interface{} `json:"caveat_context,omitempty"`
}

func toAPITuples(tuples []core.Tuple) []apiTuple {
	out := make([]apiTuple, 0, len(tuples))
	for _, t := range tuples {
		out = append(out, apiTuple{
			Namespace:        t.Namespace,
			Object:           t.Object,
			Relation:         t.Relation,
			SubjectNamespace: t.SubjectNamespace,
			SubjectID:        t.SubjectObject,
			SubjectRelation:  t.SubjectRelation,
			CaveatName:       t.CaveatName,
			CaveatContext:    t.CaveatContext,
		})
	}
	return out
}

func decodeAPITuples(r *http.Request) ([]core.Tuple, error) {
	var req struct {
		Tuples []apiTuple `json:"tuples"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}
	if len(req.Tuples) == 0 {
		return nil, errNoTuples
	}

	out := make([]core.Tuple, 0, len(req.Tuples))
	for _, t := range req.Tuples {
		if t.Namespace == "" || t.Object == "" || t.Relation == "" || t.SubjectID == "" {
			return nil, errInvalidTuple
		}
		subjectNamespace := t.SubjectNamespace
		if subjectNamespace == "" {
			subjectNamespace = "user"
		}
		out = append(out, core.Tuple{
			Namespace:        t.Namespace,
			Object:           t.Object,
			Relation:         t.Relation,
			SubjectNamespace: subjectNamespace,
			SubjectObject:    t.SubjectID,
			SubjectRelation:  t.SubjectRelation,
			CaveatName:       t.CaveatName,
			CaveatContext:    t.CaveatContext,
		})
	}
	return out, nil
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
