package credential

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

var (
	ErrInvalidHash         = errors.New("the encoded hash is not in the correct format")
	ErrIncompatibleVersion = errors.New("incompatible version of argon2")
)

type Argon2Params struct {
	Memory      uint32
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
}

// DefaultArgon2Params matches Ego's OWASP-compliant defaults for password hashing (P3-S1-T1).
var DefaultArgon2Params = &Argon2Params{
	Memory:      64 * 1024, // 64 MB
	Iterations:  3,
	Parallelism: 4,
	SaltLength:  16,
	KeyLength:   32,
}

// HashPassword generates an Argon2id hash from a plaintext password.
func HashPassword(password string) (string, error) {
	params := DefaultArgon2Params
	salt := make([]byte, params.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generating salt: %w", err)
	}

	hash := argon2.IDKey(
		[]byte(password),
		salt,
		params.Iterations,
		params.Memory,
		params.Parallelism,
		params.KeyLength,
	)

	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	encodedHash := fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		params.Memory,
		params.Iterations,
		params.Parallelism,
		b64Salt,
		b64Hash,
	)

	return encodedHash, nil
}

// VerifyPassword verifies a plaintext password against an encoded Argon2id hash.
func VerifyPassword(password, encodedHash string) (bool, error) {
	params, salt, hash, err := decodeArgon2Hash(encodedHash)
	if err != nil {
		return false, err
	}

	otherHash := argon2.IDKey(
		[]byte(password),
		salt,
		params.Iterations,
		params.Memory,
		params.Parallelism,
		params.KeyLength,
	)

	if subtle.ConstantTimeCompare(hash, otherHash) == 1 {
		return true, nil
	}

	return false, nil
}

const (
	minArgon2Memory      uint32 = 1024       // 1 MiB
	maxArgon2Memory      uint32 = 512 * 1024 // 512 MiB
	minArgon2Iterations  uint32 = 1
	maxArgon2Iterations  uint32 = 32
	minArgon2Parallelism uint8  = 1
	maxArgon2Parallelism uint8  = 32
	minArgon2SaltLength         = 8
	maxArgon2SaltLength         = 256
	minArgon2KeyLength          = 16
	maxArgon2KeyLength          = 256
)

func decodeArgon2Hash(encodedHash string) (*Argon2Params, []byte, []byte, error) {
	vals := strings.Split(encodedHash, "$")
	if len(vals) != 6 || vals[0] != "" {
		return nil, nil, nil, ErrInvalidHash
	}

	if vals[1] != "argon2id" {
		return nil, nil, nil, ErrIncompatibleVersion
	}

	if !strings.HasPrefix(vals[2], "v=") {
		return nil, nil, nil, ErrInvalidHash
	}
	version, err := strconv.Atoi(strings.TrimPrefix(vals[2], "v="))
	if err != nil || version != argon2.Version {
		return nil, nil, nil, ErrIncompatibleVersion
	}

	params := &Argon2Params{}
	var seenM, seenT, seenP bool
	for _, part := range strings.Split(vals[3], ",") {
		kv := strings.Split(part, "=")
		if len(kv) != 2 {
			return nil, nil, nil, ErrInvalidHash
		}
		switch kv[0] {
		case "m":
			if seenM {
				return nil, nil, nil, ErrInvalidHash
			}
			memory, parseErr := strconv.ParseUint(kv[1], 10, 32)
			if parseErr != nil {
				return nil, nil, nil, ErrInvalidHash
			}
			params.Memory = uint32(memory)
			seenM = true
		case "t":
			if seenT {
				return nil, nil, nil, ErrInvalidHash
			}
			iterations, parseErr := strconv.ParseUint(kv[1], 10, 32)
			if parseErr != nil {
				return nil, nil, nil, ErrInvalidHash
			}
			params.Iterations = uint32(iterations)
			seenT = true
		case "p":
			if seenP {
				return nil, nil, nil, ErrInvalidHash
			}
			parallelism, parseErr := strconv.ParseUint(kv[1], 10, 8)
			if parseErr != nil {
				return nil, nil, nil, ErrInvalidHash
			}
			params.Parallelism = uint8(parallelism)
			seenP = true
		default:
			return nil, nil, nil, ErrInvalidHash
		}
	}
	if !seenM || !seenT || !seenP {
		return nil, nil, nil, ErrInvalidHash
	}
	if params.Memory < minArgon2Memory || params.Memory > maxArgon2Memory ||
		params.Iterations < minArgon2Iterations || params.Iterations > maxArgon2Iterations ||
		params.Parallelism < minArgon2Parallelism || params.Parallelism > maxArgon2Parallelism {
		return nil, nil, nil, ErrInvalidHash
	}

	salt, err := base64.RawStdEncoding.Strict().DecodeString(vals[4])
	if err != nil || len(salt) < minArgon2SaltLength || len(salt) > maxArgon2SaltLength {
		return nil, nil, nil, ErrInvalidHash
	}
	params.SaltLength = uint32(len(salt)) //nolint:gosec // length bounded by maxArgon2SaltLength

	hash, err := base64.RawStdEncoding.Strict().DecodeString(vals[5])
	if err != nil || len(hash) < minArgon2KeyLength || len(hash) > maxArgon2KeyLength {
		return nil, nil, nil, ErrInvalidHash
	}
	params.KeyLength = uint32(len(hash)) //nolint:gosec // length bounded by maxArgon2KeyLength

	return params, salt, hash, nil
}
