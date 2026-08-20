package core_test

import (
	"testing"
	"time"

	"github.com/autorix/argus/internal/core"
)

func at(secondsAgo int, now time.Time) *time.Time {
	t := now.Add(-time.Duration(secondsAgo) * time.Second)
	return &t
}

func TestEvaluateStatus_PendingWhenNeverHeartbeated(t *testing.T) {
	now := time.Now()
	got := core.EvaluateStatus(core.StatusPending, nil, true, now)
	if got != core.StatusPending {
		t.Fatalf("got %q, want %q", got, core.StatusPending)
	}
}

func TestEvaluateStatus_HealthyWithCurrentBeatAndReadiness(t *testing.T) {
	now := time.Now()
	got := core.EvaluateStatus(core.StatusHealthy, at(5, now), true, now)
	if got != core.StatusHealthy {
		t.Fatalf("got %q, want %q", got, core.StatusHealthy)
	}
}

func TestEvaluateStatus_DegradedWhenReadinessFailsEvenIfBeatIsCurrent(t *testing.T) {
	now := time.Now()
	got := core.EvaluateStatus(core.StatusHealthy, at(5, now), false, now)
	if got != core.StatusDegraded {
		t.Fatalf("got %q, want %q", got, core.StatusDegraded)
	}
}

func TestEvaluateStatus_DegradedAfterTwoMissedBeats(t *testing.T) {
	now := time.Now()
	// 2 missed beats = 30s..44s elapsed at a 15s interval.
	got := core.EvaluateStatus(core.StatusHealthy, at(31, now), true, now)
	if got != core.StatusDegraded {
		t.Fatalf("got %q, want %q", got, core.StatusDegraded)
	}
}

func TestEvaluateStatus_StaysHealthyAtOneMissedBeat(t *testing.T) {
	now := time.Now()
	got := core.EvaluateStatus(core.StatusHealthy, at(16, now), true, now)
	if got != core.StatusHealthy {
		t.Fatalf("got %q, want %q", got, core.StatusHealthy)
	}
}

func TestEvaluateStatus_UnreachableAfterFourMissedBeats(t *testing.T) {
	now := time.Now()
	got := core.EvaluateStatus(core.StatusDegraded, at(61, now), true, now)
	if got != core.StatusUnreachable {
		t.Fatalf("got %q, want %q", got, core.StatusUnreachable)
	}
}

func TestEvaluateStatus_EvictedAfterTwentyFourHours(t *testing.T) {
	now := time.Now()
	got := core.EvaluateStatus(core.StatusUnreachable, at(24*3600+1, now), true, now)
	if got != core.StatusEvicted {
		t.Fatalf("got %q, want %q", got, core.StatusEvicted)
	}
}

func TestEvaluateStatus_DeregisteredIsTerminal(t *testing.T) {
	now := time.Now()
	// A deregistered instance must never be resurrected by the evaluator
	// sweep just because time has passed — only an explicit re-registration
	// (handled elsewhere) changes it.
	got := core.EvaluateStatus(core.StatusDeregistered, at(999999, now), true, now)
	if got != core.StatusDeregistered {
		t.Fatalf("got %q, want %q", got, core.StatusDeregistered)
	}
}

func TestEvaluateStatus_EvictedIsTerminal(t *testing.T) {
	now := time.Now()
	got := core.EvaluateStatus(core.StatusEvicted, at(1, now), true, now)
	if got != core.StatusEvicted {
		t.Fatalf("got %q, want %q", got, core.StatusEvicted)
	}
}
