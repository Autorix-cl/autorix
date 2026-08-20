// Package registry is the platform/registry client every engine embeds to
// announce itself to Argus, the (optional) control plane. Argus never sits
// in the data path: an engine with no Argus configured runs exactly as it
// did before this package existed, and an unreachable Argus degrades to
// the same no-op behavior rather than blocking or failing engine startup.
package registry

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/big"
	"net"
	"net/http"
	neturl "net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	argusv1 "github.com/autorix/argus/api/autorix/argus/v1"
	"github.com/autorix/platform/version"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

// Endpoints are the REST/gRPC addresses an engine advertises to Argus so it
// can be addressed without hardcoded per-environment configuration.
type Endpoints struct {
	REST string
	GRPC string
}

// Dependency is a known downstream this engine relies on. The client probes
// it on every heartbeat and reports the result via
// HeartbeatRequest.dependencies, so Argus's fleet view reflects real
// reachability instead of just this engine's own liveness.
type Dependency struct {
	// Name identifies the dependency, e.g. "postgres" or the depended-on
	// engine_type ("nexus", "janus").
	Name string
	// Target is what gets probed: "host:port" for a plain TCP dial, or an
	// "http://"/"https://" URL for an HTTP GET (2xx/3xx/4xx counts as
	// reachable — the process answered; 5xx and transport errors do not).
	Target string
}

// HeartbeatRecord is one outcome in the client's bounded heartbeat history,
// used to build a recovery report after a streak of failures.
type HeartbeatRecord struct {
	At  time.Time
	OK  bool
	Err string
}

// Status is a point-in-time snapshot of the client's registration state,
// meant to back a future /info-style endpoint.
type Status struct {
	Enabled       bool
	Registered    bool
	InstanceID    string
	InstanceUUID  string
	EngineType    string
	Environment   string
	Capabilities  []string
	RegisteredAt  time.Time
	FailureStreak int
}

// Config configures a Client. It is a plain value type — env parsing lives
// in NewFromEnv, keeping this constructor pure and easy to unit test.
type Config struct {
	// ArgusURL is Argus's gRPC address, e.g. "argus:50051". Empty disables
	// the client.
	ArgusURL string
	// EnrollmentToken is the one-time-use token minted via Argus's REST
	// admin API. Empty disables the client.
	EnrollmentToken string
	EngineType      string
	Environment     string
	// InstanceID is stable across restarts. Left empty, New derives one
	// from the hostname, falling back to a random UUID.
	InstanceID   string
	Endpoints    Endpoints
	Capabilities []string
	// Dependencies this engine relies on, probed every heartbeat. Optional.
	Dependencies []Dependency

	// CredentialPath persists the instance credential returned by Register
	// so a restart can reuse it instead of re-registering with a (likely
	// single-use) enrollment token. Empty uses a default under
	// /var/lib/autorix/<engine_type>/registry-credential.json.
	CredentialPath string

	Logger *slog.Logger

	// HeartbeatInterval defaults to 15s per the platform/registry contract.
	HeartbeatInterval time.Duration
}

// tickerHandle abstracts time.Ticker so tests can drive the heartbeat loop
// deterministically instead of waiting on real wall-clock time.
type tickerHandle interface {
	C() <-chan time.Time
	Stop()
}

type realTicker struct{ t *time.Ticker }

func (r *realTicker) C() <-chan time.Time { return r.t.C }
func (r *realTicker) Stop()               { r.t.Stop() }

func newRealTicker(d time.Duration) tickerHandle { return &realTicker{t: time.NewTicker(d)} }

// Client registers an engine with Argus and keeps it alive with periodic
// heartbeats. A Client built from an incomplete Config is a permanent
// no-op: Start and Stop are always safe to call and never block the
// engine's own lifecycle.
type Client struct {
	cfg     Config
	enabled bool
	logger  *slog.Logger

	dialOpts []grpc.DialOption
	conn     *grpc.ClientConn
	rpc      argusv1.ArgusServiceClient

	heartbeatInterval time.Duration
	newTicker         func(d time.Duration) tickerHandle

	registerTimeout     time.Duration
	registerBackoffBase time.Duration
	registerMaxAttempts int

	heartbeatTimeout     time.Duration
	heartbeatBackoffBase time.Duration
	heartbeatMaxAttempts int

	dependencyProbeTimeout time.Duration

	mu               sync.Mutex
	registered       bool
	heartbeatStarted bool
	instanceUUID     string
	credential       string
	registeredAt     time.Time

	heartbeatHistory     []HeartbeatRecord
	consecutiveFailures  int
	lastRecoveryFailures int

	stopOnce sync.Once
	stopCh   chan struct{}
	doneCh   chan struct{}
}

// heartbeatHistoryCap bounds the in-memory heartbeat outcome ring buffer.
const heartbeatHistoryCap = 20

// New builds a Client from cfg. The client is enabled only when both
// ArgusURL and EnrollmentToken are set; otherwise every method is a
// deliberate no-op (see package doc).
func New(cfg Config) *Client {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	instanceID := cfg.InstanceID
	if instanceID == "" {
		if hostname, err := os.Hostname(); err == nil && hostname != "" {
			instanceID = hostname
		} else {
			instanceID = uuid.NewString()
		}
	}
	cfg.InstanceID = instanceID

	interval := cfg.HeartbeatInterval
	if interval <= 0 {
		interval = 15 * time.Second
	}

	c := &Client{
		cfg:                  cfg,
		enabled:              cfg.ArgusURL != "" && cfg.EnrollmentToken != "",
		logger:               logger,
		heartbeatInterval:    interval,
		newTicker:            newRealTicker,
		registerTimeout:      8 * time.Second,
		registerBackoffBase:  200 * time.Millisecond,
		registerMaxAttempts:  4,
		heartbeatTimeout:       5 * time.Second,
		heartbeatBackoffBase:   200 * time.Millisecond,
		heartbeatMaxAttempts:   3,
		dependencyProbeTimeout: 2 * time.Second,
		stopCh:                 make(chan struct{}),
		doneCh:                 make(chan struct{}),
	}
	// A disabled client's heartbeat goroutine never starts, so doneCh must
	// already look "finished" to keep Stop non-blocking.
	if !c.enabled {
		close(c.doneCh)
	}
	return c
}

// NewFromEnv builds a Config from the standard platform/registry
// environment variables and constructs a Client. engineType, endpoints,
// capabilities and dependencies are supplied by each engine's main.go since
// they are compile-time constants for that binary, not environment-driven.
func NewFromEnv(engineType string, endpoints Endpoints, capabilities []string, dependencies []Dependency, logger *slog.Logger) *Client {
	return New(Config{
		ArgusURL:        os.Getenv("AUTORIX_ARGUS_URL"),
		EnrollmentToken: os.Getenv("AUTORIX_ENROLLMENT_TOKEN"),
		EngineType:      engineType,
		Environment:     os.Getenv("AUTORIX_ENVIRONMENT"),
		InstanceID:      os.Getenv("AUTORIX_INSTANCE_ID"),
		Endpoints:       endpoints,
		Capabilities:    capabilities,
		Dependencies:    dependencies,
		CredentialPath:  os.Getenv("AUTORIX_REGISTRY_CREDENTIAL_PATH"),
		Logger:          logger,
	})
}

// PostgresDependencyTarget extracts the "host:port" authority from a
// Postgres DSN (postgres://user:pass@host:port/db?...) for use as a
// Dependency.Target TCP probe address. It returns "" if dsn cannot be
// parsed as a URL, so callers should skip declaring the dependency in that
// case rather than probe an empty address.
func PostgresDependencyTarget(dsn string) string {
	u, err := neturl.Parse(dsn)
	if err != nil {
		return ""
	}
	return u.Host
}

// Enabled reports whether this client will ever talk to Argus.
func (c *Client) Enabled() bool { return c.enabled }

// Registered reports whether Register has succeeded at least once.
func (c *Client) Registered() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.registered
}

// Status returns a point-in-time snapshot of the client's registration
// state, meant to back a future /info-style endpoint.
func (c *Client) Status() Status {
	c.mu.Lock()
	defer c.mu.Unlock()
	return Status{
		Enabled:       c.enabled,
		Registered:    c.registered,
		InstanceID:    c.cfg.InstanceID,
		InstanceUUID:  c.instanceUUID,
		EngineType:    c.cfg.EngineType,
		Environment:   c.cfg.Environment,
		Capabilities:  c.cfg.Capabilities,
		RegisteredAt:  c.registeredAt,
		FailureStreak: c.consecutiveFailures,
	}
}

// HeartbeatHistory returns a copy of the bounded recent heartbeat outcome
// buffer, oldest first.
func (c *Client) HeartbeatHistory() []HeartbeatRecord {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]HeartbeatRecord, len(c.heartbeatHistory))
	copy(out, c.heartbeatHistory)
	return out
}

// FailureStreak returns the number of consecutive heartbeat failures
// observed since the last success (0 if the last heartbeat succeeded or none
// has been sent yet).
func (c *Client) FailureStreak() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.consecutiveFailures
}

// LastRecoveryFailureCount returns how many consecutive failures preceded
// the most recent recovery (a failed streak followed by a success), or 0 if
// no recovery has happened yet.
func (c *Client) LastRecoveryFailureCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.lastRecoveryFailures
}

// Start registers the engine with Argus and, on success, begins sending
// periodic heartbeats in a background goroutine. Start never blocks the
// caller longer than its bounded registerTimeout, never panics, and never
// prevents the engine from finishing startup — a failed or skipped
// registration just leaves the client in its no-op/degraded state.
func (c *Client) Start(ctx context.Context) {
	if !c.enabled {
		c.logger.Info("argus registry client disabled (no AUTORIX_ARGUS_URL/AUTORIX_ENROLLMENT_TOKEN); running standalone")
		return
	}

	dialOpts := append([]grpc.DialOption{}, c.dialOpts...)
	if len(c.dialOpts) == 0 {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	conn, err := grpc.NewClient(c.cfg.ArgusURL, dialOpts...)
	if err != nil {
		c.logger.Warn("argus registry client: failed to build grpc client; continuing standalone", "error", err)
		return
	}
	c.conn = conn
	c.rpc = argusv1.NewArgusServiceClient(conn)

	if pc, loadErr := c.loadPersistedCredential(); loadErr == nil {
		c.mu.Lock()
		c.registered = true
		c.instanceUUID = pc.InstanceUUID
		c.credential = pc.Credential
		c.registeredAt = pc.RegisteredAt
		c.mu.Unlock()
		c.logger.Info("reusing persisted argus registry credential; skipping registration",
			"instance_uuid", c.instanceUUID, "engine_type", c.cfg.EngineType)
	} else {
		regCtx, cancel := context.WithTimeout(ctx, c.registerTimeout)
		defer cancel()

		err = retryWithBackoff(regCtx, c.registerMaxAttempts, c.registerBackoffBase, func() error {
			return c.register(regCtx)
		})
		if err != nil {
			c.logger.Warn("argus registry client: registration failed after retries; continuing standalone (degraded)", "error", err)
			return
		}

		c.logger.Info("registered with argus", "instance_uuid", c.instanceUUID, "engine_type", c.cfg.EngineType)
		c.persistCredential()
	}

	c.doneCh = make(chan struct{})
	c.mu.Lock()
	c.heartbeatStarted = true
	c.mu.Unlock()
	go c.heartbeatLoop()
}

func (c *Client) register(ctx context.Context) error {
	resp, err := c.rpc.Register(ctx, &argusv1.RegisterRequest{
		EnrollmentToken: c.cfg.EnrollmentToken,
		EngineType:      c.cfg.EngineType,
		InstanceId:      c.cfg.InstanceID,
		Environment:     c.cfg.Environment,
		Version:         version.Version,
		BuildSha:        version.BuildSHA,
		Capabilities:    c.cfg.Capabilities,
		Endpoints: &argusv1.Endpoints{
			RestUrl: c.cfg.Endpoints.REST,
			GrpcUrl: c.cfg.Endpoints.GRPC,
		},
	})
	if err != nil {
		return err
	}

	c.mu.Lock()
	c.registered = true
	c.instanceUUID = resp.GetInstanceUuid()
	c.credential = resp.GetInstanceCredential()
	c.registeredAt = time.Now()
	c.mu.Unlock()
	return nil
}

// persistedCredential is the on-disk shape written after a successful
// Register and read back on a later Start to skip re-registering with a
// (likely single-use) enrollment token.
type persistedCredential struct {
	InstanceID   string    `json:"instance_id"`
	EngineType   string    `json:"engine_type"`
	Environment  string    `json:"environment"`
	InstanceUUID string    `json:"instance_uuid"`
	Credential   string    `json:"credential"`
	RegisteredAt time.Time `json:"registered_at"`
}

// credentialPath returns the configured path, or a default derived from
// this engine's type.
func (c *Client) credentialPath() string {
	if c.cfg.CredentialPath != "" {
		return c.cfg.CredentialPath
	}
	return filepath.Join("/var/lib/autorix", c.cfg.EngineType, "registry-credential.json")
}

// loadPersistedCredential reads and validates the on-disk credential. It
// returns an error for anything that makes the credential unusable:
// missing/unreadable file, malformed JSON, incomplete fields, or a mismatch
// against this client's instance_id/engine_type (e.g. a credential left
// over from a different instance).
func (c *Client) loadPersistedCredential() (*persistedCredential, error) {
	data, err := os.ReadFile(c.credentialPath())
	if err != nil {
		return nil, err
	}

	var pc persistedCredential
	if err := json.Unmarshal(data, &pc); err != nil {
		return nil, fmt.Errorf("persisted credential is not valid JSON: %w", err)
	}
	if pc.InstanceUUID == "" || pc.Credential == "" || pc.InstanceID == "" {
		return nil, fmt.Errorf("persisted credential is missing required fields")
	}
	if pc.InstanceID != c.cfg.InstanceID || pc.EngineType != c.cfg.EngineType {
		return nil, fmt.Errorf("persisted credential does not match this instance (instance_id=%q engine_type=%q)", pc.InstanceID, pc.EngineType)
	}
	return &pc, nil
}

// persistCredential writes the current credential to disk, best-effort: any
// failure (unwritable path, read-only filesystem, ...) is logged and
// otherwise ignored, since losing the persisted copy only means the next
// restart re-registers instead of failing to start.
func (c *Client) persistCredential() {
	c.mu.Lock()
	pc := persistedCredential{
		InstanceID:   c.cfg.InstanceID,
		EngineType:   c.cfg.EngineType,
		Environment:  c.cfg.Environment,
		InstanceUUID: c.instanceUUID,
		Credential:   c.credential,
		RegisteredAt: c.registeredAt,
	}
	c.mu.Unlock()

	path := c.credentialPath()
	data, err := json.Marshal(pc)
	if err != nil {
		c.logger.Warn("argus registry client: failed to marshal credential for persistence", "error", err)
		return
	}
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			c.logger.Warn("argus registry client: failed to create credential directory; continuing without persistence", "error", err, "path", dir)
			return
		}
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		c.logger.Warn("argus registry client: failed to persist credential; continuing without persistence", "error", err, "path", path)
	}
}

func (c *Client) heartbeatLoop() {
	defer close(c.doneCh)

	t := c.newTicker(c.heartbeatInterval)
	defer t.Stop()

	for {
		select {
		case <-c.stopCh:
			return
		case <-t.C():
			c.sendHeartbeat()
		}
	}
}

func (c *Client) sendHeartbeat() {
	defer func() {
		if r := recover(); r != nil {
			c.logger.Warn("argus registry client: heartbeat goroutine recovered from panic", "panic", r)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), c.heartbeatTimeout)
	defer cancel()

	deps := c.probeDependencies(ctx)

	err := retryWithBackoff(ctx, c.heartbeatMaxAttempts, c.heartbeatBackoffBase, func() error {
		c.mu.Lock()
		uuidStr := c.instanceUUID
		cred := c.credential
		c.mu.Unlock()

		_, err := c.rpc.Heartbeat(ctx, &argusv1.HeartbeatRequest{
			InstanceUuid:  uuidStr,
			Live:          true,
			Ready:         true,
			Dependencies:  deps,
			TimestampUnix: time.Now().Unix(),
			Nonce:         uuid.NewString(),
			// The current Argus server checks this field directly against
			// the instance's stored credential hash; it is not yet a
			// message-bound HMAC signature (see argus/internal/transport
			// /grpc/server.go Heartbeat).
			Signature: cred,
		})
		return err
	})
	c.recordHeartbeatOutcome(err)
	if err != nil {
		c.logger.Warn("argus registry client: heartbeat failed after retries", "error", err, "failure_streak", c.FailureStreak())
	}
}

// recordHeartbeatOutcome appends err's outcome to the bounded heartbeat
// history and, when a streak of failures is followed by a success, logs and
// records a recovery report (how many consecutive heartbeats were lost).
func (c *Client) recordHeartbeatOutcome(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	rec := HeartbeatRecord{At: time.Now(), OK: err == nil}
	if err != nil {
		rec.Err = err.Error()
	}
	c.heartbeatHistory = append(c.heartbeatHistory, rec)
	if len(c.heartbeatHistory) > heartbeatHistoryCap {
		c.heartbeatHistory = c.heartbeatHistory[len(c.heartbeatHistory)-heartbeatHistoryCap:]
	}

	if err != nil {
		c.consecutiveFailures++
		return
	}
	if c.consecutiveFailures > 0 {
		c.lastRecoveryFailures = c.consecutiveFailures
		c.logger.Warn("argus registry client: heartbeat recovered after a failure streak",
			"failed_heartbeats", c.consecutiveFailures)
		c.consecutiveFailures = 0
	}
}

// probeDependencies probes every declared dependency concurrently and
// returns their status for inclusion in the next heartbeat. A nil/empty
// Dependencies config is the common case and returns nil immediately. A
// dependency that cannot be reached is reported as such, never causes an
// error, and never blocks the heartbeat loop beyond
// dependencyProbeTimeout per dependency.
func (c *Client) probeDependencies(ctx context.Context) []*argusv1.DependencyStatus {
	deps := c.cfg.Dependencies
	if len(deps) == 0 {
		return nil
	}

	results := make([]*argusv1.DependencyStatus, len(deps))
	var wg sync.WaitGroup
	for i, dep := range deps {
		wg.Add(1)
		go func(i int, dep Dependency) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					results[i] = &argusv1.DependencyStatus{Name: dep.Name, Reachable: false}
				}
			}()
			results[i] = probeDependency(ctx, dep, c.dependencyProbeTimeout)
		}(i, dep)
	}
	wg.Wait()
	return results
}

// probeDependency probes a single dependency: an HTTP(S) GET expecting any
// non-5xx status for an "http://"/"https://" target, otherwise a plain TCP
// dial.
func probeDependency(ctx context.Context, dep Dependency, timeout time.Duration) *argusv1.DependencyStatus {
	pctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	start := time.Now()
	reachable := false

	if strings.HasPrefix(dep.Target, "http://") || strings.HasPrefix(dep.Target, "https://") {
		if req, err := http.NewRequestWithContext(pctx, http.MethodGet, dep.Target, nil); err == nil {
			if resp, err := http.DefaultClient.Do(req); err == nil {
				reachable = resp.StatusCode < 500
				_ = resp.Body.Close()
			}
		}
	} else {
		var d net.Dialer
		if conn, err := d.DialContext(pctx, "tcp", dep.Target); err == nil {
			reachable = true
			_ = conn.Close()
		}
	}

	return &argusv1.DependencyStatus{
		Name:      dep.Name,
		Reachable: reachable,
		LatencyMs: float64(time.Since(start).Microseconds()) / 1000.0,
	}
}

// Stop stops the heartbeat goroutine (if running) and makes a best-effort
// Deregister call. It never blocks on Argus being unreachable or on
// Deregister being unimplemented — both are logged, not treated as errors.
func (c *Client) Stop(ctx context.Context) error {
	if !c.enabled {
		return nil
	}

	c.stopOnce.Do(func() { close(c.stopCh) })

	c.mu.Lock()
	heartbeatStarted := c.heartbeatStarted
	registered := c.registered
	uuidStr := c.instanceUUID
	c.mu.Unlock()

	if heartbeatStarted {
		select {
		case <-c.doneCh:
		case <-ctx.Done():
		case <-time.After(2 * time.Second):
		}
	}

	if registered && c.rpc != nil {
		deregCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		_, err := c.rpc.Deregister(deregCtx, &argusv1.DeregisterRequest{
			InstanceUuid: uuidStr,
			Reason:       "sigterm",
		})
		cancel()
		if err != nil && status.Code(err) != codes.Unimplemented {
			c.logger.Warn("argus registry client: deregister failed (best-effort)", "error", err)
		}
	}

	if c.conn != nil {
		_ = c.conn.Close()
	}
	return nil
}

// retryWithBackoff calls fn until it succeeds, ctx is done, or maxAttempts
// is reached, sleeping an exponentially growing, jittered backoff between
// attempts. It returns the last error observed.
func retryWithBackoff(ctx context.Context, maxAttempts int, base time.Duration, fn func() error) error {
	if maxAttempts < 1 {
		maxAttempts = 1
	}

	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			if lastErr != nil {
				return lastErr
			}
			return err
		}

		lastErr = fn()
		if lastErr == nil {
			return nil
		}
		if attempt == maxAttempts-1 {
			break
		}

		wait := jitteredBackoff(base, attempt)
		select {
		case <-time.After(wait):
		case <-ctx.Done():
			return lastErr
		}
	}
	return fmt.Errorf("after %d attempts: %w", maxAttempts, lastErr)
}

// jitteredBackoff returns base*2^attempt plus up to base of random jitter,
// so a fleet of engines restarting together does not hammer Argus in
// lockstep.
func jitteredBackoff(base time.Duration, attempt int) time.Duration {
	backoff := base << attempt
	if backoff <= 0 || backoff > 30*time.Second {
		backoff = 30 * time.Second
	}
	jitter, err := rand.Int(rand.Reader, big.NewInt(int64(base)+1))
	if err != nil {
		return backoff
	}
	return backoff + time.Duration(jitter.Int64())
}
