package macaroon

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"path"
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
	if m == nil || rootSignatureKey == "" {
		return false, ErrInvalidSignature
	}

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

	// 2. Evaluate all Caveats against Context (fail-closed if caveats exist without context)
	if len(m.Caveats) > 0 {
		if vCtx == nil {
			return false, fmt.Errorf("%w: verification context is required when caveats are present", ErrCaveatFailed)
		}
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
		if vCtx.Now.IsZero() {
			return errors.New("verification time context is required for time_before caveat")
		}
		t, err := time.Parse(time.RFC3339, val)
		if err != nil {
			return fmt.Errorf("invalid time format: %w", err)
		}
		if vCtx.Now.After(t) {
			return fmt.Errorf("token expired at %s (now: %s)", val, vCtx.Now.Format(time.RFC3339))
		}

	case "ip":
		if val == "" {
			return errors.New("ip caveat value cannot be empty")
		}
		if vCtx.IPAddress == "" {
			return errors.New("request IP is required for ip caveat")
		}
		if vCtx.IPAddress != val {
			return fmt.Errorf("request IP %s does not match allowed IP %s", vCtx.IPAddress, val)
		}

	case "method":
		if val == "" {
			return errors.New("method caveat value cannot be empty")
		}
		if vCtx.Method == "" {
			return errors.New("request method is required for method caveat")
		}
		if !strings.EqualFold(vCtx.Method, val) {
			return fmt.Errorf("request method %s not allowed (expected %s)", vCtx.Method, val)
		}

	case "path_prefix":
		if val == "" {
			return errors.New("path_prefix caveat value cannot be empty")
		}
		if vCtx.Path == "" {
			return errors.New("request path is required for path_prefix caveat")
		}
		if !matchPathPrefix(vCtx.Path, val) {
			return fmt.Errorf("request path %s does not match prefix %s", vCtx.Path, val)
		}

	default:
		return fmt.Errorf("unknown caveat predicate: %s", key)
	}

	return nil
}

// matchPathPrefix normalizes both paths via path.Clean and enforces path
// segment boundaries so traversal payloads (e.g. /public/../admin) or sibling
// prefixes (e.g. /public-secret for /public) are rejected.
func matchPathPrefix(reqPath, prefix string) bool {
	if strings.Contains(reqPath, "\\") || strings.Contains(prefix, "\\") {
		return false
	}
	if !strings.HasPrefix(reqPath, "/") || !strings.HasPrefix(prefix, "/") {
		return false
	}
	cleanPath := path.Clean(reqPath)
	cleanPrefix := path.Clean(prefix)
	if cleanPrefix == "/" {
		return strings.HasPrefix(cleanPath, "/")
	}
	return cleanPath == cleanPrefix || strings.HasPrefix(cleanPath, cleanPrefix+"/")
}
