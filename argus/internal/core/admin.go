package core

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

// ForceRemoveInstance is the administrative, immediate exit (P2-S5-T5):
// unlike a graceful Deregister, the instance is evicted at once (no grace
// period, no waiting for the evaluator sweep to notice a missed
// heartbeat) and its credential is revoked in the same call, so a zombie
// process holding the old secret cannot keep heartbeating or silently
// re-register under the same identity. Shared by both the gRPC and REST
// admin surfaces so the two transports can never disagree on what
// force-remove does.
func ForceRemoveInstance(ctx context.Context, repo Repository, instanceID uuid.UUID, reason string) error {
	if err := repo.SetInstanceStatusWithEvent(ctx, instanceID, StatusEvicted, EventEvicted, map[string]interface{}{
		"reason": reason, "forced": true,
	}); err != nil {
		return fmt.Errorf("force-evicting instance: %w", err)
	}
	if err := repo.RevokeInstanceCredential(ctx, instanceID); err != nil && !errors.Is(err, ErrNotFound) {
		return fmt.Errorf("revoking instance credential: %w", err)
	}
	return nil
}
