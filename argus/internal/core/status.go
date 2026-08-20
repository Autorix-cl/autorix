// Package core holds Argus's domain model, framework-free (P2-S2-T2): no
// SQL, no HTTP, no gRPC, so the lifecycle state machine can be unit-tested
// with a synthetic clock instead of a database.
package core

import "time"

// InstanceStatus is an instance's lifecycle state (P2-S5-T1).
type InstanceStatus string

const (
	StatusPending      InstanceStatus = "pending"
	StatusRegistering  InstanceStatus = "registering"
	StatusHealthy      InstanceStatus = "healthy"
	StatusDegraded     InstanceStatus = "degraded"
	StatusUnreachable  InstanceStatus = "unreachable"
	StatusEvicted      InstanceStatus = "evicted"
	StatusDeregistered InstanceStatus = "deregistered"
)

// HeartbeatInterval is the cadence every engine's registration client
// sends heartbeats on (P2-S4-T3). The missed-beat thresholds below are
// expressed in multiples of it.
const HeartbeatInterval = 15 * time.Second

// evictAfter is how long without contact moves an instance out of the
// active fleet entirely, while retaining its history.
const evictAfter = 24 * time.Hour

// EvaluateStatus derives the status an instance should have right now,
// given when it last heartbeated and whether that heartbeat's self-reported
// readiness passed. now is an explicit parameter — never time.Now() — so
// this function is deterministically testable with a synthetic clock and
// safe to call from the evaluator sweep (P2-S5-T2) without hidden state.
//
// deregistered and evicted are terminal: only an explicit transition
// elsewhere (re-registration, force-remove) leaves them — the sweep must
// never resurrect or re-evict a terminal instance just because time passed.
func EvaluateStatus(current InstanceStatus, lastHeartbeatAt *time.Time, lastReportedReady bool, now time.Time) InstanceStatus {
	if current == StatusDeregistered || current == StatusEvicted {
		return current
	}
	if lastHeartbeatAt == nil {
		return StatusPending
	}

	elapsed := now.Sub(*lastHeartbeatAt)
	if elapsed < 0 {
		elapsed = 0
	}
	missedBeats := int(elapsed / HeartbeatInterval)

	switch {
	case elapsed >= evictAfter:
		return StatusEvicted
	case missedBeats >= 4:
		return StatusUnreachable
	case missedBeats >= 2 || !lastReportedReady:
		return StatusDegraded
	default:
		return StatusHealthy
	}
}
