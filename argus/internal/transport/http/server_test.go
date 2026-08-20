package http_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/autorix/argus/internal/core"
	"github.com/autorix/argus/internal/credential"
	"github.com/autorix/argus/internal/storage/postgres"
	transporthttp "github.com/autorix/argus/internal/transport/http"
	"github.com/autorix/platform/health"
	"github.com/autorix/platform/pgtest"
	"github.com/google/uuid"
)

func newTestHealthHandler(failing bool) *health.Handler {
	c := health.NewChecker()
	c.Register("postgres", func(ctx context.Context) error {
		if failing {
			return errors.New("db down")
		}
		return nil
	})
	return health.NewHandler(c, func() health.Info { return health.Info{Engine: "argus"} })
}

func TestHTTP_HealthAlive_Returns200(t *testing.T) {
	server := transporthttp.NewServer(newTestHealthHandler(true), nil)
	req := httptest.NewRequest(http.MethodGet, "/health/alive", nil)
	rec := httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestHTTP_HealthReady_503WhenCheckFails(t *testing.T) {
	server := transporthttp.NewServer(newTestHealthHandler(true), nil)
	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	rec := httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
}

// newContractMux satisfies the shared platform/health.Contract suite
// (P1-S1-T10), adopted here the same way every other engine does.
func newContractMux(checkErr error) http.Handler {
	return transporthttp.NewServer(newTestHealthHandler(checkErr != nil), nil).Routes()
}

func TestHealthContract(t *testing.T) {
	health.Contract(t, newContractMux)
}

func newTestServerWithRepo(t *testing.T) (*transporthttp.Server, *postgres.Repository) {
	t.Helper()
	pool := pgtest.StartPostgres(t, "../../../migrations")
	repo := postgres.NewRepository(pool)
	return transporthttp.NewServer(newTestHealthHandler(false), repo), repo
}

func TestHTTP_ListInstances_FiltersAndPaginates(t *testing.T) {
	server, repo := newTestServerWithRepo(t)
	ctx := context.Background()

	env, err := repo.CreateEnvironment(ctx, "REST list", "rest-list-env", "", false)
	if err != nil {
		t.Fatalf("CreateEnvironment: %v", err)
	}
	for i := 0; i < 2; i++ {
		req := core.RegistrationRequest{EngineType: "ego", InstanceID: uuid.NewString(), EnvironmentID: env.ID}
		if _, err := repo.UpsertInstance(ctx, req, "admin", false); err != nil {
			t.Fatalf("UpsertInstance: %v", err)
		}
	}

	httpReq := httptest.NewRequest(http.MethodGet, "/v1/instances?filter.environment=rest-list-env&filter.engine_type=ego&limit=1", nil)
	rec := httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, httpReq)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var body struct {
		Data       []map[string]interface{} `json:"data"`
		NextCursor string                    `json:"next_cursor"`
		HasMore    bool                      `json:"has_more"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if len(body.Data) != 1 {
		t.Fatalf("expected 1 instance on the first page, got %d", len(body.Data))
	}
	if !body.HasMore || body.NextCursor == "" {
		t.Fatalf("expected has_more with a cursor, got %+v", body)
	}
}

func TestHTTP_ListInstances_UnknownEnvironmentIsBadRequest(t *testing.T) {
	server, _ := newTestServerWithRepo(t)

	httpReq := httptest.NewRequest(http.MethodGet, "/v1/instances?filter.environment=does-not-exist", nil)
	rec := httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, httpReq)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unknown environment, got %d", rec.Code)
	}
}

func TestHTTP_GetInstance_ReturnsDetail(t *testing.T) {
	server, repo := newTestServerWithRepo(t)
	ctx := context.Background()

	env, err := repo.CreateEnvironment(ctx, "REST get", "rest-get-env", "", false)
	if err != nil {
		t.Fatalf("CreateEnvironment: %v", err)
	}
	inst, err := repo.UpsertInstance(ctx, core.RegistrationRequest{
		EngineType: "ego", InstanceID: "rest-host-1", EnvironmentID: env.ID,
	}, "admin", false)
	if err != nil {
		t.Fatalf("UpsertInstance: %v", err)
	}

	httpReq := httptest.NewRequest(http.MethodGet, "/v1/instances/"+inst.ID.String(), nil)
	rec := httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, httpReq)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var body map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	got, ok := body["instance"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected an 'instance' object in the response, got %+v", body)
	}
	if got["id"] != inst.ID.String() {
		t.Fatalf("expected instance id %s, got %v", inst.ID, got["id"])
	}
	if got["environment"] != "rest-get-env" {
		t.Fatalf("expected environment slug rest-get-env, got %v", got["environment"])
	}
	if _, ok := body["events"]; !ok {
		t.Fatal("expected an 'events' key in the response")
	}
	if _, ok := body["dependencies"]; !ok {
		t.Fatal("expected a 'dependencies' key in the response")
	}
}

func TestHTTP_GetInstance_UnknownIDIs404(t *testing.T) {
	server, _ := newTestServerWithRepo(t)

	httpReq := httptest.NewRequest(http.MethodGet, "/v1/instances/"+uuid.NewString(), nil)
	rec := httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, httpReq)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown instance id, got %d", rec.Code)
	}
}

func TestHTTP_GetInstance_InvalidIDIsBadRequest(t *testing.T) {
	server, _ := newTestServerWithRepo(t)

	httpReq := httptest.NewRequest(http.MethodGet, "/v1/instances/not-a-uuid", nil)
	rec := httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, httpReq)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for a malformed instance id, got %d", rec.Code)
	}
}

func TestHTTP_GetTopology_ReturnsGraph(t *testing.T) {
	server, repo := newTestServerWithRepo(t)
	ctx := context.Background()

	env, err := repo.CreateEnvironment(ctx, "Topology Env", "topo-env", "", false)
	if err != nil {
		t.Fatalf("CreateEnvironment: %v", err)
	}

	inst, err := repo.UpsertInstance(ctx, core.RegistrationRequest{
		EngineType:    "aegis",
		InstanceID:    "aegis-1",
		EnvironmentID: env.ID,
	}, "admin", false)
	if err != nil {
		t.Fatalf("UpsertInstance: %v", err)
	}

	if err := repo.DeclareDependency(ctx, inst.ID, "nexus"); err != nil {
		t.Fatalf("DeclareDependency: %v", err)
	}

	httpReq := httptest.NewRequest(http.MethodGet, "/v1/topology?environment=topo-env", nil)
	rec := httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, httpReq)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var graph core.TopologyGraph
	if err := json.NewDecoder(rec.Body).Decode(&graph); err != nil {
		t.Fatalf("decoding topology response: %v", err)
	}
	if len(graph.Nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(graph.Nodes))
	}
	if len(graph.Edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(graph.Edges))
	}
	if graph.Edges[0].TargetEngineType != "nexus" {
		t.Fatalf("expected target nexus, got %s", graph.Edges[0].TargetEngineType)
	}
}

func TestHTTP_Stream_EmitsEvents(t *testing.T) {
	server, repo := newTestServerWithRepo(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	env, err := repo.CreateEnvironment(ctx, "Stream Env", "stream-env", "", false)
	if err != nil {
		t.Fatalf("CreateEnvironment: %v", err)
	}

	_, err = repo.UpsertInstance(ctx, core.RegistrationRequest{
		EngineType:    "ego",
		InstanceID:    "ego-stream-1",
		EnvironmentID: env.ID,
	}, "admin", false)
	if err != nil {
		t.Fatalf("UpsertInstance: %v", err)
	}

	httpReq := httptest.NewRequest(http.MethodGet, "/v1/stream", nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	server.Routes().ServeHTTP(rec, httpReq)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if contentType := rec.Header().Get("Content-Type"); contentType != "text/event-stream" {
		t.Fatalf("expected text/event-stream, got %s", contentType)
	}
}

func TestHTTP_Auth_BootstrapAndLoginFlow(t *testing.T) {
	server, repo := newTestServerWithRepo(t)
	ctx := context.Background()

	// 1. Initial status -> unbootstrapped
	req := httptest.NewRequest(http.MethodGet, "/v1/auth/status", nil)
	rec := httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("auth status expected 200, got %d", rec.Code)
	}

	var statusRes map[string]interface{}
	_ = json.NewDecoder(rec.Body).Decode(&statusRes)
	if statusRes["bootstrapped"] != false {
		t.Fatalf("expected bootstrapped=false initially")
	}

	// 2. Create bootstrap token
	rawBootstrapToken := "abt_testbootstrap123456789"
	err := repo.CreateBootstrapToken(ctx, credential.HashSecret(rawBootstrapToken))
	if err != nil {
		t.Fatalf("CreateBootstrapToken failed: %v", err)
	}

	// 3. Perform bootstrap setup
	bootstrapPayload := `{"bootstrap_token":"abt_testbootstrap123456789","name":"Master Owner","email":"owner@autorix.internal","password":"MasterPassword123!"}`
	req = httptest.NewRequest(http.MethodPost, "/v1/auth/bootstrap", strings.NewReader(bootstrapPayload))
	rec = httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("bootstrap expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var bootRes map[string]interface{}
	_ = json.NewDecoder(rec.Body).Decode(&bootRes)
	sessionToken, ok := bootRes["session_token"].(string)
	if !ok || sessionToken == "" {
		t.Fatalf("expected session_token in bootstrap response")
	}

	// 4. Validate session
	req = httptest.NewRequest(http.MethodGet, "/v1/auth/session", nil)
	req.Header.Set("Authorization", "Bearer "+sessionToken)
	rec = httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("validate session expected 200, got %d", rec.Code)
	}

	// 5. Test login with correct password
	loginPayload := `{"email":"owner@autorix.internal","password":"MasterPassword123!"}`
	req = httptest.NewRequest(http.MethodPost, "/v1/auth/login", strings.NewReader(loginPayload))
	rec = httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("login expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// 6. Test login with wrong password
	badLoginPayload := `{"email":"owner@autorix.internal","password":"WrongPassword!"}`
	req = httptest.NewRequest(http.MethodPost, "/v1/auth/login", strings.NewReader(badLoginPayload))
	rec = httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("bad login expected 401, got %d", rec.Code)
	}

	// 7. Test Logout
	req = httptest.NewRequest(http.MethodDelete, "/v1/auth/session", nil)
	req.Header.Set("Authorization", "Bearer "+sessionToken)
	rec = httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("logout expected 204, got %d", rec.Code)
	}
}

