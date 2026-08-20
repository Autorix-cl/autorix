package credential

import (
	"encoding/base32"
	"testing"
	"time"
)

func TestTOTP_GenerateAndVerify(t *testing.T) {
	secret, uri, err := GenerateTOTPSecret("admin@autorix.internal")
	if err != nil {
		t.Fatalf("GenerateTOTPSecret failed: %v", err)
	}
	if secret == "" || uri == "" {
		t.Fatal("expected non-empty secret and uri")
	}

	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secret)
	if err != nil {
		t.Fatalf("decoding base32 secret: %v", err)
	}

	currentStep := uint64(time.Now().UTC().Unix() / totpPeriod)
	codeInt := generateHOTP(key, currentStep)
	codeStr := string([]byte{
		byte('0' + (codeInt/100000)%10),
		byte('0' + (codeInt/10000)%10),
		byte('0' + (codeInt/1000)%10),
		byte('0' + (codeInt/100)%10),
		byte('0' + (codeInt/10)%10),
		byte('0' + codeInt%10),
	})

	if !VerifyTOTPCode(secret, codeStr) {
		t.Errorf("expected valid TOTP code %s to verify against secret %s", codeStr, secret)
	}

	if VerifyTOTPCode(secret, "000000") && codeStr != "000000" {
		t.Error("expected invalid TOTP code to fail")
	}
}

func TestRecoveryCodes(t *testing.T) {
	raw, hashed, err := GenerateRecoveryCodes(5)
	if err != nil {
		t.Fatalf("GenerateRecoveryCodes failed: %v", err)
	}
	if len(raw) != 5 || len(hashed) != 5 {
		t.Fatalf("expected 5 codes, got raw=%d, hashed=%d", len(raw), len(hashed))
	}
	for i := 0; i < 5; i++ {
		if HashSecret(raw[i]) != hashed[i] {
			t.Errorf("hash mismatch for code %d", i)
		}
	}
}
