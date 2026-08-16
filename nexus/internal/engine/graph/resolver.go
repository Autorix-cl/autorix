package graph

import (
	"context"
	"fmt"
	"sync"

	"github.com/autorix/nexus/internal/core"
)

// Repository is the interface required by the graph engine to fetch tuples
type Repository interface {
	ReadTuples(ctx context.Context, filter core.Tuple) ([]core.Tuple, error)
}

// Resolver implements the core.GraphEngine using concurrent in-memory resolution
type Resolver struct {
	repo            Repository
	caveatEvaluator core.CaveatEvaluator // In reality, this should be a Caveat Registry/Cache
}

func NewResolver(repo Repository, ce core.CaveatEvaluator) *Resolver {
	return &Resolver{
		repo:            repo,
		caveatEvaluator: ce,
	}
}

// Check evaluates if a subject has a specific relation to an object
func (r *Resolver) Check(ctx context.Context, req core.CheckRequest) (core.CheckResult, error) {
	// Base case: check direct relation
	filter := core.Tuple{
		Namespace: req.Namespace,
		Object:    req.Object,
		Relation:  req.Relation,
	}

	tuples, err := r.repo.ReadTuples(ctx, filter)
	if err != nil {
		return core.CheckResult{}, fmt.Errorf("failed to read tuples: %w", err)
	}

	// We use a channel to receive successful checks from goroutines
	// and a context with cancel to short-circuit when the first match is found
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	resultChan := make(chan core.CheckResult, len(tuples))
	var wg sync.WaitGroup

	for _, t := range tuples {
		wg.Add(1)
		go func(tuple core.Tuple) {
			defer wg.Done()

			// 1. Check if the tuple has a Caveat (ABAC Rule)
			if tuple.CaveatName != "" {
				// In a full implementation, we'd fetch the Caveat expression from DB using CaveatName
				// For this prototype, we assume the expression is already compiled in caveatEvaluator
				// Or we'd use a CaveatRegistry.
				allowed, err := r.caveatEvaluator.Evaluate(req.RequestContext, tuple.CaveatContext)
				if err != nil || !allowed {
					// Caveat failed, this path is dead
					return 
				}
			}

			// 2. Direct Subject Match
			if tuple.SubjectNamespace == req.Subject.Namespace && 
			   tuple.SubjectObject == req.Subject.Object && 
			   tuple.SubjectRelation == req.Subject.Relation {
				
				resultChan <- core.CheckResult{Allowed: true, Reason: "direct match"}
				cancel() // Short-circuit other checks
				return
			}

			// 3. Indirect Subject Match (Usersets expansion)
			// e.g., if tuple subject is "group:admins#member", we must check if our req.Subject
			// has the "member" relation on the "group:admins" object.
			if tuple.SubjectRelation != "" {
				subReq := core.CheckRequest{
					Namespace:      tuple.SubjectNamespace,
					Object:         tuple.SubjectObject,
					Relation:       tuple.SubjectRelation,
					Subject:        req.Subject,
					RequestContext: req.RequestContext, // Propagate dynamic context
				}

				subRes, err := r.Check(ctx, subReq)
				if err == nil && subRes.Allowed {
					resultChan <- core.CheckResult{Allowed: true, Reason: "indirect match"}
					cancel()
					return
				}
			}

		}(t)
	}

	// Wait for all goroutines to finish in a separate goroutine
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Wait for either the first success, or for the channel to close (all failed)
	for res := range resultChan {
		if res.Allowed {
			return res, nil
		}
	}

	return core.CheckResult{Allowed: false, Reason: "no matching path found"}, nil
}

// Expand returns the full relation tree (omitted for brevity in this step)
func (r *Resolver) Expand(ctx context.Context, req core.ExpandRequest) (*core.ExpandTree, error) {
	return nil, fmt.Errorf("not implemented yet")
}
