package grpc

import (
	"context"
	"fmt"

	pb "github.com/autorix/nexus/api/autorix/nexus/v1"
	"github.com/autorix/nexus/internal/core"
	"google.golang.org/protobuf/types/known/structpb"
)

// Repository defines tuple and caveat storage required by gRPC operations
type Repository interface {
	WriteTuples(ctx context.Context, tuples []core.Tuple) error
	DeleteTuples(ctx context.Context, tuples []core.Tuple) error
	WriteCaveat(ctx context.Context, caveat core.CaveatDefinition) error
}

// Option configures the gRPC Server
type Option func(*Server)

// WithRepository sets the repository for persistence operations
func WithRepository(repo Repository) Option {
	return func(s *Server) {
		s.repo = repo
	}
}

// WithCaveatEvaluator sets the caveat evaluator for CEL validation
func WithCaveatEvaluator(ce core.CaveatEvaluator) Option {
	return func(s *Server) {
		s.caveatEvaluator = ce
	}
}

// Server implements the NexusService gRPC server
type Server struct {
	pb.UnimplementedNexusServiceServer
	engine          core.GraphEngine
	repo            Repository
	caveatEvaluator core.CaveatEvaluator
}

func NewServer(engine core.GraphEngine, opts ...Option) *Server {
	s := &Server{
		engine: engine,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Check maps the gRPC request to the domain layer and executes graph resolution
func (s *Server) Check(ctx context.Context, req *pb.CheckRequest) (*pb.CheckResponse, error) {
	var reqCtx map[string]interface{}
	if req.RequestContext != nil {
		reqCtx = req.RequestContext.AsMap()
	}

	domainReq := core.CheckRequest{
		Namespace: req.Namespace,
		Object:    req.Object,
		Relation:  req.Relation,
		Subject: core.Tuple{
			Namespace: req.SubjectNamespace,
			Object:    req.SubjectId,
			Relation:  req.SubjectRelation,
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

// WriteTuples inserts a batch of relation tuples
func (s *Server) WriteTuples(ctx context.Context, req *pb.WriteTuplesRequest) (*pb.WriteTuplesResponse, error) {
	tuples := make([]core.Tuple, 0, len(req.Tuples))
	for _, t := range req.Tuples {
		var caveatCtx map[string]interface{}
		if t.CaveatContext != nil {
			caveatCtx = t.CaveatContext.AsMap()
		}
		subjectNs := t.SubjectNamespace
		if subjectNs == "" {
			subjectNs = "user"
		}
		tuples = append(tuples, core.Tuple{
			Namespace:        t.Namespace,
			Object:           t.Object,
			Relation:         t.Relation,
			SubjectNamespace: subjectNs,
			SubjectObject:    t.SubjectId,
			SubjectRelation:  t.SubjectRelation,
			CaveatName:       t.CaveatName,
			CaveatContext:    caveatCtx,
		})
	}

	if s.repo != nil {
		if err := s.repo.WriteTuples(ctx, tuples); err != nil {
			return nil, fmt.Errorf("failed to write tuples: %w", err)
		}
	}

	return &pb.WriteTuplesResponse{}, nil
}

// DeleteTuples removes a batch of relation tuples
func (s *Server) DeleteTuples(ctx context.Context, req *pb.DeleteTuplesRequest) (*pb.DeleteTuplesResponse, error) {
	tuples := make([]core.Tuple, 0, len(req.Tuples))
	for _, t := range req.Tuples {
		subjectNs := t.SubjectNamespace
		if subjectNs == "" {
			subjectNs = "user"
		}
		tuples = append(tuples, core.Tuple{
			Namespace:        t.Namespace,
			Object:           t.Object,
			Relation:         t.Relation,
			SubjectNamespace: subjectNs,
			SubjectObject:    t.SubjectId,
			SubjectRelation:  t.SubjectRelation,
		})
	}

	if s.repo != nil {
		if err := s.repo.DeleteTuples(ctx, tuples); err != nil {
			return nil, fmt.Errorf("failed to delete tuples: %w", err)
		}
	}

	return &pb.DeleteTuplesResponse{}, nil
}

// WriteCaveats compiles and saves caveat definitions
func (s *Server) WriteCaveats(ctx context.Context, req *pb.WriteCaveatsRequest) (*pb.WriteCaveatsResponse, error) {
	for _, c := range req.Caveats {
		if s.caveatEvaluator != nil {
			if err := s.caveatEvaluator.Validate(c.CelExpression); err != nil {
				return nil, fmt.Errorf("invalid CEL expression for caveat %q: %w", c.Name, err)
			}
		}

		if s.repo != nil {
			if err := s.repo.WriteCaveat(ctx, core.CaveatDefinition{
				Name:          c.Name,
				CELExpression: c.CelExpression,
			}); err != nil {
				return nil, fmt.Errorf("failed to write caveat %q: %w", c.Name, err)
			}
		}
	}

	return &pb.WriteCaveatsResponse{}, nil
}

// Expand computes the full relationship tree for a resource
func (s *Server) Expand(ctx context.Context, req *pb.ExpandRequest) (*pb.ExpandResponse, error) {
	tree, err := s.engine.Expand(ctx, core.ExpandRequest{
		Namespace: req.Namespace,
		Object:    req.Object,
		Relation:  req.Relation,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to expand relation: %w", err)
	}

	return &pb.ExpandResponse{
		Tree: toProtoTreeNode(tree),
	}, nil
}

func toProtoTreeNode(node *core.ExpandTree) *pb.ExpandResponse_TreeNode {
	if node == nil {
		return nil
	}

	pbNode := &pb.ExpandResponse_TreeNode{
		Type: node.Type,
	}

	if node.Tuple != nil {
		var caveatCtx *structpb.Struct
		if node.Tuple.CaveatContext != nil {
			caveatCtx, _ = structpb.NewStruct(node.Tuple.CaveatContext)
		}
		pbNode.Tuple = &pb.RelationTuple{
			Namespace:        node.Tuple.Namespace,
			Object:           node.Tuple.Object,
			Relation:         node.Tuple.Relation,
			SubjectNamespace: node.Tuple.SubjectNamespace,
			SubjectId:        node.Tuple.SubjectObject,
			SubjectRelation:  node.Tuple.SubjectRelation,
			CaveatName:       node.Tuple.CaveatName,
			CaveatContext:    caveatCtx,
		}
	}

	for _, child := range node.Children {
		pbNode.Children = append(pbNode.Children, toProtoTreeNode(child))
	}

	return pbNode
}
