package graph

import (
	"context"
	"testing"

	"github.com/autorix/nexus/internal/core"
)

// mockRepository implements graph.Repository in-memory for testing
type mockRepository struct {
	tuples []core.Tuple
}

func (m *mockRepository) ReadTuples(ctx context.Context, filter core.Tuple) ([]core.Tuple, error) {
	var matches []core.Tuple
	for _, t := range m.tuples {
		if t.Namespace == filter.Namespace && t.Object == filter.Object && t.Relation == filter.Relation {
			matches = append(matches, t)
		}
	}
	return matches, nil
}

// mockCaveatEvaluator implements core.CaveatEvaluator
type mockCaveatEvaluator struct {
	allowed bool
}

func (m *mockCaveatEvaluator) Compile(expression string) error { return nil }
func (m *mockCaveatEvaluator) Evaluate(ctx map[string]interface{}, caveatCtx map[string]interface{}) (bool, error) {
	return m.allowed, nil
}

func TestResolver_Check(t *testing.T) {
	tests := []struct {
		name        string
		tuples      []core.Tuple
		evalAllowed bool
		req         core.CheckRequest
		wantAllowed bool
	}{
		{
			name: "direct relation match",
			tuples: []core.Tuple{
				{
					Namespace:        "document",
					Object:           "doc-1",
					Relation:         "viewer",
					SubjectNamespace: "user",
					SubjectObject:    "alice",
					SubjectRelation:  "",
				},
			},
			evalAllowed: true,
			req: core.CheckRequest{
				Namespace: "document",
				Object:    "doc-1",
				Relation:  "viewer",
				Subject: core.Tuple{
					Namespace: "user",
					Object:    "alice",
				},
			},
			wantAllowed: true,
		},
		{
			name: "direct relation mismatch (no access)",
			tuples: []core.Tuple{
				{
					Namespace:        "document",
					Object:           "doc-1",
					Relation:         "viewer",
					SubjectNamespace: "user",
					SubjectObject:    "alice",
				},
			},
			evalAllowed: true,
			req: core.CheckRequest{
				Namespace: "document",
				Object:    "doc-1",
				Relation:  "viewer",
				Subject: core.Tuple{
					Namespace: "user",
					Object:    "bob",
				},
			},
			wantAllowed: false,
		},
		{
			name: "indirect relation match through group (userset)",
			tuples: []core.Tuple{
				// doc-1 viewer is group:engineering#member
				{
					Namespace:        "document",
					Object:           "doc-1",
					Relation:         "viewer",
					SubjectNamespace: "group",
					SubjectObject:    "engineering",
					SubjectRelation:  "member",
				},
				// bob is member of group:engineering
				{
					Namespace:        "group",
					Object:           "engineering",
					Relation:         "member",
					SubjectNamespace: "user",
					SubjectObject:    "bob",
				},
			},
			evalAllowed: true,
			req: core.CheckRequest{
				Namespace: "document",
				Object:    "doc-1",
				Relation:  "viewer",
				Subject: core.Tuple{
					Namespace: "user",
					Object:    "bob",
				},
			},
			wantAllowed: true,
		},
		{
			name: "caveat denies despite valid relationship",
			tuples: []core.Tuple{
				{
					Namespace:        "document",
					Object:           "doc-financial",
					Relation:         "viewer",
					SubjectNamespace: "user",
					SubjectObject:    "charlie",
					CaveatName:       "is_office_ip",
				},
			},
			evalAllowed: false, // Caveat evaluation returns false
			req: core.CheckRequest{
				Namespace: "document",
				Object:    "doc-financial",
				Relation:  "viewer",
				Subject: core.Tuple{
					Namespace: "user",
					Object:    "charlie",
				},
			},
			wantAllowed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockRepository{tuples: tt.tuples}
			eval := &mockCaveatEvaluator{allowed: tt.evalAllowed}
			resolver := NewResolver(repo, eval)

			res, err := resolver.Check(context.Background(), tt.req)
			if err != nil {
				t.Fatalf("Check() unexpected error: %v", err)
			}
			if res.Allowed != tt.wantAllowed {
				t.Errorf("Check() allowed = %v, want %v (reason: %s)", res.Allowed, tt.wantAllowed, res.Reason)
			}
		})
	}
}
