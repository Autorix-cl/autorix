import type { TupleData } from "@/app/nexus/tuple-builder";
import type { Tuple as GridTuple } from "@/app/nexus/tuple-grid";
import type { CheckResponse, Tuple as ApiTuple } from "./schemas/nexus";
import { getCsrfHeaders } from "./client";

export interface DeleteTuplesResult {
  status: string;
}

type RawTupleRecord = Partial<ApiTuple> & Partial<GridTuple>;

function normalizeTuple(raw: RawTupleRecord): GridTuple {
  const objectType = raw.objectType ?? raw.namespace ?? "";
  const objectId = raw.objectId ?? raw.object ?? "";
  const relation = raw.relation ?? "";
  const subjectType = raw.subjectType ?? raw.subject_namespace ?? "user";
  const subjectId = raw.subjectId ?? raw.subject_id ?? "";
  const subjectRelation = raw.subjectRelation ?? (raw.subject_relation || undefined);
  const subjectSuffix = subjectRelation ? `#${subjectRelation}` : "";
  const id = raw.id ?? `${objectType}:${objectId}#${relation}@${subjectType}:${subjectId}${subjectSuffix}`;

  return {
    id,
    objectType,
    objectId,
    relation,
    subjectType,
    subjectId,
    subjectRelation,
  };
}

export async function fetchSchema(): Promise<string> {
  const res = await fetch("/api/nexus/schema");
  if (!res.ok) throw new Error("Failed to fetch schema");
  const data = (await res.json()) as { schema?: string };
  return data.schema ?? "";
}

export async function updateSchema(schema: string): Promise<string> {
  const res = await fetch("/api/nexus/schema", {
    method: "PUT",
    headers: { "Content-Type": "application/json", ...getCsrfHeaders() },
    body: JSON.stringify({ schema }),
  });
  if (!res.ok) throw new Error("Failed to update schema");
  const data = (await res.json()) as { schema?: string };
  return data.schema ?? "";
}

export async function fetchTuples(): Promise<GridTuple[]> {
  const res = await fetch("/api/nexus/tuples");
  if (!res.ok) throw new Error("Failed to fetch tuples");
  const data = (await res.json()) as RawTupleRecord[] | { tuples?: RawTupleRecord[]; data?: RawTupleRecord[] };
  const list = Array.isArray(data) ? data : (data.tuples ?? data.data ?? []);
  return list.map(normalizeTuple);
}

export async function addTuple(tuple: TupleData): Promise<ApiTuple[]> {
  const res = await fetch("/api/nexus/tuples", {
    method: "POST",
    headers: { "Content-Type": "application/json", ...getCsrfHeaders() },
    body: JSON.stringify(tuple),
  });
  if (!res.ok) throw new Error("Failed to add tuple");
  return (await res.json()) as ApiTuple[];
}

export async function deleteTuples(ids: string[]): Promise<DeleteTuplesResult> {
  const encoded = ids.map((id) => encodeURIComponent(id)).join(",");
  const res = await fetch(`/api/nexus/tuples?ids=${encoded}`, {
    method: "DELETE",
    headers: { ...getCsrfHeaders() },
  });
  if (!res.ok) throw new Error("Failed to delete tuples");
  return (await res.json()) as DeleteTuplesResult;
}

export async function checkAccess(query: {
  subject: string;
  relation: string;
  object: string;
  context?: Record<string, unknown>;
}): Promise<CheckResponse> {
  const res = await fetch("/api/nexus/check", {
    method: "POST",
    headers: { "Content-Type": "application/json", ...getCsrfHeaders() },
    body: JSON.stringify(query),
  });
  if (!res.ok) throw new Error("Failed to run check");
  return (await res.json()) as CheckResponse;
}
