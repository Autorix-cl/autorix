package core_test

import (
	"context"
	"testing"

	"github.com/autorix/themis/internal/core"
)

type mockEvaluator struct {
	compileErr  error
	evalResult  bool
	evalErr     error
	validateRes *core.ValidationResult
	validateErr error
}

func (m *mockEvaluator) Compile(expression string) error { return m.compileErr }
func (m *mockEvaluator) Evaluate(expression string, payload map[string]interface{}) (bool, error) {
	return m.evalResult, m.evalErr
}
func (m *mockEvaluator) Validate(expression string) (*core.ValidationResult, error) {
	return m.validateRes, m.validateErr
}

type mockRepo struct {
	policies []core.Policy
	versions []core.PolicyVersion
	fixtures []core.PolicyFixture
}

func (m *mockRepo) Create(ctx context.Context, policy *core.Policy) error {
	m.policies = append(m.policies, *policy)
	return nil
}
func (m *mockRepo) GetByID(ctx context.Context, tenantID, policyID string) (*core.Policy, error) {
	for _, p := range m.policies {
		if p.TenantID == tenantID && p.ID == policyID {
			return &p, nil
		}
	}
	return nil, nil
}
func (m *mockRepo) List(ctx context.Context, filter core.ListFilter) ([]core.Policy, error) {
	return m.policies, nil
}
func (m *mockRepo) ListPage(ctx context.Context, filter core.ListFilter, limit int, cursor string) ([]core.Policy, bool, error) {
	return m.policies, false, nil
}
func (m *mockRepo) Update(ctx context.Context, policy *core.Policy) error { return nil }
func (m *mockRepo) Delete(ctx context.Context, tenantID, policyID string) error  { return nil }
func (m *mockRepo) ListVersions(ctx context.Context, tenantID, policyID string) ([]core.PolicyVersion, error) {
	return m.versions, nil
}
func (m *mockRepo) GetVersion(ctx context.Context, tenantID, policyID string, version int32) (*core.PolicyVersion, error) {
	for _, v := range m.versions {
		if v.TenantID == tenantID && v.PolicyID == policyID && v.Version == version {
			return &v, nil
		}
	}
	return nil, nil
}
func (m *mockRepo) Rollback(ctx context.Context, tenantID, policyID string, version int32) (*core.Policy, error) {
	return &core.Policy{ID: policyID, TenantID: tenantID, Name: "restored"}, nil
}
func (m *mockRepo) CreateFixture(ctx context.Context, fixture *core.PolicyFixture) error {
	m.fixtures = append(m.fixtures, *fixture)
	return nil
}
func (m *mockRepo) ListFixtures(ctx context.Context, tenantID, policyID string) ([]core.PolicyFixture, error) {
	return m.fixtures, nil
}
func (m *mockRepo) DeleteFixture(ctx context.Context, tenantID, policyID, fixtureID string) error {
	return nil
}

func TestService_DryRun(t *testing.T) {
	ctx := context.Background()
	eval := &mockEvaluator{evalResult: true}
	repo := &mockRepo{}
	svc := core.NewService(repo, eval)

	res, err := svc.DryRun(ctx, core.DryRunRequest{
		Expression: `request.role == "admin"`,
		Payload:    map[string]interface{}{"request": map[string]interface{}{"role": "admin"}},
	})
	if err != nil {
		t.Fatalf("DryRun unexpected error: %v", err)
	}
	if !res.Passed {
		t.Fatalf("expected dry-run to pass")
	}
}

func TestService_RunTestSuite(t *testing.T) {
	ctx := context.Background()
	eval := &mockEvaluator{evalResult: true}
	repo := &mockRepo{
		policies: []core.Policy{
			{ID: "pol-1", TenantID: "t1", Expression: "true", Enabled: true},
		},
		fixtures: []core.PolicyFixture{
			{ID: "fix-1", PolicyID: "pol-1", TenantID: "t1", Name: "pass test", ExpectedResult: true},
		},
	}
	svc := core.NewService(repo, eval)

	res, err := svc.RunTestSuite(ctx, "t1", "pol-1")
	if err != nil {
		t.Fatalf("RunTestSuite unexpected error: %v", err)
	}
	if !res.AllPassed {
		t.Fatalf("expected all tests to pass, got: %+v", res)
	}
	if res.TotalTests != 1 || res.PassedTests != 1 {
		t.Fatalf("unexpected test counts: %+v", res)
	}
}
