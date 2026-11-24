package grpc

import (
	"context"
	cryptorand "crypto/rand"
	"fmt"
	"testing"
	"time"

	argusv1 "github.com/autorix/argus/api/autorix/argus/v1"
	"github.com/autorix/argus/internal/core"
	"github.com/autorix/argus/internal/credential"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type stubHeartbeatRepo struct {
	core.Repository
	cred           core.InstanceCredentialRecord
	instance       core.Instance
	knownEngines   map[string]bool
	recordedProbes []core.DependencyProbe
	lastReport     core.HeartbeatReport
}

func (r *stubHeartbeatRepo) GetInstanceCredential(ctx context.Context, instanceID uuid.UUID) (core.InstanceCredentialRecord, error) {
	if instanceID != r.instance.ID {
		return core.InstanceCredentialRecord{}, core.ErrNotFound
	}
	return r.cred, nil
}

func (r *stubHeartbeatRepo) GetInstance(ctx context.Context, id uuid.UUID) (core.Instance, error) {
	if id != r.instance.ID {
		return core.Instance{}, core.ErrNotFound
	}
	return r.instance, nil
}

func (r *stubHeartbeatRepo) RecordHeartbeat(ctx context.Context, id uuid.UUID, report core.HeartbeatReport) error {
	r.lastReport = report
	return nil
}

func (r *stubHeartbeatRepo) GetEngineType(ctx context.Context, slug string) (core.EngineType, error) {
	if r.knownEngines[slug] {
		return core.EngineType{Slug: slug}, nil
	}
	return core.EngineType{}, core.ErrNotFound
}

func (r *stubHeartbeatRepo) RecordDependencyProbe(ctx context.Context, instanceID uuid.UUID, dependsOnEngineType string, reachable bool, latencyMs float64) error {
	r.recordedProbes = append(r.recordedProbes, core.DependencyProbe{
		EngineType: dependsOnEngineType,
		Reachable:  reachable,
		LatencyMs:  latencyMs,
	})
	return nil
}

func (r *stubHeartbeatRepo) SetInstanceStatusWithEvent(ctx context.Context, id uuid.UUID, status core.InstanceStatus, eventType core.InstanceEventType, detail map[string]interface{}) error {
	r.instance.Status = status
	return nil
}

func (r *stubHeartbeatRepo) SetInstanceStatus(ctx context.Context, id uuid.UUID, status core.InstanceStatus) error {
	r.instance.Status = status
	return nil
}

func TestNonceStore_PrunesEmptyBucketsAndEnforcesCap(t *testing.T) {
	store := newNonceStoreWithCap(1*time.Minute, 3)
	idA := uuid.New()
	idB := uuid.New()
	now := time.Now().UTC()

	for i := 0; i < 3; i++ {
		if !store.checkAndRecord(idA, fmt.Sprintf("n-%d", i), now) {
			t.Fatalf("expected nonce %d within cap to succeed", i)
		}
	}
	// Cap reached: new nonce within same TTL window must fail closed.
	if store.checkAndRecord(idA, "n-overflow", now) {
		t.Fatal("expected nonce exceeding per-instance cap to be rejected")
	}

	// Advance past TTL and record for another instance; idA's empty bucket must be pruned.
	later := now.Add(2 * time.Minute)
	if !store.checkAndRecord(idB, "n-b1", later) {
		t.Fatal("expected fresh nonce for idB to succeed")
	}

	store.mu.Lock()
	_, existsA := store.seen[idA]
	store.mu.Unlock()
	if existsA {
		t.Fatal("expected expired empty instance bucket idA to be pruned from nonceStore")
	}
}

func TestHeartbeat_InvalidSignatureDoesNotConsumeNonce(t *testing.T) {
	encKey := make([]byte, credential.KeySize)
	if _, err := cryptorand.Read(encKey); err != nil {
		t.Fatalf("generating key: %v", err)
	}
	rawSecret, secretHash, err := credential.GenerateSecret("")
	if err != nil {
		t.Fatalf("GenerateSecret: %v", err)
	}
	sealed, err := credential.Encrypt(encKey, rawSecret)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	instID := uuid.New()
	repo := &stubHeartbeatRepo{
		cred: core.InstanceCredentialRecord{
			SecretHash:      secretHash,
			SecretEncrypted: sealed,
		},
		instance: core.Instance{
			ID:     instID,
			Status: core.StatusRegistering,
		},
		knownEngines: map[string]bool{"nexus": true},
	}
	srv := NewServer(repo, encKey)

	ts := time.Now().UTC().Unix()
	const sharedNonce = "victim-nonce-1"

	// 1. Attacker sends forged heartbeat with victim's nonce and bad signature.
	_, err = srv.Heartbeat(context.Background(), &argusv1.HeartbeatRequest{
		InstanceUuid:  instID.String(),
		Live:          true,
		Ready:         true,
		TimestampUnix: ts,
		Nonce:         sharedNonce,
		Signature:     "forged-signature",
	})
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expected Unauthenticated for forged signature, got %v", err)
	}

	// 2. Legitimate engine sends properly signed heartbeat using the same nonce; must succeed.
	validSig := credential.HeartbeatSignature(rawSecret, instID.String(), ts, sharedNonce, true, true)
	resp, err := srv.Heartbeat(context.Background(), &argusv1.HeartbeatRequest{
		InstanceUuid:  instID.String(),
		Live:          true,
		Ready:         true,
		TimestampUnix: ts,
		Nonce:         sharedNonce,
		Signature:     validSig,
	})
	if err != nil {
		t.Fatalf("expected valid heartbeat after forged attempt to succeed, got %v", err)
	}
	if resp.GetStatus() != string(core.StatusHealthy) {
		t.Fatalf("expected healthy status, got %s", resp.GetStatus())
	}

	// 3. Replaying the valid signed heartbeat must now be rejected.
	_, err = srv.Heartbeat(context.Background(), &argusv1.HeartbeatRequest{
		InstanceUuid:  instID.String(),
		Live:          true,
		Ready:         true,
		TimestampUnix: ts,
		Nonce:         sharedNonce,
		Signature:     validSig,
	})
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expected Unauthenticated on replay, got %v", err)
	}
}

func TestHeartbeat_PersistsEngineDependenciesAndSkipsNonEngineProbes(t *testing.T) {
	encKey := make([]byte, credential.KeySize)
	if _, err := cryptorand.Read(encKey); err != nil {
		t.Fatalf("generating key: %v", err)
	}
	rawSecret, secretHash, err := credential.GenerateSecret("")
	if err != nil {
		t.Fatalf("GenerateSecret: %v", err)
	}
	sealed, err := credential.Encrypt(encKey, rawSecret)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	instID := uuid.New()
	repo := &stubHeartbeatRepo{
		cred: core.InstanceCredentialRecord{
			SecretHash:      secretHash,
			SecretEncrypted: sealed,
		},
		instance: core.Instance{
			ID:     instID,
			Status: core.StatusHealthy,
		},
		knownEngines: map[string]bool{"nexus": true, "ego": true},
	}
	srv := NewServer(repo, encKey)

	ts := time.Now().UTC().Unix()
	nonce := "dep-nonce-1"
	sig := credential.HeartbeatSignature(rawSecret, instID.String(), ts, nonce, true, true)

	_, err = srv.Heartbeat(context.Background(), &argusv1.HeartbeatRequest{
		InstanceUuid:  instID.String(),
		Live:          true,
		Ready:         true,
		TimestampUnix: ts,
		Nonce:         nonce,
		Signature:     sig,
		Dependencies: []*argusv1.DependencyStatus{
			{Name: "nexus", Reachable: true, LatencyMs: 4.2},
			{Name: "postgres", Reachable: true, LatencyMs: 1.1},
			{Name: "ego", Reachable: false, LatencyMs: 50.0},
		},
	})
	if err != nil {
		t.Fatalf("Heartbeat with dependencies failed: %v", err)
	}

	if len(repo.recordedProbes) != 2 {
		t.Fatalf("expected 2 persisted engine dependency probes (nexus, ego), got %+v", repo.recordedProbes)
	}
	if repo.recordedProbes[0].EngineType != "nexus" || !repo.recordedProbes[0].Reachable || repo.recordedProbes[0].LatencyMs != 4.2 {
		t.Fatalf("unexpected first probe: %+v", repo.recordedProbes[0])
	}
	if repo.recordedProbes[1].EngineType != "ego" || repo.recordedProbes[1].Reachable || repo.recordedProbes[1].LatencyMs != 50.0 {
		t.Fatalf("unexpected second probe: %+v", repo.recordedProbes[1])
	}
}
