package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/autorix/argus/internal/core"
	"github.com/autorix/argus/internal/credential"
	transporthttp "github.com/autorix/argus/internal/transport/http"
	"github.com/autorix/platform/httpx"
	"github.com/google/uuid"
)

type fakeArgusRepo struct {
	core.Repository
	envs                  []core.Environment
	instances             []core.Instance
	events                []core.InstanceEvent
	orgs                  []core.Organisation
	projects              []core.Project
	complianceReport      core.ComplianceEvidenceReport
	operator              *core.OperatorWithCredential
	failedLoginCalls      int
	evictedInstanceID     uuid.UUID
	revokedCredentialID   uuid.UUID
	lastEventFilterInstID *uuid.UUID
	lastEventLimit        int
}

func (f *fakeArgusRepo) ListEnvironments(ctx context.Context) ([]core.Environment, error) {
	return f.envs, nil
}

func (f *fakeArgusRepo) GetEnvironmentBySlug(ctx context.Context, slug string) (core.Environment, error) {
	for _, e := range f.envs {
		if e.Slug == slug {
			return e, nil
		}
	}
	return core.Environment{}, core.ErrNotFound
}

func (f *fakeArgusRepo) GetEnvironmentByID(ctx context.Context, id uuid.UUID) (core.Environment, error) {
	for _, e := range f.envs {
		if e.ID == id {
			return e, nil
		}
	}
	return core.Environment{}, core.ErrNotFound
}

func (f *fakeArgusRepo) CreateEnvironment(ctx context.Context, name, slug, description string, isProduction bool) (core.Environment, error) {
	env := core.Environment{
		ID:           uuid.New(),
		Name:         name,
		Slug:         slug,
		Description:  description,
		IsProduction: isProduction,
		CreatedAt:    time.Now().UTC(),
	}
	f.envs = append(f.envs, env)
	return env, nil
}

func (f *fakeArgusRepo) ListInstances(ctx context.Context, filter core.InstanceFilter) ([]core.Instance, string, bool, error) {
	var out []core.Instance
	for _, inst := range f.instances {
		if filter.EnvironmentID != uuid.Nil && inst.EnvironmentID != filter.EnvironmentID {
			continue
		}
		if filter.EngineType != "" && inst.EngineType != filter.EngineType {
			continue
		}
		if filter.Status != "" && inst.Status != filter.Status {
			continue
		}
		out = append(out, inst)
	}
	return out, "", false, nil
}

func (f *fakeArgusRepo) ListEvents(ctx context.Context, instanceID *uuid.UUID, limit int) ([]core.InstanceEvent, error) {
	f.lastEventFilterInstID = instanceID
	f.lastEventLimit = limit
	var out []core.InstanceEvent
	for _, ev := range f.events {
		if instanceID != nil && ev.InstanceID != *instanceID {
			continue
		}
		out = append(out, ev)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (f *fakeArgusRepo) SetInstanceStatusWithEvent(ctx context.Context, id uuid.UUID, status core.InstanceStatus, eventType core.InstanceEventType, detail map[string]interface{}) error {
	for i := range f.instances {
		if f.instances[i].ID == id {
			f.instances[i].Status = status
			f.evictedInstanceID = id
			return nil
		}
	}
	return core.ErrNotFound
}

func (f *fakeArgusRepo) RevokeInstanceCredential(ctx context.Context, instanceID uuid.UUID) error {
	f.revokedCredentialID = instanceID
	return nil
}

func (f *fakeArgusRepo) ListOrganisations(ctx context.Context) ([]core.Organisation, error) {
	return f.orgs, nil
}

func (f *fakeArgusRepo) CreateOrganisation(ctx context.Context, org core.Organisation) (core.Organisation, error) {
	if org.ID == uuid.Nil {
		org.ID = uuid.New()
	}
	org.CreatedAt = time.Now().UTC()
	f.orgs = append(f.orgs, org)
	return org, nil
}

func (f *fakeArgusRepo) ListProjects(ctx context.Context, orgID *uuid.UUID) ([]core.Project, error) {
	var out []core.Project
	for _, p := range f.projects {
		if orgID != nil && p.OrgID != *orgID {
			continue
		}
		out = append(out, p)
	}
	return out, nil
}

func (f *fakeArgusRepo) CreateProject(ctx context.Context, project core.Project) (core.Project, error) {
	if project.ID == uuid.Nil {
		project.ID = uuid.New()
	}
	project.CreatedAt = time.Now().UTC()
	f.projects = append(f.projects, project)
	return project, nil
}

func (f *fakeArgusRepo) GetComplianceEvidence(ctx context.Context) (core.ComplianceEvidenceReport, error) {
	return f.complianceReport, nil
}

func (f *fakeArgusRepo) GetOperatorByEmail(ctx context.Context, email string) (core.OperatorWithCredential, error) {
	if f.operator != nil && f.operator.Email == email {
		return *f.operator, nil
	}
	return core.OperatorWithCredential{}, core.ErrNotFound
}

func (f *fakeArgusRepo) RecordOperatorFailedLogin(ctx context.Context, id uuid.UUID, maxAttempts int, lockDuration time.Duration) (int, *time.Time, error) {
	f.failedLoginCalls++
	if f.operator != nil && f.operator.ID == id {
		f.operator.FailedAttempts++
		return f.operator.FailedAttempts, f.operator.LockedUntil, nil
	}
	return 0, nil, core.ErrNotFound
}

func TestHTTP_ListEngines_RollsUpByEnvironmentAndComputesCapabilities(t *testing.T) {
	prodEnv := core.Environment{ID: uuid.New(), Name: "Production", Slug: "prod", IsProduction: true}
	stgEnv := core.Environment{ID: uuid.New(), Name: "Staging", Slug: "staging"}

	repo := &fakeArgusRepo{
		envs: []core.Environment{prodEnv, stgEnv},
		instances: []core.Instance{
			{
				ID:            uuid.New(),
				EngineType:    "ego",
				InstanceID:    "ego-1",
				EnvironmentID: prodEnv.ID,
				Version:       "1.2.0",
				Capabilities:  []string{"argon2id", "sessions"},
				Status:        core.StatusHealthy,
			},
			{
				ID:            uuid.New(),
				EngineType:    "ego",
				InstanceID:    "ego-2",
				EnvironmentID: prodEnv.ID,
				Version:       "1.2.0",
				Capabilities:  []string{"webauthn", "sessions"},
				Status:        core.StatusDegraded,
			},
			{
				ID:            uuid.New(),
				EngineType:    "janus",
				InstanceID:    "janus-1",
				EnvironmentID: stgEnv.ID,
				Version:       "1.0.0",
				Capabilities:  []string{"oidc"},
				Status:        core.StatusHealthy,
			},
		},
	}

	srv := transporthttp.NewServer(newTestHealthHandler(false), repo)

	// Filter to prod environment
	req := httptest.NewRequest(http.MethodGet, "/v1/engines?environment=prod", nil)
	rec := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var engines []struct {
		EngineType    string   `json:"engine_type"`
		Environment   string   `json:"environment"`
		Status        string   `json:"status"`
		InstanceCount int      `json:"instance_count"`
		Version       string   `json:"version"`
		Capabilities  []string `json:"capabilities"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&engines); err != nil {
		t.Fatalf("decode engines: %v", err)
	}
	if len(engines) != 1 {
		t.Fatalf("expected 1 rolled-up engine in prod, got %d (%+v)", len(engines), engines)
	}
	if engines[0].EngineType != "ego" || engines[0].Environment != "prod" {
		t.Fatalf("unexpected engine metadata: %+v", engines[0])
	}
	if engines[0].Status != string(core.StatusDegraded) {
		t.Fatalf("expected worst-status rollup 'degraded', got %q", engines[0].Status)
	}
	if engines[0].InstanceCount != 2 || engines[0].Version != "1.2.0" {
		t.Fatalf("unexpected count/version: %+v", engines[0])
	}
	if len(engines[0].Capabilities) != 3 {
		t.Fatalf("expected 3 union capabilities, got %v", engines[0].Capabilities)
	}

	// Unknown environment returns 400
	badReq := httptest.NewRequest(http.MethodGet, "/v1/engines?environment=nonexistent", nil)
	badRec := httptest.NewRecorder()
	srv.Routes().ServeHTTP(badRec, badReq)
	if badRec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unknown environment, got %d", badRec.Code)
	}
}

func TestHTTP_ListEvents_FiltersByInstanceAndLimit(t *testing.T) {
	instA := uuid.New()
	instB := uuid.New()
	now := time.Now().UTC()

	repo := &fakeArgusRepo{
		events: []core.InstanceEvent{
			{ID: 1, InstanceID: instA, Type: core.EventRegistered, OccurredAt: now},
			{ID: 2, InstanceID: instB, Type: core.EventDegraded, OccurredAt: now},
		},
	}
	srv := transporthttp.NewServer(newTestHealthHandler(false), repo)

	req := httptest.NewRequest(http.MethodGet, "/v1/events?instance_id="+instA.String()+"&limit=5", nil)
	rec := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var payload struct {
		Data []struct {
			ID         int64  `json:"id"`
			InstanceID string `json:"instance_id"`
			Type       string `json:"type"`
		} `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode events: %v", err)
	}
	if len(payload.Data) != 1 || payload.Data[0].InstanceID != instA.String() || payload.Data[0].Type != "registered" {
		t.Fatalf("unexpected events payload: %+v", payload)
	}
	if repo.lastEventLimit != 5 {
		t.Fatalf("expected limit 5 passed to repo, got %d", repo.lastEventLimit)
	}

	// Malformed UUID rejected with 400
	badReq := httptest.NewRequest(http.MethodGet, "/v1/events?instance_id=not-a-uuid", nil)
	badRec := httptest.NewRecorder()
	srv.Routes().ServeHTTP(badRec, badReq)
	if badRec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for malformed instance_id, got %d", badRec.Code)
	}
}

func TestHTTP_DeleteInstance_ForceRemovesAndRevokesCredential(t *testing.T) {
	instID := uuid.New()
	repo := &fakeArgusRepo{
		instances: []core.Instance{
			{ID: instID, EngineType: "aegis", InstanceID: "aegis-1", Status: core.StatusHealthy},
		},
	}
	srv := transporthttp.NewServer(newTestHealthHandler(false), repo)

	req := httptest.NewRequest(http.MethodDelete, "/v1/instances/"+instID.String(), nil)
	rec := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204 No Content on DELETE /v1/instances/{id}, got %d: %s", rec.Code, rec.Body.String())
	}
	if repo.evictedInstanceID != instID || repo.revokedCredentialID != instID {
		t.Fatalf("expected instance eviction and credential revocation for %s, got evicted=%s revoked=%s", instID, repo.evictedInstanceID, repo.revokedCredentialID)
	}
}

func TestHTTP_AdminGovernanceEndpointsAndBodyLimit(t *testing.T) {
	repo := &fakeArgusRepo{
		complianceReport: core.ComplianceEvidenceReport{
			Data: []core.ComplianceEvidenceItem{
				{ID: "CTRL-AUD-001", Status: core.ComplianceStatusCompliant},
			},
			Summary: core.ComplianceEvidenceSummary{
				TotalControls:     1,
				CompliantControls: 1,
				ScorePercent:      100,
			},
		},
	}
	srv := transporthttp.NewServer(newTestHealthHandler(false), repo)
	h := srv.Routes()

	// 1. POST /admin/organisations
	req := httptest.NewRequest(http.MethodPost, "/admin/organisations", strings.NewReader(`{"name":"Acme Platform"}`))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /admin/organisations expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var createdOrg core.Organisation
	if err := json.NewDecoder(rec.Body).Decode(&createdOrg); err != nil {
		t.Fatalf("decode org: %v", err)
	}
	if createdOrg.Slug != "acme-platform" {
		t.Fatalf("expected derived slug acme-platform, got %q", createdOrg.Slug)
	}

	// 2. GET /admin/organisations
	req = httptest.NewRequest(http.MethodGet, "/admin/organisations", nil)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /admin/organisations expected 200, got %d", rec.Code)
	}

	// 3. POST /admin/projects
	projBody := `{"org_id":"` + createdOrg.ID.String() + `","name":"Core IAM","slug":"core-iam"}`
	req = httptest.NewRequest(http.MethodPost, "/admin/projects", strings.NewReader(projBody))
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /admin/projects expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	// 4. GET /admin/projects?org_id=...
	req = httptest.NewRequest(http.MethodGet, "/admin/projects?org_id="+createdOrg.ID.String(), nil)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /admin/projects expected 200, got %d", rec.Code)
	}
	var projects []core.Project
	if err := json.NewDecoder(rec.Body).Decode(&projects); err != nil {
		t.Fatalf("decode projects: %v", err)
	}
	if len(projects) != 1 || projects[0].Slug != "core-iam" {
		t.Fatalf("unexpected projects: %+v", projects)
	}

	// 5. POST /admin/environments
	req = httptest.NewRequest(http.MethodPost, "/admin/environments", strings.NewReader(`{"name":"Production EU","type":"production","description":"Primary EU"}`))
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /admin/environments expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var env core.Environment
	if err := json.NewDecoder(rec.Body).Decode(&env); err != nil {
		t.Fatalf("decode env: %v", err)
	}
	if env.Slug != "production-eu" || !env.IsProduction {
		t.Fatalf("unexpected created environment: %+v", env)
	}

	// 6. POST /admin/compliance/evidence
	req = httptest.NewRequest(http.MethodPost, "/admin/compliance/evidence", strings.NewReader(`{}`))
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("POST /admin/compliance/evidence expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var report core.ComplianceEvidenceReport
	if err := json.NewDecoder(rec.Body).Decode(&report); err != nil {
		t.Fatalf("decode compliance evidence: %v", err)
	}
	if report.Summary.TotalControls != 1 || report.Summary.CompliantControls != 1 {
		t.Fatalf("unexpected compliance report: %+v", report)
	}

	// 7. Oversized request body (> 1 MiB) is rejected by httpx.MaxBodyBytes.
	oversized := `{"name":"` + strings.Repeat("a", (1<<20)+512) + `"}`
	req = httptest.NewRequest(http.MethodPost, "/admin/organisations", strings.NewReader(oversized))
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest && rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 400 or 413 for oversized body, got %d", rec.Code)
	}
}

func TestHTTP_Login_UsesAtomicFailedLoginCounter(t *testing.T) {
	hash, err := credential.HashPassword("CorrectPassword123!")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	repo := &fakeArgusRepo{
		operator: &core.OperatorWithCredential{
			Operator: core.Operator{
				ID:       uuid.New(),
				Email:    "ops@autorix.io",
				Name:     "Ops",
				Role:     core.RoleAdmin,
				IsActive: true,
			},
			PasswordHash: hash,
		},
	}
	srv := transporthttp.NewServer(newTestHealthHandler(false), repo)

	req := httptest.NewRequest(http.MethodPost, "/v1/auth/login", strings.NewReader(`{"email":"ops@autorix.io","password":"WrongPassword!"}`))
	rec := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
	if repo.failedLoginCalls != 1 {
		t.Fatalf("expected atomic RecordOperatorFailedLogin to be called once, got %d", repo.failedLoginCalls)
	}
}

func TestHTTP_Stream_SerializesAPIInstanceThroughAccessLog(t *testing.T) {
	env := core.Environment{ID: uuid.New(), Name: "Prod", Slug: "prod"}
	repo := &fakeArgusRepo{
		envs: []core.Environment{env},
		instances: []core.Instance{
			{
				ID:            uuid.New(),
				EngineType:    "nexus",
				InstanceID:    "nexus-01",
				EnvironmentID: env.ID,
				Version:       "2.1.0",
				Status:        core.StatusHealthy,
			},
		},
	}

	var logBuf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuf, nil))
	srv := transporthttp.NewServer(newTestHealthHandler(false), repo)
	handler := httpx.AccessLog(logger)(srv.Routes())

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(40 * time.Millisecond)
		cancel()
	}()

	req := httptest.NewRequest(http.MethodGet, "/v1/stream", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 through AccessLog, got %d: %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "event: instances\ndata: ") {
		t.Fatalf("expected SSE instances frame, got: %s", body)
	}
	// Verify toAPIInstance snake_case fields and resolved environment slug are serialized.
	for _, want := range []string{`"engine_type":"nexus"`, `"instance_id":"nexus-01"`, `"environment":"prod"`, `"status":"healthy"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %s in SSE payload, got: %s", want, body)
		}
	}
	if strings.Contains(body, `"EngineType"`) {
		t.Fatalf("expected snake_case apiInstance JSON instead of raw Go struct fields, got: %s", body)
	}
}
