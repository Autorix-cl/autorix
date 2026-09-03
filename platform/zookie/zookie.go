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
	currentVer  = "v1"
)

// Token represents a Zanzibar consistency snapshot token (Zookie).
type Token struct {
	Version   string
	Timestamp time.Time
}

// New creates a new consistency token with the specified commit timestamp.
func New(t time.Time) Token {
	return Token{
		Version:   currentVer,
		Timestamp: t.UTC(),
	}
}

// Now mints a new Zookie at the current UTC nanosecond.
func Now() Token {
	return New(time.Now().UTC())
}

// String serializes the token into an opaque, URL-safe string (e.g., zk_v1_MTc4ODQ2NTMzNzEyMzQ1Njc4OQ).
func (t Token) String() string {
	nanoStr := strconv.FormatInt(t.Timestamp.UnixNano(), 10)
	encoded := base64.RawURLEncoding.EncodeToString([]byte(nanoStr))
	return fmt.Sprintf("%s%s_%s", tokenPrefix, t.Version, encoded)
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

	nano, err := strconv.ParseInt(string(decoded), 10, 64)
	if err != nil {
		return Token{}, fmt.Errorf("%w: invalid timestamp int", ErrInvalidToken)
	}

	return Token{
		Version:   ver,
		Timestamp: time.Unix(0, nano).UTC(),
	}, nil
}

// IsAtLeastFresh checks whether a target snapshot timestamp is at least as fresh as this token.
func (t Token) IsAtLeastFresh(candidate time.Time) bool {
	if t.Timestamp.IsZero() {
		return true
	}
	return !candidate.UTC().Before(t.Timestamp)
}
