package grpc

import (
	"context"

	"github.com/autorix/nexus/internal/core"
	pb "github.com/autorix/nexus/api/autorix/nexus/v1"
)

// Server implements the NexusService gRPC server
type Server struct {
	pb.UnimplementedNexusServiceServer
	engine core.GraphEngine
}

func NewServer(engine core.GraphEngine) *Server {
	return &Server{
		engine: engine,
	}
}

// Check maps the gRPC request to the domain layer and executes graph resolution
func (s *Server) Check(ctx context.Context, req *pb.CheckRequest) (*pb.CheckResponse, error) {
	// Parse dynamic request context if present
	var reqCtx map[string]interface{}
	if req.RequestContext != nil {
		reqCtx = req.RequestContext.AsMap()
	}

	domainReq := core.CheckRequest{
		Namespace: req.Namespace,
		Object:    req.Object,
		Relation:  req.Relation,
		Subject: core.Tuple{
			Namespace:       req.SubjectNamespace,
			Object:          req.SubjectId,
			Relation:        req.SubjectRelation,
		},
		RequestContext: reqCtx,
	}

	res, err := s.engine.Check(ctx, domainReq)
	if err != nil {
		return nil, err
	}

	return &pb.CheckResponse{
		Allowed: res.Allowed,
		Reason:  res.Reason,
	}, nil
}

// WriteTuples is mocked here for brevity, usually writes to DB
func (s *Server) WriteTuples(ctx context.Context, req *pb.WriteTuplesRequest) (*pb.WriteTuplesResponse, error) {
	// Map pb.RelationTuple to core.Tuple and pass to engine/repo
	return &pb.WriteTuplesResponse{}, nil
}

func (s *Server) WriteCaveats(ctx context.Context, req *pb.WriteCaveatsRequest) (*pb.WriteCaveatsResponse, error) {
	return &pb.WriteCaveatsResponse{}, nil
}

func (s *Server) Expand(ctx context.Context, req *pb.ExpandRequest) (*pb.ExpandResponse, error) {
	return &pb.ExpandResponse{}, nil
}
