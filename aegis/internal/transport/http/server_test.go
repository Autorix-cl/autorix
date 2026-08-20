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

	"github.com/autorix/aegis/internal/core"
	"github.com/autorix/platform/health"
	"github.com/autorix/platform/paging"
)

type stubRuleStore struct {
	rules    []core.Rule
	versions []core.RuleVersion
}

func (stubRuleStore) List() []core.Rule                                { return nil }
func (stubRuleStore) Get(id string) (core.Rule, error)                 { return core.Rule{ID: id}, nil }
func (stubRuleStore) Create(r core.Rule) (core.Rule, error)            { return r, nil }
func (stubRuleStore) Update(id string, r core.Rule) (core.Rule, error) { return r, nil }
func (stubRuleStore) Delete(id string) error                           { return nil }
func (stubRuleStore) TestMatch(method, path string) (*core.Rule, error) {
	return &core.Rule{ID: "matched-rule"}, nil
}
func (stubRuleStore) Reorder(ids []string) error { return nil }
func (stubRuleStore) Rollback(version int) error {
	if version == 999 {
		return errors.New("version 999 not found")
	}
	return nil
}
func (s stubRuleStore) GetVersions() ([]core.RuleVersion, error) { return s.versions, nil }
func (stubRuleStore) Import(rules []core.Rule) error            { return nil }
func (s stubRuleStore) Export() []core.Rule                     { return s.rules }

type stubDryRunner struct{}

func (stubDryRunner) DryRun(r *http.Request) (*core.PipelineTrace, error) {
	allowed := true
	return &core.PipelineTrace{
		MatchedRuleID: "traced-rule",
		Steps: []core.PipelineTraceStep{
			{Stage: "match", Status: "success", Details: "Matched rule"},
			{Stage: "authenticator", Handler: "anonymous", Status: "success", Session: &core.Session{Subject: "anon"}},
			{Stage: "authorizer", Handler: "allow", Status: "success", Allowed: &allowed},
			{Stage: "mutator", Handler: "header", Status: "success"},
			{Stage: "upstream", Status: "success", TargetURL: "http://backend.internal/test"},
		},
		FinalVerdict: "allow",
	}, nil
}

func (stubDryRunner) Catalogue() core.HandlerCatalogue {
	return core.HandlerCatalogue{
		Authenticators: []core.HandlerInfo{{Name: "jwt", Description: "JWT Auth"}},
		Authorizers:    []core.HandlerInfo{{Name: "allow", Description: "Allow Authz"}},
		Mutators:       []core.HandlerInfo{{Name: "header", Description: "Header Mutator"}},
	}
}

// ListPage is a minimal in-memory keyset over s.rules, mirroring
// rule.Store.ListPage's contract (cursor wraps the last rule ID via
// paging.EncodeCursor) closely enough to exercise the HTTP handler without
// a real rule.Store.
func (s stubRuleStore) ListPage(limit int, cursor string) ([]core.Rule, bool, error) {
	start := 0
	if cursor != "" {
		id, err := paging.DecodeCursor(cursor)
		if err != nil {
			return nil, false, err
		}
		idx := -1
		for i, r := range s.rules {
			if r.ID == id {
				idx = i
				break
			}
		}
		if idx == -1 {
			return nil, false, errors.New("unknown cursor")
		}
		start = idx + 1
	}
	if start > len(s.rules) {
		start = len(s.rules)
	}
	end := start + limit
	hasMore := end < len(s.rules)
	if end > len(s.rules) {
		end = len(s.rules)
	}
	out := make([]core.Rule, end-start)
	copy(out, s.rules[start:end])
	return out, hasMore, nil
}

func newTestServer(checker *health.Checker) *Server {
	handler := health.NewHandler(checker, func() health.Info {
		return health.Info{Engine: "aegis", StartedAt: time.Now()}
	})
	return NewServer(stubRuleStore{}, handler, stubDryRunner{})
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
	checker.Register("boom", func(ctx context.Context) error {
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

// newContractMux builds a real aegis Server with its readiness check rigged
// to fail iff checkErr is non-nil, for the shared platform/health.Contract
// suite.
func newContractMux(checkErr error) http.Handler {
	checker := health.NewChecker()
	checker.Register("boom", func(ctx context.Context) error {
		return checkErr
	})
	return newTestServer(checker).Routes()
}

func TestHealthContract(t *testing.T) {
	health.Contract(t, newContractMux)
}

func TestHandleListRules_CursorPagination(t *testing.T) {
	const total = 5
	rules := make([]core.Rule, total)
	for i := range rules {
		rules[i] = core.Rule{ID: fmt.Sprintf("rule-%d", i)}
	}

	handler := health.NewHandler(health.NewChecker(), func() health.Info {
		return health.Info{Engine: "aegis", StartedAt: time.Now()}
	})
	srv := NewServer(stubRuleStore{rules: rules}, handler, stubDryRunner{})
	router := srv.Routes()

	type envelope struct {
		Data       []core.Rule `json:"data"`
		NextCursor string      `json:"next_cursor"`
		HasMore    bool        `json:"has_more"`
	}

	var all []core.Rule
	cursor := ""
	for i := 0; i < 10; i++ {
		url := "/rules?limit=2"
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
			t.Fatalf("page returned %d rules, want at most 2", len(got.Data))
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
		t.Fatalf("expected %d total rules across pages, got %d", total, len(all))
	}
	seen := map[string]bool{}
	for i, r := range all {
		if seen[r.ID] {
			t.Fatalf("duplicate rule %q across pages", r.ID)
		}
		seen[r.ID] = true
		want := fmt.Sprintf("rule-%d", i)
		if r.ID != want {
			t.Fatalf("page order broken or rule skipped: position %d = %q, want %q", i, r.ID, want)
		}
	}
}

func TestHandleReorder(t *testing.T) {
	srv := newTestServer(health.NewChecker())
	router := srv.Routes()

	body := `{"rule_ids": ["rule-2", "rule-1"]}`
	req := httptest.NewRequest(http.MethodPut, "/admin/rules/reorder", strings.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, rec.Body.String())
	}
}

func TestHandleImportExport(t *testing.T) {
	srv := newTestServer(health.NewChecker())
	router := srv.Routes()

	// 1. Test Import
	importYAML := `- id: "imported-1"
  match:
    url: "/api/test"
    methods: ["GET"]
  upstream:
    url: "http://backend.internal"
`
	req := httptest.NewRequest(http.MethodPost, "/admin/rules/import", strings.NewReader(importYAML))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("import: expected 200, got %d (body: %s)", rec.Code, rec.Body.String())
	}

	// 2. Test Export
	reqExport := httptest.NewRequest(http.MethodGet, "/admin/rules/export", nil)
	recExport := httptest.NewRecorder()
	router.ServeHTTP(recExport, reqExport)

	if recExport.Code != http.StatusOK {
		t.Fatalf("export: expected 200, got %d", recExport.Code)
	}
	if !strings.Contains(recExport.Header().Get("Content-Type"), "yaml") {
		t.Errorf("expected Content-Type yaml, got %s", recExport.Header().Get("Content-Type"))
	}
}

func TestHandleCatalogue(t *testing.T) {
	srv := newTestServer(health.NewChecker())
	router := srv.Routes()

	req := httptest.NewRequest(http.MethodGet, "/admin/handlers", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var cat core.HandlerCatalogue
	if err := json.NewDecoder(rec.Body).Decode(&cat); err != nil {
		t.Fatalf("decode catalogue: %v", err)
	}
	if len(cat.Authenticators) == 0 || len(cat.Authorizers) == 0 || len(cat.Mutators) == 0 {
		t.Errorf("expected non-empty catalogue handlers: %+v", cat)
	}
}

func TestHandleTestMatch_Trace(t *testing.T) {
	srv := newTestServer(health.NewChecker())
	router := srv.Routes()

	body := `{"method": "GET", "path": "/api/users/123", "headers": {"Authorization": "Bearer tok"}}`
	req := httptest.NewRequest(http.MethodPost, "/admin/rules/test-match", strings.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, rec.Body.String())
	}

	var resp struct {
		Matched bool                `json:"matched"`
		Rule    *core.Rule          `json:"rule"`
		Trace   *core.PipelineTrace `json:"trace"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode test match response: %v", err)
	}
	if !resp.Matched {
		t.Errorf("expected matched true")
	}
	if resp.Trace == nil || len(resp.Trace.Steps) == 0 {
		t.Errorf("expected pipeline trace steps in dry-run response")
	}
}

func TestHandleRollbackAndVersions(t *testing.T) {
	store := stubRuleStore{
		versions: []core.RuleVersion{
			{Version: 1, Description: "Initial version", Rules: []core.Rule{{ID: "v1-rule"}}},
		},
	}
	handler := health.NewHandler(health.NewChecker(), func() health.Info {
		return health.Info{Engine: "aegis", StartedAt: time.Now()}
	})
	srv := NewServer(store, handler, stubDryRunner{})
	router := srv.Routes()

	// 1. Get Versions
	reqV := httptest.NewRequest(http.MethodGet, "/admin/rules/versions", nil)
	recV := httptest.NewRecorder()
	router.ServeHTTP(recV, reqV)

	if recV.Code != http.StatusOK {
		t.Fatalf("get versions: expected 200, got %d", recV.Code)
	}

	// 2. Rollback valid version
	reqRB := httptest.NewRequest(http.MethodPost, "/admin/rules/rollback/1", nil)
	recRB := httptest.NewRecorder()
	router.ServeHTTP(recRB, reqRB)

	if recRB.Code != http.StatusOK {
		t.Fatalf("rollback 1: expected 200, got %d (body: %s)", recRB.Code, recRB.Body.String())
	}

	// 3. Rollback invalid version
	reqRBInvalid := httptest.NewRequest(http.MethodPost, "/admin/rules/rollback/999", nil)
	recRBInvalid := httptest.NewRecorder()
	router.ServeHTTP(recRBInvalid, reqRBInvalid)

	if recRBInvalid.Code != http.StatusNotFound {
		t.Fatalf("rollback 999: expected 404, got %d", recRBInvalid.Code)
	}
}

func TestMetricsEndpoint(t *testing.T) {
	handler := health.NewHandler(health.NewChecker(), func() health.Info {
		return health.Info{Engine: "aegis", StartedAt: time.Now()}
	})
	srv := NewServer(stubRuleStore{}, handler, stubDryRunner{})
	router := srv.Routes()

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /metrics: expected 200, got %d", rec.Code)
	}
}

