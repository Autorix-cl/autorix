import { describe, it, expect } from "vitest";
import { z } from "zod";
import { defineResourceDescriptor } from "./descriptor";

const identitySchema = z.object({
  id: z.string().uuid(),
  email: z.string().email(),
  status: z.enum(["active", "suspended", "invited"]),
  created_at: z.string(),
});

describe("Resource Descriptor (P4-S2-T1)", () => {
  it("creates a valid typed resource descriptor with fields, columns and permissions", () => {
    const descriptor = defineResourceDescriptor({
      id: "identities",
      name: "Identity",
      pluralName: "Identities",
      service: "ego",
      basePath: "/api/identities",
      schema: identitySchema,
      requiredPermissions: {
        read: "identities:read",
        create: "identities:write",
        update: "identities:write",
        delete: "identities:write",
      },
      columns: [
        { key: "email", label: "Email", sortable: true, filterable: true },
        { key: "status", label: "Status", sortable: true, filterable: true },
        { key: "created_at", label: "Created At", sortable: true },
      ],
      defaultSort: { field: "created_at", order: "desc" },
    });

    expect(descriptor.id).toBe("identities");
    expect(descriptor.pluralName).toBe("Identities");
    expect(descriptor.service).toBe("ego");
    expect(descriptor.columns).toHaveLength(3);
    expect(descriptor.requiredPermissions?.read).toBe("identities:read");
  });

  it("validates that columns map to fields declared in schema", () => {
    expect(() =>
      defineResourceDescriptor({
        id: "invalid-resource",
        name: "Invalid",
        pluralName: "Invalids",
        service: "ego",
        basePath: "/api/invalid",
        schema: identitySchema,
        columns: [
          { key: "unknown_column" as unknown as "email", label: "Unknown" },
        ],
      })
    ).toThrow();
  });
});
