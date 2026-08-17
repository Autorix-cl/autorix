package core

import (
	"context"
	"time"
)

// Policy represents a business rule owned by a Tenant.
// Each policy contains a CEL expression that is evaluated against
// a dynamic JSON payload at runtime.
type Policy struct {
	ID          string
	TenantID    string
	Name        string
	Description string
	Expression  string
	Priority    int32
	Enabled     bool
	Labels      map[string]string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// PolicyResult holds the outcome of evaluating a single policy.
type PolicyResult struct {
	PolicyID   string
	PolicyName string
	Passed     bool
	Error      string
	Expression string
}

// EvaluateRequest captures everything needed to run an evaluation.
type EvaluateRequest struct {
	TenantID    string
	PolicyID    string // Empty = evaluate all enabled policies for tenant
	Payload     map[string]interface{}
	LabelFilter map[string]string
}

// EvaluateResponse aggregates results from one or more policy evaluations.
type EvaluateResponse struct {
	AllPassed      bool
	Results        []PolicyResult
	TotalEvaluated int32
}

// ListFilter constrains which policies to return.
type ListFilter struct {
	TenantID    string
	EnabledOnly bool
	Labels      map[string]string
}

// PolicyRepository defines the persistence contract for policies.
type PolicyRepository interface {
	Create(ctx context.Context, policy *Policy) error
	GetByID(ctx context.Context, tenantID, policyID string) (*Policy, error)
	List(ctx context.Context, filter ListFilter) ([]Policy, error)
	Update(ctx context.Context, policy *Policy) error
	Delete(ctx context.Context, tenantID, policyID string) error
}

// PolicyEvaluator compiles and evaluates CEL expressions.
type PolicyEvaluator interface {
	// Compile verifies that the CEL expression is syntactically valid and returns boolean.
	Compile(expression string) error

	// Evaluate compiles (or retrieves from cache) the expression and
	// evaluates it against the given payload. Returns true if the
	// expression evaluates to boolean true.
	Evaluate(expression string, payload map[string]interface{}) (bool, error)
}

// PolicyEngine orchestrates CRUD and evaluation.
type PolicyEngine interface {
	CreatePolicy(ctx context.Context, policy *Policy) error
	GetPolicy(ctx context.Context, tenantID, policyID string) (*Policy, error)
	ListPolicies(ctx context.Context, filter ListFilter) ([]Policy, error)
	UpdatePolicy(ctx context.Context, policy *Policy) error
	DeletePolicy(ctx context.Context, tenantID, policyID string) error
	Evaluate(ctx context.Context, req EvaluateRequest) (*EvaluateResponse, error)
}
