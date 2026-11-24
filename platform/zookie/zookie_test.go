package zookie

import (
	"testing"
	"time"
)

func TestZookie_LSNRoundtripSerialization(t *testing.T) {
	lsnStr := "0/3A13F50"
	lsn, err := ParseLSN(lsnStr)
	if err != nil {
		t.Fatalf("failed to parse LSN %s: %v", lsnStr, err)
	}

	formatted := FormatLSN(lsn)
	if formatted != lsnStr {
		t.Errorf("LSN formatting mismatch: got %s, want %s", formatted, lsnStr)
	}

	token := NewWithLSN(lsn)
	serialized := token.String()
	if len(serialized) == 0 {
		t.Fatalf("expected non-empty serialized token")
	}

	parsed, err := Parse(serialized)
	if err != nil {
		t.Fatalf("failed to parse serialized LSN token %s: %v", serialized, err)
	}

	if parsed.LSN != lsn {
		t.Errorf("LSN mismatch: got %d, want %d", parsed.LSN, lsn)
	}
	if parsed.Version != "v2" {
		t.Errorf("version mismatch: got %s, want v2", parsed.Version)
	}
}

func TestZookie_LSNFreshness(t *testing.T) {
	older := NewWithLSN(100)
	newer := NewWithLSN(200)
	equal := NewWithLSN(100)

	// newer candidate is fresh relative to older requested
	if !older.IsAtLeastFresh(newer) {
		t.Errorf("expected newer candidate to be fresh relative to older")
	}

	// equal candidate is fresh
	if !older.IsAtLeastFresh(equal) {
		t.Errorf("expected equal candidate to be fresh")
	}

	// older candidate is NOT fresh relative to newer requested (must bypass cache)
	if newer.IsAtLeastFresh(older) {
		t.Errorf("expected older candidate to be stale relative to newer")
	}
}

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

	if !zk.IsAtLeastFreshTime(t1) {
		t.Errorf("expected t1 (%v) to be fresh relative to zk (%v)", t1, t0)
	}
	if !zk.IsAtLeastFreshTime(t0) {
		t.Errorf("expected t0 (%v) to be fresh relative to zk (%v)", t0, t0)
	}
	if zk.IsAtLeastFreshTime(tPast) {
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
		"zk_v2_not_hex!!!",
		"zk_v3_MTIzNDU",
		"zk_unknown_MTIzNDU",
	}

	for _, raw := range invalidTokens {
		_, err := Parse(raw)
		if err == nil {
			t.Errorf("expected error parsing invalid token %q, got nil", raw)
		}
	}
}

func TestZookie_IsAtLeastFresh_IncomparableFailsClosed(t *testing.T) {
	v2Parsed, err := Parse(NewWithLSN(200).String())
	if err != nil {
		t.Fatalf("failed to parse v2 token: %v", err)
	}
	v1Token := New(time.Now().UTC().Add(time.Hour))
	v2InMem := NewWithLSN(200)

	if v2Parsed.IsAtLeastFresh(v1Token) {
		t.Errorf("expected parsed v2 token vs v1 token to fail closed")
	}
	if v1Token.IsAtLeastFresh(v2Parsed) {
		t.Errorf("expected v1 token vs parsed v2 token to fail closed")
	}
	if v2InMem.IsAtLeastFresh(v1Token) {
		t.Errorf("expected in-memory v2 token vs v1 token to fail closed")
	}
	if (Token{}).IsAtLeastFresh(Token{}) {
		t.Errorf("expected zero tokens to fail closed")
	}

	earlier := New(time.Now().UTC())
	later := New(earlier.Timestamp.Add(time.Second))
	if !earlier.IsAtLeastFresh(later) {
		t.Errorf("expected later v1 token to be fresh relative to earlier v1 token")
	}
	if later.IsAtLeastFresh(earlier) {
		t.Errorf("expected earlier v1 token to be stale relative to later v1 token")
	}
}
