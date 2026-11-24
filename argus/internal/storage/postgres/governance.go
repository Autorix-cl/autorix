package postgres

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/autorix/argus/internal/core"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ---------------------------------------------------------------------
// Audit Log & Hash Chaining (P8-S1)
// ---------------------------------------------------------------------

// auditChainAdvisoryLockID serializes audit chain head reads and inserts
// inside RecordAudit so concurrent writes never fork prev_hash.
const auditChainAdvisoryLockID int64 = 0x6172677573617564

const maxAuditExportLimit = 10000

func decodeAuditStateJSON(raw []byte) (map[string]interface{}, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var out map[string]interface{}
	if err := dec.Decode(&out); err != nil {
		return nil, err
	}
	return out, nil
}

// SanitizeCSVCell neutralizes spreadsheet formula injection prefixes (=, +, -, @, \t, \r)
// by prepending a single quote before CSV encoding.
func SanitizeCSVCell(value string) string {
	if value == "" {
		return value
	}
	switch value[0] {
	case '=', '+', '-', '@', '\t', '\r':
		return "'" + value
	}
	trimmed := strings.TrimLeft(value, " \t\r\n")
	if len(trimmed) > 0 {
		switch trimmed[0] {
		case '=', '+', '-', '@':
			return "'" + value
		}
	}
	return value
}

func (r *Repository) RecordAudit(ctx context.Context, record core.AuditRecord) (core.AuditRecord, error) {
	if record.ID == uuid.Nil {
		record.ID = uuid.New()
	}
	if record.Outcome == "" {
		record.Outcome = "success"
	}

	var beforeStateJSON, afterStateJSON []byte
	var err error
	if record.BeforeState != nil {
		beforeStateJSON, err = json.Marshal(record.BeforeState)
		if err != nil {
			return core.AuditRecord{}, fmt.Errorf("marshaling before_state: %w", err)
		}
	}
	if record.AfterState != nil {
		afterStateJSON, err = json.Marshal(record.AfterState)
		if err != nil {
			return core.AuditRecord{}, fmt.Errorf("marshaling after_state: %w", err)
		}
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return core.AuditRecord{}, fmt.Errorf("beginning audit transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, auditChainAdvisoryLockID); err != nil {
		return core.AuditRecord{}, fmt.Errorf("acquiring audit chain lock: %w", err)
	}

	// Fetch latest record_hash under advisory lock to serialize the hash chain.
	var prevHash string
	var prevCreatedAt *time.Time
	err = tx.QueryRow(ctx, `
		SELECT record_hash, created_at FROM audit_records
		ORDER BY created_at DESC, id DESC
		LIMIT 1
	`).Scan(&prevHash, &prevCreatedAt)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return core.AuditRecord{}, fmt.Errorf("fetching previous audit hash: %w", err)
	}

	if record.CreatedAt.IsZero() {
		record.CreatedAt = time.Now().UTC().Truncate(time.Microsecond)
		if prevCreatedAt != nil {
			prevUTC := prevCreatedAt.UTC().Truncate(time.Microsecond)
			if !record.CreatedAt.After(prevUTC) {
				record.CreatedAt = prevUTC.Add(time.Microsecond)
			}
		}
	} else {
		record.CreatedAt = record.CreatedAt.UTC().Truncate(time.Microsecond)
	}

	record.PrevHash = prevHash
	record.RecordHash = core.CalculateAuditRecordHash(record)

	_, err = tx.Exec(ctx, `
		INSERT INTO audit_records (
			id, actor_id, actor_type, action, resource_type, resource_id, environment,
			before_state, after_state, request_id, source_ip, user_agent, outcome,
			prev_hash, record_hash, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			$8, $9, $10, $11, $12, $13,
			$14, $15, $16
		)
	`,
		record.ID, record.ActorID, record.ActorType, record.Action, record.ResourceType, record.ResourceID, record.Environment,
		beforeStateJSON, afterStateJSON, record.RequestID, record.SourceIP, record.UserAgent, record.Outcome,
		record.PrevHash, record.RecordHash, record.CreatedAt,
	)
	if err != nil {
		return core.AuditRecord{}, fmt.Errorf("inserting audit record: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return core.AuditRecord{}, fmt.Errorf("committing audit record: %w", err)
	}

	return record, nil
}

func (r *Repository) ListAuditRecords(ctx context.Context, filter core.AuditRecordFilter) ([]core.AuditRecord, string, bool, error) {
	query := `
		SELECT id, actor_id, actor_type, action, resource_type, resource_id, environment,
		       before_state, after_state, request_id, source_ip, user_agent, outcome,
		       prev_hash, record_hash, created_at
		FROM audit_records
		WHERE 1=1
	`
	args := []interface{}{}
	argIdx := 1

	if filter.ActorID != "" {
		query += fmt.Sprintf(" AND actor_id = $%d", argIdx)
		args = append(args, filter.ActorID)
		argIdx++
	}
	if filter.Action != "" {
		query += fmt.Sprintf(" AND action = $%d", argIdx)
		args = append(args, filter.Action)
		argIdx++
	}
	if filter.ResourceType != "" {
		query += fmt.Sprintf(" AND resource_type = $%d", argIdx)
		args = append(args, filter.ResourceType)
		argIdx++
	}
	if filter.ResourceID != "" {
		query += fmt.Sprintf(" AND resource_id = $%d", argIdx)
		args = append(args, filter.ResourceID)
		argIdx++
	}
	if filter.Environment != "" {
		query += fmt.Sprintf(" AND environment = $%d", argIdx)
		args = append(args, filter.Environment)
		argIdx++
	}
	if filter.Outcome != "" {
		query += fmt.Sprintf(" AND outcome = $%d", argIdx)
		args = append(args, filter.Outcome)
		argIdx++
	}
	if filter.Cursor != "" {
		cursorTime, err := time.Parse(time.RFC3339Nano, filter.Cursor)
		if err == nil {
			query += fmt.Sprintf(" AND created_at < $%d", argIdx)
			args = append(args, cursorTime)
			argIdx++
		}
	}

	limit := filter.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	query += fmt.Sprintf(" ORDER BY created_at DESC, id DESC LIMIT $%d", argIdx)
	args = append(args, limit+1)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, "", false, fmt.Errorf("querying audit records: %w", err)
	}
	defer rows.Close()

	var records []core.AuditRecord
	for rows.Next() {
		var rec core.AuditRecord
		var beforeJSON, afterJSON []byte
		if err := rows.Scan(
			&rec.ID, &rec.ActorID, &rec.ActorType, &rec.Action, &rec.ResourceType, &rec.ResourceID, &rec.Environment,
			&beforeJSON, &afterJSON, &rec.RequestID, &rec.SourceIP, &rec.UserAgent, &rec.Outcome,
			&rec.PrevHash, &rec.RecordHash, &rec.CreatedAt,
		); err != nil {
			return nil, "", false, fmt.Errorf("scanning audit record: %w", err)
		}
		if len(beforeJSON) > 0 {
			rec.BeforeState, err = decodeAuditStateJSON(beforeJSON)
			if err != nil {
				return nil, "", false, fmt.Errorf("decoding audit before_state: %w", err)
			}
		}
		if len(afterJSON) > 0 {
			rec.AfterState, err = decodeAuditStateJSON(afterJSON)
			if err != nil {
				return nil, "", false, fmt.Errorf("decoding audit after_state: %w", err)
			}
		}
		records = append(records, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, "", false, fmt.Errorf("iterating audit records: %w", err)
	}

	hasMore := len(records) > limit
	var nextCursor string
	if hasMore {
		records = records[:limit]
		nextCursor = records[len(records)-1].CreatedAt.UTC().Format(time.RFC3339Nano)
	}

	return records, nextCursor, hasMore, nil
}

func (r *Repository) VerifyAuditChain(ctx context.Context) (bool, error) {
	res, err := r.VerifyAuditChainDetailed(ctx)
	if err != nil {
		return false, err
	}
	return res.Verified, nil
}

func (r *Repository) VerifyAuditChainDetailed(ctx context.Context) (core.AuditVerificationResult, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, actor_id, actor_type, action, resource_type, resource_id, environment,
		       before_state, after_state, source_ip, outcome, prev_hash, record_hash, created_at
		FROM audit_records
		ORDER BY created_at ASC, id ASC
	`)
	if err != nil {
		return core.AuditVerificationResult{}, fmt.Errorf("querying audit chain: %w", err)
	}
	defer rows.Close()

	var count int
	var expectedPrev string
	var headHash string
	var genesisHash string
	now := time.Now().UTC()

	for rows.Next() {
		var rec core.AuditRecord
		var beforeJSON, afterJSON []byte

		if err := rows.Scan(
			&rec.ID, &rec.ActorID, &rec.ActorType, &rec.Action, &rec.ResourceType, &rec.ResourceID, &rec.Environment,
			&beforeJSON, &afterJSON, &rec.SourceIP, &rec.Outcome, &rec.PrevHash, &rec.RecordHash, &rec.CreatedAt,
		); err != nil {
			return core.AuditVerificationResult{}, fmt.Errorf("scanning audit chain row: %w", err)
		}

		if len(beforeJSON) > 0 {
			rec.BeforeState, err = decodeAuditStateJSON(beforeJSON)
			if err != nil {
				return core.AuditVerificationResult{}, fmt.Errorf("decoding audit chain before_state: %w", err)
			}
		}
		if len(afterJSON) > 0 {
			rec.AfterState, err = decodeAuditStateJSON(afterJSON)
			if err != nil {
				return core.AuditVerificationResult{}, fmt.Errorf("decoding audit chain after_state: %w", err)
			}
		}

		if count == 0 {
			genesisHash = rec.RecordHash
		}

		if rec.PrevHash != expectedPrev {
			return core.AuditVerificationResult{
				Verified:    false,
				ChainLength: count,
				HeadHash:    headHash,
				GenesisHash: genesisHash,
				BrokenLink: &core.BrokenLink{
					ID:               rec.ID,
					Sequence:         count,
					ExpectedPrevHash: expectedPrev,
					ActualPrevHash:   rec.PrevHash,
				},
				VerifiedAt: now,
				Algorithm:  "SHA-256",
			}, nil
		}

		recalculated := core.CalculateAuditRecordHash(rec)
		if rec.RecordHash != recalculated {
			return core.AuditVerificationResult{
				Verified:    false,
				ChainLength: count,
				HeadHash:    headHash,
				GenesisHash: genesisHash,
				BrokenLink: &core.BrokenLink{
					ID:               rec.ID,
					Sequence:         count,
					ExpectedPrevHash: recalculated,
					ActualPrevHash:   rec.RecordHash,
				},
				VerifiedAt: now,
				Algorithm:  "SHA-256",
			}, nil
		}

		expectedPrev = rec.RecordHash
		headHash = rec.RecordHash
		count++
	}
	if err := rows.Err(); err != nil {
		return core.AuditVerificationResult{}, fmt.Errorf("iterating audit chain: %w", err)
	}

	return core.AuditVerificationResult{
		Verified:    true,
		ChainLength: count,
		HeadHash:    headHash,
		GenesisHash: genesisHash,
		VerifiedAt:  now,
		Algorithm:   "SHA-256",
	}, nil
}

func (r *Repository) ExportAuditRecords(ctx context.Context, format string) ([]byte, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, actor_id, actor_type, action, resource_type, resource_id, environment,
		       before_state, after_state, request_id, source_ip, user_agent, outcome,
		       prev_hash, record_hash, created_at
		FROM audit_records
		ORDER BY created_at DESC, id DESC
		LIMIT $1
	`, maxAuditExportLimit)
	if err != nil {
		return nil, fmt.Errorf("querying audit records for export: %w", err)
	}
	defer rows.Close()

	records := make([]core.AuditRecord, 0)
	for rows.Next() {
		var rec core.AuditRecord
		var beforeJSON, afterJSON []byte
		if err := rows.Scan(
			&rec.ID, &rec.ActorID, &rec.ActorType, &rec.Action, &rec.ResourceType, &rec.ResourceID, &rec.Environment,
			&beforeJSON, &afterJSON, &rec.RequestID, &rec.SourceIP, &rec.UserAgent, &rec.Outcome,
			&rec.PrevHash, &rec.RecordHash, &rec.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning audit record for export: %w", err)
		}
		if len(beforeJSON) > 0 {
			rec.BeforeState, err = decodeAuditStateJSON(beforeJSON)
			if err != nil {
				return nil, fmt.Errorf("decoding audit before_state for export: %w", err)
			}
		}
		if len(afterJSON) > 0 {
			rec.AfterState, err = decodeAuditStateJSON(afterJSON)
			if err != nil {
				return nil, fmt.Errorf("decoding audit after_state for export: %w", err)
			}
		}
		records = append(records, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating audit records for export: %w", err)
	}

	if strings.ToLower(format) == "csv" {
		return FormatAuditRecordsCSV(records)
	}

	return json.MarshalIndent(records, "", "  ")
}

// FormatAuditRecordsCSV serializes audit records to CSV while neutralizing
// spreadsheet formula prefixes in cell values.
func FormatAuditRecordsCSV(records []core.AuditRecord) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	if err := w.Write([]string{
		"id", "actor_id", "actor_type", "action", "resource_type", "resource_id",
		"environment", "outcome", "prev_hash", "record_hash", "created_at",
	}); err != nil {
		return nil, fmt.Errorf("writing csv header: %w", err)
	}
	for _, rec := range records {
		if err := w.Write([]string{
			rec.ID.String(),
			SanitizeCSVCell(rec.ActorID),
			SanitizeCSVCell(rec.ActorType),
			SanitizeCSVCell(rec.Action),
			SanitizeCSVCell(rec.ResourceType),
			SanitizeCSVCell(rec.ResourceID),
			SanitizeCSVCell(rec.Environment),
			SanitizeCSVCell(rec.Outcome),
			SanitizeCSVCell(rec.PrevHash),
			SanitizeCSVCell(rec.RecordHash),
			rec.CreatedAt.UTC().Format(time.RFC3339Nano),
		}); err != nil {
			return nil, fmt.Errorf("writing csv row: %w", err)
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, fmt.Errorf("flushing csv export: %w", err)
	}
	return buf.Bytes(), nil
}

// ---------------------------------------------------------------------
// Config Revisions & Rollback (P8-S2)
// ---------------------------------------------------------------------

func (r *Repository) RecordConfigRevision(ctx context.Context, rev core.ConfigRevision) (core.ConfigRevision, error) {
	if rev.ID == uuid.Nil {
		rev.ID = uuid.New()
	}
	if rev.CreatedAt.IsZero() {
		rev.CreatedAt = time.Now().UTC()
	}
	rev.CreatedAt = rev.CreatedAt.UTC().Truncate(time.Microsecond)

	if rev.RevisionNum <= 0 {
		var nextRev int
		err := r.pool.QueryRow(ctx, `
			SELECT COALESCE(MAX(revision_num), 0) + 1
			FROM config_revisions
			WHERE engine = $1 AND resource_type = $2 AND resource_id = $3
		`, rev.Engine, rev.ResourceType, rev.ResourceID).Scan(&nextRev)
		if err != nil {
			return core.ConfigRevision{}, fmt.Errorf("calculating next revision number: %w", err)
		}
		rev.RevisionNum = nextRev
	}

	configJSON, err := json.Marshal(rev.Config)
	if err != nil {
		return core.ConfigRevision{}, fmt.Errorf("marshaling config revision: %w", err)
	}

	_, err = r.pool.Exec(ctx, `
		INSERT INTO config_revisions (
			id, engine, resource_type, resource_id, revision_num, author, note, config, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, rev.ID, rev.Engine, rev.ResourceType, rev.ResourceID, rev.RevisionNum, rev.Author, rev.Note, configJSON, rev.CreatedAt)
	if err != nil {
		return core.ConfigRevision{}, fmt.Errorf("inserting config revision: %w", err)
	}

	return rev, nil
}

func (r *Repository) ListConfigRevisions(ctx context.Context, engine, resourceType, resourceID string) ([]core.ConfigRevision, error) {
	query := `
		SELECT id, engine, resource_type, resource_id, revision_num, author, note, config, created_at
		FROM config_revisions
		WHERE 1=1
	`
	args := []interface{}{}
	argIdx := 1

	if engine != "" {
		query += fmt.Sprintf(" AND engine = $%d", argIdx)
		args = append(args, engine)
		argIdx++
	}
	if resourceType != "" {
		query += fmt.Sprintf(" AND resource_type = $%d", argIdx)
		args = append(args, resourceType)
		argIdx++
	}
	if resourceID != "" {
		query += fmt.Sprintf(" AND resource_id = $%d", argIdx)
		args = append(args, resourceID)
	}

	query += " ORDER BY revision_num DESC, created_at DESC"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying config revisions: %w", err)
	}
	defer rows.Close()

	var revisions []core.ConfigRevision
	for rows.Next() {
		var rev core.ConfigRevision
		var configJSON []byte
		if err := rows.Scan(
			&rev.ID, &rev.Engine, &rev.ResourceType, &rev.ResourceID,
			&rev.RevisionNum, &rev.Author, &rev.Note, &configJSON, &rev.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning config revision: %w", err)
		}
		if len(configJSON) > 0 {
			if err := json.Unmarshal(configJSON, &rev.Config); err != nil {
				return nil, fmt.Errorf("unmarshaling config revision: %w", err)
			}
		}
		revisions = append(revisions, rev)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating config revisions: %w", err)
	}

	return revisions, nil
}

func (r *Repository) GetConfigRevision(ctx context.Context, id uuid.UUID) (core.ConfigRevision, error) {
	var rev core.ConfigRevision
	var configJSON []byte
	err := r.pool.QueryRow(ctx, `
		SELECT id, engine, resource_type, resource_id, revision_num, author, note, config, created_at
		FROM config_revisions WHERE id = $1
	`, id).Scan(
		&rev.ID, &rev.Engine, &rev.ResourceType, &rev.ResourceID,
		&rev.RevisionNum, &rev.Author, &rev.Note, &configJSON, &rev.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return core.ConfigRevision{}, core.ErrNotFound
		}
		return core.ConfigRevision{}, fmt.Errorf("getting config revision: %w", err)
	}
	if len(configJSON) > 0 {
		_ = json.Unmarshal(configJSON, &rev.Config)
	}
	return rev, nil
}

func (r *Repository) RollbackConfig(ctx context.Context, targetRevisionID uuid.UUID, author, note string) (core.ConfigRevision, error) {
	target, err := r.GetConfigRevision(ctx, targetRevisionID)
	if err != nil {
		return core.ConfigRevision{}, fmt.Errorf("retrieving target revision for rollback: %w", err)
	}

	rollbackNote := fmt.Sprintf("Rollback to revision %d", target.RevisionNum)
	if note != "" {
		rollbackNote = fmt.Sprintf("Rollback to revision %d: %s", target.RevisionNum, note)
	}

	newRev := core.ConfigRevision{
		Engine:       target.Engine,
		ResourceType: target.ResourceType,
		ResourceID:   target.ResourceID,
		Author:       author,
		Note:         rollbackNote,
		Config:       target.Config,
	}

	return r.RecordConfigRevision(ctx, newRev)
}

// ---------------------------------------------------------------------
// Tenancy & Hierarchy (P8-S3)
// ---------------------------------------------------------------------

func (r *Repository) CreateOrganisation(ctx context.Context, org core.Organisation) (core.Organisation, error) {
	if org.ID == uuid.Nil {
		org.ID = uuid.New()
	}
	if org.CreatedAt.IsZero() {
		org.CreatedAt = time.Now().UTC()
	}
	org.CreatedAt = org.CreatedAt.UTC().Truncate(time.Microsecond)

	err := r.pool.QueryRow(ctx, `
		INSERT INTO organisations (id, name, slug, created_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id, name, slug, created_at
	`, org.ID, org.Name, org.Slug, org.CreatedAt).Scan(&org.ID, &org.Name, &org.Slug, &org.CreatedAt)
	if err != nil {
		return core.Organisation{}, fmt.Errorf("creating organisation: %w", err)
	}
	return org, nil
}

func (r *Repository) GetOrganisation(ctx context.Context, id uuid.UUID) (core.Organisation, error) {
	var org core.Organisation
	err := r.pool.QueryRow(ctx, `
		SELECT id, name, slug, created_at
		FROM organisations WHERE id = $1
	`, id).Scan(&org.ID, &org.Name, &org.Slug, &org.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return core.Organisation{}, core.ErrNotFound
		}
		return core.Organisation{}, fmt.Errorf("getting organisation: %w", err)
	}
	return org, nil
}

func (r *Repository) GetOrganisationBySlug(ctx context.Context, slug string) (core.Organisation, error) {
	var org core.Organisation
	err := r.pool.QueryRow(ctx, `
		SELECT id, name, slug, created_at
		FROM organisations WHERE slug = $1
	`, slug).Scan(&org.ID, &org.Name, &org.Slug, &org.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return core.Organisation{}, core.ErrNotFound
		}
		return core.Organisation{}, fmt.Errorf("getting organisation by slug: %w", err)
	}
	return org, nil
}

func (r *Repository) ListOrganisations(ctx context.Context) ([]core.Organisation, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, name, slug, created_at
		FROM organisations ORDER BY name ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("listing organisations: %w", err)
	}
	defer rows.Close()

	var orgs []core.Organisation
	for rows.Next() {
		var org core.Organisation
		if err := rows.Scan(&org.ID, &org.Name, &org.Slug, &org.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning organisation: %w", err)
		}
		orgs = append(orgs, org)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating organisations: %w", err)
	}
	return orgs, nil
}

func (r *Repository) DeleteOrganisation(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM organisations WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("deleting organisation: %w", err)
	}
	return nil
}

func (r *Repository) CreateProject(ctx context.Context, project core.Project) (core.Project, error) {
	if project.ID == uuid.Nil {
		project.ID = uuid.New()
	}
	if project.CreatedAt.IsZero() {
		project.CreatedAt = time.Now().UTC()
	}
	project.CreatedAt = project.CreatedAt.UTC().Truncate(time.Microsecond)

	err := r.pool.QueryRow(ctx, `
		INSERT INTO projects (id, org_id, name, slug, created_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, org_id, name, slug, created_at
	`, project.ID, project.OrgID, project.Name, project.Slug, project.CreatedAt).Scan(
		&project.ID, &project.OrgID, &project.Name, &project.Slug, &project.CreatedAt,
	)
	if err != nil {
		return core.Project{}, fmt.Errorf("creating project: %w", err)
	}
	return project, nil
}

func (r *Repository) GetProject(ctx context.Context, id uuid.UUID) (core.Project, error) {
	var p core.Project
	err := r.pool.QueryRow(ctx, `
		SELECT id, org_id, name, slug, created_at
		FROM projects WHERE id = $1
	`, id).Scan(&p.ID, &p.OrgID, &p.Name, &p.Slug, &p.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return core.Project{}, core.ErrNotFound
		}
		return core.Project{}, fmt.Errorf("getting project: %w", err)
	}
	return p, nil
}

func (r *Repository) GetProjectBySlug(ctx context.Context, orgID uuid.UUID, slug string) (core.Project, error) {
	var p core.Project
	err := r.pool.QueryRow(ctx, `
		SELECT id, org_id, name, slug, created_at
		FROM projects WHERE org_id = $1 AND slug = $2
	`, orgID, slug).Scan(&p.ID, &p.OrgID, &p.Name, &p.Slug, &p.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return core.Project{}, core.ErrNotFound
		}
		return core.Project{}, fmt.Errorf("getting project by slug: %w", err)
	}
	return p, nil
}

func (r *Repository) ListProjects(ctx context.Context, orgID *uuid.UUID) ([]core.Project, error) {
	query := `SELECT id, org_id, name, slug, created_at FROM projects`
	args := []interface{}{}
	if orgID != nil {
		query += ` WHERE org_id = $1`
		args = append(args, *orgID)
	}
	query += ` ORDER BY name ASC`

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing projects: %w", err)
	}
	defer rows.Close()

	var projects []core.Project
	for rows.Next() {
		var p core.Project
		if err := rows.Scan(&p.ID, &p.OrgID, &p.Name, &p.Slug, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning project: %w", err)
		}
		projects = append(projects, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating projects: %w", err)
	}
	return projects, nil
}

func (r *Repository) DeleteProject(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM projects WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("deleting project: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------
// Change Requests & Maintenance Windows (P8-S4)
// ---------------------------------------------------------------------

func (r *Repository) CreateChangeRequest(ctx context.Context, req core.ChangeRequest) (core.ChangeRequest, error) {
	if req.ID == uuid.Nil {
		req.ID = uuid.New()
	}
	if req.CreatedAt.IsZero() {
		req.CreatedAt = time.Now().UTC()
	}
	req.CreatedAt = req.CreatedAt.UTC().Truncate(time.Microsecond)
	if req.Status == "" {
		req.Status = core.ChangeRequestStatusPending
	}

	payloadJSON, err := json.Marshal(req.Payload)
	if err != nil {
		return core.ChangeRequest{}, fmt.Errorf("marshaling change request payload: %w", err)
	}

	err = r.pool.QueryRow(ctx, `
		INSERT INTO change_requests (
			id, requester_id, approver_id, action, target_resource, payload, status, expires_at, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, requester_id, approver_id, action, target_resource, payload, status, expires_at, created_at
	`, req.ID, req.RequesterID, req.ApproverID, req.Action, req.TargetResource, payloadJSON, req.Status, req.ExpiresAt, req.CreatedAt).Scan(
		&req.ID, &req.RequesterID, &req.ApproverID, &req.Action, &req.TargetResource, &payloadJSON, &req.Status, &req.ExpiresAt, &req.CreatedAt,
	)
	if err != nil {
		return core.ChangeRequest{}, fmt.Errorf("creating change request: %w", err)
	}
	if len(payloadJSON) > 0 {
		_ = json.Unmarshal(payloadJSON, &req.Payload)
	}
	return req, nil
}

func (r *Repository) GetChangeRequest(ctx context.Context, id uuid.UUID) (core.ChangeRequest, error) {
	var req core.ChangeRequest
	var payloadJSON []byte
	err := r.pool.QueryRow(ctx, `
		SELECT id, requester_id, approver_id, action, target_resource, payload, status, expires_at, created_at
		FROM change_requests WHERE id = $1
	`, id).Scan(
		&req.ID, &req.RequesterID, &req.ApproverID, &req.Action, &req.TargetResource, &payloadJSON, &req.Status, &req.ExpiresAt, &req.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return core.ChangeRequest{}, core.ErrNotFound
		}
		return core.ChangeRequest{}, fmt.Errorf("getting change request: %w", err)
	}
	if len(payloadJSON) > 0 {
		_ = json.Unmarshal(payloadJSON, &req.Payload)
	}
	return req, nil
}

func (r *Repository) ListChangeRequests(ctx context.Context, status core.ChangeRequestStatus) ([]core.ChangeRequest, error) {
	query := `SELECT id, requester_id, approver_id, action, target_resource, payload, status, expires_at, created_at FROM change_requests`
	args := []interface{}{}
	if status != "" {
		query += ` WHERE status = $1`
		args = append(args, status)
	}
	query += ` ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing change requests: %w", err)
	}
	defer rows.Close()

	var reqs []core.ChangeRequest
	for rows.Next() {
		var req core.ChangeRequest
		var payloadJSON []byte
		if err := rows.Scan(
			&req.ID, &req.RequesterID, &req.ApproverID, &req.Action, &req.TargetResource, &payloadJSON, &req.Status, &req.ExpiresAt, &req.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning change request: %w", err)
		}
		if len(payloadJSON) > 0 {
			if err := json.Unmarshal(payloadJSON, &req.Payload); err != nil {
				return nil, fmt.Errorf("unmarshaling change request payload: %w", err)
			}
		}
		reqs = append(reqs, req)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating change requests: %w", err)
	}
	return reqs, nil
}

func (r *Repository) ApproveChangeRequest(ctx context.Context, id uuid.UUID, approverID string) (core.ChangeRequest, error) {
	approverID = strings.TrimSpace(approverID)
	if approverID == "" {
		return core.ChangeRequest{}, errors.New("approver_id is required")
	}

	req, err := r.GetChangeRequest(ctx, id)
	if err != nil {
		return core.ChangeRequest{}, err
	}
	if req.RequesterID == approverID {
		return core.ChangeRequest{}, errors.New("requester cannot approve their own change request")
	}
	if req.Status != core.ChangeRequestStatusPending {
		return core.ChangeRequest{}, fmt.Errorf("cannot approve change request in status %s", req.Status)
	}
	if req.ExpiresAt != nil && !time.Now().UTC().Before(req.ExpiresAt.UTC()) {
		return core.ChangeRequest{}, errors.New("cannot approve expired change request")
	}

	req.Status = core.ChangeRequestStatusApproved
	req.ApproverID = &approverID

	tag, err := r.pool.Exec(ctx, `
		UPDATE change_requests
		SET status = $1, approver_id = $2
		WHERE id = $3
		  AND status = 'pending'
		  AND requester_id <> $2
		  AND (expires_at IS NULL OR expires_at > now())
	`, req.Status, approverID, req.ID)
	if err != nil {
		return core.ChangeRequest{}, fmt.Errorf("approving change request: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return core.ChangeRequest{}, errors.New("change request is no longer pending, expired, or ineligible for approval")
	}

	return req, nil
}

func (r *Repository) RejectChangeRequest(ctx context.Context, id uuid.UUID, approverID string) (core.ChangeRequest, error) {
	approverID = strings.TrimSpace(approverID)
	if approverID == "" {
		return core.ChangeRequest{}, errors.New("approver_id is required")
	}

	req, err := r.GetChangeRequest(ctx, id)
	if err != nil {
		return core.ChangeRequest{}, err
	}
	if req.Status != core.ChangeRequestStatusPending {
		return core.ChangeRequest{}, fmt.Errorf("cannot reject change request in status %s", req.Status)
	}

	req.Status = core.ChangeRequestStatusRejected
	req.ApproverID = &approverID

	tag, err := r.pool.Exec(ctx, `
		UPDATE change_requests
		SET status = $1, approver_id = $2
		WHERE id = $3 AND status = 'pending'
	`, req.Status, approverID, req.ID)
	if err != nil {
		return core.ChangeRequest{}, fmt.Errorf("rejecting change request: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return core.ChangeRequest{}, errors.New("change request is no longer pending")
	}

	return req, nil
}

func (r *Repository) CreateMaintenanceWindow(ctx context.Context, mw core.MaintenanceWindow) (core.MaintenanceWindow, error) {
	if mw.ID == uuid.Nil {
		mw.ID = uuid.New()
	}
	if mw.CreatedAt.IsZero() {
		mw.CreatedAt = time.Now().UTC()
	}
	mw.CreatedAt = mw.CreatedAt.UTC().Truncate(time.Microsecond)
	mw.StartsAt = mw.StartsAt.UTC().Truncate(time.Microsecond)
	mw.EndsAt = mw.EndsAt.UTC().Truncate(time.Microsecond)

	err := r.pool.QueryRow(ctx, `
		INSERT INTO maintenance_windows (id, name, description, starts_at, ends_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, name, description, starts_at, ends_at, created_at
	`, mw.ID, mw.Name, mw.Description, mw.StartsAt, mw.EndsAt, mw.CreatedAt).Scan(
		&mw.ID, &mw.Name, &mw.Description, &mw.StartsAt, &mw.EndsAt, &mw.CreatedAt,
	)
	if err != nil {
		return core.MaintenanceWindow{}, fmt.Errorf("creating maintenance window: %w", err)
	}
	return mw, nil
}

func (r *Repository) GetMaintenanceWindow(ctx context.Context, id uuid.UUID) (core.MaintenanceWindow, error) {
	var mw core.MaintenanceWindow
	err := r.pool.QueryRow(ctx, `
		SELECT id, name, description, starts_at, ends_at, created_at
		FROM maintenance_windows WHERE id = $1
	`, id).Scan(&mw.ID, &mw.Name, &mw.Description, &mw.StartsAt, &mw.EndsAt, &mw.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return core.MaintenanceWindow{}, core.ErrNotFound
		}
		return core.MaintenanceWindow{}, fmt.Errorf("getting maintenance window: %w", err)
	}
	return mw, nil
}

func (r *Repository) ListMaintenanceWindows(ctx context.Context) ([]core.MaintenanceWindow, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, name, description, starts_at, ends_at, created_at
		FROM maintenance_windows
		ORDER BY starts_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("listing maintenance windows: %w", err)
	}
	defer rows.Close()

	var windows []core.MaintenanceWindow
	for rows.Next() {
		var mw core.MaintenanceWindow
		if err := rows.Scan(&mw.ID, &mw.Name, &mw.Description, &mw.StartsAt, &mw.EndsAt, &mw.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning maintenance window: %w", err)
		}
		windows = append(windows, mw)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating maintenance windows: %w", err)
	}
	return windows, nil
}

func (r *Repository) IsInMaintenanceWindow(ctx context.Context, at time.Time) (bool, error) {
	var active bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM maintenance_windows
			WHERE $1 >= starts_at AND $1 <= ends_at
		)
	`, at.UTC()).Scan(&active)
	if err != nil {
		return false, fmt.Errorf("checking maintenance window: %w", err)
	}
	return active, nil
}

// ---------------------------------------------------------------------
// Compliance Evidence Aggregation (P8-S4)
// ---------------------------------------------------------------------

// ComplianceMetrics captures the live signals used to evaluate compliance controls.
type ComplianceMetrics struct {
	AuditVerification     core.AuditVerificationResult
	TotalOperators        int64
	ActiveOperators       int64
	MFAEnabledOperators   int64
	TotalInstances        int
	HealthyInstances      int
	DegradedInstances     int
	UnreachableInstances  int
	EvictedInstances      int
	PendingInstances      int
	RegisteringInstances  int
	DeregisteredInstances int
}

// EvaluateComplianceEvidence computes honest control statuses from live signals.
func EvaluateComplianceEvidence(metrics ComplianceMetrics, now time.Time) core.ComplianceEvidenceReport {
	auditStatus := core.ComplianceStatusCompliant
	if !metrics.AuditVerification.Verified {
		auditStatus = core.ComplianceStatusNonCompliant
	} else if metrics.AuditVerification.ChainLength == 0 {
		auditStatus = core.ComplianceStatusNeedsReview
	}

	iamStatus := core.ComplianceStatusCompliant
	switch {
	case metrics.ActiveOperators == 0:
		iamStatus = core.ComplianceStatusNonCompliant
	case metrics.MFAEnabledOperators < metrics.ActiveOperators:
		iamStatus = core.ComplianceStatusNeedsReview
	}

	opsStatus := core.ComplianceStatusCompliant
	activeInstances := metrics.TotalInstances - metrics.DeregisteredInstances
	switch {
	case activeInstances <= 0:
		opsStatus = core.ComplianceStatusNeedsReview
	case metrics.UnreachableInstances > 0 || metrics.EvictedInstances > 0:
		opsStatus = core.ComplianceStatusNonCompliant
	case metrics.DegradedInstances > 0 || metrics.PendingInstances > 0 || metrics.RegisteringInstances > 0 || metrics.HealthyInstances == 0:
		opsStatus = core.ComplianceStatusNeedsReview
	}

	items := []core.ComplianceEvidenceItem{
		{
			ID:              "CTRL-AUD-001",
			Framework:       "SOC2",
			ControlID:       "CC6.1",
			ControlName:     "Cryptographic Audit Trail Integrity",
			Status:          auditStatus,
			EvidenceType:    "hash_chain_verification",
			Description:     "All security events and administrative actions are hashed and chained using SHA-256.",
			Engine:          "argus",
			LastEvaluatedAt: now,
			Evaluator:       "argus-compliance-engine",
			ArtifactsCount:  metrics.AuditVerification.ChainLength,
			Details: map[string]interface{}{
				"chain_length":   metrics.AuditVerification.ChainLength,
				"chain_verified": metrics.AuditVerification.Verified,
				"head_hash":      metrics.AuditVerification.HeadHash,
				"algorithm":      "SHA-256",
			},
		},
		{
			ID:              "CTRL-IAM-001",
			Framework:       "SOC2",
			ControlID:       "CC6.2",
			ControlName:     "Administrative Operator Access Review",
			Status:          iamStatus,
			EvidenceType:    "operator_inventory",
			Description:     "Operator accounts and privileged credentials enrolled in control plane with MFA verification.",
			Engine:          "argus",
			LastEvaluatedAt: now,
			Evaluator:       "argus-compliance-engine",
			ArtifactsCount:  int(metrics.ActiveOperators),
			Details: map[string]interface{}{
				"total_operators":       metrics.TotalOperators,
				"active_operators":      metrics.ActiveOperators,
				"mfa_enabled_operators": metrics.MFAEnabledOperators,
			},
		},
		{
			ID:              "CTRL-OPS-001",
			Framework:       "SOC2",
			ControlID:       "CC7.1",
			ControlName:     "Fleet Infrastructure Health & Inventory",
			Status:          opsStatus,
			EvidenceType:    "instance_inventory",
			Description:     "Continuous heartbeat monitoring and registry for all fleet instances.",
			Engine:          "argus",
			LastEvaluatedAt: now,
			Evaluator:       "argus-compliance-engine",
			ArtifactsCount:  metrics.TotalInstances,
			Details: map[string]interface{}{
				"registered_instances":   metrics.TotalInstances,
				"healthy_instances":      metrics.HealthyInstances,
				"degraded_instances":     metrics.DegradedInstances,
				"unreachable_instances":  metrics.UnreachableInstances,
				"evicted_instances":      metrics.EvictedInstances,
				"pending_instances":      metrics.PendingInstances,
				"registering_instances":  metrics.RegisteringInstances,
				"deregistered_instances": metrics.DeregisteredInstances,
			},
		},
	}

	compliantCount := 0
	for _, it := range items {
		if it.Status == core.ComplianceStatusCompliant {
			compliantCount++
		}
	}

	score := float64(0)
	if len(items) > 0 {
		score = (float64(compliantCount) / float64(len(items))) * 100.0
	}

	return core.ComplianceEvidenceReport{
		Data: items,
		Summary: core.ComplianceEvidenceSummary{
			TotalControls:     len(items),
			CompliantControls: compliantCount,
			ReviewRequired:    len(items) - compliantCount,
			ScorePercent:      score,
		},
	}
}

func (r *Repository) GetComplianceEvidence(ctx context.Context) (core.ComplianceEvidenceReport, error) {
	now := time.Now().UTC()

	// 1. Audit chain verification
	auditVerification, err := r.VerifyAuditChainDetailed(ctx)
	if err != nil {
		return core.ComplianceEvidenceReport{}, fmt.Errorf("verifying audit chain: %w", err)
	}

	// 2. Real operator and MFA counts
	var metrics ComplianceMetrics
	metrics.AuditVerification = auditVerification
	if err := r.pool.QueryRow(ctx, `
		SELECT
			COUNT(*) AS total_operators,
			COUNT(*) FILTER (WHERE o.is_active = TRUE) AS active_operators,
			COUNT(*) FILTER (WHERE o.is_active = TRUE AND COALESCE(c.totp_enabled, FALSE) = TRUE) AS mfa_enabled_operators
		FROM operators o
		LEFT JOIN operator_credentials c ON c.operator_id = o.id
	`).Scan(&metrics.TotalOperators, &metrics.ActiveOperators, &metrics.MFAEnabledOperators); err != nil {
		return core.ComplianceEvidenceReport{}, fmt.Errorf("counting operators for compliance: %w", err)
	}

	// 3. Real fleet instance health breakdown across all registered instances
	rows, err := r.pool.Query(ctx, `SELECT status, COUNT(*) FROM instances GROUP BY status`)
	if err != nil {
		return core.ComplianceEvidenceReport{}, fmt.Errorf("counting instances by status for compliance: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var statusStr string
		var cnt int
		if err := rows.Scan(&statusStr, &cnt); err != nil {
			return core.ComplianceEvidenceReport{}, fmt.Errorf("scanning instance compliance status count: %w", err)
		}
		metrics.TotalInstances += cnt
		switch core.InstanceStatus(statusStr) {
		case core.StatusHealthy:
			metrics.HealthyInstances += cnt
		case core.StatusDegraded:
			metrics.DegradedInstances += cnt
		case core.StatusUnreachable:
			metrics.UnreachableInstances += cnt
		case core.StatusEvicted:
			metrics.EvictedInstances += cnt
		case core.StatusPending:
			metrics.PendingInstances += cnt
		case core.StatusRegistering:
			metrics.RegisteringInstances += cnt
		case core.StatusDeregistered:
			metrics.DeregisteredInstances += cnt
		}
	}
	if err := rows.Err(); err != nil {
		return core.ComplianceEvidenceReport{}, fmt.Errorf("iterating instance compliance status counts: %w", err)
	}

	return EvaluateComplianceEvidence(metrics, now), nil
}
