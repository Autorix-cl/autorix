package grpc

import (
	"context"
	"errors"
	"testing"

	"github.com/autorix/platform/grpchealth"
	"github.com/autorix/platform/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

// TestGRPCHealth_NotServingOnFailingCheck confirms the standard
// grpc.health.v1.Health service reports NOT_SERVING when the underlying
// health.Checker has a failing registered check (ADR 0001 / P1-S1-T6). No
// real gRPC server needs to be started: Check() is called directly.
func TestGRPCHealth_NotServingOnFailingCheck(t *testing.T) {
	checker := health.NewChecker()
	checker.Register("postgres", func(ctx context.Context) error {
		return errors.New("dependency unreachable")
	})

	srv := grpchealth.Register(nil, checker, "autorix.nexus.v1.NexusService")

	resp, err := srv.Check(context.Background(), &healthpb.HealthCheckRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != healthpb.HealthCheckResponse_NOT_SERVING {
		t.Errorf("expected NOT_SERVING, got %v", resp.Status)
	}
}
