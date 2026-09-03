package zookie

import (
	"testing"
	"time"
)

func TestZookie_RoundtripSerialization(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Nanosecond)
	token := New(now)

	serialized := token.String()
	if len(serialized) == 0 {
		t.Fatalf("expected non-empty serialized token")
	}

	parsed, err := Parse(serialized)
	if err != nil {
		t.Fatalf("failed to parse serialized token %s: %v", serialized, err)
	}

	if parsed.Timestamp.UnixNano() != now.UnixNano() {
		t.Errorf("timestamp mismatch: got %v, want %v", parsed.Timestamp, now)
	}
	if parsed.Version != "v1" {
		t.Errorf("version mismatch: got %s, want v1", parsed.Version)
	}
}

func TestZookie_IsAtLeastFresh(t *testing.T) {
	t0 := time.Now().UTC()
	t1 := t0.Add(100 * time.Millisecond)
	tPast := t0.Add(-100 * time.Millisecond)

	zk := New(t0)

	// A candidate timestamp newer than zk is fresh
	if !zk.IsAtLeastFresh(t1) {
		t.Errorf("expected t1 (%v) to be fresh relative to zk (%v)", t1, t0)
	}

	// An equal timestamp is fresh
	if !zk.IsAtLeastFresh(t0) {
		t.Errorf("expected t0 (%v) to be fresh relative to zk (%v)", t0, t0)
	}

	// An older candidate is stale
	if zk.IsAtLeastFresh(tPast) {
		t.Errorf("expected tPast (%v) to be stale relative to zk (%v)", tPast, t0)
	}
}

func TestZookie_ParseInvalid(t *testing.T) {
	invalidTokens := []string{
		"",
		"not_a_token",
		"zk_invalid",
		"zk_v1_not_base64!!!",
		"zk_v1_bm90X2FuX2ludA", // "not_an_int" base64
	}

	for _, raw := range invalidTokens {
		_, err := Parse(raw)
		if err == nil {
			t.Errorf("expected error parsing invalid token %q, got nil", raw)
		}
	}
}
