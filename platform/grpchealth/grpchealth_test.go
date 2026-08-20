package grpchealth_test

import (
	"context"
	"errors"
	"testing"

	"github.com/autorix/platform/grpchealth"
	"github.com/autorix/platform/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

func TestRegister_ServingWhenChecksPass(t *testing.T) {
	c := health.NewChecker()
	c.Register("postgres", func(ctx context.Context) error { return nil })

	srv := grpchealth.Register(nil, c, "autorix.nexus.v1.NexusService")

	resp, err := srv.Check(context.Background(), &healthpb.HealthCheckRequest{Service: "autorix.nexus.v1.NexusService"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != healthpb.HealthCheckResponse_SERVING {
		t.Fatalf("expected SERVING, got %v", resp.Status)
	}
}

func TestRegister_NotServingWhenACheckFails(t *testing.T) {
	c := health.NewChecker()
	c.Register("postgres", func(ctx context.Context) error { return errors.New("down") })

	srv := grpchealth.Register(nil, c, "autorix.nexus.v1.NexusService")

	resp, err := srv.Check(context.Background(), &healthpb.HealthCheckRequest{Service: "autorix.nexus.v1.NexusService"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != healthpb.HealthCheckResponse_NOT_SERVING {
		t.Fatalf("expected NOT_SERVING, got %v", resp.Status)
	}
}

func TestRegister_EmptyServiceNameAlsoReportsOverallStatus(t *testing.T) {
	c := health.NewChecker()
	c.Register("postgres", func(ctx context.Context) error { return nil })

	srv := grpchealth.Register(nil, c, "autorix.nexus.v1.NexusService")

	resp, err := srv.Check(context.Background(), &healthpb.HealthCheckRequest{Service: ""})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != healthpb.HealthCheckResponse_SERVING {
		t.Fatalf("expected SERVING for the empty (overall) service name, got %v", resp.Status)
	}
}

func TestRegister_UnknownServiceNameIsUnimplemented(t *testing.T) {
	c := health.NewChecker()

	srv := grpchealth.Register(nil, c, "autorix.nexus.v1.NexusService")

	_, err := srv.Check(context.Background(), &healthpb.HealthCheckRequest{Service: "not.registered"})
	if err == nil {
		t.Fatalf("expected an error for an unregistered service name")
	}
}
