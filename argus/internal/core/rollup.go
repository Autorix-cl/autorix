package core

import "github.com/google/uuid"

// LogicalEngine is N physical Instances of the same engine_type running in
// the same environment, rolled up into one operator-facing unit (P2-S6-T2).
// Its Status is derived from its members, not stored — a logical engine is
// healthy only when every one of its instances is, so a single degraded
// replica correctly drags the whole logical engine's reported status down
// instead of hiding behind an average.
type LogicalEngine struct {
	EngineType    string
	EnvironmentID uuid.UUID
	Status        InstanceStatus
	InstanceCount int
	Instances     []Instance
}

// statusSeverity ranks InstanceStatus from worst to best for rollup
// purposes: the logical engine's status is the worst status among its
// members. Healthy is the only status considered "all good"; every other
// status — including terminal ones like evicted/deregistered — is treated
// as worse than healthy so a rollup never silently reports healthy while a
// member is not.
var statusSeverity = map[InstanceStatus]int{
	StatusUnreachable:  0,
	StatusEvicted:      1,
	StatusDegraded:     2,
	StatusDeregistered: 3,
	StatusRegistering:  4,
	StatusPending:      5,
	StatusHealthy:      6,
}

func severityOf(s InstanceStatus) int {
	if v, ok := statusSeverity[s]; ok {
		return v
	}
	// Unknown status values are treated as maximally severe rather than
	// silently defaulting to healthy.
	return -1
}

type logicalKey struct {
	engineType    string
	environmentID uuid.UUID
}

// RollupInstances groups instances by engine_type + environment_id into
// LogicalEngines, deriving each group's Status as the worst status among its
// members. Group order follows each group's first appearance in instances.
func RollupInstances(instances []Instance) []LogicalEngine {
	order := make([]logicalKey, 0)
	groups := make(map[logicalKey][]Instance)

	for _, inst := range instances {
		key := logicalKey{engineType: inst.EngineType, environmentID: inst.EnvironmentID}
		if _, seen := groups[key]; !seen {
			order = append(order, key)
		}
		groups[key] = append(groups[key], inst)
	}

	out := make([]LogicalEngine, 0, len(order))
	for _, key := range order {
		members := groups[key]
		out = append(out, LogicalEngine{
			EngineType:    key.engineType,
			EnvironmentID: key.environmentID,
			Status:        deriveLogicalStatus(members),
			InstanceCount: len(members),
			Instances:     members,
		})
	}
	return out
}

// deriveLogicalStatus returns the worst (least healthy) status among
// members. An empty member list has no meaningful status; callers should
// never produce a LogicalEngine with zero members.
func deriveLogicalStatus(members []Instance) InstanceStatus {
	if len(members) == 0 {
		return StatusPending
	}
	worst := members[0].Status
	worstSeverity := severityOf(worst)
	for _, m := range members[1:] {
		if s := severityOf(m.Status); s < worstSeverity {
			worst = m.Status
			worstSeverity = s
		}
	}
	return worst
}

// AggregateCapabilities returns the union and intersection of the
// capabilities declared by instances — the union is every capability
// declared by at least one instance, the intersection is every capability
// declared by all of them. Both are unordered; empty input yields two empty
// (non-nil) sets.
func AggregateCapabilities(instances []Instance) (union []string, intersection []string) {
	union = []string{}
	intersection = []string{}
	if len(instances) == 0 {
		return union, intersection
	}

	counts := make(map[string]int)
	for _, inst := range instances {
		seen := make(map[string]bool, len(inst.Capabilities))
		for _, cap := range inst.Capabilities {
			if seen[cap] {
				continue // a duplicate within one instance must not inflate its count
			}
			seen[cap] = true
			counts[cap]++
		}
	}

	for cap, n := range counts {
		union = append(union, cap)
		if n == len(instances) {
			intersection = append(intersection, cap)
		}
	}
	return union, intersection
}
