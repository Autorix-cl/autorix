package graph

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/autorix/nexus/internal/core"
	"github.com/autorix/platform/cache"
	"github.com/autorix/platform/zookie"
)

const (
	maxCheckDepth            = 25
	maxExpandDepth           = 20
	maxLookupWorkers         = 16
	maxLookupResourceObjects = 1000
	cacheEpochKey            = "nexus:cache:epoch"
)

var (
	ErrInvalidSnapToken = errors.New("invalid snap_token")
	ErrMaxDepthExceeded = errors.New("maximum graph traversal depth exceeded")
)

// Repository is the interface required by the graph engine to fetch and query tuples
type Repository interface {
	ReadTuples(ctx context.Context, filter core.Tuple) ([]core.Tuple, error)
	QueryTuples(ctx context.Context, filter core.Tuple) ([]core.Tuple, error)
}

// LSNReader is an optional interface to read the current database LSN
type LSNReader interface {
	CurrentLSN(ctx context.Context) (uint64, error)
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

// WithCache sets the distributed/local cache for authorization decisions
func WithCache(c cache.Cache) Option {
	return func(r *Resolver) {
		r.cache = c
	}
}

// Resolver implements core.GraphEngine using bounded graph traversal
type Resolver struct {
	repo            Repository
	caveatEvaluator core.CaveatEvaluator
	namespaceGetter NamespaceGetter
	cache           cache.Cache
	epoch           atomic.Uint64
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

// InvalidateCache advances the cache generation epoch so all previously cached
// Check decisions (including transitive/rewritten paths) are bypassed immediately.
func (r *Resolver) InvalidateCache(ctx context.Context) {
	next := r.epoch.Add(1)
	if r.cache != nil {
		if ctx == nil {
			ctx = context.Background()
		}
		_ = r.cache.Set(ctx, cacheEpochKey, []byte(strconv.FormatUint(next, 10)), 24*time.Hour)
		_ = r.cache.PublishInvalidation(ctx, "nexus:invalidate", cacheEpochKey)
	}
}

func (r *Resolver) currentEpoch(ctx context.Context) uint64 {
	local := r.epoch.Load()
	if r.cache == nil {
		return local
	}
	val, err := r.cache.Get(ctx, cacheEpochKey)
	if err != nil {
		return local
	}
	remote, parseErr := strconv.ParseUint(strings.TrimSpace(string(val)), 10, 64)
	if parseErr != nil {
		return local
	}
	if remote > local {
		r.epoch.Store(remote)
		return remote
	}
	return local
}

// Check evaluates whether a subject has a relation to an object
func (r *Resolver) Check(ctx context.Context, req core.CheckRequest) (core.CheckResult, error) {
	start := time.Now()

	// Validate requested consistency snapshot token (Zookie) strictly
	var requestedZk *zookie.Token
	if req.SnapToken != "" {
		parsed, err := zookie.Parse(req.SnapToken)
		if err != nil {
			return core.CheckResult{}, fmt.Errorf("%w: %v", ErrInvalidSnapToken, err)
		}
		requestedZk = &parsed
	}

	// 1. Fast path: In-memory/Redis distributed cache lookup
	// Skip cache when Explain is requested or dynamic RequestContext is supplied.
	epoch := r.currentEpoch(ctx)
	cacheKey := fmt.Sprintf("chk:e%d:%s:%s#%s@%s:%s#%s",
		epoch,
		url.QueryEscape(req.Namespace), url.QueryEscape(req.Object), url.QueryEscape(req.Relation),
		url.QueryEscape(req.Subject.Namespace), url.QueryEscape(req.Subject.Object), url.QueryEscape(req.Subject.Relation))

	if r.cache != nil && len(req.RequestContext) == 0 && !req.Explain {
		if val, err := r.cache.Get(ctx, cacheKey); err == nil {
			strVal := string(val)
			parts := strings.SplitN(strVal, ":", 2)
			allowed := parts[0] == "1"
			var cachedToken zookie.Token
			if len(parts) > 1 {
				cachedToken, _ = zookie.Parse(parts[1])
			}

			// Verify causal consistency against requested Zookie (JD-01)
			isFresh := true
			if requestedZk != nil {
				isFresh = requestedZk.IsAtLeastFresh(cachedToken)
			}

			if isFresh {
				decision := "deny"
				if allowed {
					decision = "allow"
				}
				nexusCheckTotal.WithLabelValues(decision).Inc()
				return core.CheckResult{
					Allowed:   allowed,
					Reason:    "cached authorization decision",
					SnapToken: cachedToken.String(),
				}, nil
			}
		}
	}

	visited := make(map[string]bool)
	caveatEvaluated := false
	res, err := r.checkInternal(ctx, req, visited, 0, &caveatEvaluated)
	duration := time.Since(start).Seconds()

	nexusCheckDuration.Observe(duration)
	decision := "deny"
	if err == nil && res.Allowed {
		decision = "allow"
	}
	nexusCheckTotal.WithLabelValues(decision).Inc()

	if err != nil {
		return core.CheckResult{}, err
	}

	// Mint evaluated snapshot Zookie: use database LSN if repository supports it (JD-01)
	var evalToken zookie.Token
	if lsnReader, ok := r.repo.(LSNReader); ok {
		lsn, lsnErr := lsnReader.CurrentLSN(ctx)
		if lsnErr != nil {
			return core.CheckResult{}, fmt.Errorf("failed to determine database snapshot lsn: %w", lsnErr)
		}
		if lsn > 0 {
			evalToken = zookie.NewWithLSN(lsn)
		}
	}
	if evalToken.Version == "" {
		evalToken = zookie.Now()
	}
	res.SnapToken = evalToken.String()

	// 2. Cache successful non-caveat, non-explain evaluation with serialized Zookie token
	if r.cache != nil && len(req.RequestContext) == 0 && !req.Explain && !caveatEvaluated {
		decisionCode := "0"
		if res.Allowed {
			decisionCode = "1"
		}
		val := []byte(fmt.Sprintf("%s:%s", decisionCode, evalToken.String()))
		_ = r.cache.Set(ctx, cacheKey, val, 30*time.Second)
	}

	return res, nil
}

func (r *Resolver) checkInternal(ctx context.Context, req core.CheckRequest, visited map[string]bool, depth int, caveatEvaluated *bool) (core.CheckResult, error) {
	if err := ctx.Err(); err != nil {
		return core.CheckResult{}, err
	}
	if depth > maxCheckDepth {
		return core.CheckResult{
			Allowed: false,
			Reason:  "maximum traversal depth exceeded",
		}, ErrMaxDepthExceeded
	}

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
		if err != nil {
			return core.CheckResult{}, fmt.Errorf("failed to get namespace %q: %w", req.Namespace, err)
		}
		if schema != nil {
			if relDef, ok := schema.Relations[req.Relation]; ok {
				rewrite = relDef.Rewrite
			}
		}
	}

	if rewrite != nil {
		return r.evaluateRewrite(ctx, req, rewrite, visitedCopy, depth, caveatEvaluated)
	}

	// Default to direct tuple matching ("this")
	return r.evaluateThis(ctx, req, visitedCopy, depth, caveatEvaluated)
}

func (r *Resolver) evaluateRewrite(ctx context.Context, req core.CheckRequest, rule *core.RewriteRule, visited map[string]bool, depth int, caveatEvaluated *bool) (core.CheckResult, error) {
	if err := ctx.Err(); err != nil {
		return core.CheckResult{}, err
	}
	if depth > maxCheckDepth {
		return core.CheckResult{
			Allowed: false,
			Reason:  "maximum traversal depth exceeded",
		}, ErrMaxDepthExceeded
	}

	traceNode := &core.DecisionNode{
		Namespace:   req.Namespace,
		Object:      req.Object,
		Relation:    req.Relation,
		Subject:     req.Subject,
		RewriteType: rule.Type,
	}

	switch rule.Type {
	case "this", "":
		res, err := r.evaluateThis(ctx, req, visited, depth, caveatEvaluated)
		if err != nil {
			return core.CheckResult{}, err
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
		res, err := r.checkInternal(ctx, subReq, visited, depth+1, caveatEvaluated)
		if err != nil {
			return core.CheckResult{}, err
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
			if t.CaveatName != "" {
				*caveatEvaluated = true
				allowed, cavErr := r.evalCaveat(ctx, t.CaveatName, req.RequestContext, t.CaveatContext)
				if cavErr != nil || !allowed {
					continue
				}
			}

			subReq := core.CheckRequest{
				Namespace:      t.SubjectNamespace,
				Object:         t.SubjectObject,
				Relation:       rule.ComputedRelation,
				Subject:        req.Subject,
				RequestContext: req.RequestContext,
				Explain:        req.Explain,
			}
			subRes, err := r.checkInternal(ctx, subReq, visited, depth+1, caveatEvaluated)
			if err != nil {
				return core.CheckResult{}, err
			}
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

		traceNode.Allowed = false
		traceNode.Reason = "no matching tuple_to_userset path"
		return core.CheckResult{
			Allowed: false,
			Reason:  traceNode.Reason,
			Trace:   traceNode,
		}, nil

	case "union":
		var anyAllowed bool
		for _, child := range rule.Children {
			childRes, err := r.evaluateRewrite(ctx, req, child, visited, depth+1, caveatEvaluated)
			if err != nil {
				return core.CheckResult{}, err
			}
			if req.Explain && childRes.Trace != nil {
				traceNode.Children = append(traceNode.Children, childRes.Trace)
			}
			if childRes.Allowed {
				anyAllowed = true
				if !req.Explain {
					break
				}
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
			childRes, err := r.evaluateRewrite(ctx, req, child, visited, depth+1, caveatEvaluated)
			if err != nil {
				return core.CheckResult{}, err
			}
			if !childRes.Allowed {
				allAllowed = false
			}
			if req.Explain && childRes.Trace != nil {
				traceNode.Children = append(traceNode.Children, childRes.Trace)
			}
			if !allAllowed && !req.Explain {
				break
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

func (r *Resolver) evaluateThis(ctx context.Context, req core.CheckRequest, visited map[string]bool, depth int, caveatEvaluated *bool) (core.CheckResult, error) {
	if err := ctx.Err(); err != nil {
		return core.CheckResult{}, err
	}

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
		if err := ctx.Err(); err != nil {
			return core.CheckResult{}, err
		}

		childTrace := &core.DecisionNode{
			Namespace: tuple.SubjectNamespace,
			Object:    tuple.SubjectObject,
			Relation:  tuple.SubjectRelation,
			Subject:   req.Subject,
		}

		// 1. Evaluate Caveat if present
		if tuple.CaveatName != "" {
			*caveatEvaluated = true
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

			subRes, err := r.checkInternal(ctx, subReq, visited, depth+1, caveatEvaluated)
			if err != nil {
				return core.CheckResult{}, err
			}
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
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if depth > maxExpandDepth {
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
		if err != nil {
			return nil, fmt.Errorf("failed to get namespace %q: %w", req.Namespace, err)
		}
		if schema != nil {
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
	if err := ctx.Err(); err != nil {
		return nil, err
	}

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
			tCopy := t
			subReq := core.ExpandRequest{
				Namespace: t.SubjectNamespace,
				Object:    t.SubjectObject,
				Relation:  rule.ComputedRelation,
			}
			child, err := r.expandInternal(ctx, subReq, visited, depth+1)
			if err != nil {
				return nil, err
			}
			if child != nil {
				if tCopy.CaveatName != "" {
					node.Children = append(node.Children, &core.ExpandTree{
						Type:     "userset",
						Tuple:    &tCopy,
						Children: []*core.ExpandTree{child},
					})
				} else {
					node.Children = append(node.Children, child)
				}
			}
		}
		return node, nil
	case "union":
		node := &core.ExpandTree{Type: "union"}
		for _, child := range rule.Children {
			ch, err := r.expandRewrite(ctx, req, child, visited, depth+1)
			if err != nil {
				return nil, err
			}
			if ch != nil {
				node.Children = append(node.Children, ch)
			}
		}
		return node, nil
	case "intersection":
		node := &core.ExpandTree{Type: "intersection"}
		for _, child := range rule.Children {
			ch, err := r.expandRewrite(ctx, req, child, visited, depth+1)
			if err != nil {
				return nil, err
			}
			if ch != nil {
				node.Children = append(node.Children, ch)
			}
		}
		return node, nil
	default:
		return &core.ExpandTree{Type: "leaf"}, nil
	}
}

func (r *Resolver) expandThis(ctx context.Context, req core.ExpandRequest, visited map[string]bool, depth int) (*core.ExpandTree, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

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
			if err != nil {
				return nil, err
			}
			if child != nil {
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

// LookupSubjects finds all subject identities that possess relation to object,
// accurately evaluating union/intersection set algebra and caveat restrictions.
func (r *Resolver) LookupSubjects(ctx context.Context, req core.LookupSubjectsRequest) ([]core.Tuple, error) {
	tree, err := r.Expand(ctx, core.ExpandRequest(req))
	if err != nil {
		return nil, err
	}

	subjectSet, err := r.collectSubjects(ctx, tree)
	if err != nil {
		return nil, err
	}

	keys := make([]string, 0, len(subjectSet))
	for k := range subjectSet {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	subjects := make([]core.Tuple, 0, len(keys))
	for _, k := range keys {
		subjects = append(subjects, subjectSet[k])
	}
	return subjects, nil
}

func (r *Resolver) collectSubjects(ctx context.Context, node *core.ExpandTree) (map[string]core.Tuple, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if node == nil {
		return map[string]core.Tuple{}, nil
	}

	switch node.Type {
	case "leaf":
		if node.Tuple == nil {
			return map[string]core.Tuple{}, nil
		}
		if node.Tuple.CaveatName != "" {
			allowed, err := r.evalCaveat(ctx, node.Tuple.CaveatName, nil, node.Tuple.CaveatContext)
			if err != nil || !allowed {
				return map[string]core.Tuple{}, nil
			}
		}
		key := subjectTupleKey(*node.Tuple)
		return map[string]core.Tuple{key: *node.Tuple}, nil

	case "userset":
		if node.Tuple != nil && node.Tuple.CaveatName != "" {
			allowed, err := r.evalCaveat(ctx, node.Tuple.CaveatName, nil, node.Tuple.CaveatContext)
			if err != nil || !allowed {
				return map[string]core.Tuple{}, nil
			}
		}
		unionSet := make(map[string]core.Tuple)
		for _, child := range node.Children {
			childSet, err := r.collectSubjects(ctx, child)
			if err != nil {
				return nil, err
			}
			for k, v := range childSet {
				if _, exists := unionSet[k]; !exists {
					unionSet[k] = v
				}
			}
		}
		return unionSet, nil

	case "union":
		unionSet := make(map[string]core.Tuple)
		for _, child := range node.Children {
			childSet, err := r.collectSubjects(ctx, child)
			if err != nil {
				return nil, err
			}
			for k, v := range childSet {
				if _, exists := unionSet[k]; !exists {
					unionSet[k] = v
				}
			}
		}
		return unionSet, nil

	case "intersection":
		if len(node.Children) == 0 {
			return map[string]core.Tuple{}, nil
		}
		acc, err := r.collectSubjects(ctx, node.Children[0])
		if err != nil {
			return nil, err
		}
		for _, child := range node.Children[1:] {
			if len(acc) == 0 {
				break
			}
			childSet, err := r.collectSubjects(ctx, child)
			if err != nil {
				return nil, err
			}
			for k := range acc {
				if _, ok := childSet[k]; !ok {
					delete(acc, k)
				}
			}
		}
		return acc, nil

	default:
		return map[string]core.Tuple{}, nil
	}
}

func subjectTupleKey(t core.Tuple) string {
	return fmt.Sprintf("%s:%s#%s", url.QueryEscape(t.SubjectNamespace), url.QueryEscape(t.SubjectObject), url.QueryEscape(t.SubjectRelation))
}

// LookupResources returns all resource IDs in a namespace that a subject can access
// with a given relation, using a bounded worker pool and maximum candidate limit.
func (r *Resolver) LookupResources(ctx context.Context, req core.LookupResourcesRequest) ([]string, error) {
	allTuples, err := r.repo.QueryTuples(ctx, core.Tuple{Namespace: req.Namespace})
	if err != nil {
		return nil, fmt.Errorf("failed to query tuples for namespace %q: %w", req.Namespace, err)
	}

	uniqueObjects := make(map[string]struct{})
	for _, t := range allTuples {
		uniqueObjects[t.Object] = struct{}{}
	}

	objects := make([]string, 0, len(uniqueObjects))
	for obj := range uniqueObjects {
		objects = append(objects, obj)
	}
	sort.Strings(objects)
	if len(objects) > maxLookupResourceObjects {
		objects = objects[:maxLookupResourceObjects]
	}

	results := make([]string, 0)
	if len(objects) == 0 {
		return results, nil
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var (
		mu       sync.Mutex
		wg       sync.WaitGroup
		errOnce  sync.Once
		firstErr error
	)
	sem := make(chan struct{}, maxLookupWorkers)

	for _, obj := range objects {
		if ctx.Err() != nil {
			errOnce.Do(func() { firstErr = ctx.Err() })
			break
		}
		select {
		case <-ctx.Done():
			errOnce.Do(func() { firstErr = ctx.Err() })
			continue
		case sem <- struct{}{}:
		}

		wg.Add(1)
		go func(objectID string) {
			defer wg.Done()
			defer func() { <-sem }()

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
			if err != nil {
				if !errors.Is(err, ErrMaxDepthExceeded) {
					errOnce.Do(func() {
						firstErr = err
						cancel()
					})
				}
				return
			}
			if res.Allowed {
				mu.Lock()
				results = append(results, objectID)
				mu.Unlock()
			}
		}(obj)
	}

	wg.Wait()
	if firstErr != nil {
		return nil, firstErr
	}
	sort.Strings(results)
	return results, nil
}
