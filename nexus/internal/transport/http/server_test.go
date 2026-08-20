package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/autorix/nexus/internal/core"
	"github.com/autorix/platform/health"
	"github.com/autorix/platform/paging"
)

type stubGraphEngine struct{}

func (stubGraphEngine) Check(ctx context.Context, req core.CheckRequest) (core.CheckResult, error) {
	return core.CheckResult{}, nil
}

func (stubGraphEngine) Expand(ctx context.Context, req core.ExpandRequest) (*core.ExpandTree, error) {
	return nil, nil
}

type stubRepository struct {
	tuples []core.Tuple
}

// ListTuples is a minimal in-memory keyset over s.tuples (assumed already
// in the desired stable order), mirroring the postgres repository's cursor
// contract (cursor wraps the last tuple's object via encodeTupleCursor)
// closely enough to exercise the HTTP handler without a real database.
func (s stubRepository) ListTuples(ctx context.Context, namespace string, limit int, cursor string) ([]core.Tuple, bool, error) {
	start := 0
	if cursor != "" {
		decoded, err := paging.DecodeCursor(cursor)
		if err != nil {
			return nil, false, err
		}
		var payload struct {
			Object string `json:"o"`
		}
		if err := json.Unmarshal([]byte(decoded), &payload); err != nil {
			return nil, false, err
		}
		idx := -1
		for i, t := range s.tuples {
			if t.Object == payload.Object {
				idx = i
				break
			}
		}
		if idx == -1 {
			return nil, false, errors.New("unknown cursor")
		}
		start = idx + 1
	}
	if start > len(s.tuples) {
		start = len(s.tuples)
	}
	end := start + limit
	hasMore := end < len(s.tuples)
	if end > len(s.tuples) {
		end = len(s.tuples)
	}
	out := make([]core.Tuple, end-start)
	copy(out, s.tuples[start:end])
	return out, hasMore, nil
}
func (stubRepository) WriteTuples(ctx context.Context, tuples []core.Tuple) error  { return nil }
func (stubRepository) DeleteTuples(ctx context.Context, tuples []core.Tuple) error { return nil }
func (stubRepository) WriteCaveat(ctx context.Context, caveat core.CaveatDefinition) error { return nil }
func (stubRepository) GetCaveat(ctx context.Context, name string) (*core.CaveatDefinition, error) { return nil, nil }
func (stubRepository) ListCaveats(ctx context.Context) ([]core.CaveatDefinition, error) { return nil, nil }
func (stubRepository) DeleteCaveat(ctx context.Context, name string) error { return nil }
func (stubRepository) WriteNamespace(ctx context.Context, schema core.NamespaceSchema) error { return nil }
func (stubRepository) GetNamespace(ctx context.Context, name string) (*core.NamespaceSchema, error) { return nil, nil }
func (stubRepository) ListNamespaces(ctx context.Context) ([]core.NamespaceSchema, error) { return nil, nil }
func (stubRepository) DeleteNamespace(ctx context.Context, name string) error { return nil }

func newTestServer(checker *health.Checker) *Server {
	handler := health.NewHandler(checker, func() health.Info {
		return health.Info{Engine: "nexus", StartedAt: time.Now()}
	})
	return NewServer(stubGraphEngine{}, stubRepository{}, handler)
}

func TestHealthAlive(t *testing.T) {
	srv := newTestServer(health.NewChecker())

	req := httptest.NewRequest(http.MethodGet, "/health/alive", nil)
	rec := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["status"] != "alive" {
		t.Errorf("expected status alive, got %q", body["status"])
	}
}

func TestHealthReady_FailingCheck(t *testing.T) {
	checker := health.NewChecker()
	checker.Register("postgres", func(ctx context.Context) error {
		return errors.New("dependency unreachable")
	})
	srv := newTestServer(checker)

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	rec := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
}

// newContractMux builds a real nexus Server with its readiness check rigged
// to fail iff checkErr is non-nil, for the shared platform/health.Contract
// suite.
func newContractMux(checkErr error) http.Handler {
	checker := health.NewChecker()
	checker.Register("postgres", func(ctx context.Context) error {
		return checkErr
	})
	return newTestServer(checker).Routes()
}

func TestHealthContract(t *testing.T) {
	health.Contract(t, newContractMux)
}

// TestHandleListTuples_CursorPagination proves GET /tuples paginates via
// platform/paging's envelope: a page smaller than the full set returns
// exactly N tuples plus a next_cursor and has_more=true, and following that
// cursor returns the rest with no duplicate and no skipped tuple.
func TestHandleListTuples_CursorPagination(t *testing.T) {
	const total = 5
	tuples := make([]core.Tuple, total)
	for i := range tuples {
		tuples[i] = core.Tuple{
			Namespace: "document", Object: fmt.Sprintf("doc-%d", i), Relation: "viewer",
			SubjectNamespace: "user", SubjectObject: "alice",
		}
	}

	handler := health.NewHandler(health.NewChecker(), func() health.Info {
		return health.Info{Engine: "nexus", StartedAt: time.Now()}
	})
	srv := NewServer(stubGraphEngine{}, stubRepository{tuples: tuples}, handler)
	router := srv.Routes()

	type envelope struct {
		Data       []apiTuple `json:"data"`
		NextCursor string     `json:"next_cursor"`
		HasMore    bool       `json:"has_more"`
	}

	var all []apiTuple
	cursor := ""
	for i := 0; i < 10; i++ { // bounded loop guard against an infinite pagination bug
		url := "/tuples?limit=2"
		if cursor != "" {
			url += "&cursor=" + cursor
		}
		req := httptest.NewRequest(http.MethodGet, url, nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s: expected 200, got %d (body: %s)", url, rec.Code, rec.Body.String())
		}
		var got envelope
		if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
			t.Fatalf("decode envelope: %v", err)
		}
		if len(got.Data) > 2 {
			t.Fatalf("page returned %d tuples, want at most 2", len(got.Data))
		}
		all = append(all, got.Data...)
		if !got.HasMore {
			break
		}
		if got.NextCursor == "" {
			t.Fatalf("has_more=true but next_cursor is empty")
		}
		cursor = got.NextCursor
	}

	if len(all) != total {
		t.Fatalf("expected %d total tuples across pages, got %d", total, len(all))
	}
	seen := map[string]bool{}
	for i, tup := range all {
		if seen[tup.Object] {
			t.Fatalf("duplicate tuple %q across pages", tup.Object)
		}
		seen[tup.Object] = true
		want := fmt.Sprintf("doc-%d", i)
		if tup.Object != want {
			t.Fatalf("page order broken or tuple skipped: position %d = %q, want %q", i, tup.Object, want)
		}
	}
}

func (s stubGraphEngine) LookupSubjects(ctx context.Context, req core.LookupSubjectsRequest) ([]core.Tuple, error) {
	return []core.Tuple{
		{Namespace: req.Namespace, Object: req.Object, Relation: req.Relation, SubjectNamespace: "user", SubjectObject: "alice"},
	}, nil
}

func (s stubGraphEngine) LookupResources(ctx context.Context, req core.LookupResourcesRequest) ([]string, error) {
	return []string{"doc-1", "doc-2"}, nil
}

type fullStubRepository struct {
	stubRepository
	caveats    map[string]core.CaveatDefinition
	namespaces map[string]core.NamespaceSchema
}

func newFullStubRepository() *fullStubRepository {
	return &fullStubRepository{
		caveats:    make(map[string]core.CaveatDefinition),
		namespaces: make(map[string]core.NamespaceSchema),
	}
}

func (f *fullStubRepository) WriteCaveat(ctx context.Context, caveat core.CaveatDefinition) error {
	f.caveats[caveat.Name] = caveat
	return nil
}
func (f *fullStubRepository) GetCaveat(ctx context.Context, name string) (*core.CaveatDefinition, error) {
	c, ok := f.caveats[name]
	if !ok {
		return nil, nil
	}
	return &c, nil
}
func (f *fullStubRepository) ListCaveats(ctx context.Context) ([]core.CaveatDefinition, error) {
	var list []core.CaveatDefinition
	for _, c := range f.caveats {
		list = append(list, c)
	}
	return list, nil
}
func (f *fullStubRepository) DeleteCaveat(ctx context.Context, name string) error {
	delete(f.caveats, name)
	return nil
}

func (f *fullStubRepository) WriteNamespace(ctx context.Context, schema core.NamespaceSchema) error {
	f.namespaces[schema.Name] = schema
	return nil
}
func (f *fullStubRepository) GetNamespace(ctx context.Context, name string) (*core.NamespaceSchema, error) {
	s, ok := f.namespaces[name]
	if !ok {
		return nil, nil
	}
	return &s, nil
}
func (f *fullStubRepository) ListNamespaces(ctx context.Context) ([]core.NamespaceSchema, error) {
	var list []core.NamespaceSchema
	for _, s := range f.namespaces {
		list = append(list, s)
	}
	return list, nil
}
func (f *fullStubRepository) DeleteNamespace(ctx context.Context, name string) error {
	delete(f.namespaces, name)
	return nil
}

type stubCaveatEvaluator struct{}

func (stubCaveatEvaluator) Compile(expression string) error { return nil }
func (stubCaveatEvaluator) Validate(expression string) error {
	if expression == "invalid == syntax ==" || expression == "" {
		return errors.New("invalid CEL expression")
	}
	return nil
}
func (stubCaveatEvaluator) Evaluate(ctx map[string]interface{}, caveatCtx map[string]interface{}) (bool, error) {
	return true, nil
}
func (stubCaveatEvaluator) EvaluateByName(ctx context.Context, name string, reqCtx map[string]interface{}, caveatCtx map[string]interface{}) (bool, error) {
	return true, nil
}

type explainGraphEngine struct{}

func (explainGraphEngine) Check(ctx context.Context, req core.CheckRequest) (core.CheckResult, error) {
	var trace *core.DecisionNode
	if req.Explain {
		trace = &core.DecisionNode{
			Namespace: req.Namespace,
			Object:    req.Object,
			Relation:  req.Relation,
			Subject:   req.Subject,
			Allowed:   true,
			Reason:    "direct match",
		}
	}
	return core.CheckResult{
		Allowed: true,
		Reason:  "direct match",
		Trace:   trace,
	}, nil
}

func (explainGraphEngine) Expand(ctx context.Context, req core.ExpandRequest) (*core.ExpandTree, error) {
	return &core.ExpandTree{
		Type: "union",
		Children: []*core.ExpandTree{
			{Type: "leaf", Tuple: &core.Tuple{Namespace: req.Namespace, Object: req.Object, Relation: req.Relation, SubjectNamespace: "user", SubjectObject: "alice"}},
		},
	}, nil
}
func (explainGraphEngine) LookupSubjects(ctx context.Context, req core.LookupSubjectsRequest) ([]core.Tuple, error) {
	return []core.Tuple{
		{Namespace: req.Namespace, Object: req.Object, Relation: req.Relation, SubjectNamespace: "user", SubjectObject: "alice"},
	}, nil
}
func (explainGraphEngine) LookupResources(ctx context.Context, req core.LookupResourcesRequest) ([]string, error) {
	return []string{"doc-1", "doc-2"}, nil
}

func TestHandleCheckWithExplain(t *testing.T) {
	handler := health.NewHandler(health.NewChecker(), func() health.Info { return health.Info{Engine: "nexus"} })
	srv := NewServer(explainGraphEngine{}, newFullStubRepository(), handler, WithCaveatEvaluator(stubCaveatEvaluator{}))
	router := srv.Routes()

	body := `{"namespace":"document","object":"doc-1","relation":"viewer","subject_id":"alice","explain":true}`
	req := httptest.NewRequest(http.MethodPost, "/check", strings.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, rec.Body.String())
	}

	var res struct {
		Allowed bool               `json:"allowed"`
		Reason  string             `json:"reason"`
		Trace   *core.DecisionNode `json:"trace"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&res); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !res.Allowed || res.Trace == nil {
		t.Fatalf("expected allowed=true and non-nil trace, got %+v", res)
	}
}

func TestAdminNamespacesEndpoints(t *testing.T) {
	repo := newFullStubRepository()
	handler := health.NewHandler(health.NewChecker(), func() health.Info { return health.Info{Engine: "nexus"} })
	srv := NewServer(explainGraphEngine{}, repo, handler, WithCaveatEvaluator(stubCaveatEvaluator{}))
	router := srv.Routes()

	// 1. POST /admin/namespaces
	nsBody := `{
		"name": "document",
		"relations": {
			"viewer": {
				"rewrite": {"type": "this"}
			}
		}
	}`
	req := httptest.NewRequest(http.MethodPost, "/admin/namespaces", strings.NewReader(nsBody))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /admin/namespaces: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	// 2. GET /admin/namespaces
	req = httptest.NewRequest(http.MethodGet, "/admin/namespaces", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /admin/namespaces: expected 200, got %d", rec.Code)
	}
	var list []core.NamespaceSchema
	if err := json.NewDecoder(rec.Body).Decode(&list); err != nil || len(list) != 1 {
		t.Fatalf("expected 1 namespace, got %d (err: %v)", len(list), err)
	}

	// 3. GET /admin/namespaces/document
	req = httptest.NewRequest(http.MethodGet, "/admin/namespaces/document", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /admin/namespaces/document: expected 200, got %d", rec.Code)
	}

	// 4. DELETE /admin/namespaces/document
	req = httptest.NewRequest(http.MethodDelete, "/admin/namespaces/document", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("DELETE /admin/namespaces/document: expected 200, got %d", rec.Code)
	}

	// 5. GET /admin/namespaces/document after delete -> 404
	req = httptest.NewRequest(http.MethodGet, "/admin/namespaces/document", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("GET after delete: expected 404, got %d", rec.Code)
	}
}

func TestAdminCaveatsEndpoints(t *testing.T) {
	repo := newFullStubRepository()
	handler := health.NewHandler(health.NewChecker(), func() health.Info { return health.Info{Engine: "nexus"} })
	srv := NewServer(explainGraphEngine{}, repo, handler, WithCaveatEvaluator(stubCaveatEvaluator{}))
	router := srv.Routes()

	// 1. POST /admin/caveats with invalid CEL -> 400
	badBody := `{"name": "bad", "cel_expression": "invalid == syntax =="}`
	req := httptest.NewRequest(http.MethodPost, "/admin/caveats", strings.NewReader(badBody))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("POST invalid caveat: expected 400, got %d", rec.Code)
	}

	// 2. POST /admin/caveats valid -> 201
	validBody := `{"name": "is_admin_ip", "cel_expression": "ctx.ip == \"10.0.0.1\""}`
	req = httptest.NewRequest(http.MethodPost, "/admin/caveats", strings.NewReader(validBody))
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST valid caveat: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	// 3. GET /admin/caveats
	req = httptest.NewRequest(http.MethodGet, "/admin/caveats", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /admin/caveats: expected 200, got %d", rec.Code)
	}

	// 4. GET /admin/caveats/is_admin_ip
	req = httptest.NewRequest(http.MethodGet, "/admin/caveats/is_admin_ip", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET caveat by name: expected 200, got %d", rec.Code)
	}

	// 5. DELETE /admin/caveats/is_admin_ip
	req = httptest.NewRequest(http.MethodDelete, "/admin/caveats/is_admin_ip", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("DELETE caveat: expected 200, got %d", rec.Code)
	}
}

func TestExpandAndLookupEndpoints(t *testing.T) {
	handler := health.NewHandler(health.NewChecker(), func() health.Info { return health.Info{Engine: "nexus"} })
	srv := NewServer(explainGraphEngine{}, newFullStubRepository(), handler, WithCaveatEvaluator(stubCaveatEvaluator{}))
	router := srv.Routes()

	// 1. POST /expand
	req := httptest.NewRequest(http.MethodPost, "/expand", strings.NewReader(`{"namespace":"document","object":"doc-1","relation":"viewer"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("POST /expand: expected 200, got %d", rec.Code)
	}

	// 2. POST /lookup/subjects
	req = httptest.NewRequest(http.MethodPost, "/lookup/subjects", strings.NewReader(`{"namespace":"document","object":"doc-1","relation":"viewer"}`))
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("POST /lookup/subjects: expected 200, got %d", rec.Code)
	}

	// 3. POST /lookup/resources
	req = httptest.NewRequest(http.MethodPost, "/lookup/resources", strings.NewReader(`{"namespace":"document","relation":"viewer","subject_namespace":"user","subject_id":"alice"}`))
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("POST /lookup/resources: expected 200, got %d", rec.Code)
	}
}

func TestMetricsEndpoint(t *testing.T) {
	handler := health.NewHandler(health.NewChecker(), func() health.Info { return health.Info{Engine: "nexus"} })
	srv := NewServer(stubGraphEngine{}, stubRepository{}, handler)
	router := srv.Routes()

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /metrics: expected 200, got %d", rec.Code)
	}
}

