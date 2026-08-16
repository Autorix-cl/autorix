package session

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestSessionManager(t *testing.T) {
	sm := NewManager(1 * time.Hour)
	identityID := uuid.New()

	sess, err := sm.GenerateSession(identityID)
	if err != nil {
		t.Fatalf("GenerateSession failed: %v", err)
	}

	if sess.IdentityID != identityID {
		t.Errorf("expected identityID %s, got %s", identityID, sess.IdentityID)
	}

	if len(sess.Token) == 0 {
		t.Error("expected non-empty raw token")
	}

	if len(sess.TokenHash) == 0 {
		t.Error("expected non-empty token hash")
	}

	if !sess.IsActive() {
		t.Error("expected session to be active")
	}

	// Verify HashToken deterministic behavior
	computedHash := HashToken(sess.Token)
	if computedHash != sess.TokenHash {
		t.Errorf("expected computed hash %s to match %s", computedHash, sess.TokenHash)
	}
}
