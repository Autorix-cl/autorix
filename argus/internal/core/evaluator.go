package core

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// ApplyStatusTransition writes newStatus for instanceID's current
// heartbeat-driven lifecycle transition, appending the matching timeline
// event for the transitions the domain model names (degraded, unreachable,
// evicted, deregistered). A recovery to healthy — or any other transition
// with no dedicated InstanceEventType — is written without an event rather
// than inventing one outside the tested domain model. Shared by the gRPC
// heartbeat handler and the evaluator sweep so both write transitions the
// same way.
func ApplyStatusTransition(ctx context.Context, repo Repository, id uuid.UUID, newStatus InstanceStatus, now time.Time) error {
	detail := map[string]interface{}{"transitioned_at": now.UTC().Format(time.RFC3339)}
	switch newStatus {
	case StatusDegraded:
		return repo.SetInstanceStatusWithEvent(ctx, id, newStatus, EventDegraded, detail)
	case StatusUnreachable:
		return repo.SetInstanceStatusWithEvent(ctx, id, newStatus, EventUnreachable, detail)
	case StatusEvicted:
		return repo.SetInstanceStatusWithEvent(ctx, id, newStatus, EventEvicted, detail)
	case StatusDeregistered:
		return repo.SetInstanceStatusWithEvent(ctx, id, newStatus, EventDeregistered, detail)
	default:
		return repo.SetInstanceStatus(ctx, id, newStatus)
	}
}

// Evaluator runs the background lifecycle sweep (P2-S5-T2): instances that
// stop heartbeating must decay (healthy → degraded → unreachable → evicted)
// even if no one is actively querying them — EvaluateStatus is pure and
// already unit-tested, but something has to call it on a schedule for
// instances nobody is heartbeating from or asking about right now.
type Evaluator struct {
	Repo Repository
	// Now returns the current time; overridden in tests with a synthetic
	// clock so the sweep is deterministic and never sleeps for real.
	Now func() time.Time
	// StaleAfter is how long without a heartbeat makes an instance a sweep
	// candidate at all — must be at least as generous as the shortest
	// EvaluateStatus threshold (2 missed heartbeats) so nothing is skipped.
	StaleAfter time.Duration
}

// NewEvaluator builds an Evaluator with time.Now as its clock and
// StaleAfter defaulted to 2 heartbeat intervals — the same threshold
// EvaluateStatus starts degrading at.
func NewEvaluator(repo Repository) *Evaluator {
	return &Evaluator{
		Repo:       repo,
		Now:        func() time.Time { return time.Now().UTC() },
		StaleAfter: 2 * HeartbeatInterval,
	}
}

// Sweep lists instances stale relative to e.StaleAfter and advances each
// one through EvaluateStatus, applying (and persisting, with its matching
// timeline event) any transition that results. It returns the number of
// instances it actually transitioned.
func (e *Evaluator) Sweep(ctx context.Context) (int, error) {
	now := e.Now()
	stale, err := e.Repo.ListStaleInstances(ctx, now.Add(-e.StaleAfter))
	if err != nil {
		return 0, err
	}

	transitioned := 0
	for _, inst := range stale {
		newStatus := EvaluateStatus(inst.Status, inst.LastHeartbeatAt, false, now)
		if newStatus == inst.Status {
			continue
		}
		if err := ApplyStatusTransition(ctx, e.Repo, inst.ID, newStatus, now); err != nil {
			return transitioned, err
		}
		transitioned++
	}
	return transitioned, nil
}
