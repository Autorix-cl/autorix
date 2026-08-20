/**
 * Zod parsing at the API boundary (P1-S3-T4). A contract change in an
 * engine now surfaces as a precise validation ApiError here, at the edge —
 * instead of a runtime crash deep inside a component that assumed a field
 * would always be present.
 */
import { z } from "zod";
import type { ApiResult } from "./client";
import { fetchJSON, type FetchJSONOptions } from "./client";

/** Parses data against schema, returning an ApiResult instead of throwing. */
export function parseWithSchema<T>(schema: z.ZodType<T>, data: unknown): ApiResult<T> {
  const result = schema.safeParse(data);
  if (result.success) {
    return { ok: true, data: result.data };
  }
  const message = result.error.issues
    .map((issue) => `${issue.path.length ? issue.path.join(".") : "(root)"}: ${issue.message}`)
    .join("; ");
  return {
    ok: false,
    error: { kind: "validation", message, cause: result.error },
  };
}

/**
 * Fetches url and parses the response against schema. A transport failure
 * (network, timeout, 4xx/5xx) short-circuits with that ApiError untouched;
 * only a structurally successful response is validated against schema.
 */
export async function fetchAndParse<T>(
  url: string,
  schema: z.ZodType<T>,
  options?: FetchJSONOptions,
): Promise<ApiResult<T>> {
  const result = await fetchJSON<unknown>(url, options);
  if (!result.ok) return result;
  return parseWithSchema(schema, result.data);
}

/**
 * Creates a schema that transparently handles both bare arrays and paginated
 * envelopes ({ data: [...], has_more?: boolean }).
 */
export function pagedListSchema<T extends z.ZodTypeAny>(itemSchema: T) {
  return z.union([
    z.array(itemSchema),
    z
      .object({
        data: z.array(itemSchema).nullable().optional(),
        has_more: z.boolean().optional(),
        next_cursor: z.string().optional(),
      })
      .transform((val) => val.data ?? []),
  ]);
}

