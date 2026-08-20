package core

import (
	"context"
	"time"
)

// Policy represents a business rule owned by a Tenant.
// Each policy contains a CEL expression that is evaluated against
// a dynamic JSON payload at runtime.
type Policy struct {
	ID          string            `json:"id"`
	TenantID    string            `json:"tenant_id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Expression  string            `json:"expression"`
	Priority    int32             `json:"priority"`
	Enabled     bool              `json:"enabled"`
	Labels      map[string]string `json:"labels"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// PolicyVersion captures an immutable historical snapshot of a policy.
type PolicyVersion struct {
	ID          string            `json:"id"`
	PolicyID    string            `json:"policy_id"`
	TenantID    string            `json:"tenant_id"`
	Version     int32             `json:"version"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Expression  string            `json:"expression"`
	Priority    int32             `json:"priority"`
	Enabled     bool              `json:"enabled"`
	Labels      map[string]string `json:"labels"`
	CreatedAt   time.Time         `json:"created_at"`
}

// PolicyFixture represents a test scenario attached to a policy.
type PolicyFixture struct {
	ID             string                 `json:"id"`
	PolicyID       string                 `json:"policy_id"`
	TenantID       string                 `json:"tenant_id"`
	Name           string                 `json:"name"`
	Description    string                 `json:"description"`
	Payload        map[string]interface{} `json:"payload"`
	ExpectedResult bool                   `json:"expected_result"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

// FixtureRunResult holds the outcome of running a single fixture against a policy.
type FixtureRunResult struct {
	FixtureID      string `json:"fixture_id"`
	FixtureName    string `json:"fixture_name"`
	ExpectedResult bool   `json:"expected_result"`
	ActualResult   bool   `json:"actual_result"`
	Passed         bool   `json:"passed"`
	Error          string `json:"error,omitempty"`
}

// TestSuiteResult aggregates results from evaluating all fixtures attached to a policy.
type TestSuiteResult struct {
	PolicyID    string             `json:"policy_id"`
	AllPassed   bool               `json:"all_passed"`
	TotalTests  int                `json:"total_tests"`
	PassedTests int                `json:"passed_tests"`
	FailedTests int                `json:"failed_tests"`
	Results     []FixtureRunResult `json:"results"`
}

// ValidationError detail with line and column.
type ValidationError struct {
	Line    int    `json:"line"`
	Column  int    `json:"column"`
	Message string `json:"message"`
}

// ValidationResult represents the output of CEL syntax and type validation.
type ValidationResult struct {
	Valid     bool              `json:"valid"`
	Variables []string          `json:"variables"`
	Errors    []ValidationError `json:"errors,omitempty"`
}

// DryRunRequest contains an unpersisted expression and payload to evaluate.
type DryRunRequest struct {
	Expression string                 `json:"expression"`
	Payload    map[string]interface{} `json:"payload"`
}

// DryRunResult holds the outcome of a dry-run evaluation.
type DryRunResult struct {
	Passed bool   `json:"passed"`
	Error  string `json:"error,omitempty"`
}

// PolicyResult holds the outcome of evaluating a single policy.
type PolicyResult struct {
	PolicyID   string `json:"policy_id"`
	PolicyName string `json:"policy_name"`
	Passed     bool   `json:"passed"`
	Error      string `json:"error,omitempty"`
	Expression string `json:"expression"`
}

// EvaluateRequest captures everything needed to run an evaluation.
type EvaluateRequest struct {
	TenantID    string                 `json:"tenant_id"`
	PolicyID    string                 `json:"policy_id"` // Empty = evaluate all enabled policies for tenant
	Payload     map[string]interface{} `json:"payload"`
	LabelFilter map[string]string      `json:"label_filter"`
}

// EvaluateResponse aggregates results from one or more policy evaluations.
type EvaluateResponse struct {
	AllPassed      bool           `json:"all_passed"`
	Results        []PolicyResult `json:"results"`
	TotalEvaluated int32          `json:"total_evaluated"`
}

// ListFilter constrains which policies to return.
type ListFilter struct {
	TenantID    string
	EnabledOnly bool
	Labels      map[string]string
}

// PolicyRepository defines the persistence contract for policies, versions, and fixtures.
type PolicyRepository interface {
	Create(ctx context.Context, policy *Policy) error
	GetByID(ctx context.Context, tenantID, policyID string) (*Policy, error)
	// List returns the complete filtered set, unpaginated. It exists
	// alongside ListPage because bulk policy evaluation (Service.Evaluate)
	// needs every matching policy, not one page of them.
	List(ctx context.Context, filter ListFilter) ([]Policy, error)
	// ListPage is List's paginated counterpart for the admin listing
	// endpoint: cursor is the opaque, paging.EncodeCursor-produced token
	// from the last row of a previous page (empty starts from the
	// beginning).
	ListPage(ctx context.Context, filter ListFilter, limit int, cursor string) ([]Policy, bool, error)
	Update(ctx context.Context, policy *Policy) error
	Delete(ctx context.Context, tenantID, policyID string) error

	// Versioning
	ListVersions(ctx context.Context, tenantID, policyID string) ([]PolicyVersion, error)
	GetVersion(ctx context.Context, tenantID, policyID string, version int32) (*PolicyVersion, error)
	Rollback(ctx context.Context, tenantID, policyID string, version int32) (*Policy, error)

	// Fixtures
	CreateFixture(ctx context.Context, fixture *PolicyFixture) error
	ListFixtures(ctx context.Context, tenantID, policyID string) ([]PolicyFixture, error)
	DeleteFixture(ctx context.Context, tenantID, policyID, fixtureID string) error
}

// PolicyEvaluator compiles and evaluates CEL expressions.
type PolicyEvaluator interface {
	// Compile verifies that the CEL expression is syntactically valid and returns boolean.
	Compile(expression string) error

	// Evaluate compiles (or retrieves from cache) the expression and
	// evaluates it against the given payload. Returns true if the
	// expression evaluates to boolean true.
	Evaluate(expression string, payload map[string]interface{}) (bool, error)

	// Validate analyzes a CEL expression, returning variable references and any syntax/type errors.
	Validate(expression string) (*ValidationResult, error)
}

// PolicyEngine orchestrates CRUD, evaluation, versioning, validation, and testing.
type PolicyEngine interface {
	CreatePolicy(ctx context.Context, policy *Policy) error
	GetPolicy(ctx context.Context, tenantID, policyID string) (*Policy, error)
	ListPolicies(ctx context.Context, filter ListFilter) ([]Policy, error)
	ListPoliciesPage(ctx context.Context, filter ListFilter, limit int, cursor string) ([]Policy, bool, error)
	UpdatePolicy(ctx context.Context, policy *Policy) error
	DeletePolicy(ctx context.Context, tenantID, policyID string) error
	Evaluate(ctx context.Context, req EvaluateRequest) (*EvaluateResponse, error)

	// Versioning
	ListPolicyVersions(ctx context.Context, tenantID, policyID string) ([]PolicyVersion, error)
	RollbackPolicy(ctx context.Context, tenantID, policyID string, version int32) (*Policy, error)

	// Validation & Dry-Run
	ValidateExpression(ctx context.Context, expression string) (*ValidationResult, error)
	DryRun(ctx context.Context, req DryRunRequest) (*DryRunResult, error)

	// Fixtures
	CreateFixture(ctx context.Context, fixture *PolicyFixture) error
	ListFixtures(ctx context.Context, tenantID, policyID string) ([]PolicyFixture, error)
	DeleteFixture(ctx context.Context, tenantID, policyID, fixtureID string) error
	RunTestSuite(ctx context.Context, tenantID, policyID string) (*TestSuiteResult, error)
}
