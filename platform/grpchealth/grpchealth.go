// Package grpchealth implements the standard grpc.health.v1.Health service
// (P1-S1-T6) over the same health.Checker an engine's HTTP /health/ready
// uses, so a gRPC-native probe and an HTTP probe can never disagree — both
// evaluate the identical checks live, on every call.
package grpchealth

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"

	"github.com/autorix/platform/health"
)

// Server implements healthpb.HealthServer over a health.Checker.
type Server struct {
	healthpb.UnimplementedHealthServer
	checker  *health.Checker
	services map[string]struct{}
}

// Register builds a Server for checker and, when srv is non-nil, registers
// it as the grpc.health.v1.Health implementation on srv. serviceNames are
// the fully-qualified gRPC service names this health check may be queried
// about (e.g. "autorix.nexus.v1.NexusService"); the empty service name
// always denotes overall server health and needs no declaration.
func Register(srv *grpc.Server, checker *health.Checker, serviceNames ...string) *Server {
	services := make(map[string]struct{}, len(serviceNames))
	for _, name := range serviceNames {
		services[name] = struct{}{}
	}
	s := &Server{checker: checker, services: services}
	if srv != nil {
		healthpb.RegisterHealthServer(srv, s)
	}
	return s
}

// Check evaluates the underlying readiness checks live and reports SERVING
// or NOT_SERVING. A service name that was never declared to Register is
// reported as NotFound, matching the standard health-checking protocol.
func (s *Server) Check(ctx context.Context, req *healthpb.HealthCheckRequest) (*healthpb.HealthCheckResponse, error) {
	if req.GetService() != "" {
		if _, known := s.services[req.GetService()]; !known {
			return nil, status.Errorf(codes.NotFound, "unknown service %q", req.GetService())
		}
	}

	ok, _ := s.checker.Ready(ctx)
	resp := &healthpb.HealthCheckResponse{Status: healthpb.HealthCheckResponse_SERVING}
	if !ok {
		resp.Status = healthpb.HealthCheckResponse_NOT_SERVING
	}
	return resp, nil
}

// Watch is intentionally unimplemented: callers should poll Check, which is
// as cheap as the readiness checks it wraps and always reflects live state.
func (s *Server) Watch(req *healthpb.HealthCheckRequest, stream healthpb.Health_WatchServer) error {
	return status.Error(codes.Unimplemented, "watch is not supported; poll Check")
}
