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

	"github.com/autorix/platform/health"
	"github.com/autorix/platform/paging"
	"github.com/autorix/themis/internal/core"
)

type stubPolicyEngine struct {
	policies []core.Policy
}

func (stubPolicyEngine) CreatePolicy(ctx context.Context, policy *core.Policy) error { return nil }
func (stubPolicyEngine) GetPolicy(ctx context.Context, tenantID, policyID string) (*core.Policy, error) {
	return &core.Policy{}, nil
}
func (stubPolicyEngine) ListPolicies(ctx context.Context, filter core.ListFilter) ([]core.Policy, error) {
	return nil, nil
}

// ListPoliciesPage is a minimal in-memory keyset over s.policies (assumed
// already in the desired stable order), mirroring the postgres
// repository's cursor contract (cursor wraps the last policy's ID) closely
// enough to exercise the HTTP handler without a real database.
func (s stubPolicyEngine) ListPoliciesPage(ctx context.Context, filter core.ListFilter, limit int, cursor string) ([]core.Policy, bool, error) {
	start := 0
	if cursor != "" {
		decoded, err := paging.DecodeCursor(cursor)
		if err != nil {
			return nil, false, err
		}
		// The handler encodes "priority|createdAt|id" (see
		// handleListPolicies); only the id is needed to locate position.
		parts := strings.Split(decoded, "|")
		lastID := parts[len(parts)-1]
		idx := -1
		for i, p := range s.policies {
			if p.ID == lastID {
				idx = i
				break
			}
		}
		if idx == -1 {
			return nil, false, errors.New("unknown cursor")
		}
		start = idx + 1
	}
	if start > len(s.policies) {
		start = len(s.policies)
	}
	end := start + limit
	hasMore := end < len(s.policies)
	if end > len(s.policies) {
		end = len(s.policies)
	}
	out := make([]core.Policy, end-start)
	copy(out, s.policies[start:end])
	return out, hasMore, nil
}
func (stubPolicyEngine) UpdatePolicy(ctx context.Context, policy *core.Policy) error { return nil }
func (stubPolicyEngine) DeletePolicy(ctx context.Context, tenantID, policyID string) error {
	return nil
}
func (stubPolicyEngine) Evaluate(ctx context.Context, req core.EvaluateRequest) (*core.EvaluateResponse, error) {
	return &core.EvaluateResponse{}, nil
}
func (stubPolicyEngine) ListPolicyVersions(ctx context.Context, tenantID, policyID string) ([]core.PolicyVersion, error) {
	return []core.PolicyVersion{
		{ID: "v1-id", PolicyID: policyID, TenantID: tenantID, Version: 1, Expression: "true"},
	}, nil
}
func (stubPolicyEngine) RollbackPolicy(ctx context.Context, tenantID, policyID string, version int32) (*core.Policy, error) {
	return &core.Policy{ID: policyID, TenantID: tenantID, Expression: "true"}, nil
}
func (stubPolicyEngine) ValidateExpression(ctx context.Context, expression string) (*core.ValidationResult, error) {
	if expression == "invalid" {
		return &core.ValidationResult{
			Valid:  false,
			Errors: []core.ValidationError{{Line: 1, Column: 1, Message: "syntax error"}},
		}, nil
	}
	return &core.ValidationResult{
		Valid:     true,
		Variables: []string{"request"},
	}, nil
}
func (stubPolicyEngine) DryRun(ctx context.Context, req core.DryRunRequest) (*core.DryRunResult, error) {
	return &core.DryRunResult{Passed: true}, nil
}
func (stubPolicyEngine) CreateFixture(ctx context.Context, fixture *core.PolicyFixture) error {
	fixture.ID = "fix-1"
	return nil
}
func (stubPolicyEngine) ListFixtures(ctx context.Context, tenantID, policyID string) ([]core.PolicyFixture, error) {
	return []core.PolicyFixture{
		{ID: "fix-1", PolicyID: policyID, TenantID: tenantID, Name: "test1", ExpectedResult: true},
	}, nil
}
func (stubPolicyEngine) DeleteFixture(ctx context.Context, tenantID, policyID, fixtureID string) error {
	return nil
}
func (stubPolicyEngine) RunTestSuite(ctx context.Context, tenantID, policyID string) (*core.TestSuiteResult, error) {
	return &core.TestSuiteResult{
		PolicyID:   policyID,
		AllPassed:  true,
		TotalTests: 1,
		Results: []core.FixtureRunResult{
			{FixtureID: "fix-1", ExpectedResult: true, ActualResult: true, Passed: true},
		},
	}, nil
}

func newTestServer(checker *health.Checker) *Server {
	handler := health.NewHandler(checker, func() health.Info {
		return health.Info{Engine: "themis", StartedAt: time.Now()}
	})
	return NewServer(stubPolicyEngine{}, handler)
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

// newContractMux builds a real themis Server with its readiness check
// rigged to fail iff checkErr is non-nil, for the shared
// platform/health.Contract suite.
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

// TestHandleListPolicies_CursorPagination proves GET /policies paginates
// via platform/paging's envelope: a page smaller than the full set returns
// exactly N policies plus a next_cursor and has_more=true, and following
// that cursor returns the rest with no duplicate and no skipped policy.
func TestHandleListPolicies_CursorPagination(t *testing.T) {
	const total = 5
	policies := make([]core.Policy, total)
	for i := range policies {
		policies[i] = core.Policy{ID: fmt.Sprintf("policy-%d", i), TenantID: "default", Name: fmt.Sprintf("p%d", i)}
	}

	handler := health.NewHandler(health.NewChecker(), func() health.Info {
		return health.Info{Engine: "themis", StartedAt: time.Now()}
	})
	srv := NewServer(stubPolicyEngine{policies: policies}, handler)
	router := srv.Routes()

	type envelope struct {
		Data       []core.Policy `json:"data"`
		NextCursor string        `json:"next_cursor"`
		HasMore    bool          `json:"has_more"`
	}

	var all []core.Policy
	cursor := ""
	for i := 0; i < 10; i++ { // bounded loop guard against an infinite pagination bug
		url := "/policies?limit=2"
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
			t.Fatalf("page returned %d policies, want at most 2", len(got.Data))
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
		t.Fatalf("expected %d total policies across pages, got %d", total, len(all))
	}
	seen := map[string]bool{}
	for i, p := range all {
		if seen[p.ID] {
			t.Fatalf("duplicate policy %q across pages", p.ID)
		}
		seen[p.ID] = true
		want := fmt.Sprintf("policy-%d", i)
		if p.ID != want {
			t.Fatalf("page order broken or policy skipped: position %d = %q, want %q", i, p.ID, want)
		}
	}
}

func TestAdminPolicyVersionsAndRollback(t *testing.T) {
	srv := newTestServer(health.NewChecker())
	router := srv.Routes()

	// GET /admin/policies/{id}/versions
	req := httptest.NewRequest(http.MethodGet, "/admin/policies/pol-123/versions", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET versions: expected 200, got %d", rec.Code)
	}

	var versions []core.PolicyVersion
	if err := json.NewDecoder(rec.Body).Decode(&versions); err != nil {
		t.Fatalf("decode versions: %v", err)
	}
	if len(versions) != 1 || versions[0].Version != 1 {
		t.Fatalf("unexpected versions response: %+v", versions)
	}

	// POST /admin/policies/{id}/rollback/{version}
	req = httptest.NewRequest(http.MethodPost, "/admin/policies/pol-123/rollback/1", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("POST rollback: expected 200, got %d", rec.Code)
	}
	var rolledBack core.Policy
	if err := json.NewDecoder(rec.Body).Decode(&rolledBack); err != nil {
		t.Fatalf("decode rolledBack: %v", err)
	}
	if rolledBack.ID != "pol-123" {
		t.Fatalf("unexpected rollback policy ID: %s", rolledBack.ID)
	}
}

func TestAdminCELValidation(t *testing.T) {
	srv := newTestServer(health.NewChecker())
	router := srv.Routes()

	// Valid expression
	body := strings.NewReader(`{"expression": "request.role == 'admin'"}`)
	req := httptest.NewRequest(http.MethodPost, "/admin/policies/validate", body)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("POST validate: expected 200, got %d", rec.Code)
	}
	var res core.ValidationResult
	if err := json.NewDecoder(rec.Body).Decode(&res); err != nil {
		t.Fatalf("decode validate response: %v", err)
	}
	if !res.Valid {
		t.Fatalf("expected valid=true")
	}

	// Invalid expression
	body = strings.NewReader(`{"expression": "invalid"}`)
	req = httptest.NewRequest(http.MethodPost, "/admin/policies/validate", body)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("POST validate: expected 200, got %d", rec.Code)
	}
	if err := json.NewDecoder(rec.Body).Decode(&res); err != nil {
		t.Fatalf("decode validate response: %v", err)
	}
	if res.Valid {
		t.Fatalf("expected valid=false for invalid expression")
	}
}

func TestAdminDryRun(t *testing.T) {
	srv := newTestServer(health.NewChecker())
	router := srv.Routes()

	body := strings.NewReader(`{"expression": "request.role == 'admin'", "payload": {"request": {"role": "admin"}}}`)
	req := httptest.NewRequest(http.MethodPost, "/admin/policies/dry-run", body)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("POST dry-run: expected 200, got %d", rec.Code)
	}
	var res core.DryRunResult
	if err := json.NewDecoder(rec.Body).Decode(&res); err != nil {
		t.Fatalf("decode dry-run response: %v", err)
	}
	if !res.Passed {
		t.Fatalf("expected dry-run to pass")
	}
}

func TestAdminPolicyFixturesAndTestSuite(t *testing.T) {
	srv := newTestServer(health.NewChecker())
	router := srv.Routes()

	// GET /admin/policies/{id}/fixtures
	req := httptest.NewRequest(http.MethodGet, "/admin/policies/pol-123/fixtures", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET fixtures: expected 200, got %d", rec.Code)
	}

	// POST /admin/policies/{id}/fixtures
	body := strings.NewReader(`{"name": "test-admin", "expected_result": true, "payload": {}}`)
	req = httptest.NewRequest(http.MethodPost, "/admin/policies/pol-123/fixtures", body)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("POST fixtures: expected 201, got %d", rec.Code)
	}

	// DELETE /admin/policies/{id}/fixtures/{fixture_id}
	req = httptest.NewRequest(http.MethodDelete, "/admin/policies/pol-123/fixtures/fix-1", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("DELETE fixture: expected 200, got %d", rec.Code)
	}

	// POST /admin/policies/{id}/test-suite
	req = httptest.NewRequest(http.MethodPost, "/admin/policies/pol-123/test-suite", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("POST test-suite: expected 200, got %d", rec.Code)
	}
	var suiteRes core.TestSuiteResult
	if err := json.NewDecoder(rec.Body).Decode(&suiteRes); err != nil {
		t.Fatalf("decode test suite response: %v", err)
	}
	if !suiteRes.AllPassed {
		t.Fatalf("expected all tests to pass in test suite")
	}
}

func TestMetricsEndpoint(t *testing.T) {
	handler := health.NewHandler(health.NewChecker(), func() health.Info { return health.Info{Engine: "themis"} })
	srv := NewServer(stubPolicyEngine{}, handler)
	router := srv.Routes()

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /metrics: expected 200, got %d", rec.Code)
	}
}

