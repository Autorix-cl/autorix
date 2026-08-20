package http

import (
	"time"

	"github.com/autorix/argus/internal/core"
	"github.com/google/uuid"
)

// apiInstance is the console-facing JSON shape for an Instance, mirroring
// the gRPC Instance message's fields.
type apiInstance struct {
	ID              string            `json:"id"`
	EngineType      string            `json:"engine_type"`
	InstanceID      string            `json:"instance_id"`
	Environment     string            `json:"environment"`
	Version         string            `json:"version"`
	BuildSHA        string            `json:"build_sha"`
	SchemaVersion   string            `json:"schema_version"`
	Capabilities    []string          `json:"capabilities"`
	Endpoints       core.Endpoints    `json:"endpoints"`
	Labels          map[string]string `json:"labels,omitempty"`
	Status          string            `json:"status"`
	FirstSeenAt     time.Time         `json:"first_seen_at"`
	LastHeartbeatAt *time.Time        `json:"last_heartbeat_at,omitempty"`
	Unverified      bool              `json:"unverified"`
}

func toAPIInstance(inst core.Instance, envSlug string) apiInstance {
	return apiInstance{
		ID:              inst.ID.String(),
		EngineType:      inst.EngineType,
		InstanceID:      inst.InstanceID,
		Environment:     envSlug,
		Version:         inst.Version,
		BuildSHA:        inst.BuildSHA,
		SchemaVersion:   inst.SchemaVersion,
		Capabilities:    inst.Capabilities,
		Endpoints:       inst.Endpoints,
		Labels:          inst.Labels,
		Status:          string(inst.Status),
		FirstSeenAt:     inst.FirstSeenAt,
		LastHeartbeatAt: inst.LastHeartbeatAt,
		Unverified:      inst.Unverified,
	}
}

type apiEvent struct {
	Type       string                 `json:"type"`
	Detail     map[string]interface{} `json:"detail,omitempty"`
	OccurredAt time.Time              `json:"occurred_at"`
}

func toAPIEvents(events []core.InstanceEvent) []apiEvent {
	out := make([]apiEvent, 0, len(events))
	for _, ev := range events {
		out = append(out, apiEvent{Type: string(ev.Type), Detail: ev.Detail, OccurredAt: ev.OccurredAt})
	}
	return out
}

type apiDependency struct {
	DependsOnEngineType string     `json:"depends_on_engine_type"`
	Declared            bool       `json:"declared"`
	LastProbeReachable  *bool      `json:"last_probe_reachable,omitempty"`
	LastProbeLatencyMs  *float64   `json:"last_probe_latency_ms,omitempty"`
	LastProbedAt        *time.Time `json:"last_probed_at,omitempty"`
}

// apiEnrollmentToken is the console-facing JSON shape for an enrollment
// token. Token only carries a value on the mint response — every other
// response (list) leaves it empty, since only the hash is ever persisted
// and the plaintext is shown exactly once (P2-S3-T1).
type apiEnrollmentToken struct {
	ID            string     `json:"id"`
	Token         string     `json:"token,omitempty"`
	EngineType    string     `json:"engine_type"`
	Environment   string     `json:"environment,omitempty"`
	UsesAllowed   int        `json:"uses_allowed"`
	UsesConsumed  int        `json:"uses_consumed"`
	ExpiresAt     time.Time  `json:"expires_at"`
	CreatedBy     string     `json:"created_by"`
	CreatedAt     time.Time  `json:"created_at"`
	RevokedAt     *time.Time `json:"revoked_at,omitempty"`
}

func toAPIEnrollmentToken(t core.EnrollmentToken, envSlug string, rawToken string) apiEnrollmentToken {
	return apiEnrollmentToken{
		ID:           t.ID.String(),
		Token:        rawToken,
		EngineType:   t.EngineType,
		Environment:  envSlug,
		UsesAllowed:  t.UsesAllowed,
		UsesConsumed: t.UsesConsumed,
		ExpiresAt:    t.ExpiresAt,
		CreatedBy:    t.CreatedBy,
		CreatedAt:    t.CreatedAt,
		RevokedAt:    t.RevokedAt,
	}
}

type apiEnrollmentAuditEntry struct {
	ID            int64                  `json:"id"`
	TokenID       string                 `json:"token_id,omitempty"`
	EngineType    string                 `json:"engine_type,omitempty"`
	EnvironmentID string                 `json:"environment_id,omitempty"`
	Actor         string                 `json:"actor"`
	Origin        string                 `json:"origin,omitempty"`
	Action        string                 `json:"action"`
	Detail        map[string]interface{} `json:"detail,omitempty"`
	OccurredAt    time.Time              `json:"occurred_at"`
}

func toAPIEnrollmentAuditEntry(e core.EnrollmentAuditEntry) apiEnrollmentAuditEntry {
	out := apiEnrollmentAuditEntry{
		ID:         e.ID,
		EngineType: e.EngineType,
		Actor:      e.Actor,
		Origin:     e.Origin,
		Action:     string(e.Action),
		Detail:     e.Detail,
		OccurredAt: e.OccurredAt,
	}
	if e.TokenID != nil {
		out.TokenID = e.TokenID.String()
	}
	if e.EnvironmentID != uuid.Nil {
		out.EnvironmentID = e.EnvironmentID.String()
	}
	return out
}

func toAPIDependencies(deps []core.InstanceDependency) []apiDependency {
	out := make([]apiDependency, 0, len(deps))
	for _, d := range deps {
		out = append(out, apiDependency{
			DependsOnEngineType: d.DependsOnEngineType,
			Declared:            d.Declared,
			LastProbeReachable:  d.LastProbeReachable,
			LastProbeLatencyMs:  d.LastProbeLatencyMs,
			LastProbedAt:        d.LastProbedAt,
		})
	}
	return out
}
