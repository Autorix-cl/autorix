export interface ThemisPolicyRecord {
  id: string;
  name: string;
  expression: string;
  priority: number;
  enabled: boolean;
  tenant_id?: string;
  description?: string;
  labels?: Record<string, string>;
}

export interface ThemisDb {
  getPolicies: () => ThemisPolicyRecord[];
  getPolicyById: (id: string) => ThemisPolicyRecord | null;
  addPolicy: (policy: ThemisPolicyRecord) => void;
  togglePolicy: (id: string) => boolean;
  deletePolicy: (id: string) => boolean;
}

export const themisDb: ThemisDb = {
  getPolicies: () => [],
  getPolicyById: () => null,
  addPolicy: () => {},
  togglePolicy: () => false,
  deletePolicy: () => false,
};
