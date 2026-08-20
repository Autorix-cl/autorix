package core

import (
	"context"
	"fmt"
	"sort"
	"time"
)

// Service implements the PolicyEngine interface, orchestrating the
// repository and evaluator to handle CRUD + evaluation operations.
type Service struct {
	repo      PolicyRepository
	evaluator PolicyEvaluator
}

// NewService creates a new policy engine service.
func NewService(repo PolicyRepository, evaluator PolicyEvaluator) *Service {
	return &Service{
		repo:      repo,
		evaluator: evaluator,
	}
}

func (s *Service) CreatePolicy(ctx context.Context, policy *Policy) error {
	// Validate the CEL expression compiles before persisting
	if err := s.evaluator.Compile(policy.Expression); err != nil {
		return fmt.Errorf("invalid CEL expression: %w", err)
	}
	return s.repo.Create(ctx, policy)
}

func (s *Service) GetPolicy(ctx context.Context, tenantID, policyID string) (*Policy, error) {
	return s.repo.GetByID(ctx, tenantID, policyID)
}

func (s *Service) ListPolicies(ctx context.Context, filter ListFilter) ([]Policy, error) {
	return s.repo.List(ctx, filter)
}

func (s *Service) ListPoliciesPage(ctx context.Context, filter ListFilter, limit int, cursor string) ([]Policy, bool, error) {
	return s.repo.ListPage(ctx, filter, limit, cursor)
}

func (s *Service) UpdatePolicy(ctx context.Context, policy *Policy) error {
	// Validate the new expression compiles
	if err := s.evaluator.Compile(policy.Expression); err != nil {
		return fmt.Errorf("invalid CEL expression: %w", err)
	}
	return s.repo.Update(ctx, policy)
}

func (s *Service) DeletePolicy(ctx context.Context, tenantID, policyID string) error {
	return s.repo.Delete(ctx, tenantID, policyID)
}

// Evaluate runs one or more policies against a payload.
// If PolicyID is set, only that policy is evaluated.
// Otherwise, all enabled policies for the tenant (optionally filtered by labels) are evaluated.
func (s *Service) Evaluate(ctx context.Context, req EvaluateRequest) (resp *EvaluateResponse, err error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		themisEvaluationDuration.Observe(duration)
		decision := "deny"
		if err == nil && resp != nil && resp.AllPassed {
			decision = "allow"
		}
		themisEvaluationTotal.WithLabelValues(decision).Inc()
	}()

	var policies []Policy

	if req.PolicyID != "" {
		// Single policy evaluation
		p, err := s.repo.GetByID(ctx, req.TenantID, req.PolicyID)
		if err != nil {
			return nil, fmt.Errorf("policy not found: %w", err)
		}
		if !p.Enabled {
			return nil, fmt.Errorf("policy %s is disabled", p.Name)
		}
		policies = []Policy{*p}
	} else {
		// Bulk evaluation: all enabled policies for tenant
		var err error
		policies, err = s.repo.List(ctx, ListFilter{
			TenantID:    req.TenantID,
			EnabledOnly: true,
			Labels:      req.LabelFilter,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list policies: %w", err)
		}
	}

	// Sort by priority (ascending = highest priority first)
	sort.Slice(policies, func(i, j int) bool {
		return policies[i].Priority < policies[j].Priority
	})

	resp = &EvaluateResponse{
		AllPassed:      true,
		TotalEvaluated: int32(len(policies)), // #nosec G115 -- bounded by a tenant's policy count, never near int32 range
	}

	for _, p := range policies {
		result := PolicyResult{
			PolicyID:   p.ID,
			PolicyName: p.Name,
			Expression: p.Expression,
		}

		passed, evalErr := s.evaluator.Evaluate(p.Expression, req.Payload)
		if evalErr != nil {
			result.Passed = false
			result.Error = evalErr.Error()
			resp.AllPassed = false
		} else {
			result.Passed = passed
			if !passed {
				resp.AllPassed = false
			}
		}

		resp.Results = append(resp.Results, result)
	}

	return resp, nil
}

// ListPolicyVersions retrieves the revision history for a policy.
func (s *Service) ListPolicyVersions(ctx context.Context, tenantID, policyID string) ([]PolicyVersion, error) {
	return s.repo.ListVersions(ctx, tenantID, policyID)
}

// RollbackPolicy reverts a policy to a target version.
func (s *Service) RollbackPolicy(ctx context.Context, tenantID, policyID string, version int32) (*Policy, error) {
	return s.repo.Rollback(ctx, tenantID, policyID, version)
}

// ValidateExpression analyzes candidate CEL expressions.
func (s *Service) ValidateExpression(ctx context.Context, expression string) (*ValidationResult, error) {
	return s.evaluator.Validate(expression)
}

// DryRun evaluates an unpersisted expression against a test payload.
func (s *Service) DryRun(ctx context.Context, req DryRunRequest) (*DryRunResult, error) {
	passed, err := s.evaluator.Evaluate(req.Expression, req.Payload)
	if err != nil {
		return &DryRunResult{
			Passed: false,
			Error:  err.Error(),
		}, nil
	}
	return &DryRunResult{
		Passed: passed,
		Error:  "",
	}, nil
}

// CreateFixture persists a test fixture.
func (s *Service) CreateFixture(ctx context.Context, fixture *PolicyFixture) error {
	return s.repo.CreateFixture(ctx, fixture)
}

// ListFixtures lists all test fixtures for a policy.
func (s *Service) ListFixtures(ctx context.Context, tenantID, policyID string) ([]PolicyFixture, error) {
	return s.repo.ListFixtures(ctx, tenantID, policyID)
}

// DeleteFixture removes a test fixture.
func (s *Service) DeleteFixture(ctx context.Context, tenantID, policyID, fixtureID string) error {
	return s.repo.DeleteFixture(ctx, tenantID, policyID, fixtureID)
}

// RunTestSuite executes all fixtures attached to a policy against its current expression.
func (s *Service) RunTestSuite(ctx context.Context, tenantID, policyID string) (*TestSuiteResult, error) {
	policy, err := s.repo.GetByID(ctx, tenantID, policyID)
	if err != nil {
		return nil, fmt.Errorf("policy not found: %w", err)
	}

	fixtures, err := s.repo.ListFixtures(ctx, tenantID, policyID)
	if err != nil {
		return nil, fmt.Errorf("failed to list fixtures: %w", err)
	}

	suite := &TestSuiteResult{
		PolicyID:   policyID,
		AllPassed:  true,
		TotalTests: len(fixtures),
	}

	for _, f := range fixtures {
		actual, evalErr := s.evaluator.Evaluate(policy.Expression, f.Payload)
		res := FixtureRunResult{
			FixtureID:      f.ID,
			FixtureName:    f.Name,
			ExpectedResult: f.ExpectedResult,
			ActualResult:   actual,
		}

		if evalErr != nil {
			res.Error = evalErr.Error()
			res.Passed = false
			suite.AllPassed = false
			suite.FailedTests++
		} else if actual == f.ExpectedResult {
			res.Passed = true
			suite.PassedTests++
		} else {
			res.Passed = false
			suite.AllPassed = false
			suite.FailedTests++
		}

		suite.Results = append(suite.Results, res)
	}

	return suite, nil
}
