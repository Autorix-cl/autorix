package credential

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base32"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

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

// DefaultArgon2Params provides OWASP-compliant defaults for password hashing
var DefaultArgon2Params = &Argon2Params{
	Memory:      64 * 1024, // 64 MB
	Iterations:  3,
	Parallelism: 4,
	SaltLength:  16,
	KeyLength:   32,
}

// Hasher handles Argon2id password hashing and verification
type Hasher struct {
	params *Argon2Params
}

func NewHasher(params *Argon2Params) *Hasher {
	if params == nil {
		params = DefaultArgon2Params
	}
	return &Hasher{params: params}
}

// GenerateHash hashes a plain password and returns the PHC string format
func (h *Hasher) GenerateHash(password string) (string, error) {
	salt := make([]byte, h.params.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("failed to generate random salt: %w", err)
	}

	hash := argon2.IDKey(
		[]byte(password),
		salt,
		h.params.Iterations,
		h.params.Memory,
		h.params.Parallelism,
		h.params.KeyLength,
	)

	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	encodedHash := fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		h.params.Memory,
		h.params.Iterations,
		h.params.Parallelism,
		b64Salt,
		b64Hash,
	)

	return encodedHash, nil
}

// ComparePasswordAndHash verifies a plain password against an encoded Argon2id hash
func (h *Hasher) ComparePasswordAndHash(password, encodedHash string) (bool, error) {
	params, salt, hash, err := decodeHash(encodedHash)
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

func decodeHash(encodedHash string) (*Argon2Params, []byte, []byte, error) {
	vals := strings.Split(encodedHash, "$")
	if len(vals) != 6 {
		return nil, nil, nil, ErrInvalidHash
	}

	if vals[1] != "argon2id" {
		return nil, nil, nil, ErrInvalidHash
	}

	var version int
	_, err := fmt.Sscanf(vals[2], "v=%d", &version)
	if err != nil || version != argon2.Version {
		return nil, nil, nil, ErrIncompatibleVersion
	}

	params := &Argon2Params{}
	_, err = fmt.Sscanf(vals[3], "m=%d,t=%d,p=%d", &params.Memory, &params.Iterations, &params.Parallelism)
	if err != nil {
		return nil, nil, nil, ErrInvalidHash
	}

	salt, err := base64.RawStdEncoding.DecodeString(vals[4])
	if err != nil {
		return nil, nil, nil, ErrInvalidHash
	}
	// A real Argon2id salt/hash is a few dozen bytes; anything above this
	// bound is a corrupted or hostile stored value, not a legitimate hash —
	// reject it outright rather than let the uint32 conversion below wrap.
	const maxComponentLen = 1024
	if len(salt) > maxComponentLen {
		return nil, nil, nil, ErrInvalidHash
	}
	params.SaltLength = uint32(len(salt)) // #nosec G115 -- bounded to maxComponentLen above

	hash, err := base64.RawStdEncoding.DecodeString(vals[5])
	if err != nil {
		return nil, nil, nil, ErrInvalidHash
	}
	if len(hash) > maxComponentLen {
		return nil, nil, nil, ErrInvalidHash
	}
	params.KeyLength = uint32(len(hash)) // #nosec G115 -- bounded to maxComponentLen above

	return params, salt, hash, nil
}

const alphaNumericChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// GenerateSecureRandomString generates a cryptographically secure random alphanumeric string
func GenerateSecureRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	for i := 0; i < length; i++ {
		bytes[i] = alphaNumericChars[int(bytes[i])%len(alphaNumericChars)]
	}
	return string(bytes), nil
}

// GenerateRecoveryToken generates a 32-byte hex token and its SHA256 hash
func GenerateRecoveryToken() (string, string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", fmt.Errorf("failed to generate recovery token: %w", err)
	}
	raw := hex.EncodeToString(b)
	hashBytes := sha256.Sum256([]byte(raw))
	hash := hex.EncodeToString(hashBytes[:])
	return raw, hash, nil
}

// HashToken computes a SHA-256 hex digest of a token
func HashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// GenerateTOTPSecret generates a new Base32 TOTP secret and standard otpauth URL
func GenerateTOTPSecret(issuer, accountName string) (string, string, error) {
	secretBytes := make([]byte, 20)
	if _, err := rand.Read(secretBytes); err != nil {
		return "", "", fmt.Errorf("failed to generate TOTP secret: %w", err)
	}
	secret := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secretBytes)

	label := fmt.Sprintf("%s:%s", issuer, accountName)
	v := url.Values{}
	v.Set("secret", secret)
	v.Set("issuer", issuer)
	v.Set("algorithm", "SHA1")
	v.Set("digits", "6")
	v.Set("period", "30")

	qrURI := fmt.Sprintf("otpauth://totp/%s?%s", url.PathEscape(label), v.Encode())
	return secret, qrURI, nil
}

// GenerateTOTPCode generates a 6-digit TOTP code for a secret at a given step offset
func GenerateTOTPCode(secret string, stepOffset int64) (string, error) {
	secretBytes, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(secret))
	if err != nil {
		return "", fmt.Errorf("invalid base32 secret: %w", err)
	}

	step := (time.Now().Unix() / 30) + stepOffset
	var counterBuf [8]byte
	binary.BigEndian.PutUint64(counterBuf[:], uint64(step))

	mac := hmac.New(sha1.New, secretBytes)
	mac.Write(counterBuf[:])
	hmacResult := mac.Sum(nil)

	offset := hmacResult[len(hmacResult)-1] & 0x0f
	binaryCode := (int(hmacResult[offset]&0x7f) << 24) |
		(int(hmacResult[offset+1]&0xff) << 16) |
		(int(hmacResult[offset+2]&0xff) << 8) |
		int(hmacResult[offset+3]&0xff)

	code := binaryCode % 1000000
	return fmt.Sprintf("%06d", code), nil
}

// ValidateTOTPCode verifies a 6-digit code against a secret with ±1 step skew
func ValidateTOTPCode(secret, code string) bool {
	if len(code) != 6 {
		return false
	}
	for _, offset := range []int64{-1, 0, 1} {
		expected, err := GenerateTOTPCode(secret, offset)
		if err == nil && subtle.ConstantTimeCompare([]byte(expected), []byte(code)) == 1 {
			return true
		}
	}
	return false
}

// GenerateBackupCodes creates count backup codes formatted as XXXX-XXXX and their SHA256 hashes
func GenerateBackupCodes(count int) ([]string, []string, error) {
	if count <= 0 {
		count = 8
	}
	plainCodes := make([]string, count)
	hashedCodes := make([]string, count)
	for i := 0; i < count; i++ {
		b := make([]byte, 4)
		if _, err := rand.Read(b); err != nil {
			return nil, nil, fmt.Errorf("failed to generate backup code: %w", err)
		}
		raw := hex.EncodeToString(b)
		code := fmt.Sprintf("%s-%s", raw[:4], raw[4:])
		plainCodes[i] = code
		h := sha256.Sum256([]byte(code))
		hashedCodes[i] = hex.EncodeToString(h[:])
	}
	return plainCodes, hashedCodes, nil
}

