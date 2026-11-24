package postgres_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/autorix/argus/internal/core"
	"github.com/autorix/argus/internal/storage/postgres"
	"github.com/google/uuid"
)

func TestAuditLogAndHashChaining(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// 1. Record first audit record
	r1, err := repo.RecordAudit(ctx, core.AuditRecord{
		ActorID:      "op-1",
		ActorType:    "operator",
		Action:       "policy.create",
		ResourceType: "policy",
		ResourceID:   "pol-123",
		Environment:  "production",
		Outcome:      "success",
		BeforeState:  map[string]interface{}{"exists": false},
		AfterState:   map[string]interface{}{"name": "allow_admin"},
	})
	if err != nil {
		t.Fatalf("RecordAudit 1: %v", err)
	}
	if r1.PrevHash != "" {
		t.Fatalf("expected empty prev_hash for genesis record, got %s", r1.PrevHash)
	}
	if r1.RecordHash == "" {
		t.Fatal("expected non-empty record_hash")
	}

	// 2. Record second audit record
	r2, err := repo.RecordAudit(ctx, core.AuditRecord{
		ActorID:      "op-2",
		ActorType:    "operator",
		Action:       "policy.update",
		ResourceType: "policy",
		ResourceID:   "pol-123",
		Environment:  "production",
		Outcome:      "success",
	})
	if err != nil {
		t.Fatalf("RecordAudit 2: %v", err)
	}
	if r2.PrevHash != r1.RecordHash {
		t.Fatalf("expected r2.PrevHash == r1.RecordHash (%s), got %s", r1.RecordHash, r2.PrevHash)
	}

	// 3. Verify valid chain
	res, err := repo.VerifyAuditChainDetailed(ctx)
	if err != nil {
		t.Fatalf("VerifyAuditChainDetailed: %v", err)
	}
	if !res.Verified {
		t.Fatalf("expected audit chain to verify, got broken: %+v", res.BrokenLink)
	}
	if res.ChainLength != 2 {
		t.Fatalf("expected chain length 2, got %d", res.ChainLength)
	}
	if res.HeadHash != r2.RecordHash {
		t.Fatalf("expected head hash %s, got %s", r2.RecordHash, res.HeadHash)
	}

	// 4. Test List & Filter
	records, _, _, err := repo.ListAuditRecords(ctx, core.AuditRecordFilter{
		Action: "policy.create",
	})
	if err != nil {
		t.Fatalf("ListAuditRecords: %v", err)
	}
	if len(records) != 1 || records[0].ID != r1.ID {
		t.Fatalf("expected record r1, got %v", records)
	}

	// 5. Test Export
	jsonExport, err := repo.ExportAuditRecords(ctx, "json")
	if err != nil {
		t.Fatalf("ExportAuditRecords JSON: %v", err)
	}
	if len(jsonExport) == 0 {
		t.Fatal("expected non-empty JSON export")
	}

	csvExport, err := repo.ExportAuditRecords(ctx, "csv")
	if err != nil {
		t.Fatalf("ExportAuditRecords CSV: %v", err)
	}
	if len(csvExport) == 0 {
		t.Fatal("expected non-empty CSV export")
	}
}

func TestConfigRevisionsAndRollback(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// 1. Record Revision 1
	rev1, err := repo.RecordConfigRevision(ctx, core.ConfigRevision{
		Engine:       "vulcan",
		ResourceType: "key_template",
		ResourceID:   "tmpl-1",
		Author:       "admin@autorix.io",
		Note:         "initial template",
		Config:       map[string]interface{}{"rate_limit": 1000},
	})
	if err != nil {
		t.Fatalf("RecordConfigRevision 1: %v", err)
	}
	if rev1.RevisionNum != 1 {
		t.Fatalf("expected revision_num 1, got %d", rev1.RevisionNum)
	}

	// 2. Record Revision 2
	rev2, err := repo.RecordConfigRevision(ctx, core.ConfigRevision{
		Engine:       "vulcan",
		ResourceType: "key_template",
		ResourceID:   "tmpl-1",
		Author:       "admin@autorix.io",
		Note:         "increased rate limit",
		Config:       map[string]interface{}{"rate_limit": 5000},
	})
	if err != nil {
		t.Fatalf("RecordConfigRevision 2: %v", err)
	}
	if rev2.RevisionNum != 2 {
		t.Fatalf("expected revision_num 2, got %d", rev2.RevisionNum)
	}

	// 3. List Revisions
	revisions, err := repo.ListConfigRevisions(ctx, "vulcan", "key_template", "tmpl-1")
	if err != nil {
		t.Fatalf("ListConfigRevisions: %v", err)
	}
	if len(revisions) != 2 {
		t.Fatalf("expected 2 revisions, got %d", len(revisions))
	}
	if revisions[0].RevisionNum != 2 {
		t.Fatalf("expected newest revision first, got %d", revisions[0].RevisionNum)
	}

	// 4. Rollback to Revision 1
	rollbackRev, err := repo.RollbackConfig(ctx, rev1.ID, "auditor@autorix.io", "reverting spike")
	if err != nil {
		t.Fatalf("RollbackConfig: %v", err)
	}
	if rollbackRev.RevisionNum != 3 {
		t.Fatalf("expected revision_num 3 for rollback, got %d", rollbackRev.RevisionNum)
	}
	if rollbackRev.Config["rate_limit"] != float64(1000) && rollbackRev.Config["rate_limit"] != 1000 {
		t.Fatalf("expected rolled back config to have rate_limit 1000, got %v", rollbackRev.Config["rate_limit"])
	}
}

func TestOrganisationsAndProjects(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// 1. Create Organisation
	org, err := repo.CreateOrganisation(ctx, core.Organisation{
		Name: "Acme Corp",
		Slug: "acme-corp",
	})
	if err != nil {
		t.Fatalf("CreateOrganisation: %v", err)
	}

	gotOrg, err := repo.GetOrganisationBySlug(ctx, "acme-corp")
	if err != nil {
		t.Fatalf("GetOrganisationBySlug: %v", err)
	}
	if gotOrg.ID != org.ID {
		t.Fatalf("expected org ID %s, got %s", org.ID, gotOrg.ID)
	}

	// 2. Create Project
	proj, err := repo.CreateProject(ctx, core.Project{
		OrgID: org.ID,
		Name:  "Payments Platform",
		Slug:  "payments",
	})
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	gotProj, err := repo.GetProjectBySlug(ctx, org.ID, "payments")
	if err != nil {
		t.Fatalf("GetProjectBySlug: %v", err)
	}
	if gotProj.ID != proj.ID {
		t.Fatalf("expected project ID %s, got %s", proj.ID, gotProj.ID)
	}

	// 3. List Projects
	projects, err := repo.ListProjects(ctx, &org.ID)
	if err != nil {
		t.Fatalf("ListProjects: %v", err)
	}
	if len(projects) != 1 {
		t.Fatalf("expected 1 project, got %d", len(projects))
	}
}

func TestChangeRequestsAndMaintenanceWindows(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// 1. Create Change Request
	cr, err := repo.CreateChangeRequest(ctx, core.ChangeRequest{
		RequesterID:    "op-dev",
		Action:         "schema.migrate",
		TargetResource: "db-cluster-1",
		Payload:        map[string]interface{}{"version": "v2.0"},
	})
	if err != nil {
		t.Fatalf("CreateChangeRequest: %v", err)
	}
	if cr.Status != core.ChangeRequestStatusPending {
		t.Fatalf("expected pending status, got %s", cr.Status)
	}

	// 2. Self-approval must be rejected
	if _, err := repo.ApproveChangeRequest(ctx, cr.ID, "op-dev"); err == nil {
		t.Fatal("expected self-approval of change request to be rejected")
	}

	// 2b. Expired change request approval must be rejected
	past := time.Now().UTC().Add(-1 * time.Minute)
	expiredCR, err := repo.CreateChangeRequest(ctx, core.ChangeRequest{
		RequesterID:    "op-dev",
		Action:         "schema.migrate",
		TargetResource: "db-cluster-2",
		ExpiresAt:      &past,
	})
	if err != nil {
		t.Fatalf("CreateChangeRequest expired: %v", err)
	}
	if _, err := repo.ApproveChangeRequest(ctx, expiredCR.ID, "op-lead"); err == nil {
		t.Fatal("expected expired change request approval to be rejected")
	}

	// 2c. Approve Change Request by another operator
	approved, err := repo.ApproveChangeRequest(ctx, cr.ID, "op-lead")
	if err != nil {
		t.Fatalf("ApproveChangeRequest: %v", err)
	}
	if approved.Status != core.ChangeRequestStatusApproved || *approved.ApproverID != "op-lead" {
		t.Fatalf("unexpected approved state: %+v", approved)
	}

	// Double-approve must fail
	if _, err := repo.ApproveChangeRequest(ctx, cr.ID, "op-other"); err == nil {
		t.Fatal("expected approving an already approved change request to fail")
	}

	// 3. Create Maintenance Window
	now := time.Now().UTC()
	_, err = repo.CreateMaintenanceWindow(ctx, core.MaintenanceWindow{
		Name:        "Weekly Maintenance",
		Description: "Routine cluster reboot",
		StartsAt:    now.Add(-1 * time.Hour),
		EndsAt:      now.Add(1 * time.Hour),
	})
	if err != nil {
		t.Fatalf("CreateMaintenanceWindow: %v", err)
	}

	// 4. Check IsInMaintenanceWindow
	inWindow, err := repo.IsInMaintenanceWindow(ctx, now)
	if err != nil {
		t.Fatalf("IsInMaintenanceWindow: %v", err)
	}
	if !inWindow {
		t.Fatal("expected time to be within maintenance window")
	}

	inFutureWindow, err := repo.IsInMaintenanceWindow(ctx, now.Add(5*time.Hour))
	if err != nil {
		t.Fatalf("IsInMaintenanceWindow future: %v", err)
	}
	if inFutureWindow {
		t.Fatal("expected future time to NOT be within maintenance window")
	}

	// 5. Compliance Evidence Report
	report, err := repo.GetComplianceEvidence(ctx)
	if err != nil {
		t.Fatalf("GetComplianceEvidence: %v", err)
	}
	if len(report.Data) == 0 {
		t.Fatal("expected compliance report items")
	}
	if report.Summary.TotalControls == 0 {
		t.Fatal("expected total controls > 0")
	}
}

func TestSanitizeCSVCell_NeutralizesFormulaInjection(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"=CMD|' /C calc'!A0", "'=CMD|' /C calc'!A0"},
		{"+1+1", "'+1+1"},
		{"-2+3", "'-2+3"},
		{"@SUM(1+1)", "'@SUM(1+1)"},
		{"\t=1+1", "'\t=1+1"},
		{"\r=1+1", "'\r=1+1"},
		{"  =HYPERLINK(\"http://evil\")", "'  =HYPERLINK(\"http://evil\")"},
		{"normal-actor@autorix.io", "normal-actor@autorix.io"},
		{"", ""},
	}
	for _, tc := range cases {
		if got := postgres.SanitizeCSVCell(tc.in); got != tc.want {
			t.Errorf("SanitizeCSVCell(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestFormatAuditRecordsCSV_SanitizesFormulas(t *testing.T) {
	rec := core.AuditRecord{
		ID:           uuid.MustParse("22222222-2222-2222-2222-222222222222"),
		ActorID:      "=HYPERLINK(\"http://attacker\")",
		ActorType:    "operator",
		Action:       "+policy.update",
		ResourceType: "policy",
		ResourceID:   "-pol-1",
		Environment:  "@prod",
		Outcome:      "success",
		CreatedAt:    time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC),
	}
	raw, err := postgres.FormatAuditRecordsCSV([]core.AuditRecord{rec})
	if err != nil {
		t.Fatalf("FormatAuditRecordsCSV: %v", err)
	}
	out := string(raw)
	for _, want := range []string{"'=HYPERLINK", "'+policy.update", "'-pol-1", "'@prod"} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected sanitized token %q in CSV output:\n%s", want, out)
		}
	}
}

func TestEvaluateComplianceEvidence_HonestScoring(t *testing.T) {
	now := time.Now().UTC()

	// All failing / empty state must NOT report compliant.
	emptyReport := postgres.EvaluateComplianceEvidence(postgres.ComplianceMetrics{
		AuditVerification: core.AuditVerificationResult{Verified: true, ChainLength: 0},
	}, now)
	if emptyReport.Summary.CompliantControls != 0 {
		t.Fatalf("expected 0 compliant controls on empty system, got %d (%+v)", emptyReport.Summary.CompliantControls, emptyReport.Data)
	}
	if emptyReport.Data[0].Status != core.ComplianceStatusNeedsReview {
		t.Fatalf("expected CTRL-AUD-001 needs_review with empty chain, got %s", emptyReport.Data[0].Status)
	}
	if emptyReport.Data[1].Status != core.ComplianceStatusNonCompliant {
		t.Fatalf("expected CTRL-IAM-001 non_compliant with 0 active operators, got %s", emptyReport.Data[1].Status)
	}
	if emptyReport.Data[2].Status != core.ComplianceStatusNeedsReview {
		t.Fatalf("expected CTRL-OPS-001 needs_review with 0 instances, got %s", emptyReport.Data[2].Status)
	}

	// Broken audit chain, operators lacking MFA, and unreachable/evicted instances.
	unhealthyReport := postgres.EvaluateComplianceEvidence(postgres.ComplianceMetrics{
		AuditVerification:    core.AuditVerificationResult{Verified: false, ChainLength: 5},
		TotalOperators:       2,
		ActiveOperators:      2,
		MFAEnabledOperators:  1,
		TotalInstances:       3,
		HealthyInstances:     2,
		UnreachableInstances: 1,
	}, now)
	if unhealthyReport.Data[0].Status != core.ComplianceStatusNonCompliant {
		t.Fatalf("expected CTRL-AUD-001 non_compliant for broken chain, got %s", unhealthyReport.Data[0].Status)
	}
	if unhealthyReport.Data[1].Status != core.ComplianceStatusNeedsReview {
		t.Fatalf("expected CTRL-IAM-001 needs_review when MFA < active operators, got %s", unhealthyReport.Data[1].Status)
	}
	if unhealthyReport.Data[2].Status != core.ComplianceStatusNonCompliant {
		t.Fatalf("expected CTRL-OPS-001 non_compliant when unreachable > 0, got %s", unhealthyReport.Data[2].Status)
	}

	// Fully healthy system with verified audit records, 100% MFA, and healthy instances.
	healthyReport := postgres.EvaluateComplianceEvidence(postgres.ComplianceMetrics{
		AuditVerification:   core.AuditVerificationResult{Verified: true, ChainLength: 10, HeadHash: "abc"},
		TotalOperators:      2,
		ActiveOperators:     2,
		MFAEnabledOperators: 2,
		TotalInstances:      3,
		HealthyInstances:    3,
	}, now)
	if healthyReport.Summary.CompliantControls != 3 || healthyReport.Summary.ScorePercent != 100.0 {
		t.Fatalf("expected 3/3 compliant (100%%), got %+v", healthyReport.Summary)
	}
}
