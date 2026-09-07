package jwks

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/autorix/janus/internal/storage/postgres"
	"github.com/autorix/platform/pgtest"
)

func newPersistentTestRepository(t *testing.T) *postgres.Repository {
	t.Helper()
	return postgres.NewRepository(pgtest.StartPostgres(t, "../../migrations"))
}

// TestPersistentKeyManager_ConcurrentBootstrapAndRestart proves that a fleet
// starting against an empty database converges on one durable key rather than
// independently minting incompatible JWTs. The final manager simulates a
// process restart and validates a token minted before it started.
func TestPersistentKeyManager_ConcurrentBootstrapAndRestart(t *testing.T) {
	repo := newPersistentTestRepository(t)
	ctx := context.Background()

	const instances = 4
	managers := make([]*KeyManager, instances)
	errs := make(chan error, instances)
	var wg sync.WaitGroup
	for i := range managers {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			manager, err := NewPersistentKeyManager(ctx, repo)
			managers[i] = manager
			errs <- err
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent persistent bootstrap: %v", err)
		}
	}

	wantKid := managers[0].KeyID()
	for i, manager := range managers {
		if got := manager.KeyID(); got != wantKid {
			t.Fatalf("instance %d selected kid %q, want shared kid %q", i, got, wantKid)
		}
		if got := manager.KeyCount(); got != 1 {
			t.Fatalf("instance %d loaded %d keys, want one", i, got)
		}
	}
	keys, err := repo.ListSigningKeys(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 1 {
		t.Fatalf("database contains %d bootstrap keys, want one", len(keys))
	}

	token, err := managers[0].SignJWT(GenerateClaims("issuer", "subject", "audience", nil, time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	restarted, err := NewPersistentKeyManager(ctx, repo)
	if err != nil {
		t.Fatalf("restart load: %v", err)
	}
	if restarted.KeyID() != wantKid {
		t.Fatalf("restart selected kid %q, want %q", restarted.KeyID(), wantKid)
	}
	if _, err := restarted.VerifyJWT(token); err != nil {
		t.Fatalf("restart could not verify persisted-key token: %v", err)
	}
}

// TestPersistentKeyManager_RotationRetainsVerificationKeys verifies durable
// rotation: the new signing key survives restart and a retired key keeps
// validating tokens during the rollover window.
func TestPersistentKeyManager_RotationRetainsVerificationKeys(t *testing.T) {
	repo := newPersistentTestRepository(t)
	ctx := context.Background()
	first, err := NewPersistentKeyManager(ctx, repo)
	if err != nil {
		t.Fatal(err)
	}
	oldKid := first.KeyID()
	oldToken, err := first.SignJWT(GenerateClaims("issuer", "old", "audience", nil, time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := first.RotateKey(); err != nil {
		t.Fatalf("rotate persistent key: %v", err)
	}
	if first.KeyID() == oldKid || first.KeyCount() != 2 {
		t.Fatalf("rotation state: kid=%q keys=%d", first.KeyID(), first.KeyCount())
	}
	if _, err := first.VerifyJWT(oldToken); err != nil {
		t.Fatalf("rotated manager rejected retired-key token: %v", err)
	}

	otherInstance, err := NewPersistentKeyManager(ctx, repo)
	if err != nil {
		t.Fatalf("restart after rotation: %v", err)
	}
	if otherInstance.KeyID() != first.KeyID() || otherInstance.KeyCount() != 2 {
		t.Fatalf("restart did not restore rotation state: kid=%q keys=%d", otherInstance.KeyID(), otherInstance.KeyCount())
	}
	if _, err := otherInstance.VerifyJWT(oldToken); err != nil {
		t.Fatalf("restart rejected retired-key token: %v", err)
	}
	newToken, err := otherInstance.SignJWT(GenerateClaims("issuer", "new", "audience", nil, time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if err := first.Reload(ctx); err != nil {
		t.Fatalf("reload other instance: %v", err)
	}
	if _, err := first.VerifyJWT(newToken); err != nil {
		t.Fatalf("reloaded manager rejected new-key token: %v", err)
	}
}
