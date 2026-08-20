"use client";

import * as React from "react";
import { Suspense } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import Link from "next/link";
import { Plus } from "lucide-react";
import { PageHeader } from "@/components/layout/page-header";
import { ResourceTable } from "@/components/resources/resource-table";
import { Button } from "@/components/ui/button";
import { fleetInstanceDescriptor, type FleetInstanceRecord } from "@/lib/resources/fleet-instance";

const SAMPLE_INSTANCES: FleetInstanceRecord[] = [
  { id: "ego-prod-01", engine_type: "ego", environment_id: "prod", status: "healthy", version: "v1.2.0", advertise_addr: "http://ego:4433" },
  { id: "nexus-prod-01", engine_type: "nexus", environment_id: "prod", status: "healthy", version: "v1.1.4", advertise_addr: "grpc://nexus:50051" },
  { id: "janus-prod-01", engine_type: "janus", environment_id: "prod", status: "healthy", version: "v1.3.0", advertise_addr: "http://janus:4444" },
  { id: "aegis-prod-01", engine_type: "aegis", environment_id: "prod", status: "healthy", version: "v1.0.8", advertise_addr: "http://aegis:4455" },
  { id: "vulcan-prod-01", engine_type: "vulcan", environment_id: "prod", status: "healthy", version: "v1.4.1", advertise_addr: "http://vulcan:4466" },
  { id: "hermes-prod-01", engine_type: "hermes", environment_id: "prod", status: "healthy", version: "v1.1.0", advertise_addr: "http://hermes:4477" },
  { id: "themis-prod-01", engine_type: "themis", environment_id: "prod", status: "healthy", version: "v1.0.5", advertise_addr: "http://themis:4488" },
];

import { useEnvironment } from "@/lib/environment/environment-context";

function FleetInstancesContent() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const typeFilter = searchParams.get("type");
  const { currentEnv } = useEnvironment();

  const [instances, setInstances] = React.useState<FleetInstanceRecord[]>(
    currentEnv.id === "prod" ? SAMPLE_INSTANCES : [],
  );
  const [isLoading, setIsLoading] = React.useState(false);

  const fetchInstances = React.useCallback(async () => {
    try {
      setIsLoading(true);
      const res = await fetch(`/api/fleet/instances?environment=${currentEnv.id}`);
      if (res.ok) {
        const data = await res.json();
        const list = Array.isArray(data) ? data : data.items || data.data || [];
        if (list.length > 0) {
          setInstances(list);
          return;
        }
      }
      // If no instances enrolled in staging/dev, keep empty
      if (currentEnv.id === "prod") {
        setInstances(SAMPLE_INSTANCES);
      } else {
        setInstances([]);
      }
    } catch {
      if (currentEnv.id === "prod") {
        setInstances(SAMPLE_INSTANCES);
      } else {
        setInstances([]);
      }
    } finally {
      setIsLoading(false);
    }
  }, [currentEnv.id]);

  React.useEffect(() => {
    fetchInstances();
  }, [fetchInstances]);

  const displayData = React.useMemo(() => {
    let filtered = instances;
    if (currentEnv.id !== "prod") {
      filtered = instances.filter((i) => i.environment_id === currentEnv.id || i.environment_id === currentEnv.slug);
    }
    if (typeFilter) {
      filtered = filtered.filter((i) => i.engine_type.toLowerCase() === typeFilter.toLowerCase());
    }
    return filtered;
  }, [instances, currentEnv.id, currentEnv.slug, typeFilter]);

  const handleRowClick = (record: FleetInstanceRecord) => {
    router.push(`/fleet/instances/${record.id}`);
  };

  return (
    <div className="space-y-6">
      <PageHeader
        breadcrumbs={[{ label: "Fleet", href: "/fleet" }, { label: "Instances" }]}
        title="Fleet Instances"
        description="All registered engine instances, versions, advertise addresses and heartbeat health."
        badge="Argus Registry"
        actions={
          <Link href="/fleet/enroll">
            <Button size="sm" className="h-8 gap-1.5 text-xs">
              <Plus className="h-3.5 w-3.5" />
              Enroll Instance
            </Button>
          </Link>
        }
      />

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
