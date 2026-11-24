package grpc

import (
	"context"
	"testing"

	pb "github.com/autorix/themis/api/autorix/themis/v1"
	"github.com/autorix/themis/internal/core"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

type stubGRPCEngine struct {
	batchCalls int
}

func (stubGRPCEngine) CreatePolicy(context.Context, *core.Policy) error { return nil }
func (stubGRPCEngine) GetPolicy(context.Context, string, string) (*core.Policy, error) {
	return nil, nil
}
func (stubGRPCEngine) ListPolicies(context.Context, core.ListFilter) ([]core.Policy, error) {
	return nil, nil
}
func (stubGRPCEngine) ListPoliciesPage(context.Context, core.ListFilter, int, string) ([]core.Policy, bool, error) {
	return nil, false, nil
}
func (stubGRPCEngine) UpdatePolicy(context.Context, *core.Policy) error { return nil }
func (stubGRPCEngine) DeletePolicy(context.Context, string, string) error {
	return nil
}
func (stubGRPCEngine) Evaluate(context.Context, core.EvaluateRequest) (*core.EvaluateResponse, error) {
	return &core.EvaluateResponse{AllPassed: true}, nil
}
func (s *stubGRPCEngine) EvaluateBatch(_ context.Context, _ string, _ map[string]string, payloads []map[string]interface{}) ([]*core.EvaluateResponse, error) {
	s.batchCalls++
	out := make([]*core.EvaluateResponse, len(payloads))
	for i := range payloads {
		out[i] = &core.EvaluateResponse{AllPassed: true, TotalEvaluated: 1}
	}
	return out, nil
}
func (stubGRPCEngine) ListPolicyVersions(context.Context, string, string) ([]core.PolicyVersion, error) {
	return nil, nil
}
func (stubGRPCEngine) RollbackPolicy(context.Context, string, string, int32) (*core.Policy, error) {
	return nil, nil
}
func (stubGRPCEngine) ValidateExpression(context.Context, string) (*core.ValidationResult, error) {
	return nil, nil
}
func (stubGRPCEngine) DryRun(context.Context, core.DryRunRequest) (*core.DryRunResult, error) {
	return nil, nil
}
func (stubGRPCEngine) CreateFixture(context.Context, *core.PolicyFixture) error { return nil }
func (stubGRPCEngine) ListFixtures(context.Context, string, string) ([]core.PolicyFixture, error) {
	return nil, nil
}
func (stubGRPCEngine) DeleteFixture(context.Context, string, string, string) error { return nil }
func (stubGRPCEngine) RunTestSuite(context.Context, string, string) (*core.TestSuiteResult, error) {
	return nil, nil
}

func TestBatchEvaluate_CapAndSingleLoad(t *testing.T) {
	eng := &stubGRPCEngine{}
	srv := NewServer(eng)

	tooMany := make([]*structpb.Struct, maxBatchEvaluateSize+1)
	for i := range tooMany {
		tooMany[i], _ = structpb.NewStruct(map[string]interface{}{"i": i})
	}
	_, err := srv.BatchEvaluate(context.Background(), &pb.BatchEvaluateRequest{
		TenantId: "t1",
		Payloads: tooMany,
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument for batch size > %d, got %v", maxBatchEvaluateSize, err)
	}

	okBatch := tooMany[:3]
	resp, err := srv.BatchEvaluate(context.Background(), &pb.BatchEvaluateRequest{
		TenantId: "t1",
		Payloads: okBatch,
	})
	if err != nil {
		t.Fatalf("BatchEvaluate failed: %v", err)
	}
	if len(resp.Evaluations) != 3 || eng.batchCalls != 1 {
		t.Fatalf("expected 3 evaluations with 1 EvaluateBatch call, got len=%d calls=%d", len(resp.Evaluations), eng.batchCalls)
	}
}
