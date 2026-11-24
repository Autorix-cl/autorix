package caveat

import (
	"context"
	"errors"
	"testing"
)

func TestCELEvaluator_Evaluate(t *testing.T) {
	evaluator, err := NewCELEvaluator()
	if err != nil {
		t.Fatalf("failed to create CEL evaluator: %v", err)
	}

	tests := []struct {
		name       string
		expression string
		reqCtx     map[string]interface{}
		caveatCtx  map[string]interface{}
		want       bool
		wantErr    bool
	}{
		{
			name:       "allow by matching ip in request context",
			expression: `ctx.ip == "192.168.1.100"`,
			reqCtx:     map[string]interface{}{"ip": "192.168.1.100"},
			caveatCtx:  nil,
			want:       true,
			wantErr:    false,
		},
		{
			name:       "deny by mismatching ip",
			expression: `ctx.ip == "192.168.1.100"`,
			reqCtx:     map[string]interface{}{"ip": "10.0.0.1"},
			caveatCtx:  nil,
			want:       false,
			wantErr:    false,
		},
		{
			name:       "compare request amount against caveat limit",
			expression: `ctx.amount <= caveat.max_amount`,
			reqCtx:     map[string]interface{}{"amount": 500},
			caveatCtx:  map[string]interface{}{"max_amount": 1000},
			want:       true,
			wantErr:    false,
		},
		{
			name:       "exceed caveat limit",
			expression: `ctx.amount <= caveat.max_amount`,
			reqCtx:     map[string]interface{}{"amount": 1500},
			caveatCtx:  map[string]interface{}{"max_amount": 1000},
			want:       false,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := evaluator.Compile(tt.expression); err != nil {
				t.Fatalf("Compile() failed: %v", err)
			}

			got, err := evaluator.Evaluate(tt.reqCtx, tt.caveatCtx)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Evaluate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("Evaluate() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCELEvaluator_Validate(t *testing.T) {
	evaluator, err := NewCELEvaluator()
	if err != nil {
		t.Fatalf("failed to create CEL evaluator: %v", err)
	}

	tests := []struct {
		name       string
		expression string
		wantErr    bool
	}{
		{
			name:       "valid boolean expression",
			expression: `ctx.ip == "10.0.0.1"`,
			wantErr:    false,
		},
		{
			name:       "valid comparison with caveat",
			expression: `ctx.amount <= caveat.limit`,
			wantErr:    false,
		},
		{
			name:       "invalid syntax",
			expression: `ctx.ip === "10.0.0.1"`,
			wantErr:    true,
		},
		{
			name:       "non-boolean return expression",
			expression: `ctx.amount + 10`,
			wantErr:    true,
		},
		{
			name:       "empty expression",
			expression: ``,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := evaluator.Validate(tt.expression)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate(%q) error = %v, wantErr = %v", tt.expression, err, tt.wantErr)
			}
		})
	}
}

type mockCaveatGetter struct {
	caveats map[string]string
}

func (m *mockCaveatGetter) GetCaveatExpression(ctx context.Context, name string) (string, error) {
	expr, ok := m.caveats[name]
	if !ok {
		return "", errors.New("caveat not found")
	}
	return expr, nil
}

func TestCELEvaluator_EvaluateByName(t *testing.T) {
	getter := &mockCaveatGetter{
		caveats: map[string]string{
			"is_admin_ip":   `ctx.ip == "10.0.0.1"`,
			"within_budget": `ctx.cost <= caveat.budget`,
		},
	}

	evaluator, err := NewCELEvaluator(WithCaveatGetter(getter))
	if err != nil {
		t.Fatalf("failed to create evaluator: %v", err)
	}

	// 1. Evaluate existing caveat that passes
	ok, err := evaluator.EvaluateByName(context.Background(), "is_admin_ip", map[string]interface{}{"ip": "10.0.0.1"}, nil)
	if err != nil || !ok {
		t.Errorf("expected is_admin_ip to evaluate to true, got ok=%v, err=%v", ok, err)
	}

	// 2. Evaluate existing caveat that fails
	ok, err = evaluator.EvaluateByName(context.Background(), "is_admin_ip", map[string]interface{}{"ip": "192.168.1.1"}, nil)
	if err != nil || ok {
		t.Errorf("expected is_admin_ip to evaluate to false, got ok=%v, err=%v", ok, err)
	}

	// 3. Evaluate nonexistent caveat
	_, err = evaluator.EvaluateByName(context.Background(), "nonexistent", nil, nil)
	if err == nil {
		t.Errorf("expected error for nonexistent caveat")
	}
}

func TestCELEvaluator_InvalidateAndSafety(t *testing.T) {
	getter := &mockCaveatGetter{
		caveats: map[string]string{
			"env_check": `ctx.env == "staging"`,
		},
	}

	ev, err := NewCELEvaluator(WithCaveatGetter(getter))
	if err != nil {
		t.Fatalf("NewCELEvaluator: %v", err)
	}

	// 1. Initial caveat evaluates against "staging"
	ok, err := ev.EvaluateByName(context.Background(), "env_check", map[string]interface{}{"env": "staging"}, nil)
	if err != nil || !ok {
		t.Fatalf("expected true for staging, got ok=%v err=%v", ok, err)
	}

	// 2. Update underlying caveat expression and call Invalidate
	getter.caveats["env_check"] = `ctx.env == "prod"`
	ev.Invalidate("env_check")

	ok, err = ev.EvaluateByName(context.Background(), "env_check", map[string]interface{}{"env": "staging"}, nil)
	if err != nil || ok {
		t.Fatalf("expected false for staging after Invalidate, got ok=%v err=%v", ok, err)
	}
	ok, err = ev.EvaluateByName(context.Background(), "env_check", map[string]interface{}{"env": "prod"}, nil)
	if err != nil || !ok {
		t.Fatalf("expected true for prod after Invalidate, got ok=%v err=%v", ok, err)
	}

	// 3. Compile expression does not collide with named cache
	if err := ev.Compile(`ctx.env == "prod"`); err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if _, err := ev.EvaluateByName(context.Background(), `ctx.env == "prod"`, map[string]interface{}{"env": "prod"}, nil); err == nil {
		t.Fatalf("expected named lookup for raw expression string to miss separate named cache")
	}

	// 4. Cost limit stops expensive comprehension
	nums := make([]int64, 500)
	if err := ev.Compile(`ctx.nums.all(x, ctx.nums.all(y, x + y >= 0))`); err != nil {
		t.Fatalf("Compile expensive expr: %v", err)
	}
	if _, err := ev.Evaluate(map[string]interface{}{"nums": nums}, nil); err == nil {
		t.Fatalf("expected cost limit error for quadratic comprehension")
	}

	// 5. Context cancellation stops evaluation
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := ev.Compile(`ctx.nums.all(x, x >= 0)`); err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if _, err := ev.EvaluateContext(ctx, map[string]interface{}{"nums": nums}, nil); err == nil {
		t.Fatalf("expected error on cancelled context")
	}
}
