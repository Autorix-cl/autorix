package mutator

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"strings"
	"text/template"

	"github.com/autorix/aegis/internal/core"
)

// IdentitySignatureHeader carries the HMAC-SHA256 signature computed over
// proxy-injected identity headers (X-User-ID, X-User-Email, X-User-Roles, X-User-Scopes).
const IdentitySignatureHeader = "X-Autorix-Identity-Signature"

// SignIdentityHeaders computes the canonical v1 HMAC-SHA256 signature over the
// four standard Autorix identity headers.
func SignIdentityHeaders(userID, email, roles, scopes, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = fmt.Fprintf(mac, "v1\n%s\n%s\n%s\n%s", userID, email, roles, scopes)
	return "v1=" + hex.EncodeToString(mac.Sum(nil))
}

func resolveSigningSecret(config map[string]interface{}) string {
	if config != nil {
		if raw, ok := config["signing_secret"].(string); ok && strings.TrimSpace(raw) != "" {
			return strings.TrimSpace(raw)
		}
	}
	if secret := strings.TrimSpace(os.Getenv("AEGIS_IDENTITY_SIGNING_SECRET")); secret != "" {
		return secret
	}
	return strings.TrimSpace(os.Getenv("AUTORIX_IDENTITY_SIGNING_SECRET"))
}

type HeaderMutator struct{}

func NewHeaderMutator() *HeaderMutator {
	return &HeaderMutator{}
}

func (m *HeaderMutator) Name() string {
	return "header"
}

func (m *HeaderMutator) Mutate(r *http.Request, session *core.Session, config map[string]interface{}) error {
	// Always strip any client-supplied identity signature before mutating.
	r.Header.Del(IdentitySignatureHeader)

	headersConfig, ok := config["headers"].(map[string]interface{})
	if !ok {
		// Default: inject standard X-User-ID header
		r.Header.Set("X-User-ID", session.Subject)
		if len(session.Scopes) > 0 {
			r.Header.Set("X-User-Scopes", strings.Join(session.Scopes, " "))
		} else {
			r.Header.Del("X-User-Scopes")
		}
		// Zero Trust: strip raw bearer tokens
		r.Header.Del("Authorization")
		m.attachSignature(r, config)
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
	m.attachSignature(r, config)

	return nil
}

func (m *HeaderMutator) attachSignature(r *http.Request, config map[string]interface{}) {
	secret := resolveSigningSecret(config)
	if secret == "" {
		return
	}
	sig := SignIdentityHeaders(
		r.Header.Get("X-User-ID"),
		r.Header.Get("X-User-Email"),
		r.Header.Get("X-User-Roles"),
		r.Header.Get("X-User-Scopes"),
		secret,
	)
	r.Header.Set(IdentitySignatureHeader, sig)
}

type NoopMutator struct{}

func (m *NoopMutator) Name() string { return "noop" }
func (m *NoopMutator) Mutate(r *http.Request, session *core.Session, config map[string]interface{}) error {
	return nil
}
