package mutator

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
	"text/template"

	"github.com/autorix/aegis/internal/core"
)

type HeaderMutator struct{}

func NewHeaderMutator() *HeaderMutator {
	return &HeaderMutator{}
}

func (m *HeaderMutator) Name() string {
	return "header"
}

func (m *HeaderMutator) Mutate(r *http.Request, session *core.Session, config map[string]interface{}) error {
	headersConfig, ok := config["headers"].(map[string]interface{})
	if !ok {
		// Default: inject standard X-User-ID header
		r.Header.Set("X-User-ID", session.Subject)
		if len(session.Scopes) > 0 {
			r.Header.Set("X-User-Scopes", strings.Join(session.Scopes, " "))
		}
		return nil
	}

	// Render templates for each configured header
	for headerKey, templateStr := range headersConfig {
		tmplStr, ok := templateStr.(string)
		if !ok {
			continue
		}

		tmpl, err := template.New(headerKey).Parse(tmplStr)
		if err != nil {
			return fmt.Errorf("invalid template for header %s: %w", headerKey, err)
		}

		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, session); err != nil {
			return fmt.Errorf("failed to render header %s: %w", headerKey, err)
		}

		r.Header.Set(headerKey, buf.String())
	}

	// Zero Trust: strip raw bearer tokens unless explicitly configured to keep them
	r.Header.Del("Authorization")

	return nil
}

type NoopMutator struct{}

func (m *NoopMutator) Name() string { return "noop" }
func (m *NoopMutator) Mutate(r *http.Request, session *core.Session, config map[string]interface{}) error {
	return nil
}
