package caveat

import (
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
