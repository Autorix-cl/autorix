package caveat

import (
	"fmt"

	"github.com/google/cel-go/cel"
)

// CELEvaluator is the concrete implementation of core.CaveatEvaluator
type CELEvaluator struct {
	env     *cel.Env
	program cel.Program
}

// NewCELEvaluator initializes the CEL environment
func NewCELEvaluator() (*CELEvaluator, error) {
	// In a real scenario, we'd pre-declare variables here based on a defined schema
	// For example: cel.Variable("request", cel.MapType(cel.StringType, cel.AnyType))
	env, err := cel.NewEnv(
		cel.Variable("ctx", cel.MapType(cel.StringType, cel.DynType)),
		cel.Variable("caveat", cel.MapType(cel.StringType, cel.DynType)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create CEL env: %w", err)
	}

	return &CELEvaluator{
		env: env,
	}, nil
}

// Compile compiles the string expression into an AST
func (c *CELEvaluator) Compile(expression string) error {
	ast, issues := c.env.Compile(expression)
	if issues != nil && issues.Err() != nil {
		return fmt.Errorf("compile error: %w", issues.Err())
	}

	prg, err := c.env.Program(ast)
	if err != nil {
		return fmt.Errorf("program creation error: %w", err)
	}

	c.program = prg
	return nil
}

// Evaluate runs the compiled AST against the provided contexts
func (c *CELEvaluator) Evaluate(reqCtx map[string]interface{}, caveatCtx map[string]interface{}) (bool, error) {
	if c.program == nil {
		return false, fmt.Errorf("no program compiled")
	}

	// Prepare the evaluation environment map
	evalMap := map[string]interface{}{
		"ctx":    reqCtx,
		"caveat": caveatCtx,
	}

	out, _, err := c.program.Eval(evalMap)
	if err != nil {
		return false, fmt.Errorf("evaluation error: %w", err)
	}

	// The output must be a boolean for ABAC rules
	if out.Type() != cel.BoolType {
		return false, fmt.Errorf("expected boolean result, got %v", out.Type())
	}

	return out.Value().(bool), nil
}
