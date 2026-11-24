"use client";

import * as React from "react";
import { Suspense } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import Link from "next/link";
import { Plus, AlertCircle } from "lucide-react";
import { PageHeader } from "@/components/layout/page-header";
import { ResourceTable } from "@/components/resources/resource-table";
import { Button } from "@/components/ui/button";
import { fleetInstanceDescriptor, type FleetInstanceRecord } from "@/lib/resources/fleet-instance";
import { useEnvironment } from "@/lib/environment/environment-context";
import { Can } from "@/lib/auth/use-permission";

function FleetInstancesContent() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const typeFilter = searchParams.get("type");
  const { currentEnv } = useEnvironment();

  const [instances, setInstances] = React.useState<FleetInstanceRecord[]>([]);
  const [isLoading, setIsLoading] = React.useState(true);
  const [error, setError] = React.useState<string | null>(null);

  const fetchInstances = React.useCallback(async () => {
    setError(null);
    try {
      setIsLoading(true);
      const res = await fetch(`/api/fleet/instances?environment=${encodeURIComponent(currentEnv.id)}`);
      const data = await res.json().catch(() => ({}));
      if (!res.ok) {
        throw new Error(data.error || `Failed to load fleet instances (HTTP ${res.status})`);
      }
      const list = Array.isArray(data) ? data : data.items || data.data || [];
      setInstances(list);
    } catch (err: unknown) {
      setInstances([]);
      setError(err instanceof Error ? err.message : "Failed to load fleet instances from Argus");
    } finally {
      setIsLoading(false);
    }
  }, [currentEnv.id]);

  React.useEffect(() => {
    fetchInstances();
  }, [fetchInstances]);

  const displayData = React.useMemo(() => {
    let filtered = instances;
    if (typeFilter) {
      filtered = filtered.filter((i) => i.engine_type.toLowerCase() === typeFilter.toLowerCase());
    }
    return filtered;
  }, [instances, typeFilter]);

  const handleRowClick = (record: FleetInstanceRecord) => {
    router.push(`/fleet/instances/${encodeURIComponent(record.id)}`);
  };

  return (
    <div className="space-y-6">
      <PageHeader
        breadcrumbs={[{ label: "Fleet", href: "/fleet" }, { label: "Instances" }]}
        title="Fleet Instances"
        description="All registered engine instances, versions, advertise addresses and heartbeat health."
        badge="Argus Registry"
        actions={
          <Can perform="fleet:manage">
            <Button asChild size="sm" className="h-8 gap-1.5 text-xs">
              <Link href="/fleet/enroll">
                <Plus className="h-3.5 w-3.5" aria-hidden="true" />
                Enroll Instance
              </Link>
            </Button>
          </Can>
        }
      />

      {error && (
        <div
          role="alert"
          className="p-4 rounded-xl border border-amber-500/40 bg-amber-500/10 text-amber-300 text-xs flex items-center justify-between gap-3"
        >
          <div className="flex items-center gap-2">
            <AlertCircle className="h-4 w-4 shrink-0" aria-hidden="true" />
            <span>{error}</span>
          </div>
          <Button type="button" variant="outline" size="sm" onClick={fetchInstances} className="h-7 text-xs">
            Retry
          </Button>
        </div>
      )}

      <ResourceTable
        descriptor={fleetInstanceDescriptor}
        data={displayData}
        isLoading={isLoading}
        onRefresh={fetchInstances}
        onRowClick={handleRowClick}
      />
    </div>
  );
}

export default function FleetInstancesPage() {
  return (
    <Suspense fallback={<div className="p-8 text-xs text-muted-foreground font-mono">Loading instances...</div>}>
      <FleetInstancesContent />
    </Suspense>
  );
}
