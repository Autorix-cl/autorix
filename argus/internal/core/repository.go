package core

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Repository is Argus's storage contract (P2-S2-T2), kept free of SQL and
// transport concerns so callers only depend on this interface. The
// Postgres implementation lives in internal/storage/postgres.
type Repository interface {
	// Environments

	CreateEnvironment(ctx context.Context, name, slug, description string, isProduction bool) (Environment, error)
	GetEnvironmentBySlug(ctx context.Context, slug string) (Environment, error)
	// GetEnvironmentByID resolves the slug backing one environment_id
	// column, e.g. for rendering the human-readable environment on an
	// Instance without the caller having to fan out to ListEnvironments.
	GetEnvironmentByID(ctx context.Context, id uuid.UUID) (Environment, error)
	ListEnvironments(ctx context.Context) ([]Environment, error)

	// Engine type catalog (seeded, read-mostly)

	ListEngineTypes(ctx context.Context) ([]EngineType, error)
	GetEngineType(ctx context.Context, slug string) (EngineType, error)

	// Enrollment tokens (P2-S3)

	CreateEnrollmentToken(ctx context.Context, t EnrollmentToken) (EnrollmentToken, error)
	GetEnrollmentTokenByHash(ctx context.Context, tokenHash string) (EnrollmentToken, error)
	ListEnrollmentTokens(ctx context.Context, engineType string, environmentID uuid.UUID) ([]EnrollmentToken, error)
	ConsumeEnrollmentToken(ctx context.Context, id uuid.UUID) error
	RevokeEnrollmentToken(ctx context.Context, id uuid.UUID) error

	// Instances (P2-S2-T3: upsert keyed on engine_type+instance_id+environment_id)

	UpsertInstance(ctx context.Context, req RegistrationRequest, enrolledBy string, unverified bool) (Instance, error)
	GetInstance(ctx context.Context, id uuid.UUID) (Instance, error)
	ListInstances(ctx context.Context, filter InstanceFilter) ([]Instance, string, bool, error)
	SetInstanceStatus(ctx context.Context, id uuid.UUID, status InstanceStatus) error
	RecordHeartbeat(ctx context.Context, id uuid.UUID, report HeartbeatReport) error
	Deregister(ctx context.Context, id uuid.UUID, reason string) error
	ForceRemove(ctx context.Context, id uuid.UUID) error
	// ListStaleInstances returns instances whose last_heartbeat_at is
	// older than olderThan, for the evaluator sweep (P2-S5-T2) to advance.
	ListStaleInstances(ctx context.Context, olderThan time.Time) ([]Instance, error)

	// Instance credentials (P2-S3-T3). SecretEncrypted carries an
	// AES-256-GCM sealed copy of the raw credential alongside its hash, so
	// the HMAC heartbeat signature (P2-S3-T4) can be verified without
	// storing the secret in the clear — the hash alone (a one-way digest)
	// cannot reconstruct the key material HMAC needs.

	SetInstanceCredential(ctx context.Context, instanceID uuid.UUID, cred InstanceCredentialMaterial) error
	RotateInstanceCredential(ctx context.Context, instanceID uuid.UUID, newCred InstanceCredentialMaterial, overlapUntil time.Time) error
	GetInstanceCredential(ctx context.Context, instanceID uuid.UUID) (InstanceCredentialRecord, error)
	RevokeInstanceCredential(ctx context.Context, instanceID uuid.UUID) error

	// Events (P2-S2-T4: append-only, same transaction as the state change it records)

	AppendEvent(ctx context.Context, instanceID uuid.UUID, eventType InstanceEventType, detail map[string]interface{}) error
	ListEvents(ctx context.Context, instanceID *uuid.UUID, limit int) ([]InstanceEvent, error)
	// SetInstanceStatusWithEvent writes the status column and its matching
	// timeline event in one transaction (P2-S2-T4), so the two can never
	// disagree.
	SetInstanceStatusWithEvent(ctx context.Context, id uuid.UUID, status InstanceStatus, eventType InstanceEventType, detail map[string]interface{}) error

	// RegisterInstance is the transactional writer backing the gRPC
	// Register RPC (P2-S3): identity upsert, "registering" status, the
	// freshly minted credential material, and the "registered" timeline
	// event all commit or roll back together.
	RegisterInstance(ctx context.Context, req RegistrationRequest, enrolledBy string, unverified bool, cred InstanceCredentialMaterial) (Instance, error)

	// Dependencies (P2-S6-T3 topology)

	DeclareDependency(ctx context.Context, instanceID uuid.UUID, dependsOnEngineType string) error
	RecordDependencyProbe(ctx context.Context, instanceID uuid.UUID, dependsOnEngineType string, reachable bool, latencyMs float64) error
	ListDependencies(ctx context.Context, instanceID uuid.UUID) ([]InstanceDependency, error)
	GetTopology(ctx context.Context, environmentID *uuid.UUID) (TopologyGraph, error)

	// Retention (P2-S2-T6): PruneOlderThan deletes timeline events older
	// than cutoff and evicted instances last touched before cutoff — never
	// deregistered or currently-active instances, so pruning can never
	// silently drop a live or graceful-exit record.
	PruneOlderThan(ctx context.Context, cutoff time.Time) (eventsDeleted int64, instancesDeleted int64, err error)

	// Enrollment audit trail (P2-S3-T6)

	AppendEnrollmentAudit(ctx context.Context, entry EnrollmentAuditEntry) error
	ListEnrollmentAudit(ctx context.Context, filter EnrollmentAuditFilter) ([]EnrollmentAuditEntry, string, bool, error)

	// Console Identity: Operators & Credentials (P3-S1-T1)

	CountOperators(ctx context.Context) (int64, error)
	CreateOperatorWithPassword(ctx context.Context, op Operator, passwordHash string) (Operator, error)
	GetOperatorByEmail(ctx context.Context, email string) (OperatorWithCredential, error)
	GetOperatorByID(ctx context.Context, id uuid.UUID) (Operator, error)
	ListOperators(ctx context.Context) ([]Operator, error)
	UpdateOperatorFailedAttempts(ctx context.Context, id uuid.UUID, failedAttempts int, lockedUntil *time.Time) error
	ResetOperatorFailedAttempts(ctx context.Context, id uuid.UUID) error

	// Console Identity: Sessions (P3-S2-T2)

	CreateSession(ctx context.Context, session OperatorSession) (OperatorSession, error)
	GetSessionByTokenHash(ctx context.Context, tokenHash string) (OperatorSession, Operator, error)
	TouchSession(ctx context.Context, sessionID uuid.UUID, lastActiveAt time.Time, extendExpiry time.Time) error
	RevokeSession(ctx context.Context, sessionID uuid.UUID) error
	RevokeOperatorSessions(ctx context.Context, operatorID uuid.UUID) error

	// Console Identity: Bootstrap Tokens (P3-S1-T2)

	CreateBootstrapToken(ctx context.Context, tokenHash string) error
	ConsumeBootstrapToken(ctx context.Context, tokenHash string) (bool, error)
	HasValidBootstrapToken(ctx context.Context) (bool, error)

	// Console Identity: Role Bindings (P3-S4-T2)

	CreateRoleBinding(ctx context.Context, rb RoleBinding) (RoleBinding, error)
	ListRoleBindings(ctx context.Context, operatorID uuid.UUID) ([]RoleBinding, error)
	DeleteRoleBinding(ctx context.Context, id uuid.UUID) error

	// Console Identity: Personal Access Tokens (P3-S6-T2)

	CreatePersonalAccessToken(ctx context.Context, pat PersonalAccessToken) (PersonalAccessToken, error)
	ListPersonalAccessTokens(ctx context.Context, operatorID uuid.UUID) ([]PersonalAccessToken, error)
	RevokePersonalAccessToken(ctx context.Context, id uuid.UUID, operatorID uuid.UUID) error

	// Console Identity: Service Accounts (P3-S6-T3)

	CreateServiceAccount(ctx context.Context, sa ServiceAccount) (ServiceAccount, error)
	ListServiceAccounts(ctx context.Context) ([]ServiceAccount, error)
	GetServiceAccountByTokenHash(ctx context.Context, tokenHash string) (ServiceAccount, error)
	DeleteServiceAccount(ctx context.Context, id uuid.UUID) error

	// Audit Records & Hash Chaining (P8-S1)

	RecordAudit(ctx context.Context, record AuditRecord) (AuditRecord, error)
	ListAuditRecords(ctx context.Context, filter AuditRecordFilter) ([]AuditRecord, string, bool, error)
	VerifyAuditChain(ctx context.Context) (bool, error)
	VerifyAuditChainDetailed(ctx context.Context) (AuditVerificationResult, error)
	ExportAuditRecords(ctx context.Context, format string) ([]byte, error)

	// Config Revisions & Rollback (P8-S2)

	RecordConfigRevision(ctx context.Context, rev ConfigRevision) (ConfigRevision, error)
	ListConfigRevisions(ctx context.Context, engine, resourceType, resourceID string) ([]ConfigRevision, error)
	GetConfigRevision(ctx context.Context, id uuid.UUID) (ConfigRevision, error)
	RollbackConfig(ctx context.Context, targetRevisionID uuid.UUID, author, note string) (ConfigRevision, error)

	// Tenancy & Hierarchy (P8-S3)

	CreateOrganisation(ctx context.Context, org Organisation) (Organisation, error)
	GetOrganisation(ctx context.Context, id uuid.UUID) (Organisation, error)
	GetOrganisationBySlug(ctx context.Context, slug string) (Organisation, error)
	ListOrganisations(ctx context.Context) ([]Organisation, error)
	DeleteOrganisation(ctx context.Context, id uuid.UUID) error

	CreateProject(ctx context.Context, project Project) (Project, error)
	GetProject(ctx context.Context, id uuid.UUID) (Project, error)
	GetProjectBySlug(ctx context.Context, orgID uuid.UUID, slug string) (Project, error)
	ListProjects(ctx context.Context, orgID *uuid.UUID) ([]Project, error)
	DeleteProject(ctx context.Context, id uuid.UUID) error

	// Change Requests & Maintenance Windows (P8-S4)

	CreateChangeRequest(ctx context.Context, req ChangeRequest) (ChangeRequest, error)
	GetChangeRequest(ctx context.Context, id uuid.UUID) (ChangeRequest, error)
	ListChangeRequests(ctx context.Context, status ChangeRequestStatus) ([]ChangeRequest, error)
	ApproveChangeRequest(ctx context.Context, id uuid.UUID, approverID string) (ChangeRequest, error)
	RejectChangeRequest(ctx context.Context, id uuid.UUID, approverID string) (ChangeRequest, error)

	CreateMaintenanceWindow(ctx context.Context, mw MaintenanceWindow) (MaintenanceWindow, error)
	GetMaintenanceWindow(ctx context.Context, id uuid.UUID) (MaintenanceWindow, error)
	ListMaintenanceWindows(ctx context.Context) ([]MaintenanceWindow, error)
	IsInMaintenanceWindow(ctx context.Context, at time.Time) (bool, error)

	// Compliance Evidence Aggregation (P8-S4)

	GetComplianceEvidence(ctx context.Context) (ComplianceEvidenceReport, error)
}

// Environment groups instances (e.g. "production", "staging").
type Environment struct {
	ID           uuid.UUID  `json:"id"`
	ProjectID    *uuid.UUID `json:"project_id,omitempty"`
	Name         string     `json:"name"`
	Slug         string     `json:"slug"`
	Type         string     `json:"type,omitempty"`
	Description  string     `json:"description"`
	IsProduction bool       `json:"is_production"`
	CreatedAt    time.Time  `json:"created_at"`
}

// EngineType is one row of the seeded catalog (P2-S2-T5).
type EngineType struct {
	Slug             string
	DisplayName      string
	DefaultRESTPort  *int
	DefaultGRPCPort  *int
	Protocol         string
	CapabilitySchema map[string]interface{}
}

// EnrollmentToken is a one-time (or bounded-use) credential that admits an
// engine into the fleet (P2-S3-T1). Only its hash is stored — the plaintext
// is returned exactly once, at mint time.
type EnrollmentToken struct {
	ID            uuid.UUID
	TokenHash     string
	EngineType    string
	EnvironmentID uuid.UUID
	UsesAllowed   int
	UsesConsumed  int
	ExpiresAt     time.Time
	CreatedBy     string
	CreatedAt     time.Time
	RevokedAt     *time.Time
}

// InstanceDependency is one declared/observed edge in the topology graph.
type InstanceDependency struct {
	InstanceID          uuid.UUID
	DependsOnEngineType string
	Declared            bool
	LastProbeReachable  *bool
	LastProbeLatencyMs  *float64
	LastProbedAt        *time.Time
}
