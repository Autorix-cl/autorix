package core

import (
	"time"

	"github.com/google/uuid"
)

// Endpoints an instance advertises to the registry.
type Endpoints struct {
	RESTURL string `json:"rest_url,omitempty"`
	GRPCURL string `json:"grpc_url,omitempty"`
}

// Instance is one registered engine process (P2-S2). EngineType +
// InstanceID + EnvironmentID is the upsert key: a restarting engine
// updates its existing row instead of accumulating duplicates.
type Instance struct {
	ID              uuid.UUID
	EngineType      string
	InstanceID      string
	EnvironmentID   uuid.UUID
	Version         string
	BuildSHA        string
	SchemaVersion   string
	Endpoints       Endpoints
	Capabilities    []string
	Labels          map[string]string
	Status          InstanceStatus
	FirstSeenAt     time.Time
	LastHeartbeatAt *time.Time
	DeregisteredAt  *time.Time
	EnrolledBy      string
	// Unverified is true for an instance registered via the manual
	// fallback (P2-S3-T7) rather than a minted enrollment token — the
	// trust difference must stay visible, never silently hidden.
	Unverified bool
}

// RegistrationRequest is what an engine sends to register or re-register.
type RegistrationRequest struct {
	EngineType    string
	InstanceID    string
	EnvironmentID uuid.UUID
	Version       string
	BuildSHA      string
	SchemaVersion string
	Endpoints     Endpoints
	Capabilities  []string
}

// DependencyProbe is one dependency's reachability as reported in a
// heartbeat.
type DependencyProbe struct {
	EngineType string
	Reachable  bool
	LatencyMs  float64
}

// HeartbeatReport is what an engine sends on its heartbeat interval.
type HeartbeatReport struct {
	Live         bool
	Ready        bool
	Dependencies []DependencyProbe
	ReceivedAt   time.Time
}

// InstanceFilter narrows a ListInstances call (P2-S6-T1).
type InstanceFilter struct {
	EnvironmentID uuid.UUID
	EngineType    string
	Status        InstanceStatus
	Cursor        string
	Limit         int
}

// InstanceEventType names one entry in an instance's timeline (P2-S2,
// instance_events — append-only, written in the same transaction as the
// state transition it records).
type InstanceEventType string

const (
	EventRegistered    InstanceEventType = "registered"
	EventUpgraded      InstanceEventType = "upgraded"
	EventDegraded      InstanceEventType = "degraded"
	EventUnreachable   InstanceEventType = "unreachable"
	EventDeregistered  InstanceEventType = "deregistered"
	EventEvicted       InstanceEventType = "evicted"
	EventConfigChanged InstanceEventType = "config_changed"
	EventFlapping      InstanceEventType = "flapping"
)

// InstanceEvent is one row of the append-only fleet timeline.
type InstanceEvent struct {
	ID         int64
	InstanceID uuid.UUID
	Type       InstanceEventType
	Detail     map[string]interface{}
	OccurredAt time.Time
}

// InstanceCredentialMaterial is what gets persisted for one instance
// credential (P2-S3-T3/T4): a one-way hash used for direct comparison
// (e.g. authenticating a RotateCredential request) and an AES-256-GCM
// sealed copy of the same raw secret, which the HMAC heartbeat signature
// check needs to decrypt back to key material — a hash alone cannot do
// that.
type InstanceCredentialMaterial struct {
	SecretHash      string
	SecretEncrypted []byte
}

// InstanceCredentialRecord is one instance's live credential state,
// including the still-valid previous generation during a rotation overlap
// window (P2-S3-T3).
type InstanceCredentialRecord struct {
	SecretHash              string
	SecretEncrypted         []byte
	PreviousSecretHash      string
	PreviousSecretEncrypted []byte
	PreviousValidUntil      *time.Time
}

// CredentialRotationOverlap is how long a rotated-out credential keeps
// authenticating heartbeats alongside its replacement (P2-S3-T3), so
// rotation is never a hard, synchronization-sensitive cutover.
const CredentialRotationOverlap = 10 * time.Minute

// HeartbeatTimestampWindow bounds how far a heartbeat's self-reported
// timestamp_unix may drift from the server's clock before it is rejected
// as stale (replay protection, P2-S3-T4).
const HeartbeatTimestampWindow = 30 * time.Second

// HeartbeatNonceTTL is how long a heartbeat nonce is remembered for replay
// detection — comfortably longer than HeartbeatTimestampWindow so a nonce
// cannot be replayed even at the edge of the timestamp window.
const HeartbeatNonceTTL = 2 * HeartbeatTimestampWindow

// EnrollmentAuditAction names one recorded enrollment-token lifecycle
// event (P2-S3-T6).
type EnrollmentAuditAction string

const (
	AuditActionMint          EnrollmentAuditAction = "mint"
	AuditActionConsume       EnrollmentAuditAction = "consume"
	AuditActionConsumeFailed EnrollmentAuditAction = "consume_failed"
	AuditActionRevoke        EnrollmentAuditAction = "revoke"
)

// EnrollmentAuditEntry is one row of the enrollment audit trail: every
// mint, use, failed attempt and revocation of an enrollment token
// (P2-S3-T6).
type EnrollmentAuditEntry struct {
	ID            int64
	TokenID       *uuid.UUID
	EngineType    string
	EnvironmentID uuid.UUID
	Actor         string
	Origin        string
	Action        EnrollmentAuditAction
	Detail        map[string]interface{}
	OccurredAt    time.Time
}

// EnrollmentAuditFilter narrows a ListEnrollmentAudit call.
type EnrollmentAuditFilter struct {
	TokenID uuid.UUID
	Action  EnrollmentAuditAction
	Cursor  string
	Limit   int
}

// TopologyNode represents one node (an active instance or logical engine) in the fleet dependency graph (P2-S6-T3).
type TopologyNode struct {
	ID            uuid.UUID      `json:"id"`
	InstanceID    string         `json:"instance_id"`
	EngineType    string         `json:"engine_type"`
	EnvironmentID uuid.UUID      `json:"environment_id"`
	Environment   string         `json:"environment"`
	Status        InstanceStatus `json:"status"`
	Version       string         `json:"version"`
	Endpoints     Endpoints      `json:"endpoints"`
}

// TopologyEdge represents one declared or probed dependency between engines (P2-S6-T3).
type TopologyEdge struct {
	SourceInstanceID   uuid.UUID  `json:"source_instance_id"`
	SourceEngineType   string     `json:"source_engine_type"`
	TargetEngineType   string     `json:"target_engine_type"`
	Declared           bool       `json:"declared"`
	LastProbeReachable *bool      `json:"last_probe_reachable,omitempty"`
	LastProbeLatencyMs *float64   `json:"last_probe_latency_ms,omitempty"`
	LastProbedAt       *time.Time `json:"last_probed_at,omitempty"`
}

// TopologyGraph is the fleet dependency graph with annotated edge health (P2-S6-T3).
type TopologyGraph struct {
	Nodes []TopologyNode `json:"nodes"`
	Edges []TopologyEdge `json:"edges"`
}

