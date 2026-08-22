/**
 * Zod parsing at the API boundary (P1-S3-T4).
 */
import { z } from "zod";
import type { ApiResult } from "./client";
import { fetchJSON, type FetchJSONOptions } from "./client";

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
 * Flattens paginated envelopes to just an array. (Legacy, for components not yet upgraded to server-side pagination).
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

/**
 * Retains paginated envelope data for use with server-side pagination DataTables.
 */
export function paginatedListSchema<T extends z.ZodTypeAny>(itemSchema: T) {
  return z.object({
    data: z.array(itemSchema).nullable().transform((val) => val ?? []),
    has_more: z.boolean().optional().default(false),
    next_cursor: z.string().optional().default(""),
  });
}
