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

func (m *mockRepository) QueryTuples(ctx context.Context, filter core.Tuple) ([]core.Tuple, error) {
	var matches []core.Tuple
	for _, t := range m.tuples {
		if filter.Namespace != "" && t.Namespace != filter.Namespace {
			continue
		}
		if filter.Object != "" && t.Object != filter.Object {
			continue
		}
		if filter.Relation != "" && t.Relation != filter.Relation {
			continue
		}
		if filter.SubjectNamespace != "" && t.SubjectNamespace != filter.SubjectNamespace {
			continue
		}
		if filter.SubjectObject != "" && t.SubjectObject != filter.SubjectObject {
			continue
		}
		if filter.SubjectRelation != "" && t.SubjectRelation != filter.SubjectRelation {
			continue
		}
		matches = append(matches, t)
	}
	return matches, nil
}

// mockNamespaceGetter implements graph.NamespaceGetter
type mockNamespaceGetter struct {
	schemas map[string]core.NamespaceSchema
}

func (m *mockNamespaceGetter) GetNamespace(ctx context.Context, name string) (*core.NamespaceSchema, error) {
	if s, ok := m.schemas[name]; ok {
		return &s, nil
	}
	return nil, nil
}

type mockCaveatEvaluator struct {
	allowed bool
}

func (m *mockCaveatEvaluator) Compile(expression string) error  { return nil }
func (m *mockCaveatEvaluator) Validate(expression string) error { return nil }
func (m *mockCaveatEvaluator) Evaluate(ctx map[string]interface{}, caveatCtx map[string]interface{}) (bool, error) {
	return m.allowed, nil
}
func (m *mockCaveatEvaluator) EvaluateByName(ctx context.Context, name string, reqCtx map[string]interface{}, caveatCtx map[string]interface{}) (bool, error) {
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

func TestResolver_SchemaRewrites(t *testing.T) {
	// Schema:
	// document:
	//   viewer: union(this, computed_userset(editor), tuple_to_userset(parent, viewer))
	//   editor: union(this, computed_userset(owner))
	//   owner: this
	//   auditor: intersection(computed_userset(editor), computed_userset(compliance))
	//   compliance: this
	schemas := map[string]core.NamespaceSchema{
		"document": {
			Name: "document",
			Relations: map[string]core.RelationDefinition{
				"viewer": {
					Rewrite: &core.RewriteRule{
						Type: "union",
						Children: []*core.RewriteRule{
							{Type: "this"},
							{Type: "computed_userset", Relation: "editor"},
							{Type: "tuple_to_userset", TuplesetRelation: "parent", ComputedRelation: "viewer"},
						},
					},
				},
				"editor": {
					Rewrite: &core.RewriteRule{
						Type: "union",
						Children: []*core.RewriteRule{
							{Type: "this"},
							{Type: "computed_userset", Relation: "owner"},
						},
					},
				},
				"owner": {
					Rewrite: &core.RewriteRule{Type: "this"},
				},
				"auditor": {
					Rewrite: &core.RewriteRule{
						Type: "intersection",
						Children: []*core.RewriteRule{
							{Type: "computed_userset", Relation: "editor"},
							{Type: "computed_userset", Relation: "compliance"},
						},
					},
				},
				"compliance": {
					Rewrite: &core.RewriteRule{Type: "this"},
				},
			},
		},
		"folder": {
			Name: "folder",
			Relations: map[string]core.RelationDefinition{
				"viewer": {Rewrite: &core.RewriteRule{Type: "this"}},
			},
		},
	}

	tuples := []core.Tuple{
		// alice is owner of doc-1
		{Namespace: "document", Object: "doc-1", Relation: "owner", SubjectNamespace: "user", SubjectObject: "alice"},
		// doc-2 parent is folder:f1
		{Namespace: "document", Object: "doc-2", Relation: "parent", SubjectNamespace: "folder", SubjectObject: "f1"},
		// bob is viewer of folder:f1
		{Namespace: "folder", Object: "f1", Relation: "viewer", SubjectNamespace: "user", SubjectObject: "bob"},
		// charlie is editor of doc-3
		{Namespace: "document", Object: "doc-3", Relation: "editor", SubjectNamespace: "user", SubjectObject: "charlie"},
		// dave is both editor and compliance on doc-4
		{Namespace: "document", Object: "doc-4", Relation: "editor", SubjectNamespace: "user", SubjectObject: "dave"},
		{Namespace: "document", Object: "doc-4", Relation: "compliance", SubjectNamespace: "user", SubjectObject: "dave"},
	}

	repo := &mockRepository{tuples: tuples}
	eval := &mockCaveatEvaluator{allowed: true}
	nsGetter := &mockNamespaceGetter{schemas: schemas}
	resolver := NewResolver(repo, eval, WithNamespaceGetter(nsGetter))

	ctx := context.Background()

	// 1. Computed Userset: alice is owner -> editor -> viewer
	res, err := resolver.Check(ctx, core.CheckRequest{
		Namespace: "document", Object: "doc-1", Relation: "viewer",
		Subject: core.Tuple{Namespace: "user", Object: "alice"},
	})
	if err != nil || !res.Allowed {
		t.Fatalf("expected alice to view doc-1 via owner->editor->viewer, got %v, err=%v", res.Allowed, err)
	}

	// 2. Tuple to Userset: bob is viewer of folder:f1 -> viewer of doc-2
	res, err = resolver.Check(ctx, core.CheckRequest{
		Namespace: "document", Object: "doc-2", Relation: "viewer",
		Subject: core.Tuple{Namespace: "user", Object: "bob"},
	})
	if err != nil || !res.Allowed {
		t.Fatalf("expected bob to view doc-2 via parent folder, got %v, err=%v", res.Allowed, err)
	}

	// 3. Intersection: charlie is editor on doc-3 but not compliance -> auditor is false
	res, err = resolver.Check(ctx, core.CheckRequest{
		Namespace: "document", Object: "doc-3", Relation: "auditor",
		Subject: core.Tuple{Namespace: "user", Object: "charlie"},
	})
	if err != nil || res.Allowed {
		t.Fatalf("expected charlie to NOT be auditor on doc-3, got %v, err=%v", res.Allowed, err)
	}

	// 4. Intersection: dave is both editor and compliance on doc-4 -> auditor is true
	res, err = resolver.Check(ctx, core.CheckRequest{
		Namespace: "document", Object: "doc-4", Relation: "auditor",
		Subject: core.Tuple{Namespace: "user", Object: "dave"},
	})
	if err != nil || !res.Allowed {
		t.Fatalf("expected dave to be auditor on doc-4, got %v, err=%v", res.Allowed, err)
	}
}

func TestResolver_DecisionExplanation(t *testing.T) {
	tuples := []core.Tuple{
		{
			Namespace: "document", Object: "doc-exp", Relation: "viewer",
			SubjectNamespace: "user", SubjectObject: "alice",
			CaveatName: "is_office_ip",
		},
	}

	repo := &mockRepository{tuples: tuples}
	eval := &mockCaveatEvaluator{allowed: true}
	resolver := NewResolver(repo, eval)

	res, err := resolver.Check(context.Background(), core.CheckRequest{
		Namespace: "document", Object: "doc-exp", Relation: "viewer",
		Subject: core.Tuple{Namespace: "user", Object: "alice"},
		Explain: true,
	})
	if err != nil {
		t.Fatalf("Check explain: %v", err)
	}
	if !res.Allowed {
		t.Fatalf("expected allowed=true")
	}
	if res.Trace == nil {
		t.Fatalf("expected decision trace to be populated when Explain=true")
	}
	if res.Trace.Namespace != "document" || res.Trace.Object != "doc-exp" || !res.Trace.Allowed {
		t.Fatalf("unexpected trace root: %+v", res.Trace)
	}
	if res.Trace.Caveat == nil || res.Trace.Caveat.CaveatName != "is_office_ip" || !res.Trace.Caveat.Allowed {
		t.Fatalf("expected caveat evaluation in trace: %+v", res.Trace.Caveat)
	}
}

func TestResolver_Expand(t *testing.T) {
	schemas := map[string]core.NamespaceSchema{
		"document": {
			Name: "document",
			Relations: map[string]core.RelationDefinition{
				"viewer": {
					Rewrite: &core.RewriteRule{
						Type: "union",
						Children: []*core.RewriteRule{
							{Type: "this"},
							{Type: "computed_userset", Relation: "editor"},
						},
					},
				},
				"editor": {Rewrite: &core.RewriteRule{Type: "this"}},
			},
		},
	}

	tuples := []core.Tuple{
		{Namespace: "document", Object: "doc1", Relation: "viewer", SubjectNamespace: "user", SubjectObject: "alice"},
		{Namespace: "document", Object: "doc1", Relation: "editor", SubjectNamespace: "user", SubjectObject: "bob"},
		{Namespace: "document", Object: "doc1", Relation: "viewer", SubjectNamespace: "group", SubjectObject: "devs", SubjectRelation: "member"},
	}

	repo := &mockRepository{tuples: tuples}
	eval := &mockCaveatEvaluator{allowed: true}
	resolver := NewResolver(repo, eval, WithNamespaceGetter(&mockNamespaceGetter{schemas: schemas}))

	tree, err := resolver.Expand(context.Background(), core.ExpandRequest{
		Namespace: "document", Object: "doc1", Relation: "viewer",
	})
	if err != nil {
		t.Fatalf("Expand failed: %v", err)
	}
	if tree == nil {
		t.Fatalf("expected expand tree to be non-nil")
	}
	if tree.Type != "union" {
		t.Errorf("expected tree type 'union', got %q", tree.Type)
	}
	if len(tree.Children) == 0 {
		t.Fatalf("expected union to have children")
	}
}

func TestResolver_ReverseLookup(t *testing.T) {
	tuples := []core.Tuple{
		{Namespace: "document", Object: "doc1", Relation: "viewer", SubjectNamespace: "user", SubjectObject: "alice"},
		{Namespace: "document", Object: "doc2", Relation: "viewer", SubjectNamespace: "user", SubjectObject: "alice"},
		{Namespace: "document", Object: "doc1", Relation: "viewer", SubjectNamespace: "user", SubjectObject: "bob"},
	}

	repo := &mockRepository{tuples: tuples}
	eval := &mockCaveatEvaluator{allowed: true}
	resolver := NewResolver(repo, eval)

	ctx := context.Background()

	// 1. LookupSubjects: who has viewer on doc1 -> alice and bob
	subjects, err := resolver.LookupSubjects(ctx, core.LookupSubjectsRequest{
		Namespace: "document", Object: "doc1", Relation: "viewer",
	})
	if err != nil {
		t.Fatalf("LookupSubjects: %v", err)
	}
	if len(subjects) != 2 {
		t.Fatalf("expected 2 subjects for doc1#viewer, got %d", len(subjects))
	}

	// 2. LookupResources: what documents can alice view -> doc1 and doc2
	resources, err := resolver.LookupResources(ctx, core.LookupResourcesRequest{
		Namespace: "document", Relation: "viewer",
		SubjectNamespace: "user", SubjectObject: "alice",
	})
	if err != nil {
		t.Fatalf("LookupResources: %v", err)
	}
	if len(resources) != 2 {
		t.Fatalf("expected 2 resources for user:alice, got %d: %+v", len(resources), resources)
	}
}

func TestResolver_CycleDetection(t *testing.T) {
	tuples := []core.Tuple{
		// Group A has member Group B
		{Namespace: "group", Object: "A", Relation: "member", SubjectNamespace: "group", SubjectObject: "B", SubjectRelation: "member"},
		// Group B has member Group A (cycle!)
		{Namespace: "group", Object: "B", Relation: "member", SubjectNamespace: "group", SubjectObject: "A", SubjectRelation: "member"},
	}

	repo := &mockRepository{tuples: tuples}
	eval := &mockCaveatEvaluator{allowed: true}
	resolver := NewResolver(repo, eval)

	res, err := resolver.Check(context.Background(), core.CheckRequest{
		Namespace: "group", Object: "A", Relation: "member",
		Subject: core.Tuple{Namespace: "user", Object: "outsider"},
	})
	if err != nil {
		t.Fatalf("unexpected error on cycle: %v", err)
	}
	if res.Allowed {
		t.Fatalf("expected outsider not allowed on cyclic group")
	}
}
