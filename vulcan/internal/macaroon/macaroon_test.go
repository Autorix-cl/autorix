package macaroon

import (
	"errors"
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

func TestMacaroon_FailClosedCaveats(t *testing.T) {
	rootKey := "super-secret-root-key-12345"
	base := New("https://api.autorix.io", "key-1", rootKey)

	t.Run("unknown caveat predicate rejected", func(t *testing.T) {
		m, err := Attenuate(base, "tenant_id = acme")
		if err != nil {
			t.Fatalf("Attenuate failed: %v", err)
		}
		valid, err := Verify(m, rootKey, &core.VerificationContext{Now: time.Now()})
		if valid || !errors.Is(err, ErrCaveatFailed) {
			t.Fatalf("expected ErrCaveatFailed for unknown caveat, got valid=%v err=%v", valid, err)
		}
	})

	t.Run("nil verification context rejected when caveats exist", func(t *testing.T) {
		m, err := Attenuate(base, "method = GET")
		if err != nil {
			t.Fatalf("Attenuate failed: %v", err)
		}
		valid, err := Verify(m, rootKey, nil)
		if valid || !errors.Is(err, ErrCaveatFailed) {
			t.Fatalf("expected ErrCaveatFailed when vCtx is nil, got valid=%v err=%v", valid, err)
		}
	})

	t.Run("missing context fields rejected fail-closed", func(t *testing.T) {
		cases := []struct {
			name   string
			caveat string
			ctx    *core.VerificationContext
		}{
			{"missing ip", "ip = 10.0.0.1", &core.VerificationContext{Now: time.Now()}},
			{"missing method", "method = GET", &core.VerificationContext{Now: time.Now()}},
			{"missing path", "path_prefix = /public", &core.VerificationContext{Now: time.Now()}},
			{"zero time", "time_before = " + time.Now().Add(time.Hour).Format(time.RFC3339), &core.VerificationContext{}},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				m, err := Attenuate(base, tc.caveat)
				if err != nil {
					t.Fatalf("Attenuate failed: %v", err)
				}
				valid, err := Verify(m, rootKey, tc.ctx)
				if valid || !errors.Is(err, ErrCaveatFailed) {
					t.Fatalf("expected ErrCaveatFailed for %s, got valid=%v err=%v", tc.name, valid, err)
				}
			})
		}
	})

	t.Run("path_prefix traversal and boundary checks", func(t *testing.T) {
		m, err := Attenuate(base, "path_prefix = /public")
		if err != nil {
			t.Fatalf("Attenuate failed: %v", err)
		}

		rejected := []string{
			"/public/../admin",
			"/public/a/../../admin",
			"/public-admin",
			"/public_secret/file",
			"public/file",
			`/public\..\admin`,
		}
		for _, p := range rejected {
			valid, err := Verify(m, rootKey, &core.VerificationContext{Now: time.Now(), Path: p})
			if valid || !errors.Is(err, ErrCaveatFailed) {
				t.Errorf("expected path %q to be rejected for /public, got valid=%v err=%v", p, valid, err)
			}
		}

		allowed := []string{
			"/public",
			"/public/",
			"/public/docs",
			"/public/a/../b",
		}
		for _, p := range allowed {
			valid, err := Verify(m, rootKey, &core.VerificationContext{Now: time.Now(), Path: p})
			if !valid || err != nil {
				t.Errorf("expected path %q to be allowed for /public, got valid=%v err=%v", p, valid, err)
			}
		}
	})

	t.Run("empty root key rejected", func(t *testing.T) {
		forged := New("https://api.autorix.io", "key-1", "")
		valid, err := Verify(forged, "", &core.VerificationContext{Now: time.Now()})
		if valid || !errors.Is(err, ErrInvalidSignature) {
			t.Fatalf("expected ErrInvalidSignature for empty root key, got valid=%v err=%v", valid, err)
		}
	})
}
