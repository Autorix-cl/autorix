package core_test

import (
	"testing"

	"github.com/autorix/argus/internal/core"
	"github.com/google/uuid"
)

func TestAggregateFleetMetrics(t *testing.T) {
	instances := []core.Instance{
		{
			ID:          uuid.New(),
			EngineType:  "nexus",
			Status:      core.StatusHealthy,
		},
		{
			ID:          uuid.New(),
			EngineType:  "aegis",
			Status:      core.StatusHealthy,
		},
		{
			ID:          uuid.New(),
			EngineType:  "ego",
			Status:      core.StatusDegraded,
		},
	}

	summary := core.AggregateFleetMetrics(instances)

	if summary.TotalInstances != 3 {
		t.Errorf("expected 3 total instances, got %d", summary.TotalInstances)
	}
	if summary.HealthyInstances != 2 {
		t.Errorf("expected 2 healthy instances, got %d", summary.HealthyInstances)
	}
	if summary.FleetQPS <= 0 {
		t.Errorf("expected positive FleetQPS, got %f", summary.FleetQPS)
	}
	if len(summary.Engines) < 7 {
		t.Errorf("expected at least 7 known engines in summary, got %d", len(summary.Engines))
	}

	// Verify nexus engine summary
	var nexusSummary *core.EngineMetricSummary
	for i := range summary.Engines {
		if summary.Engines[i].EngineType == "nexus" {
			nexusSummary = &summary.Engines[i]
			break
		}
	}
	if nexusSummary == nil {
		t.Fatalf("nexus summary not found")
	}
	if nexusSummary.Status != "healthy" {
		t.Errorf("expected nexus status healthy, got %s", nexusSummary.Status)
	}
	if nexusSummary.AuthDecisions <= 0 {
		t.Errorf("expected positive auth decisions for nexus, got %d", nexusSummary.AuthDecisions)
	}
}
