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

func NewWebAuthnUser(identity *core.Identity, creds []webauthn.Credential) *WebAuthnUser {
	name := "user"
	if email, ok := identity.Traits["email"].(string); ok && email != "" {
		name = email
	}
	return &WebAuthnUser{
		id:          identity.ID,
		name:        name,
		displayName: name,
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

// NewWebAuthnManager creates a WebAuthn manager from environment or defaults.
func NewWebAuthnManager() (*Manager, error) {
	rpID := os.Getenv("WEBAUTHN_RP_ID")
	if rpID == "" {
		rpID = "localhost"
	}

	rpDisplayName := os.Getenv("WEBAUTHN_RP_DISPLAY_NAME")
	if rpDisplayName == "" {
		rpDisplayName = "Autorix Identity"
	}

	originsEnv := os.Getenv("WEBAUTHN_RP_ORIGINS")
	var origins []string
	if originsEnv != "" {
		origins = strings.Split(originsEnv, ",")
	} else {
		origins = []string{
			"http://localhost:3000",
			"http://localhost:4433",
			"https://localhost:3000",
			"https://localhost:4433",
		}
	}

	cfg := &webauthn.Config{
		RPDisplayName: rpDisplayName,
		RPID:          rpID,
		RPOrigins:     origins,
	}

	wa, err := webauthn.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("initialize webauthn: %w", err)
	}

	return &Manager{wa: wa}, nil
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
