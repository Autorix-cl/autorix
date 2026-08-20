package registry

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	argusv1 "github.com/autorix/argus/api/autorix/argus/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

// fakeArgusServer is a minimal, fully in-memory ArgusService implementation
// used to test the registry client without a real Argus instance.
type fakeArgusServer struct {
	argusv1.UnimplementedArgusServiceServer

	registerCalls   atomic.Int32
	heartbeatCalls  atomic.Int32
	deregisterCalls atomic.Int32

	registerErr error
	credential  string

	// deregisterUnimplemented mirrors the real Argus server today: the RPC
	// exists but returns codes.Unimplemented.
	deregisterUnimplemented bool

	// heartbeatErrUntil, when > 0, makes Heartbeat fail with an
	// Unavailable error for the first N calls, then succeed — used to
	// simulate a control-plane outage followed by recovery.
	heartbeatErrUntil int32

	mu               sync.Mutex
	lastRegisterReq  *argusv1.RegisterRequest
	lastHeartbeatReq *argusv1.HeartbeatRequest
}

func (f *fakeArgusServer) Register(ctx context.Context, req *argusv1.RegisterRequest) (*argusv1.RegisterResponse, error) {
	f.registerCalls.Add(1)
	f.mu.Lock()
	f.lastRegisterReq = req
	f.mu.Unlock()
	if f.registerErr != nil {
		return nil, f.registerErr
	}
	cred := f.credential
	if cred == "" {
		cred = "test-credential"
	}
	return &argusv1.RegisterResponse{
		InstanceUuid:             "11111111-1111-1111-1111-111111111111",
		InstanceCredential:       cred,
		HeartbeatIntervalSeconds: 15,
	}, nil
}

func (f *fakeArgusServer) Heartbeat(ctx context.Context, req *argusv1.HeartbeatRequest) (*argusv1.HeartbeatResponse, error) {
	n := f.heartbeatCalls.Add(1)
	f.mu.Lock()
	f.lastHeartbeatReq = req
	f.mu.Unlock()
	if f.heartbeatErrUntil > 0 && n <= f.heartbeatErrUntil {
		return nil, status.Error(codes.Unavailable, "simulated control plane outage")
	}
	return &argusv1.HeartbeatResponse{Status: "healthy"}, nil
}

func (f *fakeArgusServer) getLastRegisterReq() *argusv1.RegisterRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.lastRegisterReq
}

func (f *fakeArgusServer) getLastHeartbeatReq() *argusv1.HeartbeatRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.lastHeartbeatReq
}

func (f *fakeArgusServer) Deregister(ctx context.Context, req *argusv1.DeregisterRequest) (*argusv1.DeregisterResponse, error) {
	f.deregisterCalls.Add(1)
	if f.deregisterUnimplemented {
		return nil, status.Error(codes.Unimplemented, "deregistration lands with P2-S6")
	}
	return &argusv1.DeregisterResponse{}, nil
}

// startFakeArgus spins up an in-memory (bufconn) gRPC server backed by
// srv, and returns a dialer suitable for grpc.NewClient's
// grpc.WithContextDialer, plus a stop func.
func startFakeArgus(t *testing.T, srv *fakeArgusServer) (dialer func(context.Context, string) (net.Conn, error), stop func()) {
	t.Helper()

	lis := bufconn.Listen(1024 * 1024)
	grpcServer := grpc.NewServer()
	argusv1.RegisterArgusServiceServer(grpcServer, srv)

	go func() {
		_ = grpcServer.Serve(lis)
	}()

	dialer = func(ctx context.Context, _ string) (net.Conn, error) {
		return lis.DialContext(ctx)
	}
	stop = func() {
		grpcServer.Stop()
		_ = lis.Close()
	}
	return dialer, stop
}

func newTestClient(t *testing.T, srv *fakeArgusServer) *Client {
	t.Helper()
	return newTestClientWithConfig(t, srv, func(cfg *Config) {})
}

func newTestClientWithConfig(t *testing.T, srv *fakeArgusServer, mutate func(cfg *Config)) *Client {
	t.Helper()

	dialer, stop := startFakeArgus(t, srv)
	t.Cleanup(stop)

	cfg := Config{
		ArgusURL:        "passthrough:///bufnet",
		EnrollmentToken: "test-token",
		EngineType:      "ego",
		Environment:     "test",
		InstanceID:      "test-instance",
		Endpoints:       Endpoints{REST: "http://localhost:4433", GRPC: ""},
	}
	mutate(&cfg)

	c := New(cfg)
	// Test-only hook: dial through bufconn instead of a real network
	// address, and shrink the bounded retry/backoff windows so tests run
	// fast.
	c.dialOpts = append(c.dialOpts,
		grpc.WithContextDialer(dialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	c.registerTimeout = 2 * time.Second
	c.registerBackoffBase = 5 * time.Millisecond
	c.registerMaxAttempts = 3
	c.heartbeatTimeout = 2 * time.Second
	c.heartbeatBackoffBase = 5 * time.Millisecond
	c.heartbeatMaxAttempts = 3
	return c
}

func TestNoop_WithoutArgusURLOrToken(t *testing.T) {
	c := New(Config{
		ArgusURL:        "",
		EnrollmentToken: "",
		EngineType:      "ego",
		Environment:     "test",
	})

	if c.Enabled() {
		t.Fatalf("expected client to be disabled without ArgusURL/EnrollmentToken")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// Must never block, panic, or fail engine startup.
	c.Start(ctx)

	if err := c.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() on a no-op client should never error, got %v", err)
	}
}

func TestNoop_MissingTokenOnly(t *testing.T) {
	c := New(Config{ArgusURL: "argus:50051", EnrollmentToken: "", EngineType: "ego"})
	if c.Enabled() {
		t.Fatalf("expected client to be disabled without EnrollmentToken")
	}
}

func TestNoop_MissingURLOnly(t *testing.T) {
	c := New(Config{ArgusURL: "", EnrollmentToken: "sometoken", EngineType: "ego"})
	if c.Enabled() {
		t.Fatalf("expected client to be disabled without ArgusURL")
	}
}

func TestStart_RegistersSuccessfully(t *testing.T) {
	srv := &fakeArgusServer{}
	c := newTestClient(t, srv)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c.Start(ctx)
	defer func() { _ = c.Stop(context.Background()) }()

	if got := srv.registerCalls.Load(); got != 1 {
		t.Fatalf("expected exactly 1 Register call, got %d", got)
	}
	if !c.Registered() {
		t.Fatalf("expected client to be marked as registered after a successful Register")
	}
}

func TestStart_HeartbeatFiresPeriodically(t *testing.T) {
	srv := &fakeArgusServer{}
	c := newTestClient(t, srv)

	fake := newFakeTicker()
	c.newTicker = func(d time.Duration) tickerHandle { return fake }

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c.Start(ctx)
	defer func() { _ = c.Stop(context.Background()) }()

	const n = 3
	for i := 0; i < n; i++ {
		fake.tick()
		waitForCount(t, &srv.heartbeatCalls, int32(i+1))
	}
}

func TestStart_ArgusDownAtStartupDoesNotBlockOrPanic(t *testing.T) {
	c := New(Config{
		ArgusURL:        "127.0.0.1:1", // nothing listens here
		EnrollmentToken: "test-token",
		EngineType:      "ego",
		Environment:     "test",
	})
	c.registerTimeout = 300 * time.Millisecond
	c.registerBackoffBase = 5 * time.Millisecond
	c.registerMaxAttempts = 2
	c.heartbeatTimeout = 300 * time.Millisecond

	done := make(chan struct{})
	go func() {
		defer close(done)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		c.Start(ctx) // must not panic
	}()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatalf("Start() blocked far longer than its bounded registerTimeout")
	}

	if c.Registered() {
		t.Fatalf("expected client to remain unregistered when Argus is unreachable")
	}

	// Stop must also be a safe, non-blocking no-op in this degraded state.
	stopDone := make(chan struct{})
	go func() {
		defer close(stopDone)
		_ = c.Stop(context.Background())
	}()
	select {
	case <-stopDone:
	case <-time.After(2 * time.Second):
		t.Fatalf("Stop() blocked when the client never successfully registered")
	}
}

func TestStop_DoesNotBlockWhenDeregisterUnimplemented(t *testing.T) {
	srv := &fakeArgusServer{deregisterUnimplemented: true}
	c := newTestClient(t, srv)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c.Start(ctx)

	done := make(chan struct{})
	go func() {
		defer close(done)
		if err := c.Stop(context.Background()); err != nil {
			t.Errorf("Stop() should tolerate Unimplemented Deregister, got %v", err)
		}
	}()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatalf("Stop() blocked despite Deregister returning Unimplemented")
	}

	if srv.deregisterCalls.Load() != 1 {
		t.Fatalf("expected exactly 1 Deregister call, got %d", srv.deregisterCalls.Load())
	}
}

// fakeTicker lets tests drive the heartbeat loop deterministically instead
// of waiting on a real time.Ticker.
type fakeTicker struct {
	ch chan time.Time
}

func newFakeTicker() *fakeTicker {
	return &fakeTicker{ch: make(chan time.Time, 8)}
}

func (f *fakeTicker) C() <-chan time.Time { return f.ch }
func (f *fakeTicker) Stop()               {}
func (f *fakeTicker) tick()               { f.ch <- time.Now() }

func waitForCount(t *testing.T, counter *atomic.Int32, want int32) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if counter.Load() >= want {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for counter to reach %d, got %d", want, counter.Load())
}

// --- P2-S4-T1: credential persistence -------------------------------------

func TestStart_PersistsCredentialAfterRegister(t *testing.T) {
	srv := &fakeArgusServer{credential: "persisted-cred"}
	path := filepath.Join(t.TempDir(), "registry-credential.json")

	c := newTestClientWithConfig(t, srv, func(cfg *Config) {
		cfg.CredentialPath = path
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c.Start(ctx)
	defer func() { _ = c.Stop(context.Background()) }()

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected credential file to be written at %s: %v", path, err)
	}

	pc, err := c.loadPersistedCredential()
	if err != nil {
		t.Fatalf("expected persisted credential to be loadable, got %v", err)
	}
	if pc.Credential != "persisted-cred" {
		t.Fatalf("expected persisted credential 'persisted-cred', got %q", pc.Credential)
	}
}

func TestStart_ReusesPersistedCredential_SkipsRegister(t *testing.T) {
	srv := &fakeArgusServer{}
	path := filepath.Join(t.TempDir(), "registry-credential.json")

	// Seed the credential file as if a prior process had already
	// registered this exact instance.
	seed := New(Config{EngineType: "ego", InstanceID: "test-instance", CredentialPath: path})
	seed.instanceUUID = "seeded-uuid"
	seed.credential = "seeded-cred"
	seed.registeredAt = time.Now().Add(-time.Hour)
	seed.persistCredential()

	c := newTestClientWithConfig(t, srv, func(cfg *Config) {
		cfg.CredentialPath = path
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c.Start(ctx)
	defer func() { _ = c.Stop(context.Background()) }()

	if got := srv.registerCalls.Load(); got != 0 {
		t.Fatalf("expected Register to be skipped when a valid credential is persisted, got %d calls", got)
	}
	if !c.Registered() {
		t.Fatalf("expected client to be marked registered from the persisted credential")
	}
	if got := c.Status().InstanceUUID; got != "seeded-uuid" {
		t.Fatalf("expected instance_uuid from persisted credential, got %q", got)
	}
}

func TestStart_CorruptCredentialFile_FallsBackToRegister(t *testing.T) {
	srv := &fakeArgusServer{}
	path := filepath.Join(t.TempDir(), "registry-credential.json")
	if err := os.WriteFile(path, []byte("{not valid json"), 0o600); err != nil {
		t.Fatalf("failed to seed corrupt credential file: %v", err)
	}

	c := newTestClientWithConfig(t, srv, func(cfg *Config) {
		cfg.CredentialPath = path
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c.Start(ctx)
	defer func() { _ = c.Stop(context.Background()) }()

	if got := srv.registerCalls.Load(); got != 1 {
		t.Fatalf("expected a fresh Register call when the persisted credential is corrupt, got %d", got)
	}
	if !c.Registered() {
		t.Fatalf("expected client to register successfully after falling back")
	}
}

func TestStart_CredentialForDifferentInstance_FallsBackToRegister(t *testing.T) {
	srv := &fakeArgusServer{}
	path := filepath.Join(t.TempDir(), "registry-credential.json")

	seed := New(Config{EngineType: "ego", InstanceID: "some-other-instance", CredentialPath: path})
	seed.instanceUUID = "seeded-uuid"
	seed.credential = "seeded-cred"
	seed.registeredAt = time.Now()
	seed.persistCredential()

	c := newTestClientWithConfig(t, srv, func(cfg *Config) {
		cfg.CredentialPath = path
		cfg.InstanceID = "test-instance" // differs from the seeded credential
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c.Start(ctx)
	defer func() { _ = c.Stop(context.Background()) }()

	if got := srv.registerCalls.Load(); got != 1 {
		t.Fatalf("expected a fresh Register call when the persisted credential belongs to another instance, got %d", got)
	}
}

// --- P2-S4-T3: enriched heartbeat / recovery report ------------------------

func TestRecordHeartbeatOutcome_TracksFailuresAndRecovery(t *testing.T) {
	c := New(Config{EngineType: "ego"})

	c.recordHeartbeatOutcome(errors.New("boom 1"))
	c.recordHeartbeatOutcome(errors.New("boom 2"))
	c.recordHeartbeatOutcome(errors.New("boom 3"))

	if got := c.FailureStreak(); got != 3 {
		t.Fatalf("expected failure streak of 3, got %d", got)
	}

	c.recordHeartbeatOutcome(nil)

	if got := c.FailureStreak(); got != 0 {
		t.Fatalf("expected failure streak reset to 0 after recovery, got %d", got)
	}
	if got := c.LastRecoveryFailureCount(); got != 3 {
		t.Fatalf("expected recovery report of 3 failures, got %d", got)
	}

	history := c.HeartbeatHistory()
	if len(history) != 4 {
		t.Fatalf("expected 4 recorded heartbeat outcomes, got %d", len(history))
	}
	for i := 0; i < 3; i++ {
		if history[i].OK {
			t.Fatalf("expected history[%d] to be a failure", i)
		}
	}
	if !history[3].OK {
		t.Fatalf("expected history[3] to be the recovering success")
	}
}

func TestRecordHeartbeatOutcome_RingBufferBounded(t *testing.T) {
	c := New(Config{EngineType: "ego"})
	for i := 0; i < heartbeatHistoryCap+10; i++ {
		c.recordHeartbeatOutcome(nil)
	}
	if got := len(c.HeartbeatHistory()); got != heartbeatHistoryCap {
		t.Fatalf("expected history capped at %d, got %d", heartbeatHistoryCap, got)
	}
}

func TestStart_HeartbeatFailsThenRecovers_ReportsRecovery(t *testing.T) {
	// The server fails every attempt of the first heartbeat send (all
	// heartbeatMaxAttempts retries), then succeeds on the second tick.
	srv := &fakeArgusServer{heartbeatErrUntil: 3}
	c := newTestClient(t, srv)
	c.heartbeatMaxAttempts = 3 // exactly the number of failing calls

	fake := newFakeTicker()
	c.newTicker = func(d time.Duration) tickerHandle { return fake }

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c.Start(ctx)
	defer func() { _ = c.Stop(context.Background()) }()

	fake.tick() // first send: exhausts all 3 attempts, all failing
	waitForCount(t, &srv.heartbeatCalls, 3)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && c.FailureStreak() == 0 {
		time.Sleep(2 * time.Millisecond)
	}
	if got := c.FailureStreak(); got != 1 {
		t.Fatalf("expected 1 recorded failed heartbeat send, got %d", got)
	}

	fake.tick() // second send: succeeds immediately
	waitForCount(t, &srv.heartbeatCalls, 4)

	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && c.FailureStreak() != 0 {
		time.Sleep(2 * time.Millisecond)
	}
	if got := c.FailureStreak(); got != 0 {
		t.Fatalf("expected failure streak reset after recovery, got %d", got)
	}
	if got := c.LastRecoveryFailureCount(); got != 1 {
		t.Fatalf("expected recovery report of 1 failed send, got %d", got)
	}
}

// --- P2-S4-T4: capability advertisement -------------------------------------

func TestRegister_SendsDeclaredCapabilities(t *testing.T) {
	srv := &fakeArgusServer{}
	c := newTestClientWithConfig(t, srv, func(cfg *Config) {
		cfg.Capabilities = []string{"health.v1", "paging.v1"}
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c.Start(ctx)
	defer func() { _ = c.Stop(context.Background()) }()

	req := srv.getLastRegisterReq()
	if req == nil {
		t.Fatalf("expected Register to have been called")
	}
	if got := req.GetCapabilities(); len(got) != 2 || got[0] != "health.v1" || got[1] != "paging.v1" {
		t.Fatalf("expected capabilities [health.v1 paging.v1], got %v", got)
	}
}

// --- P2-S4-T5: dependency declaration and probing ---------------------------

func TestHeartbeat_ReportsDependencyStatus(t *testing.T) {
	healthyLis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start healthy dependency listener: %v", err)
	}
	defer healthyLis.Close()
	go func() {
		for {
			conn, err := healthyLis.Accept()
			if err != nil {
				return
			}
			_ = conn.Close()
		}
	}()

	srv := &fakeArgusServer{}
	c := newTestClientWithConfig(t, srv, func(cfg *Config) {
		cfg.Dependencies = []Dependency{
			{Name: "postgres", Target: healthyLis.Addr().String()},
			{Name: "unreachable-dep", Target: "127.0.0.1:1"},
		}
	})
	c.dependencyProbeTimeout = 500 * time.Millisecond

	fake := newFakeTicker()
	c.newTicker = func(d time.Duration) tickerHandle { return fake }

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c.Start(ctx)
	defer func() { _ = c.Stop(context.Background()) }()

	fake.tick()
	waitForCount(t, &srv.heartbeatCalls, 1)

	req := srv.getLastHeartbeatReq()
	if req == nil {
		t.Fatalf("expected a heartbeat request to have been captured")
	}
	deps := req.GetDependencies()
	if len(deps) != 2 {
		t.Fatalf("expected 2 dependency statuses, got %d", len(deps))
	}
	byName := map[string]*argusv1.DependencyStatus{}
	for _, d := range deps {
		byName[d.GetName()] = d
	}
	if !byName["postgres"].GetReachable() {
		t.Fatalf("expected postgres dependency to be reported reachable")
	}
	if byName["unreachable-dep"].GetReachable() {
		t.Fatalf("expected unreachable-dep dependency to be reported unreachable")
	}
}

func TestPostgresDependencyTarget(t *testing.T) {
	got := PostgresDependencyTarget("postgres://autorix:secret@db-host:5432/autorix_ego?sslmode=disable")
	if got != "db-host:5432" {
		t.Fatalf("expected 'db-host:5432', got %q", got)
	}
}

// --- P2-S4-T8: control plane down does not affect engine operation ---------

func TestControlPlaneDown_EngineKeepsServingNormally(t *testing.T) {
	// Mirrors the pattern every main.go uses: registry.Start(ctx) is called
	// during startup but must never block engine startup or its own HTTP
	// serving, even when Argus is entirely unreachable.
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health/ready", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ready"}`))
	})
	engine := httptest.NewServer(mux)
	defer engine.Close()

	c := New(Config{
		ArgusURL:        "127.0.0.1:1", // nothing listens here: control plane down
		EnrollmentToken: "test-token",
		EngineType:      "ego",
		Environment:     "test",
		InstanceID:      "cp-down-instance",
	})
	c.registerTimeout = 300 * time.Millisecond
	c.registerBackoffBase = 5 * time.Millisecond
	c.registerMaxAttempts = 2
	c.heartbeatTimeout = 300 * time.Millisecond

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c.Start(ctx) // must return promptly, degraded but non-blocking

	resp, err := http.Get(engine.URL + "/health/ready")
	if err != nil {
		t.Fatalf("engine health endpoint should stay reachable, got error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected engine to keep serving normally with status 200, got %d", resp.StatusCode)
	}

	if c.Registered() {
		t.Fatalf("expected registration to fail against an unreachable control plane")
	}

	stopDone := make(chan struct{})
	go func() {
		defer close(stopDone)
		_ = c.Stop(context.Background())
	}()
	select {
	case <-stopDone:
	case <-time.After(3 * time.Second):
		t.Fatalf("Stop() blocked despite the client never having registered")
	}
}
