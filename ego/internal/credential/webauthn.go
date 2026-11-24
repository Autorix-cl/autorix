package credential

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/autorix/ego/internal/core"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
)

// WebAuthnUser implements webauthn.User interface for an Ego identity.
type WebAuthnUser struct {
	id          uuid.UUID
	name        string
	displayName string
	credentials []webauthn.Credential
}

// Compile-time check that WebAuthnUser satisfies webauthn.User.
var _ webauthn.User = (*WebAuthnUser)(nil)

func NewWebAuthnUser(identity *core.Identity, creds []webauthn.Credential) *WebAuthnUser {
	name := "user"
	if identity != nil && identity.Traits != nil {
		if email, ok := identity.Traits["email"].(string); ok && email != "" {
			name = email
		}
	}
	displayName := name
	if identity != nil && identity.Traits != nil {
		switch v := identity.Traits["name"].(type) {
		case string:
			if trimmed := strings.TrimSpace(v); trimmed != "" {
				displayName = trimmed
			}
		case map[string]interface{}:
			first, _ := v["first"].(string)
			last, _ := v["last"].(string)
			full := strings.TrimSpace(strings.TrimSpace(first) + " " + strings.TrimSpace(last))
			if full != "" {
				displayName = full
			}
		}
	}
	var id uuid.UUID
	if identity != nil {
		id = identity.ID
	}
	return &WebAuthnUser{
		id:          id,
		name:        name,
		displayName: displayName,
		credentials: creds,
	}
}

func (u *WebAuthnUser) WebAuthnID() []byte {
	return []byte(u.id.String())
}

func (u *WebAuthnUser) WebAuthnName() string {
	return u.name
}

func (u *WebAuthnUser) WebAuthnDisplayName() string {
	return u.displayName
}

func (u *WebAuthnUser) WebAuthnIcon() string {
	return ""
}

func (u *WebAuthnUser) WebAuthnCredentials() []webauthn.Credential {
	return u.credentials
}

// Manager wraps the webauthn instance.
type Manager struct {
	wa *webauthn.WebAuthn
}

// NewWebAuthnConfigFromEnv builds a WebAuthn Relying Party configuration from
// environment variables with safe local-development defaults.
func NewWebAuthnConfigFromEnv() *webauthn.Config {
	rpID := strings.TrimSpace(os.Getenv("WEBAUTHN_RP_ID"))
	if rpID == "" {
		rpID = "localhost"
	}

	rpDisplayName := strings.TrimSpace(os.Getenv("WEBAUTHN_RP_DISPLAY_NAME"))
	if rpDisplayName == "" {
		rpDisplayName = "Autorix"
	}

	originsEnv := strings.TrimSpace(os.Getenv("WEBAUTHN_RP_ORIGINS"))
	var origins []string
	if originsEnv != "" {
		for _, raw := range strings.Split(originsEnv, ",") {
			if trimmed := strings.TrimSpace(raw); trimmed != "" {
				origins = append(origins, trimmed)
			}
		}
	}
	if len(origins) == 0 {
		origins = []string{
			"http://localhost:3000",
			"http://localhost:4433",
		}
	}

	return &webauthn.Config{
		RPDisplayName: rpDisplayName,
		RPID:          rpID,
		RPOrigins:     origins,
	}
}

// NewWebAuthnFromEnv initializes a *webauthn.WebAuthn using environment variables.
func NewWebAuthnFromEnv() (*webauthn.WebAuthn, error) {
	wa, err := webauthn.New(NewWebAuthnConfigFromEnv())
	if err != nil {
		return nil, fmt.Errorf("initialize webauthn: %w", err)
	}
	return wa, nil
}

// NewWebAuthnManager creates a WebAuthn manager from environment or defaults.
func NewWebAuthnManager() (*Manager, error) {
	wa, err := NewWebAuthnFromEnv()
	if err != nil {
		return nil, err
	}
	return &Manager{wa: wa}, nil
}

// NewWebAuthnManagerWithInstance wraps a pre-configured *webauthn.WebAuthn instance.
func NewWebAuthnManagerWithInstance(wa *webauthn.WebAuthn) *Manager {
	return &Manager{wa: wa}
}

// BeginRegistration generates credential creation options.
func (m *Manager) BeginRegistration(user *WebAuthnUser) (*protocol.CredentialCreation, *webauthn.SessionData, error) {
	return m.wa.BeginRegistration(user)
}

// FinishRegistration verifies the attestation response.
func (m *Manager) FinishRegistration(user *WebAuthnUser, session webauthn.SessionData, r *http.Request) (*webauthn.Credential, error) {
	return m.wa.FinishRegistration(user, session, r)
}

// BeginLogin generates credential assertion options.
func (m *Manager) BeginLogin(user *WebAuthnUser) (*protocol.CredentialAssertion, *webauthn.SessionData, error) {
	return m.wa.BeginLogin(user)
}

// FinishLogin verifies the assertion response.
func (m *Manager) FinishLogin(user *WebAuthnUser, session webauthn.SessionData, r *http.Request) (*webauthn.Credential, error) {
	return m.wa.FinishLogin(user, session, r)
}

// ParseStoredCredentials converts raw JSON database records to webauthn.Credential structs.
func ParseStoredCredentials(rawCreds []map[string]interface{}) []webauthn.Credential {
	var result []webauthn.Credential
	for _, raw := range rawCreds {
		data, err := json.Marshal(raw)
		if err != nil {
			continue
		}
		var cred webauthn.Credential
		if err := json.Unmarshal(data, &cred); err == nil {
			result = append(result, cred)
		}
	}
	return result
}
