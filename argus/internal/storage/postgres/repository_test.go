package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/autorix/argus/internal/core"
	"github.com/autorix/argus/internal/storage/postgres"
	"github.com/autorix/platform/pgtest"
	"github.com/google/uuid"
)

func newTestRepo(t *testing.T) *postgres.Repository {
	t.Helper()
	pool := pgtest.StartPostgres(t, "../../../migrations")
	return postgres.NewRepository(pool)
}

func mustEnvironment(t *testing.T, repo *postgres.Repository, slug string) core.Environment {
	t.Helper()
	env, err := repo.CreateEnvironment(context.Background(), "Test "+slug, slug, "", false)
	if err != nil {
		t.Fatalf("CreateEnvironment(%s): %v", slug, err)
	}
	return env
}

func sampleRegistrationRequest(engineType string, envID uuid.UUID) core.RegistrationRequest {
	return core.RegistrationRequest{
		EngineType:    engineType,
		InstanceID:    "host-1",
		EnvironmentID: envID,
		Version:       "1.0.0",
		BuildSHA:      "abc123",
		SchemaVersion: "1",
		Endpoints:     core.Endpoints{RESTURL: "http://host-1:4433", GRPCURL: ""},
		Capabilities:  []string{"argon2id"},
	}
}

func TestCreateAndGetEnvironment(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	env := mustEnvironment(t, repo, "staging-env")

	got, err := repo.GetEnvironmentBySlug(ctx, "staging-env")
	if err != nil {
		t.Fatalf("GetEnvironmentBySlug: %v", err)
	}
	if got.ID != env.ID {
		t.Fatalf("expected id %s, got %s", env.ID, got.ID)
	}

	envs, err := repo.ListEnvironments(ctx)
	if err != nil {
		t.Fatalf("ListEnvironments: %v", err)
	}
	if len(envs) != 1 {
		t.Fatalf("expected 1 environment, got %d", len(envs))
	}
}

func TestGetEnvironmentByID(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()
	env := mustEnvironment(t, repo, "by-id-env")

	got, err := repo.GetEnvironmentByID(ctx, env.ID)
	if err != nil {
		t.Fatalf("GetEnvironmentByID: %v", err)
	}
	if got.Slug != "by-id-env" {
		t.Fatalf("expected slug by-id-env, got %s", got.Slug)
	}

	if _, err := repo.GetEnvironmentByID(ctx, uuid.New()); err == nil {
		t.Fatal("expected error for unknown environment id")
	}
}

func TestListEngineTypesSeeded(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	types, err := repo.ListEngineTypes(ctx)
	if err != nil {
		t.Fatalf("ListEngineTypes: %v", err)
	}
	if len(types) != 7 {
		t.Fatalf("expected 7 seeded engine types, got %d", len(types))
	}

	et, err := repo.GetEngineType(ctx, "ego")
	if err != nil {
		t.Fatalf("GetEngineType: %v", err)
	}
	if et.Slug != "ego" || et.Protocol != "rest" {
		t.Fatalf("unexpected engine type: %+v", et)
	}
}

func TestUpsertInstance_InsertThenUpdateSameKey(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()
	env := mustEnvironment(t, repo, "upsert-env")

	req := sampleRegistrationRequest("ego", env.ID)

	created, err := repo.UpsertInstance(ctx, req, "admin@autorix", false)
	if err != nil {
		t.Fatalf("UpsertInstance (insert): %v", err)
	}
	if created.Status != core.StatusPending {
		t.Fatalf("expected new instance to default to pending, got %s", created.Status)
	}
	if created.EngineType != "ego" || created.InstanceID != "host-1" {
		t.Fatalf("unexpected instance: %+v", created)
	}

	// Re-register with a bumped version — same upsert key must update the
	// same row instead of creating a duplicate.
	req.Version = "1.1.0"
	updated, err := repo.UpsertInstance(ctx, req, "admin@autorix", false)
	if err != nil {
		t.Fatalf("UpsertInstance (update): %v", err)
	}
	if updated.ID != created.ID {
		t.Fatalf("expected same instance id across re-registration, got %s vs %s", created.ID, updated.ID)
	}
	if updated.Version != "1.1.0" {
		t.Fatalf("expected version to be updated, got %s", updated.Version)
	}
	if updated.FirstSeenAt.Unix() != created.FirstSeenAt.Unix() {
		t.Fatalf("expected first_seen_at to be preserved across re-registration")
	}

	fetched, err := repo.GetInstance(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetInstance: %v", err)
	}
	if fetched.Version != "1.1.0" {
		t.Fatalf("expected fetched version 1.1.0, got %s", fetched.Version)
	}
}

func TestGetInstance_NotFound(t *testing.T) {
	repo := newTestRepo(t)
	_, err := repo.GetInstance(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected error for unknown instance id")
	}
}

func TestListInstances_FiltersByEngineTypeEnvironmentAndStatus(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()
	env := mustEnvironment(t, repo, "list-env")
	otherEnv := mustEnvironment(t, repo, "list-env-2")

	egoReq := sampleRegistrationRequest("ego", env.ID)
	egoInst, err := repo.UpsertInstance(ctx, egoReq, "admin", false)
	if err != nil {
		t.Fatalf("UpsertInstance ego: %v", err)
	}
	if err := repo.SetInstanceStatus(ctx, egoInst.ID, core.StatusHealthy); err != nil {
		t.Fatalf("SetInstanceStatus: %v", err)
	}

	janusReq := sampleRegistrationRequest("janus", env.ID)
	if _, err := repo.UpsertInstance(ctx, janusReq, "admin", false); err != nil {
		t.Fatalf("UpsertInstance janus: %v", err)
	}

	otherEnvReq := sampleRegistrationRequest("ego", otherEnv.ID)
	if _, err := repo.UpsertInstance(ctx, otherEnvReq, "admin", false); err != nil {
		t.Fatalf("UpsertInstance other env: %v", err)
	}

	instances, _, _, err := repo.ListInstances(ctx, core.InstanceFilter{EngineType: "ego", EnvironmentID: env.ID})
	if err != nil {
		t.Fatalf("ListInstances by engine type + env: %v", err)
	}
	if len(instances) != 1 || instances[0].EngineType != "ego" {
		t.Fatalf("expected 1 ego instance in env, got %+v", instances)
	}

	healthyOnly, _, _, err := repo.ListInstances(ctx, core.InstanceFilter{Status: core.StatusHealthy})
	if err != nil {
		t.Fatalf("ListInstances by status: %v", err)
	}
	if len(healthyOnly) != 1 || healthyOnly[0].ID != egoInst.ID {
		t.Fatalf("expected exactly the healthy instance, got %+v", healthyOnly)
	}
}

func TestRecordHeartbeatUpdatesLastHeartbeatAt(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()
	env := mustEnvironment(t, repo, "heartbeat-env")
	inst, err := repo.UpsertInstance(ctx, sampleRegistrationRequest("ego", env.ID), "admin", false)
	if err != nil {
		t.Fatalf("UpsertInstance: %v", err)
	}
	if inst.LastHeartbeatAt != nil {
		t.Fatalf("expected nil last_heartbeat_at before first heartbeat")
	}

	report := core.HeartbeatReport{Live: true, Ready: true, ReceivedAt: time.Now().UTC()}
	if err := repo.RecordHeartbeat(ctx, inst.ID, report); err != nil {
		t.Fatalf("RecordHeartbeat: %v", err)
	}

	fetched, err := repo.GetInstance(ctx, inst.ID)
	if err != nil {
		t.Fatalf("GetInstance: %v", err)
	}
	if fetched.LastHeartbeatAt == nil {
		t.Fatal("expected last_heartbeat_at to be set after RecordHeartbeat")
	}
}

func TestSetInstanceStatusWithEvent_TransactionalEventWriter(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()
	env := mustEnvironment(t, repo, "event-env")
	inst, err := repo.UpsertInstance(ctx, sampleRegistrationRequest("ego", env.ID), "admin", false)
	if err != nil {
		t.Fatalf("UpsertInstance: %v", err)
	}

	if err := repo.SetInstanceStatusWithEvent(ctx, inst.ID, core.StatusDegraded, core.EventDegraded, map[string]interface{}{"reason": "missed 2 beats"}); err != nil {
		t.Fatalf("SetInstanceStatusWithEvent: %v", err)
	}

	fetched, err := repo.GetInstance(ctx, inst.ID)
	if err != nil {
		t.Fatalf("GetInstance: %v", err)
	}
	if fetched.Status != core.StatusDegraded {
		t.Fatalf("expected status degraded, got %s", fetched.Status)
	}

	events, err := repo.ListEvents(ctx, &inst.ID, 10)
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected exactly 1 event written transactionally with the status change, got %d", len(events))
	}
	if events[0].Type != core.EventDegraded {
		t.Fatalf("expected degraded event, got %s", events[0].Type)
	}
	if events[0].Detail["reason"] != "missed 2 beats" {
		t.Fatalf("expected event detail to be persisted, got %+v", events[0].Detail)
	}
}

func TestEnrollmentToken_MintAndConsumeSingleUse(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()
	env := mustEnvironment(t, repo, "token-env")

	token := core.EnrollmentToken{
		TokenHash:     "hash-single-use",
		EngineType:    "ego",
		EnvironmentID: env.ID,
		UsesAllowed:   1,
		ExpiresAt:     time.Now().Add(1 * time.Hour),
		CreatedBy:     "admin@autorix",
	}
	created, err := repo.CreateEnrollmentToken(ctx, token)
	if err != nil {
		t.Fatalf("CreateEnrollmentToken: %v", err)
	}
	if created.ID == uuid.Nil {
		t.Fatal("expected a generated id")
	}

	fetched, err := repo.GetEnrollmentTokenByHash(ctx, "hash-single-use")
	if err != nil {
		t.Fatalf("GetEnrollmentTokenByHash: %v", err)
	}
	if fetched.ID != created.ID {
		t.Fatalf("expected same token id, got %s vs %s", fetched.ID, created.ID)
	}

	if err := repo.ConsumeEnrollmentToken(ctx, created.ID); err != nil {
		t.Fatalf("first consume should succeed: %v", err)
	}

	if err := repo.ConsumeEnrollmentToken(ctx, created.ID); err == nil {
		t.Fatal("expected second consume of a single-use token to fail")
	}
}

func TestEnrollmentToken_ExpiredIsRejected(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()
	env := mustEnvironment(t, repo, "token-expired-env")

	token := core.EnrollmentToken{
		TokenHash:     "hash-expired",
		EngineType:    "ego",
		EnvironmentID: env.ID,
		UsesAllowed:   1,
		ExpiresAt:     time.Now().Add(-1 * time.Hour), // already expired
		CreatedBy:     "admin@autorix",
	}
	created, err := repo.CreateEnrollmentToken(ctx, token)
	if err != nil {
		t.Fatalf("CreateEnrollmentToken: %v", err)
	}

	if err := repo.ConsumeEnrollmentToken(ctx, created.ID); err == nil {
		t.Fatal("expected expired token consumption to fail")
	}
}

func TestEnrollmentToken_RevokedIsRejected(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()
	env := mustEnvironment(t, repo, "token-revoked-env")

	token := core.EnrollmentToken{
		TokenHash:     "hash-revoked",
		EngineType:    "ego",
		EnvironmentID: env.ID,
		UsesAllowed:   5,
		ExpiresAt:     time.Now().Add(1 * time.Hour),
		CreatedBy:     "admin@autorix",
	}
	created, err := repo.CreateEnrollmentToken(ctx, token)
	if err != nil {
		t.Fatalf("CreateEnrollmentToken: %v", err)
	}
	if err := repo.RevokeEnrollmentToken(ctx, created.ID); err != nil {
		t.Fatalf("RevokeEnrollmentToken: %v", err)
	}
	if err := repo.ConsumeEnrollmentToken(ctx, created.ID); err == nil {
		t.Fatal("expected revoked token consumption to fail")
	}
}

func TestInstanceCredential_SetGetRotateRevoke(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()
	env := mustEnvironment(t, repo, "cred-env")
	inst, err := repo.UpsertInstance(ctx, sampleRegistrationRequest("ego", env.ID), "admin", false)
	if err != nil {
		t.Fatalf("UpsertInstance: %v", err)
	}

	if err := repo.SetInstanceCredential(ctx, inst.ID, core.InstanceCredentialMaterial{SecretHash: "hash-v1", SecretEncrypted: []byte("enc-v1")}); err != nil {
		t.Fatalf("SetInstanceCredential: %v", err)
	}

	rec, err := repo.GetInstanceCredential(ctx, inst.ID)
	if err != nil {
		t.Fatalf("GetInstanceCredential: %v", err)
	}
	if rec.SecretHash != "hash-v1" || rec.PreviousSecretHash != "" || rec.PreviousValidUntil != nil {
		t.Fatalf("unexpected credential state: %+v", rec)
	}

	overlapUntil := time.Now().Add(10 * time.Minute)
	if err := repo.RotateInstanceCredential(ctx, inst.ID, core.InstanceCredentialMaterial{SecretHash: "hash-v2", SecretEncrypted: []byte("enc-v2")}, overlapUntil); err != nil {
		t.Fatalf("RotateInstanceCredential: %v", err)
	}

	rec, err = repo.GetInstanceCredential(ctx, inst.ID)
	if err != nil {
		t.Fatalf("GetInstanceCredential after rotate: %v", err)
	}
	if rec.SecretHash != "hash-v2" || rec.PreviousSecretHash != "hash-v1" || rec.PreviousValidUntil == nil {
		t.Fatalf("unexpected credential state after rotate: %+v", rec)
	}

	if err := repo.RevokeInstanceCredential(ctx, inst.ID); err != nil {
		t.Fatalf("RevokeInstanceCredential: %v", err)
	}
	if _, err := repo.GetInstanceCredential(ctx, inst.ID); err == nil {
		t.Fatal("expected revoked credential to no longer be retrievable")
	}
}

func TestRegisterInstance_AtomicIdentityCredentialAndEvent(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()
	env := mustEnvironment(t, repo, "register-env")

	req := sampleRegistrationRequest("ego", env.ID)
	inst, err := repo.RegisterInstance(ctx, req, "admin@autorix", false, core.InstanceCredentialMaterial{SecretHash: "secret-hash-1", SecretEncrypted: []byte("secret-enc-1")})
	if err != nil {
		t.Fatalf("RegisterInstance: %v", err)
	}
	if inst.Status != core.StatusRegistering {
		t.Fatalf("expected status registering right after Register, got %s", inst.Status)
	}

	rec, err := repo.GetInstanceCredential(ctx, inst.ID)
	if err != nil {
		t.Fatalf("GetInstanceCredential: %v", err)
	}
	if rec.SecretHash != "secret-hash-1" {
		t.Fatalf("expected persisted credential hash, got %s", rec.SecretHash)
	}

	events, err := repo.ListEvents(ctx, &inst.ID, 10)
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	if len(events) != 1 || events[0].Type != core.EventRegistered {
		t.Fatalf("expected exactly one registered event, got %+v", events)
	}
}

func TestListStaleInstances(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()
	env := mustEnvironment(t, repo, "stale-env")
	inst, err := repo.UpsertInstance(ctx, sampleRegistrationRequest("ego", env.ID), "admin", false)
	if err != nil {
		t.Fatalf("UpsertInstance: %v", err)
	}
	if err := repo.RecordHeartbeat(ctx, inst.ID, core.HeartbeatReport{Live: true, Ready: true, ReceivedAt: time.Now().UTC()}); err != nil {
		t.Fatalf("RecordHeartbeat: %v", err)
	}

	stale, err := repo.ListStaleInstances(ctx, time.Now().Add(1*time.Hour))
	if err != nil {
		t.Fatalf("ListStaleInstances: %v", err)
	}
	if len(stale) != 1 || stale[0].ID != inst.ID {
		t.Fatalf("expected instance to be stale relative to a future cutoff, got %+v", stale)
	}

	notStale, err := repo.ListStaleInstances(ctx, time.Now().Add(-1*time.Hour))
	if err != nil {
		t.Fatalf("ListStaleInstances: %v", err)
	}
	if len(notStale) != 0 {
		t.Fatalf("expected no stale instances relative to a past cutoff, got %+v", notStale)
	}
}

func TestPruneOlderThan_DeletesOldEventsAndEvictedInstancesOnly(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()
	env := mustEnvironment(t, repo, "prune-env")

	// An old evicted instance: must be pruned, along with its events.
	evicted, err := repo.UpsertInstance(ctx, sampleRegistrationRequest("ego", env.ID), "admin", false)
	if err != nil {
		t.Fatalf("UpsertInstance (evicted): %v", err)
	}
	if err := repo.SetInstanceStatusWithEvent(ctx, evicted.ID, core.StatusEvicted, core.EventEvicted, nil); err != nil {
		t.Fatalf("SetInstanceStatusWithEvent: %v", err)
	}

	// A healthy instance with an old event: the event is prunable, the
	// instance itself must survive since it is not evicted.
	healthyReq := sampleRegistrationRequest("ego", env.ID)
	healthyReq.InstanceID = "host-2"
	healthy, err := repo.UpsertInstance(ctx, healthyReq, "admin", false)
	if err != nil {
		t.Fatalf("UpsertInstance (healthy): %v", err)
	}
	if err := repo.AppendEvent(ctx, healthy.ID, core.EventRegistered, nil); err != nil {
		t.Fatalf("AppendEvent: %v", err)
	}

	cutoff := time.Now().Add(1 * time.Hour) // everything above is "old" relative to this

	eventsDeleted, instancesDeleted, err := repo.PruneOlderThan(ctx, cutoff)
	if err != nil {
		t.Fatalf("PruneOlderThan: %v", err)
	}
	if instancesDeleted != 1 {
		t.Fatalf("expected exactly 1 evicted instance pruned, got %d", instancesDeleted)
	}
	if eventsDeleted < 2 {
		t.Fatalf("expected at least 2 events pruned (registered+evicted for evicted, registered for healthy), got %d", eventsDeleted)
	}

	if _, err := repo.GetInstance(ctx, evicted.ID); err == nil {
		t.Fatal("expected the evicted instance to be gone after prune")
	}
	if _, err := repo.GetInstance(ctx, healthy.ID); err != nil {
		t.Fatalf("expected the non-evicted instance to survive prune: %v", err)
	}
}

func TestEnrollmentAudit_AppendAndList(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()
	env := mustEnvironment(t, repo, "audit-env")

	token, err := repo.CreateEnrollmentToken(ctx, core.EnrollmentToken{
		TokenHash: "audit-hash", EngineType: "ego", EnvironmentID: env.ID,
		UsesAllowed: 1, ExpiresAt: time.Now().Add(time.Hour), CreatedBy: "admin",
	})
	if err != nil {
		t.Fatalf("CreateEnrollmentToken: %v", err)
	}

	entries := []core.EnrollmentAuditEntry{
		{TokenID: &token.ID, EngineType: "ego", EnvironmentID: env.ID, Actor: "admin", Action: core.AuditActionMint},
		{TokenID: &token.ID, EngineType: "ego", EnvironmentID: env.ID, Actor: "engine:ego", Action: core.AuditActionConsume},
		{TokenID: &token.ID, EngineType: "ego", EnvironmentID: env.ID, Actor: "engine:ego", Action: core.AuditActionConsumeFailed, Detail: map[string]interface{}{"reason": "expired"}},
		{TokenID: &token.ID, EngineType: "ego", EnvironmentID: env.ID, Actor: "admin", Action: core.AuditActionRevoke},
	}
	for _, e := range entries {
		if err := repo.AppendEnrollmentAudit(ctx, e); err != nil {
			t.Fatalf("AppendEnrollmentAudit(%s): %v", e.Action, err)
		}
	}

	all, _, hasMore, err := repo.ListEnrollmentAudit(ctx, core.EnrollmentAuditFilter{TokenID: token.ID})
	if err != nil {
		t.Fatalf("ListEnrollmentAudit: %v", err)
	}
	if len(all) != 4 {
		t.Fatalf("expected 4 audit entries, got %d", len(all))
	}
	if hasMore {
		t.Fatal("expected no more pages")
	}
	// Newest first.
	if all[0].Action != core.AuditActionRevoke {
		t.Fatalf("expected newest entry (revoke) first, got %s", all[0].Action)
	}

	failedOnly, _, _, err := repo.ListEnrollmentAudit(ctx, core.EnrollmentAuditFilter{Action: core.AuditActionConsumeFailed})
	if err != nil {
		t.Fatalf("ListEnrollmentAudit filtered: %v", err)
	}
	if len(failedOnly) != 1 || failedOnly[0].Detail["reason"] != "expired" {
		t.Fatalf("expected exactly 1 consume_failed entry with detail, got %+v", failedOnly)
	}
}
