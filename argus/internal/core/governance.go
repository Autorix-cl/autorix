package core

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
)

// AuditRecord is an immutable audit log entry chained via SHA-256 (P8-S1).
type AuditRecord struct {
	ID           uuid.UUID              `json:"id"`
	ActorID      string                 `json:"actor_id"`
	ActorType    string                 `json:"actor_type"`
	Action       string                 `json:"action"`
	ResourceType string                 `json:"resource_type"`
	ResourceID   string                 `json:"resource_id"`
	Environment  string                 `json:"environment"`
	BeforeState  map[string]interface{} `json:"before_state,omitempty"`
	AfterState   map[string]interface{} `json:"after_state,omitempty"`
	RequestID    string                 `json:"request_id,omitempty"`
	SourceIP     string                 `json:"source_ip,omitempty"`
	UserAgent    string                 `json:"user_agent,omitempty"`
	Outcome      string                 `json:"outcome"`
	PrevHash     string                 `json:"prev_hash"`
	RecordHash   string                 `json:"record_hash"`
	CreatedAt    time.Time              `json:"created_at"`
}

type AuditRecordFilter struct {
	ActorID      string
	Action       string
	ResourceType string
	ResourceID   string
	Environment  string
	Outcome      string
	Limit        int
	Cursor       string
}

// CalculateAuditRecordHash computes a collision-resistant SHA-256 digest over
// the full security-relevant audit payload using length-prefixed fields so
// delimiter collisions across adjacent values are impossible.
func CalculateAuditRecordHash(rec AuditRecord) string {
	ts := rec.CreatedAt.UTC().Truncate(time.Microsecond).Format(time.RFC3339Nano)
	fields := []string{
		rec.PrevHash,
		rec.ID.String(),
		rec.ActorID,
		rec.ActorType,
		rec.Action,
		rec.ResourceType,
		rec.ResourceID,
		rec.Environment,
		rec.SourceIP,
		canonicalAuditStateJSON(rec.BeforeState),
		canonicalAuditStateJSON(rec.AfterState),
		rec.Outcome,
		ts,
	}

	h := sha256.New()
	for _, f := range fields {
		_, _ = h.Write([]byte(strconv.Itoa(len(f))))
		_, _ = h.Write([]byte{':'})
		_, _ = h.Write([]byte(f))
		_, _ = h.Write([]byte{'|'})
	}
	return hex.EncodeToString(h.Sum(nil))
}

func canonicalAuditStateJSON(state map[string]interface{}) string {
	if state == nil {
		return ""
	}
	b, err := json.Marshal(state)
	if err != nil {
		return fmt.Sprintf("marshal-error:%v", err)
	}
	return string(b)
}

type AuditVerificationResult struct {
	Verified    bool        `json:"verified"`
	ChainLength int         `json:"chain_length"`
	HeadHash    string      `json:"head_hash"`
	GenesisHash string      `json:"genesis_hash,omitempty"`
	BrokenLink  *BrokenLink `json:"broken_link,omitempty"`
	VerifiedAt  time.Time   `json:"verified_at"`
	Algorithm   string      `json:"algorithm"`
}

type BrokenLink struct {
	ID               uuid.UUID `json:"id"`
	Sequence         int       `json:"sequence"`
	ExpectedPrevHash string    `json:"expected_prev_hash"`
	ActualPrevHash   string    `json:"actual_prev_hash"`
}

// ConfigRevision tracks configuration changes and enables rollback (P8-S2).
type ConfigRevision struct {
	ID           uuid.UUID              `json:"id"`
	Engine       string                 `json:"engine"`
	ResourceType string                 `json:"resource_type"`
	ResourceID   string                 `json:"resource_id"`
	RevisionNum  int                    `json:"revision_num"`
	Author       string                 `json:"author"`
	Note         string                 `json:"note"`
	Config       map[string]interface{} `json:"config"`
	CreatedAt    time.Time              `json:"created_at"`
}

// Tenancy / Hierarchy Models (P8-S3)
type Organisation struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	CreatedAt time.Time `json:"created_at"`
}

type Project struct {
	ID        uuid.UUID `json:"id"`
	OrgID     uuid.UUID `json:"org_id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	CreatedAt time.Time `json:"created_at"`
}

// Change Requests & Maintenance Windows (P8-S4)
type ChangeRequestStatus string

const (
	ChangeRequestStatusPending  ChangeRequestStatus = "pending"
	ChangeRequestStatusApproved ChangeRequestStatus = "approved"
	ChangeRequestStatusRejected ChangeRequestStatus = "rejected"
	ChangeRequestStatusExpired  ChangeRequestStatus = "expired"
)

type ChangeRequest struct {
	ID             uuid.UUID              `json:"id"`
	RequesterID    string                 `json:"requester_id"`
	ApproverID     *string                `json:"approver_id,omitempty"`
	Action         string                 `json:"action"`
	TargetResource string                 `json:"target_resource"`
	Payload        map[string]interface{} `json:"payload"`
	Status         ChangeRequestStatus    `json:"status"`
	ExpiresAt      *time.Time             `json:"expires_at,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
}

type MaintenanceWindow struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	StartsAt    time.Time `json:"starts_at"`
	EndsAt      time.Time `json:"ends_at"`
	CreatedAt   time.Time `json:"created_at"`
}

// Compliance Evidence (P8-S4)
type ComplianceControlStatus string

const (
	ComplianceStatusCompliant    ComplianceControlStatus = "compliant"
	ComplianceStatusNeedsReview  ComplianceControlStatus = "needs_review"
	ComplianceStatusNonCompliant ComplianceControlStatus = "non_compliant"
)

type ComplianceEvidenceItem struct {
	ID              string                  `json:"id"`
	Framework       string                  `json:"framework"`
	ControlID       string                  `json:"control_id"`
	ControlName     string                  `json:"control_name"`
	Status          ComplianceControlStatus `json:"status"`
	EvidenceType    string                  `json:"evidence_type"`
	Description     string                  `json:"description"`
	Engine          string                  `json:"engine,omitempty"`
	LastEvaluatedAt time.Time               `json:"last_evaluated_at"`
	Evaluator       string                  `json:"evaluator"`
	ArtifactsCount  int                     `json:"artifacts_count"`
	Details         map[string]interface{}  `json:"details,omitempty"`
}

type ComplianceEvidenceSummary struct {
	TotalControls     int     `json:"total_controls"`
	CompliantControls int     `json:"compliant_controls"`
	ReviewRequired    int     `json:"review_required"`
	ScorePercent      float64 `json:"score_percent"`
}

type ComplianceEvidenceReport struct {
	Data    []ComplianceEvidenceItem  `json:"data"`
	Summary ComplianceEvidenceSummary `json:"summary"`
}
