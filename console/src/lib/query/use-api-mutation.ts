/**
 * A TanStack Query mutation wired to the toast system by default (P1-S3-T1
 * + T6 combined): success copy names what happened, error copy names the
 * fix, sourced straight from the ApiError taxonomy every BFF route and the
 * typed client already produce.
 */
import { useMutation, type UseMutationOptions } from "@tanstack/react-query";
import type { ApiResult } from "../api/client";
import { toastApiError, toastSuccess } from "../toast";

export interface UseApiMutationOptions<TData, TVariables> extends Omit<
  UseMutationOptions<TData, Error, TVariables>,
  "mutationFn" | "onSuccess"
> {
  /** Shown as a success toast. Omit to mutate silently on success. */
  successMessage?: string | ((data: TData) => string);
  /** Called after a successful mutation, once the success toast has fired. */
  onSuccess?: (data: TData, variables: TVariables) => void;
  /** Overrides the default fix copy shown alongside the error message. */
  errorFix?: string;
}

export function useApiMutation<TData, TVariables>(
  mutationFn: (variables: TVariables) => Promise<ApiResult<TData>>,
  options: UseApiMutationOptions<TData, TVariables> = {},
) {
  const { successMessage, onSuccess, errorFix, ...rest } = options;

  return useMutation<TData, Error, TVariables>({
    ...rest,
    mutationFn: async (variables) => {
      const result = await mutationFn(variables);
      if (!result.ok) {
        toastApiError(result.error, errorFix);
        throw new Error(result.error.message);
      }
      if (successMessage) {
        toastSuccess(typeof successMessage === "function" ? successMessage(result.data) : successMessage);
      }
      onSuccess?.(result.data, variables);
      return result.data;
    },
  });
}
