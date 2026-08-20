// Package grpc is Argus's gRPC hot path (ArgusService): registration and
// heartbeat, the calls every engine's platform/registry client makes.
// Handlers land with their owning specs (P2-S3 enrollment, P2-S5
// lifecycle); this file wires the health contract and server skeleton
// (P2-S1) so it is real and testable before that lands.
package grpc

import (
	"context"
	"errors"
	"sync"
	"time"

	argusv1 "github.com/autorix/argus/api/autorix/argus/v1"
	"github.com/autorix/argus/internal/core"
	"github.com/autorix/argus/internal/credential"
	"github.com/autorix/platform/paging"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Server implements argusv1.ArgusServiceServer against a core.Repository.
type Server struct {
	argusv1.UnimplementedArgusServiceServer
	repo   core.Repository
	encKey []byte // AES-256-GCM key sealing/opening instance credentials (P2-S3-T4)
	nonces *nonceStore
}

// NewServer builds a Server backed by repo. encKey must be exactly
// credential.KeySize bytes — it seals instance credentials at rest so the
// raw secret can be recovered to verify an HMAC heartbeat signature
// without ever storing it in the clear (P2-S3-T4).
func NewServer(repo core.Repository, encKey []byte) *Server {
	return &Server{
		repo:   repo,
		encKey: encKey,
		nonces: newNonceStore(core.HeartbeatNonceTTL),
	}
}

func (s *Server) Register(ctx context.Context, req *argusv1.RegisterRequest) (*argusv1.RegisterResponse, error) {
	if req.GetEnrollmentToken() == "" || req.GetEngineType() == "" || req.GetInstanceId() == "" || req.GetEnvironment() == "" {
		return nil, status.Error(codes.InvalidArgument, "enrollment_token, engine_type, instance_id and environment are required")
	}

	env, err := s.repo.GetEnvironmentBySlug(ctx, req.GetEnvironment())
	if err != nil {
		if errors.Is(err, core.ErrNotFound) {
			return nil, status.Errorf(codes.InvalidArgument, "unknown environment %q", req.GetEnvironment())
		}
		return nil, status.Error(codes.Internal, "looking up environment")
	}

	tokenHash := credential.HashSecret(req.GetEnrollmentToken())
	token, err := s.repo.GetEnrollmentTokenByHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, core.ErrEnrollmentTokenInvalid) || errors.Is(err, core.ErrNotFound) {
			s.auditConsumeFailed(ctx, nil, req.GetEngineType(), env.ID, "unknown, expired, revoked or exhausted token")
			return nil, status.Error(codes.PermissionDenied, "enrollment token is invalid, expired, revoked or exhausted")
		}
		return nil, status.Error(codes.Internal, "looking up enrollment token")
	}
	// The token is scoped to one engine type and environment (P2-S3-T1) —
	// a valid, unconsumed token for the wrong scope must be rejected just
	// as firmly as an unknown one, never silently accepted.
	if token.EngineType != req.GetEngineType() || token.EnvironmentID != env.ID {
		s.auditConsumeFailed(ctx, &token.ID, req.GetEngineType(), env.ID, "token not scoped to this engine type/environment")
		return nil, status.Error(codes.PermissionDenied, "enrollment token is not scoped to this engine type and environment")
	}

	if err := s.repo.ConsumeEnrollmentToken(ctx, token.ID); err != nil {
		if errors.Is(err, core.ErrEnrollmentTokenInvalid) {
			s.auditConsumeFailed(ctx, &token.ID, req.GetEngineType(), env.ID, "token invalid, expired, revoked or exhausted at consume time")
			return nil, status.Error(codes.PermissionDenied, "enrollment token is invalid, expired, revoked or exhausted")
		}
		return nil, status.Error(codes.Internal, "consuming enrollment token")
	}

	rawSecret, secretHash, err := credential.GenerateSecret("")
	if err != nil {
		return nil, status.Error(codes.Internal, "generating instance credential")
	}
	secretEncrypted, err := credential.Encrypt(s.encKey, rawSecret)
	if err != nil {
		return nil, status.Error(codes.Internal, "sealing instance credential")
	}

	regReq := core.RegistrationRequest{
		EngineType:    req.GetEngineType(),
		InstanceID:    req.GetInstanceId(),
		EnvironmentID: env.ID,
		Version:       req.GetVersion(),
		BuildSHA:      req.GetBuildSha(),
		SchemaVersion: req.GetSchemaVersion(),
		Capabilities:  req.GetCapabilities(),
		Endpoints: core.Endpoints{
			RESTURL: req.GetEndpoints().GetRestUrl(),
			GRPCURL: req.GetEndpoints().GetGrpcUrl(),
		},
	}

	instance, err := s.repo.RegisterInstance(ctx, regReq, "enrollment-token:"+token.ID.String(), false, core.InstanceCredentialMaterial{
		SecretHash:      secretHash,
		SecretEncrypted: secretEncrypted,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "registering instance")
	}

	s.auditAppend(ctx, core.EnrollmentAuditEntry{
		TokenID: &token.ID, EngineType: req.GetEngineType(), EnvironmentID: env.ID,
		Actor: "engine:" + req.GetEngineType(), Action: core.AuditActionConsume,
		Detail: map[string]interface{}{"instance_id": req.GetInstanceId(), "instance_uuid": instance.ID.String()},
	})

	return &argusv1.RegisterResponse{
		InstanceUuid:             instance.ID.String(),
		InstanceCredential:       rawSecret,
		HeartbeatIntervalSeconds: int32(core.HeartbeatInterval / time.Second),
	}, nil
}

// auditConsumeFailed and auditAppend log audit failures/successes without
// letting an audit-store error mask (or delay) the real gRPC error — the
// enrollment flow's own success/failure is authoritative, auditing is
// best-effort observability on top of it.
func (s *Server) auditConsumeFailed(ctx context.Context, tokenID *uuid.UUID, engineType string, envID uuid.UUID, reason string) {
	s.auditAppend(ctx, core.EnrollmentAuditEntry{
		TokenID: tokenID, EngineType: engineType, EnvironmentID: envID,
		Actor: "engine:" + engineType, Action: core.AuditActionConsumeFailed,
		Detail: map[string]interface{}{"reason": reason},
	})
}

func (s *Server) auditAppend(ctx context.Context, entry core.EnrollmentAuditEntry) {
	_ = s.repo.AppendEnrollmentAudit(ctx, entry)
}

// decryptCredential opens an AES-256-GCM sealed instance secret with this
// server's key. A nil/empty blob (e.g. a credential set before P2-S3-T4)
// yields an error rather than a false match.
func (s *Server) decryptCredential(blob []byte) (string, error) {
	if len(blob) == 0 {
		return "", errors.New("no sealed credential available")
	}
	return credential.Decrypt(s.encKey, blob)
}

func (s *Server) Heartbeat(ctx context.Context, req *argusv1.HeartbeatRequest) (*argusv1.HeartbeatResponse, error) {
	id, err := uuid.Parse(req.GetInstanceUuid())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "instance_uuid is not a valid uuid")
	}
	if req.GetSignature() == "" || req.GetNonce() == "" || req.GetTimestampUnix() == 0 {
		return nil, status.Error(codes.Unauthenticated, "signature, nonce and timestamp_unix are required")
	}

	now := time.Now().UTC()
	sentAt := time.Unix(req.GetTimestampUnix(), 0).UTC()
	drift := now.Sub(sentAt)
	if drift < 0 {
		drift = -drift
	}
	if drift > core.HeartbeatTimestampWindow {
		return nil, status.Error(codes.Unauthenticated, "heartbeat timestamp is outside the allowed window")
	}

	if !s.nonces.checkAndRecord(id, req.GetNonce(), now) {
		return nil, status.Error(codes.Unauthenticated, "heartbeat nonce has already been used")
	}

	cred, err := s.repo.GetInstanceCredential(ctx, id)
	if err != nil {
		if errors.Is(err, core.ErrNotFound) {
			return nil, status.Error(codes.Unauthenticated, "unknown instance or revoked credential")
		}
		return nil, status.Error(codes.Internal, "looking up instance credential")
	}

	valid := s.verifyHeartbeatSignature(cred.SecretEncrypted, req)
	if !valid && cred.PreviousSecretEncrypted != nil && cred.PreviousValidUntil != nil && now.Before(*cred.PreviousValidUntil) {
		valid = s.verifyHeartbeatSignature(cred.PreviousSecretEncrypted, req)
	}
	if !valid {
		return nil, status.Error(codes.Unauthenticated, "invalid heartbeat signature")
	}

	instance, err := s.repo.GetInstance(ctx, id)
	if err != nil {
		if errors.Is(err, core.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "instance not found")
		}
		return nil, status.Error(codes.Internal, "looking up instance")
	}

	report := core.HeartbeatReport{Live: req.GetLive(), Ready: req.GetReady(), ReceivedAt: now}
	if err := s.repo.RecordHeartbeat(ctx, id, report); err != nil {
		return nil, status.Error(codes.Internal, "recording heartbeat")
	}

	// The state machine (already unit-tested in core/status.go) derives the
	// real status from this heartbeat: elapsed-since-heartbeat is zero
	// relative to the moment it was just recorded, so this reduces to
	// "healthy if ready, degraded if not" — missed-beat escalation to
	// unreachable/evicted is the evaluator sweep's job (P2-S5-T2), not this
	// RPC's.
	newStatus := core.EvaluateStatus(instance.Status, &now, req.GetReady(), now)
	if newStatus != instance.Status {
		if err := core.ApplyStatusTransition(ctx, s.repo, id, newStatus, now); err != nil {
			return nil, status.Error(codes.Internal, "updating instance status")
		}
	}

	return &argusv1.HeartbeatResponse{Status: string(newStatus)}, nil
}

// verifyHeartbeatSignature decrypts blob (a sealed instance secret) and
// checks it against req's HMAC signature in constant time. Any failure to
// decrypt (wrong/missing key material) is treated as "does not match"
// rather than propagated, since an undecryptable credential must not
// short-circuit heartbeat auth into an error a caller could use to probe
// server state.
func (s *Server) verifyHeartbeatSignature(blob []byte, req *argusv1.HeartbeatRequest) bool {
	raw, err := s.decryptCredential(blob)
	if err != nil {
		return false
	}
	expected := credential.HeartbeatSignature(raw, req.GetInstanceUuid(), req.GetTimestampUnix(), req.GetNonce(), req.GetLive(), req.GetReady())
	return credential.ConstantTimeEqual(expected, req.GetSignature())
}

// Deregister is the graceful, engine-initiated exit (P2-S6): the instance
// moves to "deregistered" with the caller-supplied reason recorded on its
// timeline. Unlike ForceRemove, the row and its history are kept — a
// deregistration is a normal event, not an incident to erase.
func (s *Server) Deregister(ctx context.Context, req *argusv1.DeregisterRequest) (*argusv1.DeregisterResponse, error) {
	id, err := uuid.Parse(req.GetInstanceUuid())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "instance_uuid is not a valid uuid")
	}
	if err := s.repo.Deregister(ctx, id, req.GetReason()); err != nil {
		if errors.Is(err, core.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "instance not found")
		}
		return nil, status.Error(codes.Internal, "deregistering instance")
	}
	return &argusv1.DeregisterResponse{}, nil
}

// RotateCredential mints a fresh instance credential (P2-S3-T3) after
// verifying the caller presents the current one. The previous credential
// keeps authenticating heartbeats for core.CredentialRotationOverlap, so
// rotation never needs to be perfectly synchronized with the engine
// picking up the new secret.
func (s *Server) RotateCredential(ctx context.Context, req *argusv1.RotateCredentialRequest) (*argusv1.RotateCredentialResponse, error) {
	id, err := uuid.Parse(req.GetInstanceUuid())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "instance_uuid is not a valid uuid")
	}
	if req.GetCurrentCredential() == "" {
		return nil, status.Error(codes.InvalidArgument, "current_credential is required")
	}

	cred, err := s.repo.GetInstanceCredential(ctx, id)
	if err != nil {
		if errors.Is(err, core.ErrNotFound) {
			return nil, status.Error(codes.Unauthenticated, "unknown instance or revoked credential")
		}
		return nil, status.Error(codes.Internal, "looking up instance credential")
	}

	presentedHash := credential.HashSecret(req.GetCurrentCredential())
	if !credential.ConstantTimeEqual(presentedHash, cred.SecretHash) {
		return nil, status.Error(codes.Unauthenticated, "current_credential does not match")
	}

	rawSecret, secretHash, err := credential.GenerateSecret("")
	if err != nil {
		return nil, status.Error(codes.Internal, "generating instance credential")
	}
	secretEncrypted, err := credential.Encrypt(s.encKey, rawSecret)
	if err != nil {
		return nil, status.Error(codes.Internal, "sealing instance credential")
	}

	overlapUntil := time.Now().UTC().Add(core.CredentialRotationOverlap)
	if err := s.repo.RotateInstanceCredential(ctx, id, core.InstanceCredentialMaterial{
		SecretHash:      secretHash,
		SecretEncrypted: secretEncrypted,
	}, overlapUntil); err != nil {
		if errors.Is(err, core.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "instance credential not found")
		}
		return nil, status.Error(codes.Internal, "rotating instance credential")
	}

	return &argusv1.RotateCredentialResponse{
		NewCredential: rawSecret,
		OverlapUntil:  timestamppb.New(overlapUntil),
	}, nil
}

// GetInstance returns one instance's current state, its most recent
// timeline events and its dependency edges in a single response (P2-S6-T1)
// — a console detail view needs all three, and fetching them here instead
// of three separate calls keeps that a single round trip instead of a
// client-side fan-out.
func (s *Server) GetInstance(ctx context.Context, req *argusv1.GetInstanceRequest) (*argusv1.GetInstanceResponse, error) {
	id, err := uuid.Parse(req.GetInstanceUuid())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "instance_uuid is not a valid uuid")
	}

	inst, err := s.repo.GetInstance(ctx, id)
	if err != nil {
		if errors.Is(err, core.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "instance not found")
		}
		return nil, status.Error(codes.Internal, "getting instance")
	}

	envSlug := ""
	if env, err := s.repo.GetEnvironmentByID(ctx, inst.EnvironmentID); err == nil {
		envSlug = env.Slug
	} else if !errors.Is(err, core.ErrNotFound) {
		return nil, status.Error(codes.Internal, "getting environment")
	}

	events, err := s.repo.ListEvents(ctx, &id, 20)
	if err != nil {
		return nil, status.Error(codes.Internal, "listing events")
	}

	deps, err := s.repo.ListDependencies(ctx, id)
	if err != nil {
		return nil, status.Error(codes.Internal, "listing dependencies")
	}

	return &argusv1.GetInstanceResponse{
		Instance:     toProtoInstance(inst, envSlug),
		Events:       toProtoEvents(events),
		Dependencies: toProtoDependencies(deps),
	}, nil
}

// ListInstances returns a filtered, cursor-paginated page of instances
// (P2-S6-T1). Environment/status filters are optional; an unknown
// environment slug is rejected as InvalidArgument rather than silently
// returning an empty page, the same way Register treats it.
func (s *Server) ListInstances(ctx context.Context, req *argusv1.ListInstancesRequest) (*argusv1.ListInstancesResponse, error) {
	filter := core.InstanceFilter{
		EngineType: req.GetEngineType(),
		Status:     core.InstanceStatus(req.GetStatus()),
		Limit:      int(req.GetLimit()),
	}

	if envSlug := req.GetEnvironment(); envSlug != "" {
		env, err := s.repo.GetEnvironmentBySlug(ctx, envSlug)
		if err != nil {
			if errors.Is(err, core.ErrNotFound) {
				return nil, status.Errorf(codes.InvalidArgument, "unknown environment %q", envSlug)
			}
			return nil, status.Error(codes.Internal, "looking up environment")
		}
		filter.EnvironmentID = env.ID
	}

	if req.GetCursor() != "" {
		raw, err := paging.DecodeCursor(req.GetCursor())
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "cursor is not valid")
		}
		filter.Cursor = raw
	}

	instances, nextCursor, hasMore, err := s.repo.ListInstances(ctx, filter)
	if err != nil {
		return nil, status.Error(codes.Internal, "listing instances")
	}

	// Resolve every environment slug once instead of once per instance, so
	// a full page of results costs one extra query, not N.
	envs, err := s.repo.ListEnvironments(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "listing environments")
	}
	envSlugByID := make(map[uuid.UUID]string, len(envs))
	for _, e := range envs {
		envSlugByID[e.ID] = e.Slug
	}

	out := make([]*argusv1.Instance, 0, len(instances))
	for _, inst := range instances {
		out = append(out, toProtoInstance(inst, envSlugByID[inst.EnvironmentID]))
	}

	resp := &argusv1.ListInstancesResponse{Instances: out, HasMore: hasMore}
	if hasMore {
		resp.NextCursor = paging.EncodeCursor(nextCursor)
	}
	return resp, nil
}

// ---------------------------------------------------------------------
// Heartbeat replay protection (P2-S3-T4)
// ---------------------------------------------------------------------

// nonceStore remembers recently-seen heartbeat nonces per instance, bounded
// by TTL expiry rather than an unbounded count: every checkAndRecord call
// prunes entries older than ttl before checking, so memory use tracks
// actual heartbeat traffic within the replay window instead of growing
// forever.
type nonceStore struct {
	mu   sync.Mutex
	seen map[uuid.UUID]map[string]time.Time
	ttl  time.Duration
}

func newNonceStore(ttl time.Duration) *nonceStore {
	return &nonceStore{seen: make(map[uuid.UUID]map[string]time.Time), ttl: ttl}
}

// checkAndRecord returns false if nonce was already recorded for instance
// id and has not yet expired (a replay); otherwise it records it and
// returns true.
func (n *nonceStore) checkAndRecord(id uuid.UUID, nonce string, now time.Time) bool {
	n.mu.Lock()
	defer n.mu.Unlock()

	m, ok := n.seen[id]
	if !ok {
		m = make(map[string]time.Time)
		n.seen[id] = m
	}
	for existing, expiresAt := range m {
		if now.After(expiresAt) {
			delete(m, existing)
		}
	}

	if expiresAt, exists := m[nonce]; exists && now.Before(expiresAt) {
		return false
	}
	m[nonce] = now.Add(n.ttl)
	return true
}
