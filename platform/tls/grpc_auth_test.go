package autortls

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"net/url"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

func TestUnaryInternalAuthInterceptor(t *testing.T) {
	okHandler := func(ctx context.Context, _ any) (any, error) {
		return PeerIdentity(ctx), nil
	}

	t.Run("disabled when unconfigured", func(t *testing.T) {
		ic := UnaryInternalAuthInterceptor(GRPCInternalAuthConfig{})
		if _, err := ic(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/nexus.v1.Check/Check"}, okHandler); err != nil {
			t.Fatalf("expected pass-through, got %v", err)
		}
	})

	t.Run("health probe bypasses auth", func(t *testing.T) {
		ic := UnaryInternalAuthInterceptor(GRPCInternalAuthConfig{Required: true})
		if _, err := ic(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/grpc.health.v1.Health/Check"}, okHandler); err != nil {
			t.Fatalf("expected health probe bypass, got %v", err)
		}
	})

	t.Run("rejects missing token", func(t *testing.T) {
		ic := UnaryInternalAuthInterceptor(GRPCInternalAuthConfig{Secret: "grpc-secret"})
		_, err := ic(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/nexus.v1.Check/Check"}, okHandler)
		if status.Code(err) != codes.Unauthenticated {
			t.Fatalf("expected Unauthenticated, got %v", err)
		}
	})

	t.Run("accepts valid metadata token", func(t *testing.T) {
		ic := UnaryInternalAuthInterceptor(GRPCInternalAuthConfig{Secret: "grpc-secret"})
		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcInternalTokenMetadata, "grpc-secret"))
		if _, err := ic(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/nexus.v1.Check/Check"}, okHandler); err != nil {
			t.Fatalf("expected success with valid metadata token, got %v", err)
		}
	})

	t.Run("accepts allowed mTLS SPIFFE peer and denies unauthorized peer", func(t *testing.T) {
		ic := UnaryInternalAuthInterceptor(GRPCInternalAuthConfig{
			Required:              true,
			AllowedSPIFFEServices: []string{"aegis"},
		})

		aegisURI, _ := url.Parse("spiffe://autorix.internal/service/aegis")
		goodCtx := peer.NewContext(context.Background(), &peer.Peer{
			AuthInfo: credentials.TLSInfo{
				State: tls.ConnectionState{
					VerifiedChains: [][]*x509.Certificate{{
						{
							Subject: pkix.Name{CommonName: "aegis"},
							URIs:    []*url.URL{aegisURI},
						},
					}},
				},
			},
		})
		res, err := ic(goodCtx, nil, &grpc.UnaryServerInfo{FullMethod: "/nexus.v1.Check/Check"}, okHandler)
		if err != nil || res != "aegis" {
			t.Fatalf("expected allowed mTLS peer, got res=%v err=%v", res, err)
		}

		badURI, _ := url.Parse("spiffe://autorix.internal/service/rogue")
		badCtx := peer.NewContext(context.Background(), &peer.Peer{
			AuthInfo: credentials.TLSInfo{
				State: tls.ConnectionState{
					VerifiedChains: [][]*x509.Certificate{{
						{
							Subject: pkix.Name{CommonName: "rogue"},
							URIs:    []*url.URL{badURI},
						},
					}},
				},
			},
		})
		_, err = ic(badCtx, nil, &grpc.UnaryServerInfo{FullMethod: "/nexus.v1.Check/Check"}, okHandler)
		if status.Code(err) != codes.PermissionDenied {
			t.Fatalf("expected PermissionDenied for rogue mTLS peer, got %v", err)
		}
	})

	t.Run("accepts bearer and console metadata tokens and CN-only mTLS cert", func(t *testing.T) {
		ic := UnaryInternalAuthInterceptor(GRPCInternalAuthConfig{Secret: "grpc-secret", AllowedSPIFFEServices: []string{"console"}})
		bearerCtx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer grpc-secret"))
		if _, err := ic(bearerCtx, nil, &grpc.UnaryServerInfo{FullMethod: "/nexus.v1.Check/Check"}, okHandler); err != nil {
			t.Fatalf("expected bearer metadata token to succeed, got %v", err)
		}
		consoleCtx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcConsoleTokenMetadata, "grpc-secret"))
		if _, err := ic(consoleCtx, nil, &grpc.UnaryServerInfo{FullMethod: "/nexus.v1.Check/Check"}, okHandler); err != nil {
			t.Fatalf("expected console metadata token to succeed, got %v", err)
		}
		cnCtx := peer.NewContext(context.Background(), &peer.Peer{
			AuthInfo: credentials.TLSInfo{
				State: tls.ConnectionState{
					VerifiedChains: [][]*x509.Certificate{{{Subject: pkix.Name{CommonName: "console"}}}},
				},
			},
		})
		res, err := ic(cnCtx, nil, &grpc.UnaryServerInfo{FullMethod: "/nexus.v1.Check/Check"}, okHandler)
		if err != nil || res != "console" {
			t.Fatalf("expected CN-only mTLS cert to succeed, res=%v err=%v", res, err)
		}
	})
}

func TestGRPCInternalAuthConfigFromEnv(t *testing.T) {
	t.Setenv("AUTORIX_INTERNAL_AUTH_SECRET", "")
	t.Setenv("AUTORIX_CONSOLE_INTERNAL_TOKEN", "console-fallback")
	t.Setenv("AUTORIX_REQUIRE_INTERNAL_AUTH", "1")
	t.Setenv("AUTORIX_ALLOWED_SPIFFE_SERVICES", "aegis, console")

	cfg := GRPCInternalAuthConfigFromEnv()
	if cfg.Secret != "console-fallback" || !cfg.Required {
		t.Fatalf("unexpected env config: %+v", cfg)
	}
	if len(cfg.AllowedSPIFFEServices) != 2 || cfg.AllowedSPIFFEServices[0] != "aegis" || cfg.AllowedSPIFFEServices[1] != "console" {
		t.Fatalf("unexpected allowed SPIFFE services: %+v", cfg.AllowedSPIFFEServices)
	}
}
