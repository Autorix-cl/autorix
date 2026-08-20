package grpc

import (
	"context"
	"testing"

	pb "github.com/autorix/nexus/api/autorix/nexus/v1"
	"github.com/autorix/nexus/internal/core"
	"google.golang.org/protobuf/types/known/structpb"
)

type mockGraphEngine struct {
	allowed bool
	reason  string
}

func (m *mockGraphEngine) Check(ctx context.Context, req core.CheckRequest) (core.CheckResult, error) {
	return core.CheckResult{Allowed: m.allowed, Reason: m.reason}, nil
}

func (m *mockGraphEngine) Expand(ctx context.Context, req core.ExpandRequest) (*core.ExpandTree, error) {
	return nil, nil
}

func (m *mockGraphEngine) LookupSubjects(ctx context.Context, req core.LookupSubjectsRequest) ([]core.Tuple, error) {
	return nil, nil
}

func (m *mockGraphEngine) LookupResources(ctx context.Context, req core.LookupResourcesRequest) ([]string, error) {
	return nil, nil
}

type mockRepo struct {
	writtenTuples []core.Tuple
	deletedTuples []core.Tuple
	caveats       []core.CaveatDefinition
}

func (m *mockRepo) WriteTuples(ctx context.Context, tuples []core.Tuple) error {
	m.writtenTuples = append(m.writtenTuples, tuples...)
	return nil
}

func (m *mockRepo) DeleteTuples(ctx context.Context, tuples []core.Tuple) error {
	m.deletedTuples = append(m.deletedTuples, tuples...)
	return nil
}

func (m *mockRepo) WriteCaveat(ctx context.Context, caveat core.CaveatDefinition) error {
	m.caveats = append(m.caveats, caveat)
	return nil
}

type mockCELEvaluator struct{}

func (mockCELEvaluator) Compile(expression string) error  { return nil }
func (mockCELEvaluator) Validate(expression string) error { return nil }
func (mockCELEvaluator) Evaluate(ctx map[string]interface{}, caveatCtx map[string]interface{}) (bool, error) {
	return true, nil
}
func (mockCELEvaluator) EvaluateByName(ctx context.Context, name string, reqCtx map[string]interface{}, caveatCtx map[string]interface{}) (bool, error) {
	return true, nil
}

func TestServer_Check(t *testing.T) {
	engine := &mockGraphEngine{allowed: true, reason: "mock allow"}
	server := NewServer(engine)

	reqCtx, _ := structpb.NewStruct(map[string]interface{}{
		"ip": "192.168.1.1",
	})

	req := &pb.CheckRequest{
		Namespace:        "document",
		Object:           "doc-100",
		Relation:         "editor",
		SubjectId:        "user-42",
		SubjectNamespace: "user",
		RequestContext:   reqCtx,
	}

	resp, err := server.Check(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !resp.Allowed {
		t.Errorf("expected allowed = true, got false")
	}

	if resp.Reason != "mock allow" {
		t.Errorf("expected reason 'mock allow', got %s", resp.Reason)
	}
}

func TestServer_WriteAndDeleteTuples(t *testing.T) {
	repo := &mockRepo{}
	engine := &mockGraphEngine{}
	server := NewServer(engine, WithRepository(repo))

	ctx := context.Background()

	// 1. WriteTuples
	writeReq := &pb.WriteTuplesRequest{
		Tuples: []*pb.RelationTuple{
			{
				Namespace:        "document",
				Object:           "doc-1",
				Relation:         "viewer",
				SubjectNamespace: "user",
				SubjectId:        "alice",
			},
		},
	}
	_, err := server.WriteTuples(ctx, writeReq)
	if err != nil {
		t.Fatalf("WriteTuples failed: %v", err)
	}
	if len(repo.writtenTuples) != 1 || repo.writtenTuples[0].SubjectObject != "alice" {
		t.Fatalf("expected written tuple for alice, got: %+v", repo.writtenTuples)
	}

	// 2. DeleteTuples
	deleteReq := &pb.DeleteTuplesRequest{
		Tuples: []*pb.RelationTuple{
			{
				Namespace:        "document",
				Object:           "doc-1",
				Relation:         "viewer",
				SubjectNamespace: "user",
				SubjectId:        "alice",
			},
		},
	}
	_, err = server.DeleteTuples(ctx, deleteReq)
	if err != nil {
		t.Fatalf("DeleteTuples failed: %v", err)
	}
	if len(repo.deletedTuples) != 1 {
		t.Fatalf("expected deleted tuple for alice, got: %+v", repo.deletedTuples)
	}
}

func TestServer_WriteCaveats(t *testing.T) {
	repo := &mockRepo{}
	engine := &mockGraphEngine{}
	evaluator := &mockCELEvaluator{}
	server := NewServer(engine, WithRepository(repo), WithCaveatEvaluator(evaluator))

	req := &pb.WriteCaveatsRequest{
		Caveats: []*pb.CaveatDefinition{
			{
				Name:          "is_admin",
				CelExpression: `ctx.role == "admin"`,
			},
		},
	}

	_, err := server.WriteCaveats(context.Background(), req)
	if err != nil {
		t.Fatalf("WriteCaveats failed: %v", err)
	}
	if len(repo.caveats) != 1 || repo.caveats[0].Name != "is_admin" {
		t.Fatalf("expected caveat is_admin written, got %+v", repo.caveats)
	}
}

type expandMockEngine struct {
	mockGraphEngine
}

func (expandMockEngine) Expand(ctx context.Context, req core.ExpandRequest) (*core.ExpandTree, error) {
	return &core.ExpandTree{
		Type: "union",
		Children: []*core.ExpandTree{
			{
				Type: "leaf",
				Tuple: &core.Tuple{
					Namespace:        req.Namespace,
					Object:           req.Object,
					Relation:         req.Relation,
					SubjectNamespace: "user",
					SubjectObject:    "alice",
				},
			},
		},
	}, nil
}

func TestServer_Expand(t *testing.T) {
	engine := &expandMockEngine{}
	server := NewServer(engine)

	req := &pb.ExpandRequest{
		Namespace: "document",
		Object:    "doc-1",
		Relation:  "viewer",
	}

	resp, err := server.Expand(context.Background(), req)
	if err != nil {
		t.Fatalf("Expand failed: %v", err)
	}
	if resp.Tree == nil || resp.Tree.Type != "union" || len(resp.Tree.Children) != 1 {
		t.Fatalf("unexpected expand tree in response: %+v", resp.Tree)
	}
	if resp.Tree.Children[0].Tuple.SubjectId != "alice" {
		t.Fatalf("expected leaf subject alice, got %s", resp.Tree.Children[0].Tuple.SubjectId)
	}
}
