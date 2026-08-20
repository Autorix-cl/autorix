/**
 * Mutation feedback (P1-S3-T6): success copy names what happened, error
 * copy names the fix. Mutations previously gave no feedback at all.
 */
import { toast } from "sonner";
import type { ApiError } from "./api/client";

export function toastSuccess(message: string): void {
  toast.success(message);
}

function defaultFixFor(kind: ApiError["kind"]): string {
  switch (kind) {
    case "engine-unreachable":
      return "Check that the engine is running and reachable, then try again.";
    case "unauthorized":
      return "Sign in again and retry.";
    case "validation":
      return "Fix the highlighted fields and try again.";
    case "engine-error":
      return "The engine reported an internal error. Try again in a moment.";
    default:
      return "Try again, or check the engine's logs if this keeps happening.";
  }
}

export function toastApiError(error: ApiError, fix?: string): void {
  toast.error(error.message, { description: fix ?? defaultFixFor(error.kind) });
}
