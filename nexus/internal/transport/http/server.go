// Package http exposes a thin REST admin/console gateway over the Nexus
// hybrid ReBAC+ABAC engine. It talks directly to the domain layer
// (core.GraphEngine + the tuple repository) rather than through gRPC, since
// the gRPC WriteTuples/WriteCaveats/Expand handlers are not yet implemented.
// gRPC (internal/transport/grpc) remains the primary/canonical transport for
// service-to-service calls (e.g. from Aegis); this server exists so the
// admin console can manage relation tuples and run Check simulations over
// plain HTTP.
package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/autorix/nexus/internal/core"
)

var (
	errNoTuples     = errors.New("at least one tuple is required")
	errInvalidTuple = errors.New("each tuple requires namespace, object, relation and subject_id")
)

// Repository is the subset of tuple persistence needed by the admin API.
type Repository interface {
	ListTuples(ctx context.Context, namespace string, limit int) ([]core.Tuple, error)
	WriteTuples(ctx context.Context, tuples []core.Tuple) error
	DeleteTuples(ctx context.Context, tuples []core.Tuple) error
}

type Server struct {
	engine core.GraphEngine
	repo   Repository
}

func NewServer(engine core.GraphEngine, repo Repository) *Server {
	return &Server{engine: engine, repo: repo}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("POST /check", s.handleCheck)
	mux.HandleFunc("GET /tuples", s.handleListTuples)
	mux.HandleFunc("POST /tuples", s.handleWriteTuples)
	mux.HandleFunc("DELETE /tuples", s.handleDeleteTuples)

	return mux
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "healthy",
		"service": "nexus",
		"engine":  "Hybrid ReBAC (Zanzibar) + ABAC (CEL)",
	})
}

// handleCheck evaluates whether a subject has a relation on a namespace:object,
// mirroring the gRPC Check RPC but over plain JSON/HTTP.
func (s *Server) handleCheck(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Namespace        string                 `json:"namespace"`
		Object           string                 `json:"object"`
		Relation         string                 `json:"relation"`
		SubjectNamespace string                 `json:"subject_namespace"`
		SubjectID        string                 `json:"subject_id"`
		SubjectRelation  string                 `json:"subject_relation"`
		RequestContext   map[string]interface{} `json:"request_context"`
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
	}

	res, err := s.engine.Check(r.Context(), domainReq)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Check failed: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"allowed": res.Allowed,
		"reason":  res.Reason,
	})
}

// handleListTuples lists relation tuples for the console table view,
// optionally filtered by namespace.
func (s *Server) handleListTuples(w http.ResponseWriter, r *http.Request) {
	namespace := r.URL.Query().Get("namespace")

	tuples, err := s.repo.ListTuples(r.Context(), namespace, 200)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, toAPITuples(tuples))
}

// handleWriteTuples creates one or more relation tuples.
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

// handleDeleteTuples removes one or more relation tuples, matched by their
// full primary key.
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

// apiTuple is the console-facing JSON shape for a Zanzibar relation tuple,
// using the "namespace:object#relation@subject" convention from the docs.
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
