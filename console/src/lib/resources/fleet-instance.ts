import { z } from "zod";
import { defineResourceDescriptor } from "./descriptor";

export const fleetInstanceSchema = z.object({
  id: z.string(),
  engine_type: z.string(),
  environment_id: z.string(),
  status: z.enum(["healthy", "degraded", "unreachable", "evicted"]),
  version: z.string().optional(),
  advertise_addr: z.string().optional(),
  last_heartbeat_at: z.string().optional(),
});

export type FleetInstanceRecord = z.infer<typeof fleetInstanceSchema>;

export const fleetInstanceDescriptor = defineResourceDescriptor<FleetInstanceRecord>({
  id: "instances",
  name: "Instance",
  pluralName: "Fleet Instances",
  service: "argus",
  basePath: "/api/fleet/instances",
  schema: fleetInstanceSchema,
  requiredPermissions: {
    read: "fleet:read",
    delete: "fleet:admin",
  },
  columns: [
    { key: "id", label: "Instance ID", sortable: true },
    { key: "engine_type", label: "Engine Type", sortable: true, filterable: true },
    { key: "environment_id", label: "Environment", sortable: true, filterable: true },
    { key: "status", label: "Status", sortable: true, filterable: true },
    { key: "version", label: "Version", sortable: true },
    { key: "advertise_addr", label: "Address" },
  ],
  defaultSort: { field: "engine_type", order: "asc" },
});
