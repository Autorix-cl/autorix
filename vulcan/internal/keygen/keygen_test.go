package keygen

import (
	"strings"
	"testing"
)

func TestGenerateAPIKey(t *testing.T) {
	// 1. Live Key
	liveKey, err := GenerateAPIKey(true)
	if err != nil {
		t.Fatalf("GenerateAPIKey(true) failed: %v", err)
	}

	if !strings.HasPrefix(liveKey.RawToken, "av_live_") {
		t.Errorf("expected prefix 'av_live_', got %s", liveKey.RawToken)
	}

	if len(liveKey.KeyHash) != 64 { // SHA-256 hex is 64 chars
		t.Errorf("expected 64-char key hash, got %d", len(liveKey.KeyHash))
	}

	if len(liveKey.Hint) != 4 {
		t.Errorf("expected 4-char hint, got %d", len(liveKey.Hint))
	}

	// 2. Test Key
	testKey, err := GenerateAPIKey(false)
	if err != nil {
		t.Fatalf("GenerateAPIKey(false) failed: %v", err)
	}

	if !strings.HasPrefix(testKey.RawToken, "av_test_") {
		t.Errorf("expected prefix 'av_test_', got %s", testKey.RawToken)
	}
}
