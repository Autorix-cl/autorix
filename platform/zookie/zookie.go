package zookie

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

var (
	ErrInvalidToken = errors.New("zookie: invalid token format")
	ErrExpiredToken = errors.New("zookie: snapshot token has expired")
)

const (
	tokenPrefix = "zk_"
	v1Ver       = "v1"
	v2Ver       = "v2" // Database LSN / Transaction sequence (PostgreSQL linearizable)
)

// Token represents a Zanzibar consistency snapshot token (Zookie).
// It supports PostgreSQL Log Sequence Numbers (LSN) for database-enforced linearizability,
// and monotonic timestamps as a portable fallback.
type Token struct {
	Version   string
	LSN       uint64    // PostgreSQL Log Sequence Number (LSN) or transaction sequence
	Timestamp time.Time // Fallback UTC timestamp
}

// ParseLSN parses a PostgreSQL WAL LSN string ("X/Y" hex format) into a uint64.
func ParseLSN(lsn string) (uint64, error) {
	parts := strings.Split(strings.TrimSpace(lsn), "/")
	if len(parts) != 2 {
		return 0, errors.New("invalid postgres lsn format: expected X/Y")
	}
	high, err := strconv.ParseUint(parts[0], 16, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid lsn high word: %w", err)
	}
	low, err := strconv.ParseUint(parts[1], 16, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid lsn low word: %w", err)
	}
	return (high << 32) | low, nil
}

// FormatLSN formats a uint64 into a PostgreSQL WAL LSN string ("X/Y" hex format).
func FormatLSN(lsn uint64) string {
	high := uint32(lsn >> 32)
	low := uint32(lsn & 0xFFFFFFFF)
	return fmt.Sprintf("%X/%X", high, low)
}

// New creates a new consistency token with the specified commit timestamp (v1).
func New(t time.Time) Token {
	return Token{
		Version:   v1Ver,
		Timestamp: t.UTC(),
	}
}

// NewWithLSN creates a Zookie backed by a PostgreSQL Log Sequence Number (LSN) (v2).
// This guarantees strict linearizable causal ordering without relying on application clocks.
func NewWithLSN(lsn uint64) Token {
	return Token{
		Version:   v2Ver,
		LSN:       lsn,
		Timestamp: time.Now().UTC(),
	}
}

// Now mints a new Zookie at the current UTC nanosecond.
func Now() Token {
	return New(time.Now().UTC())
}

// String serializes the token into an opaque, URL-safe string.
func (t Token) String() string {
	if t.Version == v2Ver || t.LSN > 0 {
		lsnHex := strconv.FormatUint(t.LSN, 16)
		encoded := base64.RawURLEncoding.EncodeToString([]byte(lsnHex))
		return fmt.Sprintf("%s%s_%s", tokenPrefix, v2Ver, encoded)
	}

	nanoStr := strconv.FormatInt(t.Timestamp.UnixNano(), 10)
	encoded := base64.RawURLEncoding.EncodeToString([]byte(nanoStr))
	return fmt.Sprintf("%s%s_%s", tokenPrefix, v1Ver, encoded)
}

// Parse decodes a raw string into a Token.
func Parse(raw string) (Token, error) {
	if !strings.HasPrefix(raw, tokenPrefix) {
		return Token{}, ErrInvalidToken
	}

	parts := strings.Split(raw, "_")
	if len(parts) != 3 {
		return Token{}, ErrInvalidToken
	}

	ver := parts[1]
	encoded := parts[2]

	decoded, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return Token{}, fmt.Errorf("%w: base64 decode error", ErrInvalidToken)
	}

	if ver == v2Ver {
		lsn, err := strconv.ParseUint(string(decoded), 16, 64)
		if err != nil {
			return Token{}, fmt.Errorf("%w: invalid lsn hex", ErrInvalidToken)
		}
		return Token{
			Version: ver,
			LSN:     lsn,
		}, nil
	}

	nano, err := strconv.ParseInt(string(decoded), 10, 64)
	if err != nil {
		return Token{}, fmt.Errorf("%w: invalid timestamp int", ErrInvalidToken)
	}

	return Token{
		Version:   ver,
		Timestamp: time.Unix(0, nano).UTC(),
	}, nil
}

// IsAtLeastFresh checks whether candidate (a database snapshot or cached token) is at least as fresh as requested token t.
func (t Token) IsAtLeastFresh(candidate Token) bool {
	// If the requested token has a database LSN, evaluate strictly by LSN sequence
	if t.LSN > 0 && candidate.LSN > 0 {
		return candidate.LSN >= t.LSN
	}

	// Fallback to timestamp comparison if LSN is not present
	if !t.Timestamp.IsZero() && !candidate.Timestamp.IsZero() {
		return !candidate.Timestamp.UTC().Before(t.Timestamp)
	}

	return true
}

// IsAtLeastFreshTime checks whether a target snapshot timestamp is at least as fresh as this token (v1 compatibility).
func (t Token) IsAtLeastFreshTime(candidate time.Time) bool {
	if t.Timestamp.IsZero() {
		return true
	}
	return !candidate.UTC().Before(t.Timestamp)
}
