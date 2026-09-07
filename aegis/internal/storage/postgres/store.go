package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/autorix/aegis/internal/core"
	"github.com/autorix/aegis/internal/rule"
	"github.com/autorix/platform/paging"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"gopkg.in/yaml.v3"
)

// PostgresStore implements rule.Store backed by PostgreSQL with an in-memory cache and hot-reload.
type PostgresStore struct {
	pool    *pgxpool.Pool
	mu      sync.RWMutex
	rules   []core.Rule
	matcher *rule.Matcher

	listenerCancel context.CancelFunc
	listenerDone   chan struct{}
	listenerOnce   sync.Once
}

const rulesChangeChannel = "aegis_rules_changed"

// NewPostgresStore initializes a PostgresStore, loads existing rules from DB into memory, and compiles the Matcher.
func NewPostgresStore(ctx context.Context, pool *pgxpool.Pool) (*PostgresStore, error) {
	s := &PostgresStore{pool: pool}
	if err := s.reloadLocked(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize rules cache from postgres: %w", err)
	}
	s.startRuleChangeListener()
	return s, nil
}

const rulesBootstrapAdvisoryLock int64 = 0x414547495352554c // "AEGISRUL"

// NewPostgresStoreWithBootstrap initializes the database-backed rule store and,
// when the rules table is empty, seeds it exactly once from the operator-supplied
// YAML file. Existing database rules are never imported over or otherwise
// changed. The advisory lock makes startup safe when multiple replicas begin at
// the same time.
func NewPostgresStoreWithBootstrap(ctx context.Context, pool *pgxpool.Pool, rulesPath string) (*PostgresStore, error) {
	seedRules, err := loadBootstrapRules(rulesPath)
	if err != nil {
		return nil, err
	}
	if err := bootstrapRules(ctx, pool, seedRules); err != nil {
		return nil, err
	}
	return NewPostgresStore(ctx, pool)
}

func loadBootstrapRules(rulesPath string) ([]core.Rule, error) {
	if strings.TrimSpace(rulesPath) == "" {
		return nil, errors.New("rules bootstrap path is required for postgres storage")
	}
	data, err := os.ReadFile(rulesPath) // #nosec G304 -- operator-supplied startup configuration
	if err != nil {
		return nil, fmt.Errorf("read bootstrap rules file %s: %w", rulesPath, err)
	}
	var rules []core.Rule
	if err := yaml.Unmarshal(data, &rules); err != nil {
		return nil, fmt.Errorf("parse bootstrap rules YAML: %w", err)
	}
	if len(rules) == 0 {
		return nil, errors.New("bootstrap rules file must contain at least one rule")
	}
	if _, err := rule.NewMatcher(rules); err != nil {
		return nil, fmt.Errorf("validate bootstrap rules: %w", err)
	}
	return rules, nil
}

func bootstrapRules(ctx context.Context, pool *pgxpool.Pool, rules []core.Rule) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin rules bootstrap: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", rulesBootstrapAdvisoryLock); err != nil {
		return fmt.Errorf("lock rules bootstrap: %w", err)
	}
	var hasRules bool
	if err := tx.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM rules)").Scan(&hasRules); err != nil {
		return fmt.Errorf("check existing rules: %w", err)
	}
	if hasRules {
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit skipped rules bootstrap: %w", err)
		}
		return nil
	}

	for index, r := range rules {
		if r.ID == "" {
			return fmt.Errorf("bootstrap rule at index %d has no id", index)
		}
		// YAML file order is the evaluation order. Zero is the conventional
		// unspecified order value, so normalize it to its file position.
		if r.OrderIdx == 0 {
			r.OrderIdx = index
		}
		if err := insertRuleTx(ctx, tx, r); err != nil {
			return fmt.Errorf("insert bootstrap rule %s: %w", r.ID, err)
		}
	}
	if err := recordVersionSnapshotTx(ctx, tx, "Bootstrapped default rules"); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit rules bootstrap: %w", err)
	}
	return nil
}

func insertRuleTx(ctx context.Context, tx pgx.Tx, r core.Rule) error {
	matchJSON, err := json.Marshal(r.Match)
	if err != nil {
		return fmt.Errorf("marshal match: %w", err)
	}
	authsJSON, err := json.Marshal(r.Authenticators)
	if err != nil {
		return fmt.Errorf("marshal authenticators: %w", err)
	}
	authzJSON, err := json.Marshal(r.Authorizer)
	if err != nil {
		return fmt.Errorf("marshal authorizer: %w", err)
	}
	mutsJSON, err := json.Marshal(r.Mutators)
	if err != nil {
		return fmt.Errorf("marshal mutators: %w", err)
	}
	upstreamJSON, err := json.Marshal(r.Upstream)
	if err != nil {
		return fmt.Errorf("marshal upstream: %w", err)
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO rules (id, description, order_idx, match, authenticators, authorizer, mutators, upstream, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
	`, r.ID, r.Description, r.OrderIdx, matchJSON, authsJSON, authzJSON, mutsJSON, upstreamJSON)
	return err
}

// Close stops the background listener and waits for its database connection to
// be released. It is safe to call more than once.
func (s *PostgresStore) Close() {
	s.listenerOnce.Do(func() {})
	if s.listenerCancel != nil {
		s.listenerCancel()
	}
	if s.listenerDone != nil {
		<-s.listenerDone
	}
}

func (s *PostgresStore) startRuleChangeListener() {
	ctx, cancel := context.WithCancel(context.Background())
	s.listenerCancel = cancel
	s.listenerDone = make(chan struct{})
	go func() {
		defer close(s.listenerDone)
		s.listenForRuleChanges(ctx)
	}()
}

// listenForRuleChanges owns a dedicated pooled connection because PostgreSQL
// LISTEN state is session-scoped. On every reconnect it reloads the cache,
// closing the gap for notifications that may have been missed while the
// database connection was unavailable.
func (s *PostgresStore) listenForRuleChanges(ctx context.Context) {
	backoff := 50 * time.Millisecond
	for ctx.Err() == nil {
		err := s.listenForRuleChangesOnce(ctx)
		if ctx.Err() != nil {
			return
		}
		_ = err // reconnecting is the resilient path; callers keep serving cached rules.
		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
		if backoff < time.Second {
			backoff *= 2
		}
	}
}

func (s *PostgresStore) listenForRuleChangesOnce(ctx context.Context) error {
	conn, err := s.pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire rules listener connection: %w", err)
	}
	defer conn.Release()

	if _, err := conn.Exec(ctx, "LISTEN "+rulesChangeChannel); err != nil {
		return fmt.Errorf("listen for rules changes: %w", err)
	}
	if err := s.Reload(ctx); err != nil {
		return fmt.Errorf("reload rules after listener reconnect: %w", err)
	}
	for {
		notification, err := conn.Conn().WaitForNotification(ctx)
		if err != nil {
			return fmt.Errorf("wait for rules notification: %w", err)
		}
		if notification.Channel != rulesChangeChannel {
			continue
		}
		if err := s.Reload(ctx); err != nil {
			return fmt.Errorf("reload rules after notification: %w", err)
		}
	}
}

// Pool exposes the underlying connection pool for health checks and diagnostics.
func (s *PostgresStore) Pool() *pgxpool.Pool {
	return s.pool
}

// Match evaluates incoming requests against the cached compiled Matcher with zero DB latency.
func (s *PostgresStore) Match(r *http.Request) (*core.Rule, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.matcher == nil {
		return nil, errors.New("no compiled matcher available")
	}
	return s.matcher.Match(r)
}

// TestMatch simulates rule matching for a given method and path without allocating a full request.
func (s *PostgresStore) TestMatch(method, path string) (*core.Rule, error) {
	req := &http.Request{
		Method: strings.ToUpper(method),
		URL:    &url.URL{Path: path},
	}
	return s.Match(req)
}

// List returns a snapshot copy of the current in-memory rule set.
func (s *PostgresStore) List() []core.Rule {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]core.Rule, len(s.rules))
	copy(out, s.rules)
	return out
}

// ListPage returns a paginated slice of rules starting after cursor in stable order.
func (s *PostgresStore) ListPage(limit int, cursor string) ([]core.Rule, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	start := 0
	if cursor != "" {
		id, err := paging.DecodeCursor(cursor)
		if err != nil {
			return nil, false, fmt.Errorf("invalid cursor: %w", err)
		}
		idx := -1
		for i, r := range s.rules {
			if r.ID == id {
				idx = i
				break
			}
		}
		if idx == -1 {
			return nil, false, fmt.Errorf("invalid cursor: %w", rule.ErrRuleNotFound)
		}
		start = idx + 1
	}

	if start > len(s.rules) {
		start = len(s.rules)
	}
	end := start + limit
	hasMore := end < len(s.rules)
	if end > len(s.rules) {
		end = len(s.rules)
	}

	out := make([]core.Rule, end-start)
	copy(out, s.rules[start:end])
	return out, hasMore, nil
}

// Get returns a single rule by ID from the in-memory cache.
func (s *PostgresStore) Get(id string) (core.Rule, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, r := range s.rules {
		if r.ID == id {
			return r, nil
		}
	}
	return core.Rule{}, rule.ErrRuleNotFound
}

// Create persists a new rule to Postgres, takes a version snapshot, and hot-reloads the in-memory matcher.
func (s *PostgresStore) Create(r core.Rule) (core.Rule, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := context.Background()

	if r.ID == "" {
		r.ID = s.generateIDLocked(r.Description)
	}

	// Check collision in memory
	for _, existing := range s.rules {
		if existing.ID == r.ID {
			return core.Rule{}, rule.ErrDuplicateID
		}
	}

	matchJSON, err := json.Marshal(r.Match)
	if err != nil {
		return core.Rule{}, fmt.Errorf("marshal match: %w", err)
	}
	authsJSON, err := json.Marshal(r.Authenticators)
	if err != nil {
		return core.Rule{}, fmt.Errorf("marshal authenticators: %w", err)
	}
	authzJSON, err := json.Marshal(r.Authorizer)
	if err != nil {
		return core.Rule{}, fmt.Errorf("marshal authorizer: %w", err)
	}
	mutsJSON, err := json.Marshal(r.Mutators)
	if err != nil {
		return core.Rule{}, fmt.Errorf("marshal mutators: %w", err)
	}
	upstreamJSON, err := json.Marshal(r.Upstream)
	if err != nil {
		return core.Rule{}, fmt.Errorf("marshal upstream: %w", err)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return core.Rule{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if r.OrderIdx == 0 {
		var maxOrder int
		_ = tx.QueryRow(ctx, "SELECT COALESCE(MAX(order_idx), -1) + 1 FROM rules").Scan(&maxOrder)
		r.OrderIdx = maxOrder
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO rules (id, description, order_idx, match, authenticators, authorizer, mutators, upstream, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
	`, r.ID, r.Description, r.OrderIdx, matchJSON, authsJSON, authzJSON, mutsJSON, upstreamJSON)
	if err != nil {
		return core.Rule{}, fmt.Errorf("insert rule: %w", err)
	}

	if err := s.recordVersionSnapshotTx(ctx, tx, fmt.Sprintf("Created rule %s", r.ID)); err != nil {
		return core.Rule{}, err
	}
	if err := notifyRulesChangedTx(ctx, tx); err != nil {
		return core.Rule{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return core.Rule{}, fmt.Errorf("commit tx: %w", err)
	}

	if err := s.reloadLocked(ctx); err != nil {
		return core.Rule{}, fmt.Errorf("reload rules: %w", err)
	}

	return r, nil
}

// Update modifies an existing rule in Postgres, saves a version snapshot, and hot-reloads the cache.
func (s *PostgresStore) Update(id string, r core.Rule) (core.Rule, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := context.Background()
	r.ID = id

	matchJSON, err := json.Marshal(r.Match)
	if err != nil {
		return core.Rule{}, fmt.Errorf("marshal match: %w", err)
	}
	authsJSON, err := json.Marshal(r.Authenticators)
	if err != nil {
		return core.Rule{}, fmt.Errorf("marshal authenticators: %w", err)
	}
	authzJSON, err := json.Marshal(r.Authorizer)
	if err != nil {
		return core.Rule{}, fmt.Errorf("marshal authorizer: %w", err)
	}
	mutsJSON, err := json.Marshal(r.Mutators)
	if err != nil {
		return core.Rule{}, fmt.Errorf("marshal mutators: %w", err)
	}
	upstreamJSON, err := json.Marshal(r.Upstream)
	if err != nil {
		return core.Rule{}, fmt.Errorf("marshal upstream: %w", err)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return core.Rule{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx, `
		UPDATE rules
		SET description = $2, match = $3, authenticators = $4, authorizer = $5, mutators = $6, upstream = $7, updated_at = NOW()
		WHERE id = $1
	`, id, r.Description, matchJSON, authsJSON, authzJSON, mutsJSON, upstreamJSON)
	if err != nil {
		return core.Rule{}, fmt.Errorf("update rule: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return core.Rule{}, rule.ErrRuleNotFound
	}

	if err := s.recordVersionSnapshotTx(ctx, tx, fmt.Sprintf("Updated rule %s", id)); err != nil {
		return core.Rule{}, err
	}
	if err := notifyRulesChangedTx(ctx, tx); err != nil {
		return core.Rule{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return core.Rule{}, fmt.Errorf("commit tx: %w", err)
	}

	if err := s.reloadLocked(ctx); err != nil {
		return core.Rule{}, fmt.Errorf("reload rules: %w", err)
	}

	return r, nil
}

// Delete removes a rule from Postgres, saves a version snapshot, and hot-reloads the cache.
func (s *PostgresStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := context.Background()

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx, "DELETE FROM rules WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete rule: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return rule.ErrRuleNotFound
	}

	if err := s.recordVersionSnapshotTx(ctx, tx, fmt.Sprintf("Deleted rule %s", id)); err != nil {
		return err
	}
	if err := notifyRulesChangedTx(ctx, tx); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return s.reloadLocked(ctx)
}

// Reorder updates the order_idx for rules in Postgres, saves a version snapshot, and hot-reloads.
func (s *PostgresStore) Reorder(ids []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := context.Background()

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	for idx, id := range ids {
		_, err := tx.Exec(ctx, "UPDATE rules SET order_idx = $1, updated_at = NOW() WHERE id = $2", idx, id)
		if err != nil {
			return fmt.Errorf("reorder rule %s: %w", id, err)
		}
	}

	if err := s.recordVersionSnapshotTx(ctx, tx, "Reordered rules"); err != nil {
		return err
	}
	if err := notifyRulesChangedTx(ctx, tx); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return s.reloadLocked(ctx)
}

// Rollback restores a specific rule version snapshot into the active rules table.
func (s *PostgresStore) Rollback(version int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := context.Background()

	var snapshotJSON []byte
	err := s.pool.QueryRow(ctx, "SELECT rules_snapshot FROM rule_versions WHERE version = $1", version).Scan(&snapshotJSON)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("version %d not found: %w", version, rule.ErrRuleNotFound)
		}
		return fmt.Errorf("query rule version: %w", err)
	}

	var snapshotRules []core.Rule
	if err := json.Unmarshal(snapshotJSON, &snapshotRules); err != nil {
		return fmt.Errorf("unmarshal snapshot rules: %w", err)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, "DELETE FROM rules"); err != nil {
		return fmt.Errorf("clear rules: %w", err)
	}

	for idx, r := range snapshotRules {
		order := r.OrderIdx
		if order == 0 {
			order = idx
		}
		matchJSON, _ := json.Marshal(r.Match)
		authsJSON, _ := json.Marshal(r.Authenticators)
		authzJSON, _ := json.Marshal(r.Authorizer)
		mutsJSON, _ := json.Marshal(r.Mutators)
		upstreamJSON, _ := json.Marshal(r.Upstream)

		_, err := tx.Exec(ctx, `
			INSERT INTO rules (id, description, order_idx, match, authenticators, authorizer, mutators, upstream, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
		`, r.ID, r.Description, order, matchJSON, authsJSON, authzJSON, mutsJSON, upstreamJSON)
		if err != nil {
			return fmt.Errorf("insert rule on rollback: %w", err)
		}
	}

	if err := s.recordVersionSnapshotTx(ctx, tx, fmt.Sprintf("Rollback to version %d", version)); err != nil {
		return err
	}
	if err := notifyRulesChangedTx(ctx, tx); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return s.reloadLocked(ctx)
}

// GetVersions returns all historical rule version snapshots.
func (s *PostgresStore) GetVersions() ([]core.RuleVersion, error) {
	ctx := context.Background()
	rows, err := s.pool.Query(ctx, "SELECT version, description, rules_snapshot, created_at FROM rule_versions ORDER BY version DESC")
	if err != nil {
		return nil, fmt.Errorf("query rule_versions: %w", err)
	}
	defer rows.Close()

	var versions []core.RuleVersion
	for rows.Next() {
		var v core.RuleVersion
		var snapshotJSON []byte
		if err := rows.Scan(&v.Version, &v.Description, &snapshotJSON, &v.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan rule_version: %w", err)
		}
		if len(snapshotJSON) > 0 {
			_ = json.Unmarshal(snapshotJSON, &v.Rules)
		}
		versions = append(versions, v)
	}

	return versions, nil
}

// Import replaces all rules with the provided list in Postgres.
func (s *PostgresStore) Import(rules []core.Rule) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := context.Background()

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, "DELETE FROM rules"); err != nil {
		return fmt.Errorf("clear rules: %w", err)
	}

	for idx, r := range rules {
		matchJSON, _ := json.Marshal(r.Match)
		authsJSON, _ := json.Marshal(r.Authenticators)
		authzJSON, _ := json.Marshal(r.Authorizer)
		mutsJSON, _ := json.Marshal(r.Mutators)
		upstreamJSON, _ := json.Marshal(r.Upstream)

		order := r.OrderIdx
		if order == 0 {
			order = idx
		}

		_, err := tx.Exec(ctx, `
			INSERT INTO rules (id, description, order_idx, match, authenticators, authorizer, mutators, upstream, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
		`, r.ID, r.Description, order, matchJSON, authsJSON, authzJSON, mutsJSON, upstreamJSON)
		if err != nil {
			return fmt.Errorf("insert rule during import: %w", err)
		}
	}

	if err := s.recordVersionSnapshotTx(ctx, tx, "Imported rules"); err != nil {
		return err
	}
	if err := notifyRulesChangedTx(ctx, tx); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return s.reloadLocked(ctx)
}

// Export returns the current list of rules in evaluation order.
func (s *PostgresStore) Export() []core.Rule {
	return s.List()
}

// Reload forces a fresh reload from Postgres into the in-memory cache and recompiles the matcher.
func (s *PostgresStore) Reload(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.reloadLocked(ctx)
}

func (s *PostgresStore) reloadLocked(ctx context.Context) error {
	rows, err := s.pool.Query(ctx, `
		SELECT id, description, order_idx, match, authenticators, authorizer, mutators, upstream
		FROM rules
		ORDER BY order_idx ASC, created_at ASC, id ASC
	`)
	if err != nil {
		return fmt.Errorf("query rules: %w", err)
	}
	defer rows.Close()

	var rules []core.Rule
	for rows.Next() {
		var r core.Rule
		var matchJSON, authsJSON, authzJSON, mutsJSON, upstreamJSON []byte
		if err := rows.Scan(&r.ID, &r.Description, &r.OrderIdx, &matchJSON, &authsJSON, &authzJSON, &mutsJSON, &upstreamJSON); err != nil {
			return fmt.Errorf("scan rule: %w", err)
		}

		if len(matchJSON) > 0 {
			_ = json.Unmarshal(matchJSON, &r.Match)
		}
		if len(authsJSON) > 0 {
			_ = json.Unmarshal(authsJSON, &r.Authenticators)
		}
		if len(authzJSON) > 0 {
			_ = json.Unmarshal(authzJSON, &r.Authorizer)
		}
		if len(mutsJSON) > 0 {
			_ = json.Unmarshal(mutsJSON, &r.Mutators)
		}
		if len(upstreamJSON) > 0 {
			_ = json.Unmarshal(upstreamJSON, &r.Upstream)
		}

		rules = append(rules, r)
	}

	matcher, err := rule.NewMatcher(rules)
	if err != nil {
		return fmt.Errorf("compile rules matcher: %w", err)
	}

	s.rules = rules
	s.matcher = matcher
	return nil
}

func notifyRulesChangedTx(ctx context.Context, tx pgx.Tx) error {
	if _, err := tx.Exec(ctx, "SELECT pg_notify($1, '')", rulesChangeChannel); err != nil {
		return fmt.Errorf("notify rules change: %w", err)
	}
	return nil
}

func (s *PostgresStore) recordVersionSnapshotTx(ctx context.Context, tx pgx.Tx, description string) error {
	return recordVersionSnapshotTx(ctx, tx, description)
}

func recordVersionSnapshotTx(ctx context.Context, tx pgx.Tx, description string) error {
	rows, err := tx.Query(ctx, `
		SELECT id, description, order_idx, match, authenticators, authorizer, mutators, upstream
		FROM rules
		ORDER BY order_idx ASC, created_at ASC, id ASC
	`)
	if err != nil {
		return fmt.Errorf("query rules for snapshot: %w", err)
	}
	defer rows.Close()

	var rules []core.Rule
	for rows.Next() {
		var r core.Rule
		var matchJSON, authsJSON, authzJSON, mutsJSON, upstreamJSON []byte
		if err := rows.Scan(&r.ID, &r.Description, &r.OrderIdx, &matchJSON, &authsJSON, &authzJSON, &mutsJSON, &upstreamJSON); err != nil {
			return fmt.Errorf("scan rule for snapshot: %w", err)
		}
		_ = json.Unmarshal(matchJSON, &r.Match)
		_ = json.Unmarshal(authsJSON, &r.Authenticators)
		_ = json.Unmarshal(authzJSON, &r.Authorizer)
		_ = json.Unmarshal(mutsJSON, &r.Mutators)
		_ = json.Unmarshal(upstreamJSON, &r.Upstream)
		rules = append(rules, r)
	}

	snapshotJSON, err := json.Marshal(rules)
	if err != nil {
		return fmt.Errorf("marshal snapshot rules: %w", err)
	}

	_, err = tx.Exec(ctx, "INSERT INTO rule_versions (description, rules_snapshot) VALUES ($1, $2)", description, snapshotJSON)
	if err != nil {
		return fmt.Errorf("insert rule_version: %w", err)
	}

	return nil
}

func (s *PostgresStore) generateIDLocked(seed string) string {
	base := slugify(seed)
	if base == "" {
		base = "rule"
	}

	id := base
	for n := 2; ; n++ {
		found := false
		for _, r := range s.rules {
			if r.ID == id {
				found = true
				break
			}
		}
		if !found {
			return id
		}
		id = fmt.Sprintf("%s-%d", base, n)
	}
}

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	lastDash := false
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z' || r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		default:
			if !lastDash && b.Len() > 0 {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}
