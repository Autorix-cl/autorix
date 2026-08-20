package macaroon

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/autorix/vulcan/internal/core"
)

var (
	ErrInvalidSignature = errors.New("macaroon signature verification failed")
	ErrCaveatFailed     = errors.New("macaroon caveat restriction violated")
)

// New creates an initial base Macaroon without caveats
func New(location, keyID, rootSignatureKey string) *core.Macaroon {
	h := hmac.New(sha256.New, []byte(rootSignatureKey))
	h.Write([]byte(keyID))
	sig := hex.EncodeToString(h.Sum(nil))

	return &core.Macaroon{
		Location:  location,
		KeyID:     keyID,
		Caveats:   []core.Caveat{},
		Signature: sig,
	}
}

// Attenuate adds a new caveat locally and computes the chained HMAC signature
func Attenuate(m *core.Macaroon, predicate string) (*core.Macaroon, error) {
	currentSigBytes, err := hex.DecodeString(m.Signature)
	if err != nil {
		return nil, fmt.Errorf("invalid current signature hex: %w", err)
	}

	// Compute chained HMAC: HMAC(current_signature, new_predicate)
	h := hmac.New(sha256.New, currentSigBytes)
	h.Write([]byte(predicate))
	newSig := hex.EncodeToString(h.Sum(nil))

	newCaveats := make([]core.Caveat, len(m.Caveats), len(m.Caveats)+1)
	copy(newCaveats, m.Caveats)
	newCaveats = append(newCaveats, core.Caveat{Predicate: predicate})

	return &core.Macaroon{
		Location:  m.Location,
		KeyID:     m.KeyID,
		Caveats:   newCaveats,
		Signature: newSig,
	}, nil
}

// Verify validates both the chained cryptographic signature and all caveats
func Verify(m *core.Macaroon, rootSignatureKey string, vCtx *core.VerificationContext) (bool, error) {
	// 1. Reconstruct Chained Signature from Root Key
	h := hmac.New(sha256.New, []byte(rootSignatureKey))
	h.Write([]byte(m.KeyID))
	currentSig := h.Sum(nil)

	for _, c := range m.Caveats {
		hNext := hmac.New(sha256.New, currentSig)
		hNext.Write([]byte(c.Predicate))
		currentSig = hNext.Sum(nil)
	}

	expectedSigHex := hex.EncodeToString(currentSig)

	// Constant-time signature comparison
	if subtle.ConstantTimeCompare([]byte(m.Signature), []byte(expectedSigHex)) != 1 {
		return false, ErrInvalidSignature
	}

	// 2. Evaluate all Caveats against Context
	if vCtx != nil {
		for _, c := range m.Caveats {
			if err := evaluateCaveat(c.Predicate, vCtx); err != nil {
				return false, fmt.Errorf("%w: %v", ErrCaveatFailed, err)
			}
		}
	}

	return true, nil
}

func evaluateCaveat(predicate string, vCtx *core.VerificationContext) error {
	parts := strings.SplitN(predicate, "=", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid caveat syntax (expected key = val): %s", predicate)
	}

	key := strings.TrimSpace(parts[0])
	val := strings.TrimSpace(parts[1])

	switch key {
	case "time_before":
		t, err := time.Parse(time.RFC3339, val)
		if err != nil {
			return fmt.Errorf("invalid time format: %w", err)
		}
		if vCtx.Now.After(t) {
			return fmt.Errorf("token expired at %s (now: %s)", val, vCtx.Now.Format(time.RFC3339))
		}

	case "ip":
		if vCtx.IPAddress != "" && vCtx.IPAddress != val {
			return fmt.Errorf("request IP %s does not match allowed IP %s", vCtx.IPAddress, val)
		}

	case "method":
		if vCtx.Method != "" && !strings.EqualFold(vCtx.Method, val) {
			return fmt.Errorf("request method %s not allowed (expected %s)", vCtx.Method, val)
		}

	case "path_prefix":
		if vCtx.Path != "" && !strings.HasPrefix(vCtx.Path, val) {
			return fmt.Errorf("request path %s does not match prefix %s", vCtx.Path, val)
		}
	}

	return nil
}
