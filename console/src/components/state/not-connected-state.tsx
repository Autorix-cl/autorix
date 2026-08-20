"use client";

import { PlugZap } from "lucide-react";
import { useTranslation } from "@/lib/i18n";

/**
 * The shared "this engine could not be reached" state (P1-S3-T5). Per the
 * roadmap's "degrade by feature, not by page" principle: an unreachable
 * engine renders this in its own section while the rest of the console
 * keeps working — never a broken page, never invented data.
 */
export function NotConnectedState({ engineName, onRetry }: { engineName: string; onRetry?: () => void }) {
  const { t } = useTranslation();

  return (
    <div className="flex flex-col items-center justify-center gap-2 py-16 text-center">
      <PlugZap className="h-8 w-8 text-muted-foreground" aria-hidden="true" />
      <h3 className="text-sm font-medium">{t("state.notConnectedTitle", { engine: engineName })}</h3>
      <p className="max-w-sm text-sm text-muted-foreground">
        {t("state.notConnectedDescription", { engine: engineName })}
      </p>
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
