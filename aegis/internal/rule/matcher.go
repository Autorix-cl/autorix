package rule

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"

	"github.com/autorix/aegis/internal/core"
	"gopkg.in/yaml.v3"
)

// ErrInvalidPath is returned when a request path contains traversal, null bytes,
// backslashes, or forbidden percent-encoded path delimiters.
var ErrInvalidPath = errors.New("invalid request path")

// ValidateAndCanonicalizePath rejects path traversal / normalization bypasses
// and returns a canonicalized path starting with "/".
func ValidateAndCanonicalizePath(u *url.URL, requestURI string) (string, error) {
	if u == nil {
		return "", fmt.Errorf("%w: empty url", ErrInvalidPath)
	}

	candidates := []string{u.Path, u.RawPath}
	if strings.HasPrefix(requestURI, "/") {
		rawPathOnly, _, _ := strings.Cut(requestURI, "?")
		candidates = append(candidates, rawPathOnly)
	}

	for _, s := range candidates {
		if s == "" {
			continue
		}
		if strings.ContainsRune(s, '\x00') || strings.ContainsRune(s, '\\') {
			return "", fmt.Errorf("%w: forbidden character", ErrInvalidPath)
		}
		if strings.Contains(s, "..") {
			return "", fmt.Errorf("%w: path traversal not allowed", ErrInvalidPath)
		}
		lower := strings.ToLower(s)
		if strings.Contains(lower, "%2f") || strings.Contains(lower, "%5c") || strings.Contains(lower, "%2e") || strings.Contains(lower, "%00") {
			return "", fmt.Errorf("%w: encoded path delimiter not allowed", ErrInvalidPath)
		}
	}

	p := u.Path
	if p == "" {
		p = "/"
	}
	cleaned := path.Clean(p)
	if !strings.HasPrefix(cleaned, "/") {
		cleaned = "/" + cleaned
	}
	return cleaned, nil
}

// CompileURLPattern converts a rule's match.url template into a compiled regexp.
func CompileURLPattern(rawPattern string) (*regexp.Regexp, error) {
	pattern := rawPattern
	if !strings.HasPrefix(pattern, "^") {
		pattern = "^" + pattern
	}
	if !strings.HasSuffix(pattern, "$") {
		pattern = pattern + "$"
	}
	pattern = strings.ReplaceAll(pattern, "<", "(")
	pattern = strings.ReplaceAll(pattern, ">", ")")
	return regexp.Compile(pattern)
}

// ValidateRule validates a single rule's match fields, regex syntax, and upstream URL scheme.
func ValidateRule(r core.Rule) error {
	if r.Match.URL == "" {
		return errors.New("match.url is required")
	}
	if len(r.Match.Methods) == 0 {
		return errors.New("match.methods must have at least one entry")
	}
	if r.Upstream.URL == "" {
		return errors.New("upstream.url is required")
	}
	u, err := url.Parse(r.Upstream.URL)
	if err != nil {
		return fmt.Errorf("invalid upstream.url: %w", err)
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("upstream.url must use http or https scheme, got %q", u.Scheme)
	}
	if u.Host == "" {
		return errors.New("upstream.url must include a host")
	}
	if _, err := NewMatcher([]core.Rule{r}); err != nil {
		return err
	}
	return nil
}

// ValidateRules validates every rule in the set and verifies the full matcher compiles.
func ValidateRules(rules []core.Rule) error {
	for _, r := range rules {
		if err := ValidateRule(r); err != nil {
			return err
		}
	}
	if _, err := NewMatcher(rules); err != nil {
		return err
	}
	return nil
}

type compiledRule struct {
	rule    core.Rule
	urlReg  *regexp.Regexp
	methods map[string]bool
}

type Matcher struct {
	rules []compiledRule
}

func NewMatcherFromYAML(data []byte) (*Matcher, error) {
	var rawRules []core.Rule
	if err := yaml.Unmarshal(data, &rawRules); err != nil {
		return nil, fmt.Errorf("failed to parse rules YAML: %w", err)
	}

	return compileRules(rawRules)
}

// NewMatcher compiles an already-parsed rule set (used by Store when rules
// come from the admin API rather than a YAML file on disk).
func NewMatcher(rawRules []core.Rule) (*Matcher, error) {
	return compileRules(rawRules)
}

// compileRules builds the regex-matched lookup table shared by
// NewMatcherFromYAML and NewMatcher.
func compileRules(rawRules []core.Rule) (*Matcher, error) {
	matcher := &Matcher{}
	for _, r := range rawRules {
		reg, err := CompileURLPattern(r.Match.URL)
		if err != nil {
			return nil, fmt.Errorf("invalid regex for rule %s: %w", r.ID, err)
		}

		methods := make(map[string]bool)
		for _, m := range r.Match.Methods {
			methods[strings.ToUpper(m)] = true
		}

		matcher.rules = append(matcher.rules, compiledRule{
			rule:    r,
			urlReg:  reg,
			methods: methods,
		})
	}

	return matcher, nil
}

// Match finds the first matching rule for an incoming request after validating
// and canonicalizing the request path.
func (m *Matcher) Match(r *http.Request) (*core.Rule, error) {
	reqURL, err := ValidateAndCanonicalizePath(r.URL, r.RequestURI)
	if err != nil {
		return nil, err
	}
	reqMethod := strings.ToUpper(r.Method)

	for _, cr := range m.rules {
		// Check method
		if !cr.methods["*"] && !cr.methods[reqMethod] {
			continue
		}

		// Check URL
		if cr.urlReg.MatchString(reqURL) {
			ruleCopy := cr.rule
			return &ruleCopy, nil
		}
	}

	return nil, fmt.Errorf("no matching rule found for path %s [%s]", reqURL, reqMethod)
}
