package credential

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/subtle"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"math"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	totpDigits   = 6
	totpPeriod   = 30
	totpSecretLen = 20
)

// GenerateTOTPSecret returns a random base32 secret and an otpauth URL for QR codes (P3-S1-T4).
func GenerateTOTPSecret(email string) (secret string, uri string, err error) {
	b := make([]byte, totpSecretLen)
	if _, err := rand.Read(b); err != nil {
		return "", "", fmt.Errorf("generating totp secret: %w", err)
	}

	secret = base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b)
	issuer := "Autorix Console"
	account := email
	if account == "" {
		account = "operator"
	}

	uri = fmt.Sprintf(
		"otpauth://totp/%s:%s?secret=%s&issuer=%s&algorithm=SHA1&digits=6&period=30",
		url.PathEscape(issuer),
		url.PathEscape(account),
		secret,
		url.QueryEscape(issuer),
	)

	return secret, uri, nil
}

// VerifyTOTPCode verifies a 6-digit TOTP code against a base32 secret,
// allowing a +/- 1 step window (30 seconds) for clock drift.
func VerifyTOTPCode(secret string, code string) bool {
	cleanSecret := strings.ToUpper(strings.TrimSpace(secret))
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(cleanSecret)
	if err != nil {
		// try with padding
		key, err = base32.StdEncoding.DecodeString(cleanSecret)
		if err != nil {
			return false
		}
	}

	cleanCode := strings.TrimSpace(code)
	if len(cleanCode) != totpDigits {
		return false
	}
	intCode, err := strconv.Atoi(cleanCode)
	if err != nil {
		return false
	}

	currentTime := time.Now().UTC().Unix()
	currentStep := currentTime / totpPeriod

	// Check steps: current - 1, current, current + 1
	for stepOffset := int64(-1); stepOffset <= 1; stepOffset++ {
		step := currentStep + stepOffset
		expectedCode := generateHOTP(key, uint64(step))
		if subtle.ConstantTimeEq(int32(expectedCode), int32(intCode)) == 1 {
			return true
		}
	}

	return false
}

func generateHOTP(key []byte, counter uint64) int {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, counter)

	mac := hmac.New(sha1.New, key)
	mac.Write(buf)
	sum := mac.Sum(nil)

	offset := sum[len(sum)-1] & 0x0f
	binaryCode := binary.BigEndian.Uint32(sum[offset : offset+4])
	binaryCode &= 0x7fffffff

	mod := uint32(math.Pow10(totpDigits))
	return int(binaryCode % mod)
}

// GenerateRecoveryCodes produces single-use recovery codes and their hashes (P3-S1-T4).
func GenerateRecoveryCodes(count int) (rawCodes []string, hashCodes []string, err error) {
	for i := 0; i < count; i++ {
		b := make([]byte, 5)
		if _, err := rand.Read(b); err != nil {
			return nil, nil, err
		}
		raw := fmt.Sprintf("%04x-%04x", binary.BigEndian.Uint16(b[0:2]), binary.BigEndian.Uint16(b[2:4]))
		rawCodes = append(rawCodes, raw)
		hashCodes = append(hashCodes, HashSecret(raw))
	}
	return rawCodes, hashCodes, nil
}
