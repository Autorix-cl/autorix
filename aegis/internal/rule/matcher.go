package rule

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/autorix/aegis/internal/core"
	"gopkg.in/yaml.v3"
)

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

	matcher := &Matcher{}
	for _, r := range rawRules {
		// Convert URL template like <.*> or <[0-9]+> to valid regex
		pattern := r.Match.URL
		if !strings.HasPrefix(pattern, "^") {
			pattern = "^" + pattern
		}
		if !strings.HasSuffix(pattern, "$") {
			pattern = pattern + "$"
		}

		// Replace <pattern> with (pattern) for regex matching
		pattern = strings.ReplaceAll(pattern, "<", "(")
		pattern = strings.ReplaceAll(pattern, ">", ")")

		reg, err := regexp.Compile(pattern)
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

// Match finds the first matching rule for an incoming request
func (m *Matcher) Match(r *http.Request) (*core.Rule, error) {
	reqURL := r.URL.Path
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
