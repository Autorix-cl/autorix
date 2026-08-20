package grpc

import (
	"time"

	argusv1 "github.com/autorix/argus/api/autorix/argus/v1"
	"github.com/autorix/argus/internal/core"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// toProtoInstance maps a core.Instance to its wire shape. envSlug is passed
// in rather than looked up here so callers can resolve it once (or batch it
// across a whole ListInstances page) instead of every conversion issuing its
// own environment query.
func toProtoInstance(inst core.Instance, envSlug string) *argusv1.Instance {
	return &argusv1.Instance{
		Id:            inst.ID.String(),
		EngineType:    inst.EngineType,
		InstanceId:    inst.InstanceID,
		Environment:   envSlug,
		Version:       inst.Version,
		BuildSha:      inst.BuildSHA,
		SchemaVersion: inst.SchemaVersion,
		Capabilities:  inst.Capabilities,
		Endpoints: &argusv1.Endpoints{
			RestUrl: inst.Endpoints.RESTURL,
			GrpcUrl: inst.Endpoints.GRPCURL,
		},
		Status:          string(inst.Status),
		FirstSeenAt:     timestamppb.New(inst.FirstSeenAt),
		LastHeartbeatAt: optionalTimestamp(inst.LastHeartbeatAt),
	}
}

func toProtoEvents(events []core.InstanceEvent) []*argusv1.InstanceEvent {
	out := make([]*argusv1.InstanceEvent, 0, len(events))
	for _, ev := range events {
		detail, err := structpb.NewStruct(ev.Detail)
		if err != nil {
			// A detail map that cannot round-trip through structpb (e.g. an
			// unsupported value type) must not take down the whole response —
			// surface the event without its detail rather than failing GetInstance.
			detail = &structpb.Struct{}
		}
		out = append(out, &argusv1.InstanceEvent{
			Type:       string(ev.Type),
			Detail:     detail,
			OccurredAt: timestamppb.New(ev.OccurredAt),
		})
	}
	return out
}

func toProtoDependencies(deps []core.InstanceDependency) []*argusv1.InstanceDependencyEdge {
	out := make([]*argusv1.InstanceDependencyEdge, 0, len(deps))
	for _, d := range deps {
		edge := &argusv1.InstanceDependencyEdge{
			DependsOnEngineType: d.DependsOnEngineType,
			Declared:            d.Declared,
			LastProbedAt:        optionalTimestamp(d.LastProbedAt),
		}
		if d.LastProbeReachable != nil {
			edge.LastProbeReachable = *d.LastProbeReachable
		}
		if d.LastProbeLatencyMs != nil {
			edge.LastProbeLatencyMs = *d.LastProbeLatencyMs
		}
		out = append(out, edge)
	}
	return out
}

func optionalTimestamp(t *time.Time) *timestamppb.Timestamp {
	if t == nil {
		return nil
	}
	return timestamppb.New(*t)
}
