package caveat

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/google/cel-go/cel"
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

	mu    sync.RWMutex
	cache map[string]cel.Program
}

// NewCELEvaluator initializes the CEL environment
func NewCELEvaluator(opts ...Option) (*CELEvaluator, error) {
	env, err := cel.NewEnv(
		cel.Variable("ctx", cel.MapType(cel.StringType, cel.DynType)),
		cel.Variable("caveat", cel.MapType(cel.StringType, cel.DynType)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create CEL env: %w", err)
	}

	eval := &CELEvaluator{
		env:   env,
		cache: make(map[string]cel.Program),
	}

	for _, opt := range opts {
		opt(eval)
	}

	return eval, nil
}

// Validate checks whether expression is syntactically valid and returns a boolean type.
func (c *CELEvaluator) Validate(expression string) error {
	if strings.TrimSpace(expression) == "" {
		return errors.New("caveat expression cannot be empty")
	}

	ast, issues := c.env.Compile(expression)
	if issues != nil && issues.Err() != nil {
		return fmt.Errorf("compile error: %w", issues.Err())
	}

	if ast.OutputType() != cel.BoolType {
		return fmt.Errorf("expected boolean return type, got %v", ast.OutputType())
	}

	_, err := c.env.Program(ast)
	if err != nil {
		return fmt.Errorf("program creation error: %w", err)
	}

	return nil
}

// Compile compiles the string expression into an AST and caches it.
func (c *CELEvaluator) Compile(expression string) error {
	if err := c.Validate(expression); err != nil {
		return err
	}

	ast, _ := c.env.Compile(expression)
	prg, err := c.env.Program(ast)
	if err != nil {
		return fmt.Errorf("program creation error: %w", err)
	}

	c.mu.Lock()
	c.program = prg
	c.cache[expression] = prg
	c.mu.Unlock()

	return nil
}

// Evaluate runs the compiled AST against the provided contexts
func (c *CELEvaluator) Evaluate(reqCtx map[string]interface{}, caveatCtx map[string]interface{}) (bool, error) {
	c.mu.RLock()
	prg := c.program
	c.mu.RUnlock()

	if prg == nil {
		return false, fmt.Errorf("no program compiled")
	}

	return c.evalProgram(prg, reqCtx, caveatCtx)
}

// EvaluateByName loads, compiles (if not cached), and evaluates a caveat by name.
func (c *CELEvaluator) EvaluateByName(ctx context.Context, name string, reqCtx map[string]interface{}, caveatCtx map[string]interface{}) (bool, error) {
	c.mu.RLock()
	prg, ok := c.cache[name]
	c.mu.RUnlock()

	if !ok {
		if c.getter == nil {
			return false, fmt.Errorf("caveat getter not configured and caveat %q not in cache", name)
		}

		expr, err := c.getter.GetCaveatExpression(ctx, name)
		if err != nil {
			return false, fmt.Errorf("failed to get caveat %q: %w", name, err)
		}

		if err := c.Validate(expr); err != nil {
			return false, fmt.Errorf("invalid caveat %q expression: %w", name, err)
		}

		ast, _ := c.env.Compile(expr)
		compiled, err := c.env.Program(ast)
		if err != nil {
			return false, fmt.Errorf("program creation error for %q: %w", name, err)
		}

		c.mu.Lock()
		c.cache[name] = compiled
		prg = compiled
		c.mu.Unlock()
	}

	return c.evalProgram(prg, reqCtx, caveatCtx)
}

func (c *CELEvaluator) evalProgram(prg cel.Program, reqCtx map[string]interface{}, caveatCtx map[string]interface{}) (bool, error) {
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

	out, _, err := prg.Eval(evalMap)
	if err != nil {
		return false, fmt.Errorf("evaluation error: %w", err)
	}

	if out.Type() != cel.BoolType {
		return false, fmt.Errorf("expected boolean result, got %v", out.Type())
	}

	return out.Value().(bool), nil
}
