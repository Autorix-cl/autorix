package engine

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/autorix/themis/internal/core"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	exprpb "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
)

const (
	defaultCacheSize                  = 1024
	defaultCostLimit                  = uint64(100000)
	defaultInterruptCheckFrequency    = uint(100)
	defaultParserRecursionLimit       = 64
	defaultMaxExpressionCodePointSize = 16384
)

// cachedProgram holds a precompiled CEL program ready for evaluation.
type cachedProgram struct {
	program cel.Program
}

// CELEvaluator implements core.PolicyEvaluator using Google CEL.
// It maintains a thread-safe LRU cache of compiled programs keyed
// by expression string to avoid recompilation on every request.
type CELEvaluator struct {
	env   *cel.Env
	mu    sync.RWMutex
	cache map[string]*cachedProgram
	order []string // FIFO/LRU eviction order (oldest first)
	max   int
}

// NewCELEvaluator creates a CEL environment with standard operators and
// functions suitable for business rule evaluation.
func NewCELEvaluator() (*CELEvaluator, error) {
	return NewCELEvaluatorWithSize(defaultCacheSize)
}

// NewCELEvaluatorWithSize creates a CEL evaluator with a specific cache capacity.
// Non-positive capacities fall back to defaultCacheSize.
func NewCELEvaluatorWithSize(maxSize int) (*CELEvaluator, error) {
	if maxSize <= 0 {
		maxSize = defaultCacheSize
	}

	// Build the CEL environment once with parser DoS limits. We use DynType
	// for the payload so expressions can reference dynamic fields safely.
	env, err := cel.NewEnv(
		cel.ParserRecursionLimit(defaultParserRecursionLimit),
		cel.ParserExpressionSizeLimit(defaultMaxExpressionCodePointSize),
		cel.Variable("payload", cel.MapType(cel.StringType, cel.DynType)),
		cel.Variable("request", cel.MapType(cel.StringType, cel.DynType)),
		cel.Variable("auth", cel.MapType(cel.StringType, cel.DynType)),
		cel.Variable("resource", cel.MapType(cel.StringType, cel.DynType)),
		cel.Variable("context", cel.MapType(cel.StringType, cel.DynType)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create CEL environment: %w", err)
	}

	return &CELEvaluator{
		env:   env,
		cache: make(map[string]*cachedProgram, maxSize),
		order: make([]string, 0, maxSize),
		max:   maxSize,
	}, nil
}

// Compile checks that the expression is syntactically valid, returns bool, and caches it.
func (e *CELEvaluator) Compile(expression string) error {
	_, err := e.getOrCompile(expression)
	return err
}

// Evaluate compiles (or retrieves from cache) the given CEL expression
// and evaluates it against the provided payload.
func (e *CELEvaluator) Evaluate(expression string, payload map[string]interface{}) (bool, error) {
	return e.EvaluateContext(context.Background(), expression, payload)
}

// EvaluateContext compiles (or retrieves from cache) the given CEL expression
// and evaluates it using ContextEval so context cancellation/deadlines halt
// runaway evaluations.
func (e *CELEvaluator) EvaluateContext(ctx context.Context, expression string, payload map[string]interface{}) (bool, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	prog, err := e.getOrCompile(expression)
	if err != nil {
		return false, err
	}
	return e.evalProgram(ctx, prog.program, payload)
}

// EvaluateUncached compiles and evaluates a one-off expression without
// inserting it into the shared production program cache (preventing cache
// thrashing and lock contention during DryRun calls).
func (e *CELEvaluator) EvaluateUncached(ctx context.Context, expression string, payload map[string]interface{}) (bool, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	prog, err := e.compileProgram(expression)
	if err != nil {
		return false, err
	}
	return e.evalProgram(ctx, prog, payload)
}

func (e *CELEvaluator) evalProgram(ctx context.Context, prog cel.Program, payload map[string]interface{}) (bool, error) {
	if payload == nil {
		payload = make(map[string]interface{})
	}

	// Build activation with the payload and top-level aliases
	authMap, _ := payload["auth"].(map[string]interface{})
	if authMap == nil {
		authMap = make(map[string]interface{})
	}
	resourceMap, _ := payload["resource"].(map[string]interface{})
	if resourceMap == nil {
		resourceMap = make(map[string]interface{})
	}
	reqMap, _ := payload["request"].(map[string]interface{})
	if reqMap == nil {
		reqMap = payload
	}

	activation := map[string]interface{}{
		"payload":  payload,
		"request":  reqMap,
		"auth":     authMap,
		"resource": resourceMap,
		"context":  payload,
	}

	out, _, err := prog.ContextEval(ctx, activation)
	if err != nil {
		return false, fmt.Errorf("CEL evaluation error: %w", err)
	}

	// The expression MUST evaluate to a boolean
	if out.Type() != types.BoolType {
		return false, fmt.Errorf("expression must evaluate to bool, got %s", out.Type())
	}

	return out.Value().(bool), nil
}

func (e *CELEvaluator) compileProgram(expression string) (cel.Program, error) {
	ast, issues := e.env.Compile(expression)
	if issues != nil && issues.Err() != nil {
		return nil, fmt.Errorf("CEL compilation error: %w", issues.Err())
	}

	if ast.OutputType() != cel.BoolType {
		return nil, fmt.Errorf("expression must evaluate to bool, got %s", ast.OutputType())
	}

	prog, err := e.env.Program(
		ast,
		cel.CostLimit(defaultCostLimit),
		cel.InterruptCheckFrequency(defaultInterruptCheckFrequency),
	)
	if err != nil {
		return nil, fmt.Errorf("CEL program creation error: %w", err)
	}
	return prog, nil
}

// getOrCompile retrieves a cached program or compiles and caches a new one.
func (e *CELEvaluator) getOrCompile(expression string) (*cachedProgram, error) {
	// Fast path: read lock
	e.mu.RLock()
	if cached, ok := e.cache[expression]; ok {
		e.mu.RUnlock()
		return cached, nil
	}
	e.mu.RUnlock()

	// Compile outside write lock to avoid blocking readers during compilation
	prog, err := e.compileProgram(expression)
	if err != nil {
		return nil, err
	}

	// Slow path: store in cache
	e.mu.Lock()
	defer e.mu.Unlock()

	// Double-check after acquiring write lock
	if cached, ok := e.cache[expression]; ok {
		return cached, nil
	}

	cp := &cachedProgram{program: prog}

	// Evict oldest if at capacity
	if len(e.cache) >= e.max && len(e.order) > 0 {
		oldest := e.order[0]
		e.order = e.order[1:]
		delete(e.cache, oldest)
	}

	e.cache[expression] = cp
	e.order = append(e.order, expression)

	return cp, nil
}

// CacheSize returns the current number of cached programs.
func (e *CELEvaluator) CacheSize() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return len(e.cache)
}

// Validate parses and type-checks a CEL expression, extracting variables and detailed errors.
func (e *CELEvaluator) Validate(expression string) (*core.ValidationResult, error) {
	ast, issues := e.env.Compile(expression)
	if issues != nil && issues.Err() != nil {
		var validationErrors []core.ValidationError
		for _, ce := range issues.Errors() {
			validationErrors = append(validationErrors, core.ValidationError{
				Line:    ce.Location.Line(),
				Column:  ce.Location.Column(),
				Message: ce.Message,
			})
		}
		if len(validationErrors) == 0 {
			validationErrors = append(validationErrors, core.ValidationError{
				Message: issues.Err().Error(),
			})
		}
		return &core.ValidationResult{
			Valid:     false,
			Variables: []string{},
			Errors:    validationErrors,
		}, nil
	}

	if ast.OutputType() != cel.BoolType {
		return &core.ValidationResult{
			Valid:     false,
			Variables: []string{},
			Errors: []core.ValidationError{
				{
					Message: fmt.Sprintf("expression must evaluate to bool, got %s", ast.OutputType()),
				},
			},
		}, nil
	}

	// Extract referenced variables from parsed AST
	varsSet := make(map[string]struct{})
	parsedExpr, err := cel.AstToParsedExpr(ast)
	if err == nil && parsedExpr != nil && parsedExpr.GetExpr() != nil {
		extractVariables(parsedExpr.GetExpr(), varsSet)
	}

	vars := make([]string, 0, len(varsSet))
	for v := range varsSet {
		vars = append(vars, v)
	}
	sort.Strings(vars)

	return &core.ValidationResult{
		Valid:     true,
		Variables: vars,
		Errors:    nil,
	}, nil
}

func extractVariables(expr *exprpb.Expr, vars map[string]struct{}) {
	if expr == nil {
		return
	}

	switch k := expr.GetExprKind().(type) {
	case *exprpb.Expr_IdentExpr:
		if k.IdentExpr != nil && k.IdentExpr.GetName() != "" {
			vars[k.IdentExpr.GetName()] = struct{}{}
		}
	case *exprpb.Expr_SelectExpr:
		if k.SelectExpr != nil {
			extractVariables(k.SelectExpr.GetOperand(), vars)
		}
	case *exprpb.Expr_CallExpr:
		if k.CallExpr != nil {
			if k.CallExpr.GetTarget() != nil {
				extractVariables(k.CallExpr.GetTarget(), vars)
			}
			for _, arg := range k.CallExpr.GetArgs() {
				extractVariables(arg, vars)
			}
		}
	case *exprpb.Expr_ListExpr:
		if k.ListExpr != nil {
			for _, elem := range k.ListExpr.GetElements() {
				extractVariables(elem, vars)
			}
		}
	case *exprpb.Expr_StructExpr:
		if k.StructExpr != nil {
			for _, entry := range k.StructExpr.GetEntries() {
				extractVariables(entry.GetValue(), vars)
			}
		}
	case *exprpb.Expr_ComprehensionExpr:
		if k.ComprehensionExpr != nil {
			extractVariables(k.ComprehensionExpr.GetIterRange(), vars)
			extractVariables(k.ComprehensionExpr.GetAccuInit(), vars)
			extractVariables(k.ComprehensionExpr.GetLoopStep(), vars)
			extractVariables(k.ComprehensionExpr.GetLoopCondition(), vars)
			extractVariables(k.ComprehensionExpr.GetResult(), vars)
		}
	}
}
