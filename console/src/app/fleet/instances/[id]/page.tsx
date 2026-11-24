"use client";

import * as React from "react";
import { useParams, useRouter } from "next/navigation";
import { Server, Activity, AlertCircle, Loader2 } from "lucide-react";
import { toast } from "sonner";
import { ResourceDetailScaffold } from "@/components/resources/resource-detail-scaffold";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { fleetInstanceDescriptor, type FleetInstanceRecord } from "@/lib/resources/fleet-instance";
import { formatDateTime } from "@/lib/format/date-time";
import { usePermission } from "@/lib/auth/use-permission";

function readCsrfCookie(): string | undefined {
  if (typeof document === "undefined") return undefined;
  const match = document.cookie.match(/(?:^|;\s*)autorix_csrf=([^;]+)/);
  return match ? decodeURIComponent(match[1]) : undefined;
}

export default function FleetInstanceDetailPage() {
  const params = useParams();
  const router = useRouter();
  const id = String(params?.id || "");
  const { can } = usePermission();
  const canManageFleet = can("fleet:manage");

  const [record, setRecord] = React.useState<FleetInstanceRecord | null>(null);
  const [loading, setLoading] = React.useState(true);
  const [error, setError] = React.useState<string | null>(null);

  const fetchInstance = React.useCallback(async () => {
    if (!id) return;
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`/api/fleet/instances/${encodeURIComponent(id)}`);
      const data = await res.json().catch(() => ({}));
      if (!res.ok) {
        throw new Error(data.error || `Failed to load instance (HTTP ${res.status})`);
      }
      setRecord(data as FleetInstanceRecord);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "Instance not found or Argus unavailable");
      setRecord(null);
    } finally {
      setLoading(false);
    }
  }, [id]);

  React.useEffect(() => {
    fetchInstance();
  }, [fetchInstance]);

  const handleDelete = async () => {
    try {
      const csrf = readCsrfCookie();
      const res = await fetch(`/api/fleet/instances/${encodeURIComponent(id)}`, {
        method: "DELETE",
        headers: csrf ? { "X-CSRF-Token": csrf } : undefined,
      });
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        throw new Error(data.error || `Failed to remove instance (HTTP ${res.status})`);
      }
      toast.success(`Instance ${id} removed from fleet`);
      router.push("/fleet/instances");
    } catch (err: unknown) {
      toast.error(err instanceof Error ? err.message : "Failed to remove instance");
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center py-20 text-xs text-muted-foreground font-mono gap-2">
        <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
        <span>Loading instance telemetry...</span>
      </div>
    );
  }

  if (error || !record) {
    return (
      <Card className="max-w-lg mx-auto my-12 border-destructive/40 bg-card">
        <CardContent className="p-6 text-center space-y-4">
          <AlertCircle className="h-8 w-8 text-destructive mx-auto" aria-hidden="true" />
          <div className="space-y-1">
            <h2 className="text-sm font-semibold text-foreground">Unable to Load Fleet Instance</h2>
            <p className="text-xs text-muted-foreground">{error || `Instance "${id}" could not be found.`}</p>
          </div>
          <div className="flex items-center justify-center gap-2">
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => router.push("/fleet/instances")}
              className="text-xs"
            >
              Back to Instances
            </Button>
            <Button type="button" size="sm" onClick={fetchInstance} className="text-xs">
              Retry
            </Button>
          </div>
        </CardContent>
      </Card>
    );
  }

  return (
    <ResourceDetailScaffold
      descriptor={fleetInstanceDescriptor}
      record={record}
      title={record.id}
      subtitle={`arn:autorix:${record.engine_type}:${record.environment_id || "prod"}:default:instance/${record.id}`}
      onBack={() => router.push("/fleet/instances")}
      onDelete={canManageFleet ? handleDelete : undefined}
    >
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <Card className="border-border/80 bg-card">
          <CardHeader className="pb-3">
            <CardTitle className="text-sm font-semibold flex items-center gap-2">
              <Server className="h-4 w-4 text-primary" aria-hidden="true" />
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
              <span className="font-mono">{record.environment_id || "prod"}</span>
            </div>
            <div className="flex justify-between py-1.5 border-b border-border/50">
              <span className="text-muted-foreground">Status</span>
              <Badge
                variant="outline"
                className={
                  record.status === "healthy"
                    ? "text-[10px] font-mono text-emerald-400 border-emerald-500/30"
                    : "text-[10px] font-mono text-amber-400 border-amber-500/30"
                }
              >
                {record.status}
              </Badge>
            </div>
            <div className="flex justify-between py-1.5">
              <span className="text-muted-foreground">Advertise Address</span>
              <span className="font-mono text-primary">{record.advertise_addr || "—"}</span>
            </div>
          </CardContent>
        </Card>

        <Card className="border-border/80 bg-card">
          <CardHeader className="pb-3">
            <CardTitle className="text-sm font-semibold flex items-center gap-2">
              <Activity className="h-4 w-4 text-emerald-400" aria-hidden="true" />
              Heartbeat & Registration
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-2 text-xs">
            <div className="flex justify-between py-1.5 border-b border-border/50">
              <span className="text-muted-foreground">Version</span>
              <span className="font-mono font-semibold">{record.version || "—"}</span>
            </div>
            <div className="flex justify-between py-1.5 border-b border-border/50">
              <span className="text-muted-foreground">Last Heartbeat</span>
              <span className="font-mono">
                {record.last_heartbeat_at ? formatDateTime(record.last_heartbeat_at) : "—"}
              </span>
            </div>
            <div className="flex justify-between py-1.5">
              <span className="text-muted-foreground">Trust Mode</span>
              <Badge variant="outline" className="text-[10px] font-mono text-blue-400 border-blue-500/30">
                Signed HMAC Heartbeat
              </Badge>
            </div>
          </CardContent>
        </Card>
      </div>
    </ResourceDetailScaffold>
  );
}
