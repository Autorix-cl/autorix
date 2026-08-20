package graph

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/autorix/nexus/internal/core"
)

// Repository is the interface required by the graph engine to fetch and query tuples
type Repository interface {
	ReadTuples(ctx context.Context, filter core.Tuple) ([]core.Tuple, error)
	QueryTuples(ctx context.Context, filter core.Tuple) ([]core.Tuple, error)
}

// NamespaceGetter allows retrieving namespace schemas for rewrite rules
type NamespaceGetter interface {
	GetNamespace(ctx context.Context, name string) (*core.NamespaceSchema, error)
}

// Option configures a Resolver
type Option func(*Resolver)

// WithNamespaceGetter sets the schema source for rewrite evaluation
func WithNamespaceGetter(ng NamespaceGetter) Option {
	return func(r *Resolver) {
		r.namespaceGetter = ng
	}
}

// Resolver implements core.GraphEngine using concurrent graph traversal
type Resolver struct {
	repo            Repository
	caveatEvaluator core.CaveatEvaluator
	namespaceGetter NamespaceGetter
}

// NewResolver initializes a Resolver with options
func NewResolver(repo Repository, ce core.CaveatEvaluator, opts ...Option) *Resolver {
	r := &Resolver{
		repo:            repo,
		caveatEvaluator: ce,
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// Check evaluates whether a subject has a relation to an object
func (r *Resolver) Check(ctx context.Context, req core.CheckRequest) (core.CheckResult, error) {
	start := time.Now()
	visited := make(map[string]bool)
	res, err := r.checkInternal(ctx, req, visited)
	duration := time.Since(start).Seconds()

	nexusCheckDuration.Observe(duration)
	decision := "deny"
	if res.Allowed {
		decision = "allow"
	}
	nexusCheckTotal.WithLabelValues(decision).Inc()

	return res, err
}

func (r *Resolver) checkInternal(ctx context.Context, req core.CheckRequest, visited map[string]bool) (core.CheckResult, error) {
	key := fmt.Sprintf("%s:%s#%s@%s:%s#%s",
		req.Namespace, req.Object, req.Relation,
		req.Subject.Namespace, req.Subject.Object, req.Subject.Relation)

	if visited[key] {
		// Cycle detected
		return core.CheckResult{
			Allowed: false,
			Reason:  "cycle detected",
			Trace: &core.DecisionNode{
				Namespace: req.Namespace,
				Object:    req.Object,
				Relation:  req.Relation,
				Subject:   req.Subject,
				Allowed:   false,
				Reason:    "cycle detected",
			},
		}, nil
	}

	visitedCopy := make(map[string]bool, len(visited)+1)
	for k, v := range visited {
		visitedCopy[k] = v
	}
	visitedCopy[key] = true

	// Check if schema defines rewrite rules
	var rewrite *core.RewriteRule
	if r.namespaceGetter != nil {
		schema, err := r.namespaceGetter.GetNamespace(ctx, req.Namespace)
		if err == nil && schema != nil {
			if relDef, ok := schema.Relations[req.Relation]; ok {
				rewrite = relDef.Rewrite
			}
		}
	}

	if rewrite != nil {
		return r.evaluateRewrite(ctx, req, rewrite, visitedCopy)
	}

	// Default to direct tuple matching ("this")
	return r.evaluateThis(ctx, req, visitedCopy)
}

func (r *Resolver) evaluateRewrite(ctx context.Context, req core.CheckRequest, rule *core.RewriteRule, visited map[string]bool) (core.CheckResult, error) {
	traceNode := &core.DecisionNode{
		Namespace:   req.Namespace,
		Object:      req.Object,
		Relation:    req.Relation,
		Subject:     req.Subject,
		RewriteType: rule.Type,
	}

	switch rule.Type {
	case "this", "":
		res, err := r.evaluateThis(ctx, req, visited)
		if err != nil {
			return res, err
		}
		if req.Explain {
			traceNode.Allowed = res.Allowed
			traceNode.Reason = res.Reason
			if res.Trace != nil {
				traceNode.Children = res.Trace.Children
				traceNode.Caveat = res.Trace.Caveat
			}
			res.Trace = traceNode
		}
		return res, nil

	case "computed_userset":
		subReq := req
		subReq.Relation = rule.Relation
		res, err := r.checkInternal(ctx, subReq, visited)
		if err != nil {
			return res, err
		}
		if req.Explain {
			traceNode.Allowed = res.Allowed
			traceNode.Reason = fmt.Sprintf("computed_userset (%s): %s", rule.Relation, res.Reason)
			if res.Trace != nil {
				traceNode.Children = []*core.DecisionNode{res.Trace}
			}
			res.Trace = traceNode
		}
		return res, nil

	case "tuple_to_userset":
		tuples, err := r.repo.ReadTuples(ctx, core.Tuple{
			Namespace: req.Namespace,
			Object:    req.Object,
			Relation:  rule.TuplesetRelation,
		})
		if err != nil {
			return core.CheckResult{}, err
		}

		for _, t := range tuples {
			subReq := core.CheckRequest{
				Namespace:      t.SubjectNamespace,
				Object:         t.SubjectObject,
				Relation:       rule.ComputedRelation,
				Subject:        req.Subject,
				RequestContext: req.RequestContext,
				Explain:        req.Explain,
			}
			subRes, err := r.checkInternal(ctx, subReq, visited)
			if err == nil {
				if req.Explain && subRes.Trace != nil {
					traceNode.Children = append(traceNode.Children, subRes.Trace)
				}
				if subRes.Allowed {
					traceNode.Allowed = true
					traceNode.Reason = "tuple_to_userset match"
					return core.CheckResult{
						Allowed: true,
						Reason:  "tuple_to_userset match",
						Trace:   traceNode,
					}, nil
				}
			}
		}

		traceNode.Allowed = false
		traceNode.Reason = "no matching tuple_to_userset path"
		return core.CheckResult{
			Allowed: false,
			Reason:  traceNode.Reason,
			Trace:   traceNode,
		}, nil

	case "union":
		var anyAllowed bool
		var reasons []string
		for _, child := range rule.Children {
			childRes, err := r.evaluateRewrite(ctx, req, child, visited)
			if err != nil {
				continue
			}
			if req.Explain && childRes.Trace != nil {
				traceNode.Children = append(traceNode.Children, childRes.Trace)
			}
			if childRes.Allowed {
				anyAllowed = true
				reasons = append(reasons, childRes.Reason)
			}
		}

		traceNode.Allowed = anyAllowed
		if anyAllowed {
			traceNode.Reason = "union match"
			return core.CheckResult{
				Allowed: true,
				Reason:  "union match",
				Trace:   traceNode,
			}, nil
		}

		traceNode.Reason = "union: no branches satisfied"
		return core.CheckResult{
			Allowed: false,
			Reason:  traceNode.Reason,
			Trace:   traceNode,
		}, nil

	case "intersection":
		allAllowed := len(rule.Children) > 0
		for _, child := range rule.Children {
			childRes, err := r.evaluateRewrite(ctx, req, child, visited)
			if err != nil || !childRes.Allowed {
				allAllowed = false
			}
			if req.Explain && childRes.Trace != nil {
				traceNode.Children = append(traceNode.Children, childRes.Trace)
			}
		}

		traceNode.Allowed = allAllowed
		if allAllowed {
			traceNode.Reason = "intersection match"
			return core.CheckResult{
				Allowed: true,
				Reason:  "intersection match",
				Trace:   traceNode,
			}, nil
		}

		traceNode.Reason = "intersection: not all branches satisfied"
		return core.CheckResult{
			Allowed: false,
			Reason:  traceNode.Reason,
			Trace:   traceNode,
		}, nil

	default:
		return core.CheckResult{Allowed: false, Reason: fmt.Sprintf("unsupported rewrite type: %s", rule.Type)}, nil
	}
}

func (r *Resolver) evaluateThis(ctx context.Context, req core.CheckRequest, visited map[string]bool) (core.CheckResult, error) {
	filter := core.Tuple{
		Namespace: req.Namespace,
		Object:    req.Object,
		Relation:  req.Relation,
	}

	tuples, err := r.repo.ReadTuples(ctx, filter)
	if err != nil {
		return core.CheckResult{}, fmt.Errorf("failed to read tuples: %w", err)
	}

	rootTrace := &core.DecisionNode{
		Namespace: req.Namespace,
		Object:    req.Object,
		Relation:  req.Relation,
		Subject:   req.Subject,
	}

	for _, tuple := range tuples {
		childTrace := &core.DecisionNode{
			Namespace: tuple.SubjectNamespace,
			Object:    tuple.SubjectObject,
			Relation:  tuple.SubjectRelation,
			Subject:   req.Subject,
		}

		// 1. Evaluate Caveat if present
		if tuple.CaveatName != "" {
			allowed, err := r.evalCaveat(ctx, tuple.CaveatName, req.RequestContext, tuple.CaveatContext)
			caveatRes := &core.CaveatResult{
				CaveatName: tuple.CaveatName,
				Allowed:    allowed && err == nil,
				Context:    req.RequestContext,
			}
			if err != nil {
				caveatRes.Reason = err.Error()
			}
			childTrace.Caveat = caveatRes
			rootTrace.Caveat = caveatRes

			if err != nil || !allowed {
				childTrace.Allowed = false
				childTrace.Reason = "caveat denied"
				if req.Explain {
					rootTrace.Children = append(rootTrace.Children, childTrace)
				}
				continue
			}
		}

		// 2. Direct Subject Match
		if tuple.SubjectNamespace == req.Subject.Namespace &&
			tuple.SubjectObject == req.Subject.Object &&
			tuple.SubjectRelation == req.Subject.Relation {

			childTrace.Allowed = true
			childTrace.Reason = "direct match"
			rootTrace.Allowed = true
			rootTrace.Reason = "direct match"
			if req.Explain {
				rootTrace.Children = append(rootTrace.Children, childTrace)
			}
			return core.CheckResult{
				Allowed: true,
				Reason:  "direct match",
				Trace:   rootTrace,
			}, nil
		}

		// 3. Indirect Userset Match
		if tuple.SubjectRelation != "" {
			subReq := core.CheckRequest{
				Namespace:      tuple.SubjectNamespace,
				Object:         tuple.SubjectObject,
				Relation:       tuple.SubjectRelation,
				Subject:        req.Subject,
				RequestContext: req.RequestContext,
				Explain:        req.Explain,
			}

			subRes, err := r.checkInternal(ctx, subReq, visited)
			if err == nil {
				if req.Explain && subRes.Trace != nil {
					childTrace.Children = append(childTrace.Children, subRes.Trace)
				}
				if subRes.Allowed {
					childTrace.Allowed = true
					childTrace.Reason = "indirect match"
					rootTrace.Allowed = true
					rootTrace.Reason = "indirect match"
					if req.Explain {
						rootTrace.Children = append(rootTrace.Children, childTrace)
					}
					return core.CheckResult{
						Allowed: true,
						Reason:  "indirect match",
						Trace:   rootTrace,
					}, nil
				}
			}
		}

		if req.Explain {
			rootTrace.Children = append(rootTrace.Children, childTrace)
		}
	}

	rootTrace.Allowed = false
	rootTrace.Reason = "no matching path found"
	return core.CheckResult{
		Allowed: false,
		Reason:  rootTrace.Reason,
		Trace:   rootTrace,
	}, nil
}

func (r *Resolver) evalCaveat(ctx context.Context, name string, reqCtx, caveatCtx map[string]interface{}) (bool, error) {
	if r.caveatEvaluator == nil {
		return false, fmt.Errorf("no caveat evaluator configured")
	}
	return r.caveatEvaluator.EvaluateByName(ctx, name, reqCtx, caveatCtx)
}

// Expand returns the full subtree of relations
func (r *Resolver) Expand(ctx context.Context, req core.ExpandRequest) (*core.ExpandTree, error) {
	visited := make(map[string]bool)
	return r.expandInternal(ctx, req, visited, 0)
}

func (r *Resolver) expandInternal(ctx context.Context, req core.ExpandRequest, visited map[string]bool, depth int) (*core.ExpandTree, error) {
	if depth > 20 {
		return &core.ExpandTree{Type: "max_depth_reached"}, nil
	}

	key := fmt.Sprintf("%s:%s#%s", req.Namespace, req.Object, req.Relation)
	if visited[key] {
		return &core.ExpandTree{Type: "cycle"}, nil
	}

	visitedCopy := make(map[string]bool, len(visited)+1)
	for k, v := range visited {
		visitedCopy[k] = v
	}
	visitedCopy[key] = true

	var rewrite *core.RewriteRule
	if r.namespaceGetter != nil {
		schema, err := r.namespaceGetter.GetNamespace(ctx, req.Namespace)
		if err == nil && schema != nil {
			if relDef, ok := schema.Relations[req.Relation]; ok {
				rewrite = relDef.Rewrite
			}
		}
	}

	if rewrite != nil {
		return r.expandRewrite(ctx, req, rewrite, visitedCopy, depth)
	}

	return r.expandThis(ctx, req, visitedCopy, depth)
}

func (r *Resolver) expandRewrite(ctx context.Context, req core.ExpandRequest, rule *core.RewriteRule, visited map[string]bool, depth int) (*core.ExpandTree, error) {
	switch rule.Type {
	case "this", "":
		return r.expandThis(ctx, req, visited, depth)
	case "computed_userset":
		subReq := req
		subReq.Relation = rule.Relation
		return r.expandInternal(ctx, subReq, visited, depth+1)
	case "tuple_to_userset":
		tuples, err := r.repo.ReadTuples(ctx, core.Tuple{
			Namespace: req.Namespace,
			Object:    req.Object,
			Relation:  rule.TuplesetRelation,
		})
		if err != nil {
			return nil, err
		}
		node := &core.ExpandTree{Type: "userset"}
		for _, t := range tuples {
			subReq := core.ExpandRequest{
				Namespace: t.SubjectNamespace,
				Object:    t.SubjectObject,
				Relation:  rule.ComputedRelation,
			}
			child, err := r.expandInternal(ctx, subReq, visited, depth+1)
			if err == nil && child != nil {
				node.Children = append(node.Children, child)
			}
		}
		return node, nil
	case "union":
		node := &core.ExpandTree{Type: "union"}
		for _, child := range rule.Children {
			ch, err := r.expandRewrite(ctx, req, child, visited, depth+1)
			if err == nil && ch != nil {
				node.Children = append(node.Children, ch)
			}
		}
		return node, nil
	case "intersection":
		node := &core.ExpandTree{Type: "intersection"}
		for _, child := range rule.Children {
			ch, err := r.expandRewrite(ctx, req, child, visited, depth+1)
			if err == nil && ch != nil {
				node.Children = append(node.Children, ch)
			}
		}
		return node, nil
	default:
		return &core.ExpandTree{Type: "leaf"}, nil
	}
}

func (r *Resolver) expandThis(ctx context.Context, req core.ExpandRequest, visited map[string]bool, depth int) (*core.ExpandTree, error) {
	tuples, err := r.repo.ReadTuples(ctx, core.Tuple{
		Namespace: req.Namespace,
		Object:    req.Object,
		Relation:  req.Relation,
	})
	if err != nil {
		return nil, err
	}

	node := &core.ExpandTree{Type: "union"}
	for _, t := range tuples {
		tCopy := t
		if t.SubjectRelation != "" {
			subReq := core.ExpandRequest{
				Namespace: t.SubjectNamespace,
				Object:    t.SubjectObject,
				Relation:  t.SubjectRelation,
			}
			child, err := r.expandInternal(ctx, subReq, visited, depth+1)
			if err == nil && child != nil {
				node.Children = append(node.Children, &core.ExpandTree{
					Type:     "userset",
					Tuple:    &tCopy,
					Children: []*core.ExpandTree{child},
				})
			}
		} else {
			node.Children = append(node.Children, &core.ExpandTree{
				Type:  "leaf",
				Tuple: &tCopy,
			})
		}
	}
	return node, nil
}

// LookupSubjects finds all subject identities that possess relation to object
func (r *Resolver) LookupSubjects(ctx context.Context, req core.LookupSubjectsRequest) ([]core.Tuple, error) {
	tree, err := r.Expand(ctx, core.ExpandRequest{
		Namespace: req.Namespace,
		Object:    req.Object,
		Relation:  req.Relation,
	})
	if err != nil {
		return nil, err
	}

	seen := make(map[string]bool)
	var subjects []core.Tuple

	var collect func(node *core.ExpandTree)
	collect = func(node *core.ExpandTree) {
		if node == nil {
			return
		}
		if node.Type == "leaf" && node.Tuple != nil {
			key := fmt.Sprintf("%s:%s#%s", node.Tuple.SubjectNamespace, node.Tuple.SubjectObject, node.Tuple.SubjectRelation)
			if !seen[key] {
				seen[key] = true
				subjects = append(subjects, *node.Tuple)
			}
		}
		for _, child := range node.Children {
			collect(child)
		}
	}

	collect(tree)
	return subjects, nil
}

// LookupResources returns all resource IDs in a namespace that a subject can access with a given relation
func (r *Resolver) LookupResources(ctx context.Context, req core.LookupResourcesRequest) ([]string, error) {
	allTuples, err := r.repo.QueryTuples(ctx, core.Tuple{Namespace: req.Namespace})
	if err != nil {
		return nil, fmt.Errorf("failed to query tuples for namespace %q: %w", req.Namespace, err)
	}

	uniqueObjects := make(map[string]bool)
	for _, t := range allTuples {
		uniqueObjects[t.Object] = true
	}

	var results []string
	var mu sync.Mutex
	var wg sync.WaitGroup

	for obj := range uniqueObjects {
		wg.Add(1)
		go func(objectID string) {
			defer wg.Done()

			checkReq := core.CheckRequest{
				Namespace: req.Namespace,
				Object:    objectID,
				Relation:  req.Relation,
				Subject: core.Tuple{
					Namespace: req.SubjectNamespace,
					Object:    req.SubjectObject,
					Relation:  req.SubjectRelation,
				},
			}

			res, err := r.Check(ctx, checkReq)
			if err == nil && res.Allowed {
				mu.Lock()
				results = append(results, objectID)
				mu.Unlock()
			}
		}(obj)
	}

	wg.Wait()
	return results, nil
}
