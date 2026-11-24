package autortls

import (
	"context"
	"crypto/subtle"
	"os"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

const (
	grpcInternalTokenMetadata = "x-autorix-internal-token" //nolint:gosec // metadata header name, not a credential
	grpcConsoleTokenMetadata  = "x-autorix-console-token"  //nolint:gosec // metadata header name, not a credential
)

// GRPCInternalAuthConfig configures the gRPC unary internal authentication interceptor.
type GRPCInternalAuthConfig struct {
	Secret                string
	Required              bool
	AllowedSPIFFEServices []string
}

// GRPCInternalAuthConfigFromEnv reads standard Autorix internal auth env vars.
func GRPCInternalAuthConfigFromEnv() GRPCInternalAuthConfig {
	secret := strings.TrimSpace(os.Getenv("AUTORIX_INTERNAL_AUTH_SECRET"))
	if secret == "" {
		secret = strings.TrimSpace(os.Getenv("AUTORIX_CONSOLE_INTERNAL_TOKEN"))
	}
	reqEnv := strings.TrimSpace(os.Getenv("AUTORIX_REQUIRE_INTERNAL_AUTH"))
	required := strings.EqualFold(reqEnv, "true") || reqEnv == "1"

	var allowed []string
	if raw := strings.TrimSpace(os.Getenv("AUTORIX_ALLOWED_SPIFFE_SERVICES")); raw != "" {
		for _, item := range strings.Split(raw, ",") {
			if svc := strings.ToLower(strings.TrimSpace(item)); svc != "" {
				allowed = append(allowed, svc)
			}
		}
	}
	return GRPCInternalAuthConfig{
		Secret:                secret,
		Required:              required,
		AllowedSPIFFEServices: allowed,
	}
}

// UnaryInternalAuthInterceptor enforces either verified mTLS client certificates
// or constant-time shared internal token metadata on gRPC unary RPCs, while
// exempting standard gRPC health probes (/grpc.health.v1.Health/).
func UnaryInternalAuthInterceptor(cfg GRPCInternalAuthConfig) grpc.UnaryServerInterceptor {
	secretBytes := []byte(strings.TrimSpace(cfg.Secret))
	allowed := make(map[string]bool, len(cfg.AllowedSPIFFEServices))
	for _, svc := range cfg.AllowedSPIFFEServices {
		if norm := strings.ToLower(strings.TrimSpace(svc)); norm != "" {
			allowed[norm] = true
		}
	}

	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if !cfg.Required && len(secretBytes) == 0 {
			return handler(ctx, req)
		}
		if info != nil && strings.HasPrefix(info.FullMethod, "/grpc.health.v1.Health/") {
			return handler(ctx, req)
		}

		// 1. Check verified mTLS peer certificate from gRPC transport.
		if p, ok := peer.FromContext(ctx); ok && p != nil {
			if tlsInfo, ok := p.AuthInfo.(credentials.TLSInfo); ok &&
				len(tlsInfo.State.VerifiedChains) > 0 && len(tlsInfo.State.VerifiedChains[0]) > 0 {
				cert := tlsInfo.State.VerifiedChains[0][0]
				hasSpiffe := false
				var matchedSpiffe string
				for _, u := range cert.URIs {
					if u == nil || !strings.EqualFold(u.Scheme, "spiffe") {
						continue
					}
					hasSpiffe = true
					if svc, ok := ExtractServiceFromSPIFFE(u.String()); ok {
						if len(allowed) == 0 || allowed[svc] {
							matchedSpiffe = u.String()
							break
						}
					}
				}
				authorized := matchedSpiffe != ""
				if !hasSpiffe && (len(allowed) == 0 || allowed[strings.ToLower(cert.Subject.CommonName)]) {
					authorized = true
				}
				if !authorized {
					return nil, status.Error(codes.PermissionDenied, "peer mTLS identity is not authorized")
				}
				ctx = context.WithValue(ctx, PeerIdentityKey, cert.Subject.CommonName)
				if matchedSpiffe != "" {
					ctx = context.WithValue(ctx, PeerSpiffeIDKey, matchedSpiffe)
				}
				return handler(ctx, req)
			}
		}

		// 2. Check incoming gRPC metadata token.
		if len(secretBytes) > 0 {
			if md, ok := metadata.FromIncomingContext(ctx); ok {
				for _, key := range []string{grpcInternalTokenMetadata, grpcConsoleTokenMetadata} {
					for _, val := range md.Get(key) {
						if subtle.ConstantTimeCompare([]byte(val), secretBytes) == 1 {
							return handler(ctx, req)
						}
					}
				}
				for _, val := range md.Get("authorization") {
					if len(val) > 7 && strings.EqualFold(val[:7], "Bearer ") {
						if subtle.ConstantTimeCompare([]byte(strings.TrimSpace(val[7:])), secretBytes) == 1 {
							return handler(ctx, req)
						}
					}
				}
			}
		}

		return nil, status.Error(codes.Unauthenticated, "internal service authentication required")
	}
}
