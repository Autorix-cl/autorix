package core

import (
	"context"
	"fmt"
	"sort"
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
func (s *Service) Evaluate(ctx context.Context, req EvaluateRequest) (*EvaluateResponse, error) {
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

	resp := &EvaluateResponse{
		AllPassed:      true,
		TotalEvaluated: int32(len(policies)),
	}

	for _, p := range policies {
		result := PolicyResult{
			PolicyID:   p.ID,
			PolicyName: p.Name,
			Expression: p.Expression,
		}

		passed, err := s.evaluator.Evaluate(p.Expression, req.Payload)
		if err != nil {
			result.Passed = false
			result.Error = err.Error()
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
