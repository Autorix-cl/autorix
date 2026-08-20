/**
 * Bridges the ApiResult discriminated union (from fetchJSON/fetchAndParse)
 * into TanStack Query's throw-on-error model, so pages get typed ApiError
 * — with its `kind` — as `error` directly, instead of a generic Error a
 * component would have to re-derive the failure mode from (P1-S3-T1).
 */
import { useQuery, type QueryKey, type UseQueryOptions } from "@tanstack/react-query";
import type { ApiError, ApiResult } from "../api/client";

export function useApiQuery<TData>(
  queryKey: QueryKey,
  queryFn: () => Promise<ApiResult<TData>>,
  options?: Omit<UseQueryOptions<TData, ApiError, TData, QueryKey>, "queryKey" | "queryFn">,
) {
  return useQuery<TData, ApiError, TData, QueryKey>({
    queryKey,
    queryFn: async () => {
      const result = await queryFn();
      if (!result.ok) throw result.error;
      return result.data;
    },
    ...options,
  });
}
