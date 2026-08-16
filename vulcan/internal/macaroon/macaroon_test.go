package macaroon

import (
	"testing"
	"time"

	"github.com/autorix/vulcan/internal/core"
)

func TestMacaroon_LifecycleAndAttenuation(t *testing.T) {
	rootKey := "super-secret-root-key-12345"
	keyID := "api-key-uuid-100"
	location := "https://api.autorix.io"

	// 1. Create Base Macaroon
	m0 := New(location, keyID, rootKey)
	if m0.Signature == "" {
		t.Fatal("expected non-empty signature")
	}

	// Verify base macaroon
	valid, err := Verify(m0, rootKey, nil)
	if err != nil || !valid {
		t.Fatalf("base macaroon verification failed: %v", err)
	}

	// 2. Attenuate with Caveat 1: time restriction in future
	futureTime := time.Now().Add(1 * time.Hour).Format(time.RFC3339)
	m1, err := Attenuate(m0, "time_before = "+futureTime)
	if err != nil {
		t.Fatalf("Attenuate failed: %v", err)
	}

	if len(m1.Caveats) != 1 {
		t.Fatalf("expected 1 caveat, got %d", len(m1.Caveats))
	}

	// Verify m1 with current time
	vCtx := &core.VerificationContext{Now: time.Now()}
	valid, err = Verify(m1, rootKey, vCtx)
	if err != nil || !valid {
		t.Fatalf("attenuated m1 verification failed: %v", err)
	}

	// 3. Attenuate further with Caveat 2: IP restriction
	m2, err := Attenuate(m1, "ip = 192.168.1.50")
	if err != nil {
		t.Fatalf("Attenuate 2 failed: %v", err)
	}

	if len(m2.Caveats) != 2 {
		t.Fatalf("expected 2 caveats, got %d", len(m2.Caveats))
	}

	// Verify m2 matching IP
	vCtxIP := &core.VerificationContext{
		Now:       time.Now(),
		IPAddress: "192.168.1.50",
	}
	valid, err = Verify(m2, rootKey, vCtxIP)
	if err != nil || !valid {
		t.Fatalf("attenuated m2 verification failed with matching IP: %v", err)
	}

	// 4. Verify m2 with WRONG IP (must fail)
	vCtxWrongIP := &core.VerificationContext{
		Now:       time.Now(),
		IPAddress: "10.0.0.99",
	}
	valid, err = Verify(m2, rootKey, vCtxWrongIP)
	if err == nil || valid {
		t.Errorf("expected verification to fail for wrong IP, but passed")
	}

	// 5. Tamper with signature (must fail)
	m2Tampered := *m2
	m2Tampered.Signature = "00112233445566778899aabbccddeeff"
	valid, err = Verify(&m2Tampered, rootKey, vCtxIP)
	if err == nil || valid {
		t.Errorf("expected verification to fail for tampered signature")
	}
}
