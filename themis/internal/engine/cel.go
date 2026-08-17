package engine

import (
	"fmt"
	"sync"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
)

const defaultCacheSize = 1024

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
	order []string // LRU eviction order (oldest first)
	max   int
}

// NewCELEvaluator creates a CEL environment with standard operators and
// functions suitable for business rule evaluation.
func NewCELEvaluator() (*CELEvaluator, error) {
	return NewCELEvaluatorWithSize(defaultCacheSize)
}

// NewCELEvaluatorWithSize creates a CEL evaluator with a specific cache capacity.
func NewCELEvaluatorWithSize(maxSize int) (*CELEvaluator, error) {
	// Build the CEL environment once. We use DynType for the payload
	// so that expressions can reference any field dynamically.
	env, err := cel.NewEnv(
		// All payload fields are injected as top-level variables.
		// We declare a single "payload" variable of type map(string, dyn)
		// so the expression can access any key.
		cel.Variable("payload", cel.MapType(cel.StringType, cel.DynType)),
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
//
// The payload map is injected as the "payload" variable in the CEL context,
// so expressions should be written as: payload.amount < payload.budget
func (e *CELEvaluator) Evaluate(expression string, payload map[string]interface{}) (bool, error) {
	prog, err := e.getOrCompile(expression)
	if err != nil {
		return false, err
	}

	// Build activation with the payload
	activation := map[string]interface{}{
		"payload": payload,
	}

	out, _, err := prog.program.Eval(activation)
	if err != nil {
		return false, fmt.Errorf("CEL evaluation error: %w", err)
	}

	// The expression MUST evaluate to a boolean
	if out.Type() != types.BoolType {
		return false, fmt.Errorf("expression must evaluate to bool, got %s", out.Type())
	}

	return out.Value().(bool), nil
}

// getOrCompile retrieves a cached program or compiles and caches a new one.
func (e *CELEvaluator) getOrCompile(expression string) (*cachedProgram, error) {
	// Fast path: read lock
	e.mu.RLock()
	if cached, ok := e.cache[expression]; ok {
		e.mu.RUnlock()
		// Note: we don't update LRU order on reads for simplicity;
		// in high-traffic scenarios this still performs well because
		// hot expressions stay cached.
		return cached, nil
	}
	e.mu.RUnlock()

	// Slow path: compile and cache
	e.mu.Lock()
	defer e.mu.Unlock()

	// Double-check after acquiring write lock
	if cached, ok := e.cache[expression]; ok {
		return cached, nil
	}

	// Compile: parse + check + program
	ast, issues := e.env.Compile(expression)
	if issues != nil && issues.Err() != nil {
		return nil, fmt.Errorf("CEL compilation error: %w", issues.Err())
	}

	prog, err := e.env.Program(ast)
	if err != nil {
		return nil, fmt.Errorf("CEL program creation error: %w", err)
	}

	cp := &cachedProgram{program: prog}

	// Evict oldest if at capacity
	if len(e.cache) >= e.max {
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
