package core_test

import (
	"reflect"
	"sort"
	"testing"

	"github.com/autorix/argus/internal/core"
	"github.com/google/uuid"
)

func TestRollupInstances_HealthyOnlyWhenAllMembersHealthy(t *testing.T) {
	env := uuid.New()
	instances := []core.Instance{
		{ID: uuid.New(), EngineType: "ego", EnvironmentID: env, Status: core.StatusHealthy},
		{ID: uuid.New(), EngineType: "ego", EnvironmentID: env, Status: core.StatusHealthy},
	}

	rollups := core.RollupInstances(instances)
	if len(rollups) != 1 {
		t.Fatalf("expected 1 logical engine, got %d", len(rollups))
	}
	got := rollups[0]
	if got.Status != core.StatusHealthy {
		t.Fatalf("expected healthy rollup when every member is healthy, got %s", got.Status)
	}
	if got.InstanceCount != 2 {
		t.Fatalf("expected instance count 2, got %d", got.InstanceCount)
	}
}

func TestRollupInstances_DegradedWhenAnyMemberIsNotHealthy(t *testing.T) {
	env := uuid.New()
	instances := []core.Instance{
		{ID: uuid.New(), EngineType: "ego", EnvironmentID: env, Status: core.StatusHealthy},
		{ID: uuid.New(), EngineType: "ego", EnvironmentID: env, Status: core.StatusDegraded},
	}

	rollups := core.RollupInstances(instances)
	if len(rollups) != 1 {
		t.Fatalf("expected 1 logical engine, got %d", len(rollups))
	}
	if rollups[0].Status != core.StatusDegraded {
		t.Fatalf("expected degraded rollup, got %s", rollups[0].Status)
	}
}

func TestRollupInstances_UnreachableOutranksDegraded(t *testing.T) {
	env := uuid.New()
	instances := []core.Instance{
		{ID: uuid.New(), EngineType: "ego", EnvironmentID: env, Status: core.StatusDegraded},
		{ID: uuid.New(), EngineType: "ego", EnvironmentID: env, Status: core.StatusUnreachable},
		{ID: uuid.New(), EngineType: "ego", EnvironmentID: env, Status: core.StatusHealthy},
	}

	rollups := core.RollupInstances(instances)
	if rollups[0].Status != core.StatusUnreachable {
		t.Fatalf("expected unreachable to be the worst-case rollup status, got %s", rollups[0].Status)
	}
}

func TestRollupInstances_GroupsByEngineTypeAndEnvironment(t *testing.T) {
	envA := uuid.New()
	envB := uuid.New()
	instances := []core.Instance{
		{ID: uuid.New(), EngineType: "ego", EnvironmentID: envA, Status: core.StatusHealthy},
		{ID: uuid.New(), EngineType: "janus", EnvironmentID: envA, Status: core.StatusHealthy},
		{ID: uuid.New(), EngineType: "ego", EnvironmentID: envB, Status: core.StatusHealthy},
	}

	rollups := core.RollupInstances(instances)
	if len(rollups) != 3 {
		t.Fatalf("expected 3 distinct logical engines (engine_type x environment), got %d", len(rollups))
	}
}

func TestAggregateCapabilities_UnionAndIntersection(t *testing.T) {
	instances := []core.Instance{
		{Capabilities: []string{"argon2id", "mfa"}},
		{Capabilities: []string{"argon2id", "webauthn"}},
	}

	union, intersection := core.AggregateCapabilities(instances)

	sort.Strings(union)
	sort.Strings(intersection)

	wantUnion := []string{"argon2id", "mfa", "webauthn"}
	wantIntersection := []string{"argon2id"}

	if !reflect.DeepEqual(union, wantUnion) {
		t.Fatalf("union: expected %v, got %v", wantUnion, union)
	}
	if !reflect.DeepEqual(intersection, wantIntersection) {
		t.Fatalf("intersection: expected %v, got %v", wantIntersection, intersection)
	}
}

func TestAggregateCapabilities_EmptyInstancesReturnsEmptySets(t *testing.T) {
	union, intersection := core.AggregateCapabilities(nil)
	if len(union) != 0 || len(intersection) != 0 {
		t.Fatalf("expected empty union/intersection for no instances, got union=%v intersection=%v", union, intersection)
	}
}

func TestAggregateCapabilities_NoOverlapYieldsEmptyIntersection(t *testing.T) {
	instances := []core.Instance{
		{Capabilities: []string{"a"}},
		{Capabilities: []string{"b"}},
	}
	_, intersection := core.AggregateCapabilities(instances)
	if len(intersection) != 0 {
		t.Fatalf("expected empty intersection for disjoint capability sets, got %v", intersection)
	}
}
