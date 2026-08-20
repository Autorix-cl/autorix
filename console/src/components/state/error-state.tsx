"use client";

import { AlertTriangle } from "lucide-react";
import { useTranslation } from "@/lib/i18n";
import type { ApiError } from "@/lib/api/client";

/**
 * The shared "something failed" state (P1-S3-T5). Distinct from
 * NotConnectedState: this renders for a request that reached the engine and
 * got an error back (validation, unauthorized, engine-error, unknown) — an
 * engine that couldn't be reached at all gets NotConnectedState instead, so
 * the two failure modes read differently to the operator.
 */
export function ErrorState({ error, onRetry }: { error?: ApiError; onRetry?: () => void }) {
  const { t } = useTranslation();

  const description =
    error?.kind === "unauthorized" ? t("state.errorUnauthorized") : (error?.message ?? t("state.errorDefault"));

  return (
    <div role="alert" className="flex flex-col items-center justify-center gap-2 py-16 text-center">
      <AlertTriangle className="h-8 w-8 text-destructive" aria-hidden="true" />
      <h3 className="text-sm font-medium">{t("state.errorTitle")}</h3>
      <p className="max-w-sm text-sm text-muted-foreground">{description}</p>
      {onRetry ? (
        <button
          type="button"
          onClick={onRetry}
          className="mt-3 rounded-md border border-border px-3 py-1.5 text-sm font-medium hover:bg-muted"
        >
          {t("state.retry")}
        </button>
      ) : null}
    </div>
  );
}
