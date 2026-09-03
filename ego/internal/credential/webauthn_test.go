package credential

import (
	"testing"

	"github.com/autorix/ego/internal/core"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
)

func TestNewWebAuthnManager(t *testing.T) {
	m, err := NewWebAuthnManager()
	if err != nil {
		t.Fatalf("NewWebAuthnManager: %v", err)
	}
	if m == nil {
		t.Fatal("expected manager, got nil")
	}

	id := uuid.New()
	identity := &core.Identity{
		ID:     id,
		Traits: map[string]interface{}{"email": "test@example.com"},
	}

	user := NewWebAuthnUser(identity, nil)
	if string(user.WebAuthnID()) != id.String() {
		t.Errorf("WebAuthnID = %s, want %s", string(user.WebAuthnID()), id.String())
	}
	if user.WebAuthnName() != "test@example.com" {
		t.Errorf("WebAuthnName = %s, want test@example.com", user.WebAuthnName())
	}

	creation, session, err := m.BeginRegistration(user)
	if err != nil {
		t.Fatalf("BeginRegistration: %v", err)
	}
	if creation == nil || session == nil {
		t.Fatal("expected non-nil creation options and session data")
	}
	if len(session.Challenge) == 0 {
		t.Fatal("expected non-empty challenge in session data")
	}
}

func TestParseStoredCredentials(t *testing.T) {
	mockCred := webauthn.Credential{
		ID:        []byte("test-cred-id"),
		PublicKey: []byte("test-public-key"),
	}

	raw := []map[string]interface{}{
		{
			"id":        mockCred.ID,
			"publicKey": mockCred.PublicKey,
		},
	}

	parsed := ParseStoredCredentials(raw)
	if len(parsed) != 1 {
		t.Fatalf("expected 1 parsed credential, got %d", len(parsed))
	}
}
