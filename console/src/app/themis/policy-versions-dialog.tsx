"use client";

import * as React from "react";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { Can } from "@/lib/auth/use-permission";
import { formatDateTime } from "@/lib/format/date-time";
import { History, RotateCcw, Loader2, GitCommit } from "lucide-react";
import { toast } from "sonner";
import type { PolicyVersion } from "@/lib/api/schemas/themis";

function readCsrfHeaders(): Record<string, string> {
  if (typeof document === "undefined") return {};
  const m = document.cookie.match(/(?:^|;\s*)autorix_csrf=([^;]+)/);
  return m ? { "X-CSRF-Token": decodeURIComponent(m[1]) } : {};
}

interface PolicyVersionsDialogProps {
  policyId: string | null;
  policyName: string | null;
  isOpen: boolean;
  onOpenChange: (open: boolean) => void;
  onRolledBack?: () => void;
}

export function PolicyVersionsDialog({
  policyId,
  policyName,
  isOpen,
  onOpenChange,
  onRolledBack,
}: PolicyVersionsDialogProps) {
  const [versions, setVersions] = React.useState<PolicyVersion[]>([]);
  const [isLoading, setIsLoading] = React.useState(false);
  const [loadError, setLoadError] = React.useState<string | null>(null);
  const [isRollingBack, setIsRollingBack] = React.useState<number | null>(null);
  const [pendingRollbackVersion, setPendingRollbackVersion] = React.useState<number | null>(null);

  React.useEffect(() => {
    if (!isOpen || !policyId) return;

    const controller = new AbortController();
    setIsLoading(true);
    setLoadError(null);
    fetch(`/api/themis/policies/${encodeURIComponent(policyId)}/versions`, { signal: controller.signal })
      .then(async (res) => {
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        return res.json();
      })
      .then((data) => setVersions(Array.isArray(data) ? data : []))
      .catch((err) => {
        if (controller.signal.aborted) return;
        const msg = err instanceof Error ? err.message : "Failed to load policy versions";
        setLoadError(msg);
        toast.error("Failed to load policy versions");
      })
      .finally(() => {
        if (!controller.signal.aborted) setIsLoading(false);
      });

    return () => controller.abort();
  }, [isOpen, policyId]);

  const handleRollback = async (version: number) => {
    if (!policyId) return;
    setIsRollingBack(version);
    try {
      const res = await fetch(`/api/themis/policies/${encodeURIComponent(policyId)}/rollback/${version}`, {
        method: "POST",
        headers: readCsrfHeaders(),
      });
      if (!res.ok) {
        const err = await res.json().catch(() => ({}));
        throw new Error(err.error || "Rollback failed");
      }
      toast.success(`Successfully rolled back to version ${version}`);
      setPendingRollbackVersion(null);
      onRolledBack?.();
      onOpenChange(false);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Rollback failed");
    } finally {
      setIsRollingBack(null);
    }
  };

  return (
    <>
      <Dialog open={isOpen} onOpenChange={onOpenChange}>
        <DialogContent className="max-w-xl">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <History className="w-5 h-5 text-primary" />
              Version History · {policyName || policyId}
            </DialogTitle>
            <DialogDescription className="text-xs">
              Review immutable snapshots and rollback to any previous version.
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-3 pt-2">
            {isLoading ? (
              <div className="flex items-center justify-center py-8 text-xs text-muted-foreground gap-2">
                <Loader2 className="w-4 h-4 animate-spin" />
                Loading history snapshots...
              </div>
            ) : loadError ? (
              <div role="alert" className="text-center py-8 text-xs text-destructive">
                Failed to load version history ({loadError}).
              </div>
            ) : versions.length === 0 ? (
              <div className="text-center py-8 text-xs text-muted-foreground">
                No previous versions recorded for this policy yet.
              </div>
            ) : (
              <div className="max-h-[360px] overflow-y-auto space-y-2 pr-1">
                {versions.map((v) => (
                  <div
                    key={v.id || v.version}
                    className="p-3 rounded-lg border bg-card/60 flex items-start justify-between gap-3 text-xs"
                  >
                    <div className="space-y-1 flex-1 min-w-0">
                      <div className="flex items-center gap-2 flex-wrap">
                        <Badge variant="outline" className="font-mono text-[11px]">
                          <GitCommit className="w-3 h-3 mr-1" />v{v.version}
                        </Badge>
                        <span className="font-semibold text-foreground">{v.name}</span>
                        <span className="text-[11px] text-muted-foreground">
                          {v.created_at ? formatDateTime(v.created_at) : ""}
                        </span>
                      </div>
                      <div className="font-mono text-[11px] bg-muted/40 p-1.5 rounded text-muted-foreground truncate max-w-md">
                        {v.expression}
                      </div>
                    </div>

                    <Can perform="policies:write">
                      <Button
                        type="button"
                        variant="outline"
                        size="sm"
                        className="h-7 text-xs gap-1 shrink-0"
                        disabled={isRollingBack !== null}
                        onClick={() => setPendingRollbackVersion(v.version)}
                      >
                        {isRollingBack === v.version ? (
                          <Loader2 className="w-3 h-3 animate-spin" />
                        ) : (
                          <RotateCcw className="w-3 h-3" />
                        )}
                        Rollback
                      </Button>
                    </Can>
                  </div>
                ))}
              </div>
            )}
          </div>
        </DialogContent>
      </Dialog>

      <ConfirmDialog
        open={pendingRollbackVersion !== null}
        onOpenChange={(open) => !open && setPendingRollbackVersion(null)}
        title="Rollback Policy Version"
        description={
          <>
            Rollback policy <strong className="font-mono">{policyName || policyId}</strong> to snapshot{" "}
            <strong className="font-mono">v{pendingRollbackVersion}</strong>? Live ABAC evaluations will immediately use
            that expression.
          </>
        }
        confirmLabel={`Rollback to v${pendingRollbackVersion ?? ""}`}
        variant="destructive"
        isLoading={isRollingBack !== null}
        onConfirm={() => {
          if (pendingRollbackVersion !== null) {
            void handleRollback(pendingRollbackVersion);
          }
        }}
      />
    </>
  );
}
