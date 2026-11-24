package caveat

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/google/cel-go/cel"
)

const (
	defaultCostLimit                  = uint64(100000)
	defaultInterruptCheckFrequency    = uint(100)
	defaultParserRecursionLimit       = 64
	defaultMaxExpressionCodePointSize = 16384
	maxExprCacheSize                  = 1024
)

// CaveatGetter is an optional source for fetching caveat CEL expressions by name.
type CaveatGetter interface {
	GetCaveatExpression(ctx context.Context, name string) (string, error)
}

// Option configures a CELEvaluator.
type Option func(*CELEvaluator)

// WithCaveatGetter sets the caveat expression source.
func WithCaveatGetter(getter CaveatGetter) Option {
	return func(c *CELEvaluator) {
		c.getter = getter
	}
}

// CELEvaluator is the concrete implementation of core.CaveatEvaluator
type CELEvaluator struct {
	env     *cel.Env
	program cel.Program
	getter  CaveatGetter

	mu         sync.RWMutex
	exprCache  map[string]cel.Program
	exprOrder  []string
	namedCache map[string]cel.Program
}

// NewCELEvaluator initializes the CEL environment with parser and runtime DoS limits.
func NewCELEvaluator(opts ...Option) (*CELEvaluator, error) {
	env, err := cel.NewEnv(
		cel.ParserRecursionLimit(defaultParserRecursionLimit),
		cel.ParserExpressionSizeLimit(defaultMaxExpressionCodePointSize),
		cel.Variable("ctx", cel.MapType(cel.StringType, cel.DynType)),
		cel.Variable("caveat", cel.MapType(cel.StringType, cel.DynType)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create CEL env: %w", err)
	}

	eval := &CELEvaluator{
		env:        env,
		exprCache:  make(map[string]cel.Program),
		exprOrder:  make([]string, 0, maxExprCacheSize),
		namedCache: make(map[string]cel.Program),
	}

	for _, opt := range opts {
		opt(eval)
	}

	return eval, nil
}

func (c *CELEvaluator) compileProgram(expression string) (cel.Program, error) {
	if strings.TrimSpace(expression) == "" {
		return nil, errors.New("caveat expression cannot be empty")
	}

	ast, issues := c.env.Compile(expression)
	if issues != nil && issues.Err() != nil {
		return nil, fmt.Errorf("compile error: %w", issues.Err())
	}

	if ast.OutputType() != cel.BoolType {
		return nil, fmt.Errorf("expected boolean return type, got %v", ast.OutputType())
	}

	prg, err := c.env.Program(
		ast,
		cel.CostLimit(defaultCostLimit),
		cel.InterruptCheckFrequency(defaultInterruptCheckFrequency),
	)
	if err != nil {
		return nil, fmt.Errorf("program creation error: %w", err)
	}

	return prg, nil
}

// Validate checks whether expression is syntactically valid and returns a boolean type.
func (c *CELEvaluator) Validate(expression string) error {
	_, err := c.compileProgram(expression)
	return err
}

// Compile compiles the string expression into an AST and stores it in the
// expression cache (separate from named caveat definitions).
func (c *CELEvaluator) Compile(expression string) error {
	c.mu.RLock()
	cached, ok := c.exprCache[expression]
	c.mu.RUnlock()
	if ok {
		c.mu.Lock()
		c.program = cached
		c.mu.Unlock()
		return nil
	}

	prg, err := c.compileProgram(expression)
	if err != nil {
		return err
	}

	c.mu.Lock()
	c.program = prg
	if _, exists := c.exprCache[expression]; !exists {
		if len(c.exprCache) >= maxExprCacheSize && len(c.exprOrder) > 0 {
			oldest := c.exprOrder[0]
			c.exprOrder = c.exprOrder[1:]
			delete(c.exprCache, oldest)
		}
		c.exprCache[expression] = prg
		c.exprOrder = append(c.exprOrder, expression)
	}
	c.mu.Unlock()

	return nil
}

// Invalidate removes a named caveat from the compiled named caveat cache so
// updates or deletions take effect immediately.
func (c *CELEvaluator) Invalidate(name string) {
	c.mu.Lock()
	delete(c.namedCache, name)
	c.mu.Unlock()
}

// Evaluate runs the compiled AST against the provided contexts.
func (c *CELEvaluator) Evaluate(reqCtx map[string]interface{}, caveatCtx map[string]interface{}) (bool, error) {
	return c.EvaluateContext(context.Background(), reqCtx, caveatCtx)
}

// EvaluateContext runs the compiled AST against the provided contexts with cancellation support.
func (c *CELEvaluator) EvaluateContext(ctx context.Context, reqCtx map[string]interface{}, caveatCtx map[string]interface{}) (bool, error) {
	c.mu.RLock()
	prg := c.program
	c.mu.RUnlock()

	if prg == nil {
		return false, fmt.Errorf("no program compiled")
	}

	return c.evalProgram(ctx, prg, reqCtx, caveatCtx)
}

// EvaluateByName loads, compiles (if not cached), and evaluates a caveat by name.
func (c *CELEvaluator) EvaluateByName(ctx context.Context, name string, reqCtx map[string]interface{}, caveatCtx map[string]interface{}) (bool, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	c.mu.RLock()
	prg, ok := c.namedCache[name]
	c.mu.RUnlock()

	if !ok {
		if c.getter == nil {
			return false, fmt.Errorf("caveat getter not configured and caveat %q not in cache", name)
		}

		expr, err := c.getter.GetCaveatExpression(ctx, name)
		if err != nil {
			return false, fmt.Errorf("failed to get caveat %q: %w", name, err)
		}

		compiled, err := c.compileProgram(expr)
		if err != nil {
			return false, fmt.Errorf("invalid caveat %q expression: %w", name, err)
		}

		c.mu.Lock()
		c.namedCache[name] = compiled
		prg = compiled
		c.mu.Unlock()
	}

	return c.evalProgram(ctx, prg, reqCtx, caveatCtx)
}

func (c *CELEvaluator) evalProgram(ctx context.Context, prg cel.Program, reqCtx map[string]interface{}, caveatCtx map[string]interface{}) (bool, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if reqCtx == nil {
		reqCtx = map[string]interface{}{}
	}
	if caveatCtx == nil {
		caveatCtx = map[string]interface{}{}
	}

	evalMap := map[string]interface{}{
		"ctx":    reqCtx,
		"caveat": caveatCtx,
	}

	out, _, err := prg.ContextEval(ctx, evalMap)
	if err != nil {
		return false, fmt.Errorf("evaluation error: %w", err)
	}

	if out.Type() != cel.BoolType {
		return false, fmt.Errorf("expected boolean result, got %v", out.Type())
	}

	return out.Value().(bool), nil
}
