package grpc

import (
	"context"

	"github.com/autorix/themis/internal/core"
	pb "github.com/autorix/themis/api/autorix/themis/v1"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Server implements the ThemisService gRPC server.
type Server struct {
	pb.UnimplementedThemisServiceServer
	engine core.PolicyEngine
}

// NewServer creates a new Themis gRPC server.
func NewServer(engine core.PolicyEngine) *Server {
	return &Server{engine: engine}
}

// ────────────────────────────────────────────────────────────────────────
// CRUD Handlers
// ────────────────────────────────────────────────────────────────────────

func (s *Server) CreatePolicy(ctx context.Context, req *pb.CreatePolicyRequest) (*pb.CreatePolicyResponse, error) {
	if req.TenantId == "" || req.Name == "" || req.Expression == "" {
		return nil, status.Error(codes.InvalidArgument, "tenant_id, name, and expression are required")
	}

	policy := &core.Policy{
		TenantID:    req.TenantId,
		Name:        req.Name,
		Description: req.Description,
		Expression:  req.Expression,
		Priority:    req.Priority,
		Enabled:     req.Enabled,
		Labels:      req.Labels,
	}

	if err := s.engine.CreatePolicy(ctx, policy); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create policy: %v", err)
	}

	return &pb.CreatePolicyResponse{
		Policy: domainToPb(policy),
	}, nil
}

func (s *Server) GetPolicy(ctx context.Context, req *pb.GetPolicyRequest) (*pb.GetPolicyResponse, error) {
	if req.TenantId == "" || req.PolicyId == "" {
		return nil, status.Error(codes.InvalidArgument, "tenant_id and policy_id are required")
	}

	policy, err := s.engine.GetPolicy(ctx, req.TenantId, req.PolicyId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "policy not found: %v", err)
	}

	return &pb.GetPolicyResponse{
		Policy: domainToPb(policy),
	}, nil
}

func (s *Server) ListPolicies(ctx context.Context, req *pb.ListPoliciesRequest) (*pb.ListPoliciesResponse, error) {
	if req.TenantId == "" {
		return nil, status.Error(codes.InvalidArgument, "tenant_id is required")
	}

	policies, err := s.engine.ListPolicies(ctx, core.ListFilter{
		TenantID:    req.TenantId,
		EnabledOnly: req.EnabledOnly,
		Labels:      req.LabelFilter,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list policies: %v", err)
	}

	pbPolicies := make([]*pb.Policy, len(policies))
	for i := range policies {
		pbPolicies[i] = domainToPb(&policies[i])
	}

	return &pb.ListPoliciesResponse{
		Policies: pbPolicies,
	}, nil
}

func (s *Server) UpdatePolicy(ctx context.Context, req *pb.UpdatePolicyRequest) (*pb.UpdatePolicyResponse, error) {
	if req.TenantId == "" || req.PolicyId == "" {
		return nil, status.Error(codes.InvalidArgument, "tenant_id and policy_id are required")
	}

	policy := &core.Policy{
		ID:          req.PolicyId,
		TenantID:    req.TenantId,
		Name:        req.Name,
		Description: req.Description,
		Expression:  req.Expression,
		Priority:    req.Priority,
		Enabled:     req.Enabled,
		Labels:      req.Labels,
	}

	if err := s.engine.UpdatePolicy(ctx, policy); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update policy: %v", err)
	}

	// Re-read to get updated timestamps
	updated, err := s.engine.GetPolicy(ctx, req.TenantId, req.PolicyId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to read updated policy: %v", err)
	}

	return &pb.UpdatePolicyResponse{
		Policy: domainToPb(updated),
	}, nil
}

func (s *Server) DeletePolicy(ctx context.Context, req *pb.DeletePolicyRequest) (*pb.DeletePolicyResponse, error) {
	if req.TenantId == "" || req.PolicyId == "" {
		return nil, status.Error(codes.InvalidArgument, "tenant_id and policy_id are required")
	}

	if err := s.engine.DeletePolicy(ctx, req.TenantId, req.PolicyId); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete policy: %v", err)
	}

	return &pb.DeletePolicyResponse{}, nil
}

// ────────────────────────────────────────────────────────────────────────
// Evaluation Handlers
// ────────────────────────────────────────────────────────────────────────

func (s *Server) EvaluatePolicy(ctx context.Context, req *pb.EvaluatePolicyRequest) (*pb.EvaluatePolicyResponse, error) {
	if req.TenantId == "" {
		return nil, status.Error(codes.InvalidArgument, "tenant_id is required")
	}
	if req.Payload == nil {
		return nil, status.Error(codes.InvalidArgument, "payload is required")
	}

	evalReq := core.EvaluateRequest{
		TenantID:    req.TenantId,
		PolicyID:    req.PolicyId,
		Payload:     req.Payload.AsMap(),
		LabelFilter: req.LabelFilter,
	}

	resp, err := s.engine.Evaluate(ctx, evalReq)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "evaluation failed: %v", err)
	}

	return evalResponseToPb(resp), nil
}

func (s *Server) BatchEvaluate(ctx context.Context, req *pb.BatchEvaluateRequest) (*pb.BatchEvaluateResponse, error) {
	if req.TenantId == "" {
		return nil, status.Error(codes.InvalidArgument, "tenant_id is required")
	}
	if len(req.Payloads) == 0 {
		return nil, status.Error(codes.InvalidArgument, "at least one payload is required")
	}

	var evaluations []*pb.EvaluatePolicyResponse

	for _, payload := range req.Payloads {
		evalReq := core.EvaluateRequest{
			TenantID:    req.TenantId,
			Payload:     payload.AsMap(),
			LabelFilter: req.LabelFilter,
		}

		resp, err := s.engine.Evaluate(ctx, evalReq)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "batch evaluation failed: %v", err)
		}

		evaluations = append(evaluations, evalResponseToPb(resp))
	}

	return &pb.BatchEvaluateResponse{
		Evaluations: evaluations,
	}, nil
}

// ────────────────────────────────────────────────────────────────────────
// Mappers
// ────────────────────────────────────────────────────────────────────────

func domainToPb(p *core.Policy) *pb.Policy {
	return &pb.Policy{
		Id:          p.ID,
		TenantId:    p.TenantID,
		Name:        p.Name,
		Description: p.Description,
		Expression:  p.Expression,
		Priority:    p.Priority,
		Enabled:     p.Enabled,
		Labels:      p.Labels,
		CreatedAt:   timestamppb.New(p.CreatedAt),
		UpdatedAt:   timestamppb.New(p.UpdatedAt),
	}
}

func evalResponseToPb(resp *core.EvaluateResponse) *pb.EvaluatePolicyResponse {
	pbResults := make([]*pb.PolicyResult, len(resp.Results))
	for i, r := range resp.Results {
		pbResults[i] = &pb.PolicyResult{
			PolicyId:   r.PolicyID,
			PolicyName: r.PolicyName,
			Passed:     r.Passed,
			Error:      r.Error,
			Expression: r.Expression,
		}
	}

	return &pb.EvaluatePolicyResponse{
		AllPassed:      resp.AllPassed,
		Results:        pbResults,
		TotalEvaluated: resp.TotalEvaluated,
	}
}
