"use client";

import { Loader2 } from "lucide-react";
import { useTranslation } from "@/lib/i18n";

export function LoadingState({ label }: { label?: string }) {
  const { t } = useTranslation();
  return (
    <div role="status" className="flex flex-col items-center justify-center gap-3 py-16 text-muted-foreground">
      <Loader2 className="h-6 w-6 animate-spin" aria-hidden="true" />
      <p className="text-sm">{label ?? t("state.loading")}</p>
    </div>
  );
}
