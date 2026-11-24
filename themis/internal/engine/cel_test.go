package engine

import (
	"context"
	"strings"
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

func TestCELEvaluator_HardeningAndSafety(t *testing.T) {
	t.Run("non-positive cache size does not panic", func(t *testing.T) {
		for _, sz := range []int{0, -10} {
			ev, err := NewCELEvaluatorWithSize(sz)
			if err != nil {
				t.Fatalf("NewCELEvaluatorWithSize(%d): %v", sz, err)
			}
			ok, err := ev.Evaluate(`payload.x == 1`, map[string]interface{}{"x": 1})
			if err != nil || !ok {
				t.Fatalf("Evaluate failed for size %d: ok=%v err=%v", sz, ok, err)
			}
		}
	})

	t.Run("Compile rejects non-bool expression", func(t *testing.T) {
		ev, err := NewCELEvaluator()
		if err != nil {
			t.Fatalf("NewCELEvaluator: %v", err)
		}
		if err := ev.Compile(`payload.amount + 1`); err == nil {
			t.Fatalf("expected Compile to reject non-bool expression")
		}
	})

	t.Run("EvaluateUncached does not populate shared cache", func(t *testing.T) {
		ev, err := NewCELEvaluator()
		if err != nil {
			t.Fatalf("NewCELEvaluator: %v", err)
		}
		ok, err := ev.EvaluateUncached(context.Background(), `payload.a == "ok"`, map[string]interface{}{"a": "ok"})
		if err != nil || !ok {
			t.Fatalf("EvaluateUncached failed: ok=%v err=%v", ok, err)
		}
		if ev.CacheSize() != 0 {
			t.Fatalf("expected CacheSize() == 0 after EvaluateUncached, got %d", ev.CacheSize())
		}
	})

	t.Run("cost limit stops expensive nested comprehensions", func(t *testing.T) {
		ev, err := NewCELEvaluator()
		if err != nil {
			t.Fatalf("NewCELEvaluator: %v", err)
		}
		nums := make([]int64, 500)
		for i := range nums {
			nums[i] = int64(i)
		}
		// 500 * 500 = 250,000 iterations >> CostLimit(100000)
		expr := `payload.nums.all(x, payload.nums.all(y, x + y >= 0))`
		_, err = ev.EvaluateContext(context.Background(), expr, map[string]interface{}{"nums": nums})
		if err == nil {
			t.Fatalf("expected cost limit error for quadratic comprehension, got nil")
		}
	})

	t.Run("cancelled context interrupts comprehension", func(t *testing.T) {
		ev, err := NewCELEvaluator()
		if err != nil {
			t.Fatalf("NewCELEvaluator: %v", err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		nums := make([]int64, 500)
		expr := `payload.nums.all(x, x >= 0)`
		_, err = ev.EvaluateContext(ctx, expr, map[string]interface{}{"nums": nums})
		if err == nil {
			t.Fatalf("expected error on cancelled context, got nil")
		}
	})

	t.Run("oversized expression rejected", func(t *testing.T) {
		ev, err := NewCELEvaluator()
		if err != nil {
			t.Fatalf("NewCELEvaluator: %v", err)
		}
		huge := "true" + strings.Repeat(" && true", 3000)
		if err := ev.Compile(huge); err == nil {
			t.Fatalf("expected oversized expression (>16384 code points) to fail compilation")
		}
	})
}
