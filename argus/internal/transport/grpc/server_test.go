package grpc_test

import (
	"context"
	cryptorand "crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	argusv1 "github.com/autorix/argus/api/autorix/argus/v1"
	"github.com/autorix/argus/internal/core"
	"github.com/autorix/argus/internal/credential"
	"github.com/autorix/argus/internal/storage/postgres"
	grpchandler "github.com/autorix/argus/internal/transport/grpc"
	"github.com/autorix/platform/pgtest"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func testEncKey(t *testing.T) []byte {
	t.Helper()
	key := make([]byte, credential.KeySize)
	if _, err := cryptorand.Read(key); err != nil {
		t.Fatalf("generating test encryption key: %v", err)
	}
	return key
}

func newTestServer(t *testing.T) (*grpchandler.Server, *postgres.Repository) {
	t.Helper()
	pool := pgtest.StartPostgres(t, "../../../migrations")
	repo := postgres.NewRepository(pool)
	return grpchandler.NewServer(repo, testEncKey(t)), repo
}

func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// signHeartbeat builds a valid HMAC signature (and matching
// timestamp/nonce) for a heartbeat sent with rawCredential, mirroring what
// a real engine's registration client does (P2-S3-T4).
func signHeartbeat(rawCredential, instanceUUID string, live, ready bool, nonce string) (timestampUnix int64, sig string) {
	now := time.Now().Unix()
	return now, credential.HeartbeatSignature(rawCredential, instanceUUID, now, nonce, live, ready)
}

func TestRegister_ValidTokenCreatesInstanceAndCredential(t *testing.T) {
	ctx := context.Background()
	srv, repo := newTestServer(t)

	env, err := repo.CreateEnvironment(ctx, "Staging", "staging", "", false)
	if err != nil {
		t.Fatalf("CreateEnvironment: %v", err)
	}

	rawToken := "raw-token-abc"
	if _, err := repo.CreateEnrollmentToken(ctx, core.EnrollmentToken{
		TokenHash:     hashToken(rawToken),
		EngineType:    "ego",
		EnvironmentID: env.ID,
		UsesAllowed:   1,
		ExpiresAt:     time.Now().Add(1 * time.Hour),
		CreatedBy:     "admin@autorix",
	}); err != nil {
		t.Fatalf("CreateEnrollmentToken: %v", err)
	}

	resp, err := srv.Register(ctx, &argusv1.RegisterRequest{
		EnrollmentToken: rawToken,
		EngineType:      "ego",
		InstanceId:      "host-1",
		Environment:     "staging",
		Version:         "1.0.0",
		Endpoints:       &argusv1.Endpoints{RestUrl: "http://host-1:4433"},
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if resp.GetInstanceUuid() == "" {
		t.Fatal("expected a non-empty instance uuid")
	}
	if resp.GetInstanceCredential() == "" {
		t.Fatal("expected a non-empty instance credential")
	}
	if resp.GetHeartbeatIntervalSeconds() != int32(core.HeartbeatInterval/time.Second) {
		t.Fatalf("unexpected heartbeat interval: %d", resp.GetHeartbeatIntervalSeconds())
	}

	// The token was single-use: a second Register with the same token must
	// fail instead of silently succeeding again.
	_, err = srv.Register(ctx, &argusv1.RegisterRequest{
		EnrollmentToken: rawToken,
		EngineType:      "ego",
		InstanceId:      "host-2",
		Environment:     "staging",
	})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("expected PermissionDenied reusing a single-use token, got %v", err)
	}
}

func TestRegister_InvalidTokenIsRejectedNotPanicked(t *testing.T) {
	ctx := context.Background()
	srv, repo := newTestServer(t)

	if _, err := repo.CreateEnvironment(ctx, "Staging", "staging", "", false); err != nil {
		t.Fatalf("CreateEnvironment: %v", err)
	}

	_, err := srv.Register(ctx, &argusv1.RegisterRequest{
		EnrollmentToken: "does-not-exist",
		EngineType:      "ego",
		InstanceId:      "host-1",
		Environment:     "staging",
	})
	if err == nil {
		t.Fatal("expected an error for an unknown enrollment token")
	}
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("expected PermissionDenied for unknown token, got %v", err)
	}
}

func TestRegister_ExpiredTokenIsRejected(t *testing.T) {
	ctx := context.Background()
	srv, repo := newTestServer(t)

	env, err := repo.CreateEnvironment(ctx, "Staging", "staging", "", false)
	if err != nil {
		t.Fatalf("CreateEnvironment: %v", err)
	}
	rawToken := "expired-token"
	if _, err := repo.CreateEnrollmentToken(ctx, core.EnrollmentToken{
		TokenHash:     hashToken(rawToken),
		EngineType:    "ego",
		EnvironmentID: env.ID,
		UsesAllowed:   1,
		ExpiresAt:     time.Now().Add(-1 * time.Hour),
		CreatedBy:     "admin@autorix",
	}); err != nil {
		t.Fatalf("CreateEnrollmentToken: %v", err)
	}

	_, err = srv.Register(ctx, &argusv1.RegisterRequest{
		EnrollmentToken: rawToken,
		EngineType:      "ego",
		InstanceId:      "host-1",
		Environment:     "staging",
	})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("expected PermissionDenied for expired token, got %v", err)
	}
}

func TestRegister_TokenScopedToWrongEngineTypeIsRejected(t *testing.T) {
	ctx := context.Background()
	srv, repo := newTestServer(t)

	env, err := repo.CreateEnvironment(ctx, "Staging", "staging", "", false)
	if err != nil {
		t.Fatalf("CreateEnvironment: %v", err)
	}
	rawToken := "scoped-token"
	if _, err := repo.CreateEnrollmentToken(ctx, core.EnrollmentToken{
		TokenHash:     hashToken(rawToken),
		EngineType:    "ego",
		EnvironmentID: env.ID,
		UsesAllowed:   1,
		ExpiresAt:     time.Now().Add(1 * time.Hour),
		CreatedBy:     "admin@autorix",
	}); err != nil {
		t.Fatalf("CreateEnrollmentToken: %v", err)
	}

	_, err = srv.Register(ctx, &argusv1.RegisterRequest{
		EnrollmentToken: rawToken,
		EngineType:      "janus", // token is scoped to "ego"
		InstanceId:      "host-1",
		Environment:     "staging",
	})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("expected PermissionDenied for wrong-scope token, got %v", err)
	}
}

func registerTestInstance(t *testing.T, ctx context.Context, srv *grpchandler.Server, repo *postgres.Repository) *argusv1.RegisterResponse {
	t.Helper()
	env, err := repo.CreateEnvironment(ctx, "Staging", "staging-hb", "", false)
	if err != nil {
		t.Fatalf("CreateEnvironment: %v", err)
	}
	rawToken := "hb-token"
	if _, err := repo.CreateEnrollmentToken(ctx, core.EnrollmentToken{
		TokenHash:     hashToken(rawToken),
		EngineType:    "ego",
		EnvironmentID: env.ID,
		UsesAllowed:   1,
		ExpiresAt:     time.Now().Add(1 * time.Hour),
		CreatedBy:     "admin@autorix",
	}); err != nil {
		t.Fatalf("CreateEnrollmentToken: %v", err)
	}
	resp, err := srv.Register(ctx, &argusv1.RegisterRequest{
		EnrollmentToken: rawToken,
		EngineType:      "ego",
		InstanceId:      "host-hb",
		Environment:     "staging-hb",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	return resp
}

func TestHeartbeat_ValidCredentialUpdatesStatus(t *testing.T) {
	ctx := context.Background()
	srv, repo := newTestServer(t)
	regResp := registerTestInstance(t, ctx, srv, repo)

	ts, sig := signHeartbeat(regResp.GetInstanceCredential(), regResp.GetInstanceUuid(), true, true, "nonce-1")
	hbResp, err := srv.Heartbeat(ctx, &argusv1.HeartbeatRequest{
		InstanceUuid:  regResp.GetInstanceUuid(),
		Live:          true,
		Ready:         true,
		TimestampUnix: ts,
		Nonce:         "nonce-1",
		Signature:     sig,
	})
	if err != nil {
		t.Fatalf("Heartbeat: %v", err)
	}
	if hbResp.GetStatus() != string(core.StatusHealthy) {
		t.Fatalf("expected healthy status after ready heartbeat, got %s", hbResp.GetStatus())
	}

	// A heartbeat reporting not-ready should degrade the instance. Uses a
	// fresh nonce — reusing "nonce-1" must be rejected as a replay
	// (covered separately below).
	ts2, sig2 := signHeartbeat(regResp.GetInstanceCredential(), regResp.GetInstanceUuid(), true, false, "nonce-2")
	hbResp, err = srv.Heartbeat(ctx, &argusv1.HeartbeatRequest{
		InstanceUuid:  regResp.GetInstanceUuid(),
		Live:          true,
		Ready:         false,
		TimestampUnix: ts2,
		Nonce:         "nonce-2",
		Signature:     sig2,
	})
	if err != nil {
		t.Fatalf("Heartbeat (not ready): %v", err)
	}
	if hbResp.GetStatus() != string(core.StatusDegraded) {
		t.Fatalf("expected degraded status after not-ready heartbeat, got %s", hbResp.GetStatus())
	}
}

func TestHeartbeat_InvalidSignatureIsRejected(t *testing.T) {
	ctx := context.Background()
	srv, repo := newTestServer(t)
	regResp := registerTestInstance(t, ctx, srv, repo)

	ts := time.Now().Unix()
	_, err := srv.Heartbeat(ctx, &argusv1.HeartbeatRequest{
		InstanceUuid:  regResp.GetInstanceUuid(),
		Live:          true,
		Ready:         true,
		TimestampUnix: ts,
		Nonce:         "nonce-bad-sig",
		Signature:     "0000000000000000000000000000000000000000000000000000000000000000",
	})
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expected Unauthenticated for wrong signature, got %v", err)
	}
}

func TestHeartbeat_ReplayedNonceIsRejected(t *testing.T) {
	ctx := context.Background()
	srv, repo := newTestServer(t)
	regResp := registerTestInstance(t, ctx, srv, repo)

	ts, sig := signHeartbeat(regResp.GetInstanceCredential(), regResp.GetInstanceUuid(), true, true, "replay-nonce")
	req := &argusv1.HeartbeatRequest{
		InstanceUuid: regResp.GetInstanceUuid(), Live: true, Ready: true,
		TimestampUnix: ts, Nonce: "replay-nonce", Signature: sig,
	}
	if _, err := srv.Heartbeat(ctx, req); err != nil {
		t.Fatalf("first Heartbeat with this nonce should succeed: %v", err)
	}
	if _, err := srv.Heartbeat(ctx, req); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expected Unauthenticated replaying the same nonce, got %v", err)
	}
}

func TestHeartbeat_StaleTimestampIsRejected(t *testing.T) {
	ctx := context.Background()
	srv, repo := newTestServer(t)
	regResp := registerTestInstance(t, ctx, srv, repo)

	staleTs := time.Now().Add(-5 * time.Minute).Unix()
	sig := credential.HeartbeatSignature(regResp.GetInstanceCredential(), regResp.GetInstanceUuid(), staleTs, "stale-nonce", true, true)
	_, err := srv.Heartbeat(ctx, &argusv1.HeartbeatRequest{
		InstanceUuid: regResp.GetInstanceUuid(), Live: true, Ready: true,
		TimestampUnix: staleTs, Nonce: "stale-nonce", Signature: sig,
	})
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expected Unauthenticated for a stale timestamp, got %v", err)
	}
}

func TestGetInstance_ReturnsInstanceEventsAndDependencies(t *testing.T) {
	ctx := context.Background()
	srv, repo := newTestServer(t)
	regResp := registerTestInstance(t, ctx, srv, repo)
	id, err := uuid.Parse(regResp.GetInstanceUuid())
	if err != nil {
		t.Fatalf("parsing instance uuid: %v", err)
	}

	if err := repo.DeclareDependency(ctx, id, "vulcan"); err != nil {
		t.Fatalf("DeclareDependency: %v", err)
	}
	if err := repo.RecordDependencyProbe(ctx, id, "vulcan", true, 12.5); err != nil {
		t.Fatalf("RecordDependencyProbe: %v", err)
	}

	resp, err := srv.GetInstance(ctx, &argusv1.GetInstanceRequest{InstanceUuid: regResp.GetInstanceUuid()})
	if err != nil {
		t.Fatalf("GetInstance: %v", err)
	}
	if resp.GetInstance() == nil || resp.GetInstance().GetId() != regResp.GetInstanceUuid() {
		t.Fatalf("expected the instance in the response, got %+v", resp.GetInstance())
	}
	if resp.GetInstance().GetEnvironment() != "staging-hb" {
		t.Fatalf("expected environment slug staging-hb, got %s", resp.GetInstance().GetEnvironment())
	}
	if len(resp.GetEvents()) == 0 {
		t.Fatal("expected at least the 'registered' event")
	}
	if len(resp.GetDependencies()) != 1 || resp.GetDependencies()[0].GetDependsOnEngineType() != "vulcan" {
		t.Fatalf("expected 1 dependency edge on vulcan, got %+v", resp.GetDependencies())
	}
	if !resp.GetDependencies()[0].GetLastProbeReachable() {
		t.Fatal("expected the dependency to be reported reachable")
	}
}

func TestGetInstance_UnknownInstanceIsNotFound(t *testing.T) {
	ctx := context.Background()
	srv, _ := newTestServer(t)

	_, err := srv.GetInstance(ctx, &argusv1.GetInstanceRequest{InstanceUuid: "00000000-0000-0000-0000-000000000000"})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("expected NotFound for unknown instance, got %v", err)
	}
}

func TestGetInstance_InvalidUUIDIsInvalidArgument(t *testing.T) {
	ctx := context.Background()
	srv, _ := newTestServer(t)

	_, err := srv.GetInstance(ctx, &argusv1.GetInstanceRequest{InstanceUuid: "not-a-uuid"})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument for a malformed uuid, got %v", err)
	}
}

func TestListInstances_FiltersAndPaginates(t *testing.T) {
	ctx := context.Background()
	srv, repo := newTestServer(t)

	env, err := repo.CreateEnvironment(ctx, "List gRPC", "list-grpc-env", "", false)
	if err != nil {
		t.Fatalf("CreateEnvironment: %v", err)
	}
	for i := 0; i < 3; i++ {
		req := core.RegistrationRequest{
			EngineType:    "ego",
			InstanceID:    fmt.Sprintf("host-%d", i),
			EnvironmentID: env.ID,
			Version:       "1.0.0",
		}
		if _, err := repo.UpsertInstance(ctx, req, "admin", false); err != nil {
			t.Fatalf("UpsertInstance: %v", err)
		}
	}
	otherReq := core.RegistrationRequest{EngineType: "janus", InstanceID: "host-other", EnvironmentID: env.ID}
	if _, err := repo.UpsertInstance(ctx, otherReq, "admin", false); err != nil {
		t.Fatalf("UpsertInstance (other engine type): %v", err)
	}

	resp, err := srv.ListInstances(ctx, &argusv1.ListInstancesRequest{
		Environment: "list-grpc-env",
		EngineType:  "ego",
		Limit:       2,
	})
	if err != nil {
		t.Fatalf("ListInstances: %v", err)
	}
	if len(resp.GetInstances()) != 2 {
		t.Fatalf("expected page of 2 ego instances, got %d", len(resp.GetInstances()))
	}
	if !resp.GetHasMore() || resp.GetNextCursor() == "" {
		t.Fatalf("expected has_more with a non-empty cursor, got has_more=%v cursor=%q", resp.GetHasMore(), resp.GetNextCursor())
	}
	for _, inst := range resp.GetInstances() {
		if inst.GetEnvironment() != "list-grpc-env" {
			t.Fatalf("expected environment slug on every instance, got %+v", inst)
		}
	}

	resp2, err := srv.ListInstances(ctx, &argusv1.ListInstancesRequest{
		Environment: "list-grpc-env",
		EngineType:  "ego",
		Limit:       2,
		Cursor:      resp.GetNextCursor(),
	})
	if err != nil {
		t.Fatalf("ListInstances page 2: %v", err)
	}
	if len(resp2.GetInstances()) != 1 {
		t.Fatalf("expected exactly 1 remaining ego instance on page 2, got %d", len(resp2.GetInstances()))
	}
	if resp2.GetHasMore() {
		t.Fatal("expected no more pages after the second page")
	}
}

func TestListInstances_UnknownEnvironmentIsInvalidArgument(t *testing.T) {
	ctx := context.Background()
	srv, _ := newTestServer(t)

	_, err := srv.ListInstances(ctx, &argusv1.ListInstancesRequest{Environment: "does-not-exist"})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument for unknown environment, got %v", err)
	}
}

func TestHeartbeat_UnknownInstanceIsRejected(t *testing.T) {
	ctx := context.Background()
	srv, _ := newTestServer(t)

	_, err := srv.Heartbeat(ctx, &argusv1.HeartbeatRequest{
		InstanceUuid:  "00000000-0000-0000-0000-000000000000",
		Live:          true,
		Ready:         true,
		TimestampUnix: time.Now().Unix(),
		Nonce:         "unknown-instance-nonce",
		Signature:     "whatever",
	})
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expected Unauthenticated for unknown instance, got %v", err)
	}
}

func TestRotateCredential_NewCredentialWorksOldWorksDuringOverlap(t *testing.T) {
	ctx := context.Background()
	srv, repo := newTestServer(t)
	regResp := registerTestInstance(t, ctx, srv, repo)

	rotateResp, err := srv.RotateCredential(ctx, &argusv1.RotateCredentialRequest{
		InstanceUuid:      regResp.GetInstanceUuid(),
		CurrentCredential: regResp.GetInstanceCredential(),
	})
	if err != nil {
		t.Fatalf("RotateCredential: %v", err)
	}
	if rotateResp.GetNewCredential() == "" || rotateResp.GetNewCredential() == regResp.GetInstanceCredential() {
		t.Fatalf("expected a fresh, different credential, got %q", rotateResp.GetNewCredential())
	}
	if rotateResp.GetOverlapUntil() == nil {
		t.Fatal("expected a non-nil overlap_until")
	}

	// The new credential authenticates a heartbeat.
	ts, sig := signHeartbeat(rotateResp.GetNewCredential(), regResp.GetInstanceUuid(), true, true, "post-rotate-nonce")
	if _, err := srv.Heartbeat(ctx, &argusv1.HeartbeatRequest{
		InstanceUuid: regResp.GetInstanceUuid(), Live: true, Ready: true,
		TimestampUnix: ts, Nonce: "post-rotate-nonce", Signature: sig,
	}); err != nil {
		t.Fatalf("Heartbeat with new credential should succeed: %v", err)
	}

	// The old credential still authenticates during the overlap window.
	ts2, sig2 := signHeartbeat(regResp.GetInstanceCredential(), regResp.GetInstanceUuid(), true, true, "overlap-nonce")
	if _, err := srv.Heartbeat(ctx, &argusv1.HeartbeatRequest{
		InstanceUuid: regResp.GetInstanceUuid(), Live: true, Ready: true,
		TimestampUnix: ts2, Nonce: "overlap-nonce", Signature: sig2,
	}); err != nil {
		t.Fatalf("Heartbeat with old credential should still succeed during overlap: %v", err)
	}

	// Rotating again with the now-stale old credential must fail.
	if _, err := srv.RotateCredential(ctx, &argusv1.RotateCredentialRequest{
		InstanceUuid:      regResp.GetInstanceUuid(),
		CurrentCredential: regResp.GetInstanceCredential(),
	}); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expected Unauthenticated rotating with a stale current_credential, got %v", err)
	}
}

func TestRotateCredential_WrongCurrentCredentialIsRejected(t *testing.T) {
	ctx := context.Background()
	srv, repo := newTestServer(t)
	regResp := registerTestInstance(t, ctx, srv, repo)

	_, err := srv.RotateCredential(ctx, &argusv1.RotateCredentialRequest{
		InstanceUuid:      regResp.GetInstanceUuid(),
		CurrentCredential: "not-the-right-secret",
	})
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expected Unauthenticated for wrong current_credential, got %v", err)
	}
}

func TestDeregister_MovesInstanceToDeregistered(t *testing.T) {
	ctx := context.Background()
	srv, repo := newTestServer(t)
	regResp := registerTestInstance(t, ctx, srv, repo)
	id, err := uuid.Parse(regResp.GetInstanceUuid())
	if err != nil {
		t.Fatalf("parsing instance uuid: %v", err)
	}

	if _, err := srv.Deregister(ctx, &argusv1.DeregisterRequest{InstanceUuid: regResp.GetInstanceUuid(), Reason: "sigterm"}); err != nil {
		t.Fatalf("Deregister: %v", err)
	}

	inst, err := repo.GetInstance(ctx, id)
	if err != nil {
		t.Fatalf("GetInstance: %v", err)
	}
	if inst.Status != core.StatusDeregistered {
		t.Fatalf("expected status deregistered, got %s", inst.Status)
	}
}
