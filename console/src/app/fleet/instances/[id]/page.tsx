"use client";

import * as React from "react";
import { useParams, useRouter } from "next/navigation";
import {
  Server,
  Activity,
} from "lucide-react";
import { ResourceDetailScaffold } from "@/components/resources/resource-detail-scaffold";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { fleetInstanceDescriptor, type FleetInstanceRecord } from "@/lib/resources/fleet-instance";

export default function FleetInstanceDetailPage() {
  const params = useParams();
  const router = useRouter();
  const id = params?.id as string;

  const [record] = React.useState<FleetInstanceRecord>({
    id: id || "ego-prod-01",
    engine_type: "ego",
    environment_id: "prod",
    status: "healthy",
    version: "v1.2.0",
    advertise_addr: "http://ego:4433",
    last_heartbeat_at: new Date().toISOString(),
  });

  const handleDelete = async () => {
    try {
      await fetch(`/api/fleet/instances/${id}`, { method: "DELETE" });
      router.push("/fleet/instances");
    } catch {
      router.push("/fleet/instances");
    }
  };

  return (
    <ResourceDetailScaffold
      descriptor={fleetInstanceDescriptor}
      record={record}
      title={record.id}
      subtitle={`arn:autorix:${record.engine_type}:${record.environment_id}:default:instance/${record.id}`}
      onBack={() => router.push("/fleet/instances")}
      onDelete={handleDelete}
    >
      {/* Overview Tab Content */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <Card className="border-border/80 bg-card">
          <CardHeader className="pb-3">
            <CardTitle className="text-sm font-semibold flex items-center gap-2">
              <Server className="h-4 w-4 text-primary" />
              Instance Properties
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-2 text-xs">
            <div className="flex justify-between py-1.5 border-b border-border/50">
              <span className="text-muted-foreground">Engine Type</span>
              <span className="font-mono font-semibold uppercase">{record.engine_type}</span>
            </div>
            <div className="flex justify-between py-1.5 border-b border-border/50">
              <span className="text-muted-foreground">Environment</span>
              <span className="font-mono">{record.environment_id}</span>
            </div>
            <div className="flex justify-between py-1.5 border-b border-border/50">
              <span className="text-muted-foreground">Status</span>
              <Badge variant="outline" className="text-[10px] font-mono text-emerald-400 border-emerald-500/30">
                {record.status}
              </Badge>
            </div>
            <div className="flex justify-between py-1.5">
              <span className="text-muted-foreground">Advertise Address</span>
              <span className="font-mono text-primary">{record.advertise_addr}</span>
            </div>
          </CardContent>
        </Card>

        <Card className="border-border/80 bg-card">
          <CardHeader className="pb-3">
            <CardTitle className="text-sm font-semibold flex items-center gap-2">
              <Activity className="h-4 w-4 text-emerald-400" />
              Heartbeat & Registration
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-2 text-xs">
            <div className="flex justify-between py-1.5 border-b border-border/50">
              <span className="text-muted-foreground">Version</span>
              <span className="font-mono font-semibold">{record.version}</span>
            </div>
            <div className="flex justify-between py-1.5 border-b border-border/50">
              <span className="text-muted-foreground">Heartbeat Cadence</span>
              <span className="font-mono">10s (Signed HMAC)</span>
            </div>
            <div className="flex justify-between py-1.5">
              <span className="text-muted-foreground">Trust Mode</span>
              <Badge variant="outline" className="text-[10px] font-mono text-blue-400 border-blue-500/30">
                Verified (aet_ token)
              </Badge>
            </div>
          </CardContent>
        </Card>
      </div>
    </ResourceDetailScaffold>
  );
}
