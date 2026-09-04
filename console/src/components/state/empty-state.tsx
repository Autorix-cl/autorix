"use client";

import type { ReactNode } from "react";
import { Inbox } from "lucide-react";
import { useTranslation } from "@/lib/i18n";

export function EmptyState({
  title,
  description,
  action,
  docsUrl,
  docsLabel,
}: {
  title?: string;
  description?: string;
  action?: ReactNode;
  docsUrl?: string;
  docsLabel?: string;
}) {
  const { t } = useTranslation();
  return (
    <div className="flex flex-col items-center justify-center gap-2 py-16 text-center">
      <Inbox className="h-8 w-8 text-muted-foreground" aria-hidden="true" />
      <h3 className="text-sm font-medium">{title ?? t("state.emptyTitle")}</h3>
      <p className="max-w-sm text-sm text-muted-foreground">{description ?? t("state.emptyDescription")}</p>
      {action ? <div className="mt-3 flex items-center justify-center gap-2">{action}</div> : null}
      {docsUrl ? (
        <a
          href={docsUrl}
          target="_blank"
          rel="noopener noreferrer"
          className="mt-1 text-xs text-primary hover:underline inline-flex items-center gap-1"
        >
          {docsLabel ?? "View Documentation →"}
        </a>
      ) : null}
    </div>
  );
}
