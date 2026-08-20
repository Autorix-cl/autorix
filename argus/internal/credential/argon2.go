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

func decodeArgon2Hash(encodedHash string) (*Argon2Params, []byte, []byte, error) {
	vals := strings.Split(encodedHash, "$")
	if len(vals) != 6 {
		return nil, nil, nil, ErrInvalidHash
	}

	if vals[1] != "argon2id" {
		return nil, nil, nil, ErrIncompatibleVersion
	}

	var version int
	_, err := fmt.Sscanf(vals[2], "v=%d", &version)
	if err != nil || version != argon2.Version {
		return nil, nil, nil, ErrIncompatibleVersion
	}

	params := &Argon2Params{}
	var memory, iterations, parallelism uint64
	for _, part := range strings.Split(vals[3], ",") {
		kv := strings.Split(part, "=")
		if len(kv) != 2 {
			return nil, nil, nil, ErrInvalidHash
		}
		switch kv[0] {
		case "m":
			memory, err = strconv.ParseUint(kv[1], 10, 32)
			if err != nil {
				return nil, nil, nil, ErrInvalidHash
			}
			params.Memory = uint32(memory)
		case "t":
			iterations, err = strconv.ParseUint(kv[1], 10, 32)
			if err != nil {
				return nil, nil, nil, ErrInvalidHash
			}
			params.Iterations = uint32(iterations)
		case "p":
			parallelism, err = strconv.ParseUint(kv[1], 10, 8)
			if err != nil {
				return nil, nil, nil, ErrInvalidHash
			}
			params.Parallelism = uint8(parallelism)
		}
	}

	salt, err := base64.RawStdEncoding.Strict().DecodeString(vals[4])
	if err != nil || len(salt) > 256 {
		return nil, nil, nil, ErrInvalidHash
	}
	params.SaltLength = uint32(len(salt))

	hash, err := base64.RawStdEncoding.Strict().DecodeString(vals[5])
	if err != nil || len(hash) > 256 {
		return nil, nil, nil, ErrInvalidHash
	}
	params.KeyLength = uint32(len(hash))

	return params, salt, hash, nil
}
