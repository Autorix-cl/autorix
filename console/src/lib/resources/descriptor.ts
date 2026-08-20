import { z } from "zod";
import type { BACKEND_URLS } from "../api-config";

export interface ColumnDescriptor<T> {
  key: keyof T & string;
  label: string;
  sortable?: boolean;
  filterable?: boolean;
  render?: (value: unknown, record: T) => React.ReactNode;
}

export interface ResourcePermissions {
  read: string;
  create?: string;
  update?: string;
  delete?: string;
}

export interface ResourceDescriptor<T> {
  id: string;
  name: string;
  pluralName: string;
  service: keyof typeof BACKEND_URLS;
  basePath: string;
  schema: z.ZodObject<z.ZodRawShape>;
  requiredPermissions?: ResourcePermissions;
  columns: ColumnDescriptor<T>[];
  defaultSort?: {
    field: keyof T & string;
    order: "asc" | "desc";
  };
}

export function defineResourceDescriptor<T extends Record<string, unknown>>(
  config: ResourceDescriptor<T>
): ResourceDescriptor<T> {
  // Validate column keys exist in Zod schema shape
  const schemaKeys = Object.keys(config.schema.shape);
  for (const col of config.columns) {
    if (!schemaKeys.includes(col.key)) {
      throw new Error(
        `Invalid column key '${col.key}' in resource descriptor '${config.id}'. Must match schema shape: ${schemaKeys.join(", ")}`
      );
    }
  }

  return config;
}
