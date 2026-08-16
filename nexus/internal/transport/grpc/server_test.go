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
