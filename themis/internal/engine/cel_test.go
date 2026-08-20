package engine

import (
	"testing"
)

func TestCELEvaluator_BasicExpressions(t *testing.T) {
	evaluator, err := NewCELEvaluator()
	if err != nil {
		t.Fatalf("failed to create evaluator: %v", err)
	}

	tests := []struct {
		name       string
		expression string
		payload    map[string]interface{}
		want       bool
		wantErr    bool
	}{
		{
			name:       "simple comparison true",
			expression: `payload.amount < payload.budget`,
			payload:    map[string]interface{}{"amount": 500.0, "budget": 1000.0},
			want:       true,
		},
		{
			name:       "simple comparison false",
			expression: `payload.amount < payload.budget`,
			payload:    map[string]interface{}{"amount": 1500.0, "budget": 1000.0},
			want:       false,
		},
		{
			name:       "string equality",
			expression: `payload.status == "active"`,
			payload:    map[string]interface{}{"status": "active"},
			want:       true,
		},
		{
			name:       "string inequality",
			expression: `payload.status == "active"`,
			payload:    map[string]interface{}{"status": "inactive"},
			want:       false,
		},
		{
			name:       "compound AND true",
			expression: `payload.amount < payload.budget && payload.status == "approved"`,
			payload:    map[string]interface{}{"amount": 100.0, "budget": 500.0, "status": "approved"},
			want:       true,
		},
		{
			name:       "compound AND false",
			expression: `payload.amount < payload.budget && payload.status == "approved"`,
			payload:    map[string]interface{}{"amount": 100.0, "budget": 500.0, "status": "pending"},
			want:       false,
		},
		{
			name:       "compound OR true",
			expression: `payload.role == "admin" || payload.level > 5`,
			payload:    map[string]interface{}{"role": "user", "level": 10},
			want:       true,
		},
		{
			name:       "negation",
			expression: `!(payload.blocked)`,
			payload:    map[string]interface{}{"blocked": false},
			want:       true,
		},
		{
			name:       "arithmetic in expression",
			expression: `payload.quantity * payload.unit_price <= payload.max_total`,
			payload:    map[string]interface{}{"quantity": 5, "unit_price": 100, "max_total": 600},
			want:       true,
		},
		{
			name:       "invalid expression",
			expression: `payload.foo +++ bar`,
			payload:    map[string]interface{}{},
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := evaluator.Evaluate(tt.expression, tt.payload)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("Evaluate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCELEvaluator_CacheBehavior(t *testing.T) {
	evaluator, err := NewCELEvaluatorWithSize(3)
	if err != nil {
		t.Fatalf("failed to create evaluator: %v", err)
	}

	expressions := []string{
		`payload.a == true`,
		`payload.b == true`,
		`payload.c == true`,
	}

	payload := map[string]interface{}{"a": true, "b": true, "c": true}

	// Fill cache to capacity
	for _, expr := range expressions {
		if _, err := evaluator.Evaluate(expr, payload); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	if evaluator.CacheSize() != 3 {
		t.Errorf("cache size = %d, want 3", evaluator.CacheSize())
	}

	// Add one more — should evict the oldest
	if _, err := evaluator.Evaluate(`payload.d == true`, map[string]interface{}{"d": true}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if evaluator.CacheSize() != 3 {
		t.Errorf("cache size after eviction = %d, want 3", evaluator.CacheSize())
	}
}

func TestCELEvaluator_NonBoolExpression(t *testing.T) {
	evaluator, err := NewCELEvaluator()
	if err != nil {
		t.Fatalf("failed to create evaluator: %v", err)
	}

	// Expression that returns int, not bool
	_, err = evaluator.Evaluate(`payload.amount + 1`, map[string]interface{}{"amount": 5})
	if err == nil {
		t.Error("expected error for non-bool expression, got nil")
	}
}

func TestCELEvaluator_Validate(t *testing.T) {
	evaluator, err := NewCELEvaluator()
	if err != nil {
		t.Fatalf("failed to create evaluator: %v", err)
	}

	t.Run("valid expression with variables", func(t *testing.T) {
		res, err := evaluator.Validate(`request.role == "admin" && payload.amount < 100`)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !res.Valid {
			t.Fatalf("expected expression to be valid, got errors: %+v", res.Errors)
		}
		if len(res.Variables) < 2 {
			t.Fatalf("expected variables ['payload', 'request'], got %v", res.Variables)
		}
	})

	t.Run("invalid syntax expression with line and column", func(t *testing.T) {
		res, err := evaluator.Validate(`payload.amount < `)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Valid {
			t.Fatalf("expected expression to be invalid")
		}
		if len(res.Errors) == 0 {
			t.Fatalf("expected at least one syntax error")
		}
		if res.Errors[0].Line == 0 && res.Errors[0].Column == 0 {
			t.Fatalf("expected line/column information, got %+v", res.Errors[0])
		}
	})

	t.Run("non-boolean expression type check", func(t *testing.T) {
		res, err := evaluator.Validate(`payload.amount + 10`)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Valid {
			t.Fatalf("expected non-boolean expression to be invalid")
		}
		if len(res.Errors) == 0 {
			t.Fatalf("expected error for non-boolean return type")
		}
	})
}

func BenchmarkCELEvaluator_CachedEval(b *testing.B) {
	evaluator, err := NewCELEvaluator()
	if err != nil {
		b.Fatalf("failed to create evaluator: %v", err)
	}

	expression := `payload.amount < payload.budget && payload.status == "active" && payload.quantity > 0`
	payload := map[string]interface{}{
		"amount":   500.0,
		"budget":   1000.0,
		"status":   "active",
		"quantity": 10,
	}

	// Warm the cache
	if _, err := evaluator.Evaluate(expression, payload); err != nil {
		b.Fatalf("warm-up failed: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = evaluator.Evaluate(expression, payload)
	}
}
