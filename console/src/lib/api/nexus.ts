import { TupleData } from "@/app/nexus/tuple-builder";

export async function fetchSchema(): Promise<string> {
  const res = await fetch("/api/nexus/schema");
  if (!res.ok) throw new Error("Failed to fetch schema");
  const data = await res.json();
  return data.schema;
}

export async function updateSchema(schema: string): Promise<string> {
  const res = await fetch("/api/nexus/schema", {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ schema }),
  });
  if (!res.ok) throw new Error("Failed to update schema");
  const data = await res.json();
  return data.schema;
}

export async function fetchTuples(): Promise<unknown[]> {
  const res = await fetch("/api/nexus/tuples");
  if (!res.ok) throw new Error("Failed to fetch tuples");
  const data = await res.json();
  return data.tuples;
}

export async function addTuple(tuple: TupleData): Promise<unknown> {
  const res = await fetch("/api/nexus/tuples", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(tuple),
  });
  if (!res.ok) throw new Error("Failed to add tuple");
  return res.json();
}

export async function deleteTuples(ids: string[]): Promise<unknown> {
  const res = await fetch(`/api/nexus/tuples?ids=${ids.join(",")}`, {
    method: "DELETE",
  });
  if (!res.ok) throw new Error("Failed to delete tuples");
  return res.json();
}

export async function checkAccess(query: { subject: string, relation: string, object: string }): Promise<unknown> {
  const res = await fetch("/api/nexus/check", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(query),
  });
  if (!res.ok) throw new Error("Failed to run check");
  return res.json();
}
