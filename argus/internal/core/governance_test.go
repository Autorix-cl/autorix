package core

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestCalculateAuditRecordHash_CoversAllFieldsAndPreventsDelimiterCollisions(t *testing.T) {
	baseID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	ts := time.Date(2026, 5, 1, 12, 0, 0, 123456000, time.UTC)

	base := AuditRecord{
		ID:           baseID,
		ActorID:      "op-1",
		ActorType:    "operator",
		Action:       "policy.update",
		ResourceType: "policy",
		ResourceID:   "pol-1",
		Environment:  "production",
		SourceIP:     "10.0.0.1",
		BeforeState:  map[string]interface{}{"enabled": false, "name": "alpha"},
		AfterState:   map[string]interface{}{"enabled": true, "name": "alpha"},
		Outcome:      "success",
		PrevHash:     "prev-hash-0",
		CreatedAt:    ts,
	}

	baseHash := CalculateAuditRecordHash(base)
	if baseHash == "" {
		t.Fatal("expected non-empty audit record hash")
	}

	// Deterministic across map key order and identical inputs.
	same := base
	same.BeforeState = map[string]interface{}{"name": "alpha", "enabled": false}
	if got := CalculateAuditRecordHash(same); got != baseHash {
		t.Fatalf("expected canonical hash stability, got %s vs %s", got, baseHash)
	}

	mutations := []struct {
		name   string
		mutate func(r *AuditRecord)
	}{
		{"actor_id", func(r *AuditRecord) { r.ActorID = "op-2" }},
		{"actor_type", func(r *AuditRecord) { r.ActorType = "service_account" }},
		{"action", func(r *AuditRecord) { r.Action = "policy.delete" }},
		{"resource_type", func(r *AuditRecord) { r.ResourceType = "rule" }},
		{"resource_id", func(r *AuditRecord) { r.ResourceID = "pol-2" }},
		{"environment", func(r *AuditRecord) { r.Environment = "staging" }},
		{"source_ip", func(r *AuditRecord) { r.SourceIP = "10.0.0.2" }},
		{"before_state", func(r *AuditRecord) { r.BeforeState = map[string]interface{}{"enabled": true} }},
		{"after_state", func(r *AuditRecord) { r.AfterState = map[string]interface{}{"enabled": false} }},
		{"outcome", func(r *AuditRecord) { r.Outcome = "denied" }},
		{"prev_hash", func(r *AuditRecord) { r.PrevHash = "prev-hash-1" }},
		{"created_at", func(r *AuditRecord) { r.CreatedAt = ts.Add(time.Microsecond) }},
	}

	for _, tc := range mutations {
		t.Run("tamper_"+tc.name, func(t *testing.T) {
			copyRec := base
			tc.mutate(&copyRec)
			if got := CalculateAuditRecordHash(copyRec); got == baseHash {
				t.Fatalf("expected mutating %s to change audit record hash", tc.name)
			}
		})
	}

	// Delimiter collision check: shifting characters across adjacent fields must not collide.
	rA := base
	rA.Action = "ab|3:cde"
	rA.ResourceType = "f"

	rB := base
	rB.Action = "ab"
	rB.ResourceType = "cde|1:f"

	if CalculateAuditRecordHash(rA) == CalculateAuditRecordHash(rB) {
		t.Fatal("expected length-prefixed encoding to prevent delimiter collisions")
	}
}
