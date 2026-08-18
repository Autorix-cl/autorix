package rule

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"

	"github.com/autorix/aegis/internal/core"
	"gopkg.in/yaml.v3"
)

var (
	// ErrRuleNotFound is returned by Get/Update/Delete when no rule matches
	// the given ID.
	ErrRuleNotFound = errors.New("rule not found")
	// ErrDuplicateID is returned by Create when the (explicit or generated)
	// rule ID collides with an existing one.
	ErrDuplicateID = errors.New("a rule with this id already exists")
)

// Store is a file-backed, hot-reloadable collection of Aegis routing rules.
// It is the single source of truth consumed by both the proxy pipeline
// (read-only, via Match) and the admin REST API (read/write). Every mutation
// recompiles the Matcher and persists the full rule set back to the YAML
// file before swapping in-memory state, so an invalid rule (e.g. bad regex)
// never corrupts the running pipeline or the on-disk file, and the proxy
// never observes a partially-applied change.
type Store struct {
	mu      sync.RWMutex
	path    string
	rules   []core.Rule
	matcher *Matcher
}

// NewStore loads the rule set from the YAML file at path and compiles it.
func NewStore(path string) (*Store, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read rules file %s: %w", path, err)
	}

	var rawRules []core.Rule
	if err := yaml.Unmarshal(data, &rawRules); err != nil {
		return nil, fmt.Errorf("failed to parse rules YAML: %w", err)
	}

	matcher, err := compileRules(rawRules)
	if err != nil {
		return nil, err
	}

	return &Store{path: path, rules: rawRules, matcher: matcher}, nil
}

// Match delegates to the current compiled Matcher. Safe for concurrent use
// alongside rule mutations — a write in progress never blocks a read for
// longer than the time it takes to swap a pointer.
func (s *Store) Match(r *http.Request) (*core.Rule, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.matcher.Match(r)
}

// TestMatch is a convenience for the console's rule simulator: it runs the
// same matching logic as Match without needing a real *http.Request.
func (s *Store) TestMatch(method, path string) (*core.Rule, error) {
	req := &http.Request{
		Method: strings.ToUpper(method),
		URL:    &url.URL{Path: path},
	}
	return s.Match(req)
}

// List returns a snapshot of the current rules, in file order.
func (s *Store) List() []core.Rule {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]core.Rule, len(s.rules))
	copy(out, s.rules)
	return out
}

// Get returns a single rule by ID.
func (s *Store) Get(id string) (core.Rule, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, r := range s.rules {
		if r.ID == id {
			return r, nil
		}
	}
	return core.Rule{}, ErrRuleNotFound
}

// Create appends a new rule. If rule.ID is empty, one is derived from its
// description (falling back to "rule"), disambiguated with a numeric suffix.
func (s *Store) Create(r core.Rule) (core.Rule, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if r.ID == "" {
		r.ID = s.generateIDLocked(r.Description)
	}
	for _, existing := range s.rules {
		if existing.ID == r.ID {
			return core.Rule{}, ErrDuplicateID
		}
	}

	next := append(append([]core.Rule{}, s.rules...), r)
	if err := s.applyLocked(next); err != nil {
		return core.Rule{}, err
	}
	return r, nil
}

// Update replaces the rule identified by id in place, preserving its
// position in the file.
func (s *Store) Update(id string, r core.Rule) (core.Rule, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := s.indexOfLocked(id)
	if idx == -1 {
		return core.Rule{}, ErrRuleNotFound
	}

	r.ID = id
	next := append([]core.Rule{}, s.rules...)
	next[idx] = r
	if err := s.applyLocked(next); err != nil {
		return core.Rule{}, err
	}
	return r, nil
}

// Delete removes the rule identified by id.
func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := s.indexOfLocked(id)
	if idx == -1 {
		return ErrRuleNotFound
	}

	next := append([]core.Rule{}, s.rules[:idx]...)
	next = append(next, s.rules[idx+1:]...)
	return s.applyLocked(next)
}

// indexOfLocked requires the caller to hold s.mu (read or write).
func (s *Store) indexOfLocked(id string) int {
	for i, r := range s.rules {
		if r.ID == id {
			return i
		}
	}
	return -1
}

// applyLocked recompiles the matcher and persists to disk before swapping
// state. Caller must hold s.mu for writing.
func (s *Store) applyLocked(rules []core.Rule) error {
	matcher, err := compileRules(rules)
	if err != nil {
		return fmt.Errorf("invalid rule set: %w", err)
	}

	data, err := yaml.Marshal(rules)
	if err != nil {
		return fmt.Errorf("failed to marshal rules: %w", err)
	}
	if err := os.WriteFile(s.path, data, 0o644); err != nil {
		return fmt.Errorf("failed to persist rules file: %w", err)
	}

	s.rules = rules
	s.matcher = matcher
	return nil
}

func (s *Store) generateIDLocked(seed string) string {
	base := slugify(seed)
	if base == "" {
		base = "rule"
	}

	id := base
	for n := 2; ; n++ {
		if s.indexOfLocked(id) == -1 {
			return id
		}
		id = fmt.Sprintf("%s-%d", base, n)
	}
}

// slugify lower-cases s and collapses any run of non [a-z0-9] characters
// into a single dash, trimming leading/trailing dashes.
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
