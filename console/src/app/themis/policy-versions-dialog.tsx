"use client";

import * as React from "react";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { History, RotateCcw, Loader2, GitCommit } from "lucide-react";
import { toast } from "sonner";
import type { PolicyVersion } from "@/lib/api/schemas/themis";

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
  const [isRollingBack, setIsRollingBack] = React.useState<number | null>(null);

  React.useEffect(() => {
    if (!isOpen || !policyId) return;

    setIsLoading(true);
    fetch(`/api/themis/policies/${policyId}/versions`)
      .then((res) => (res.ok ? res.json() : []))
      .then((data) => setVersions(data))
      .catch(() => toast.error("Failed to load policy versions"))
      .finally(() => setIsLoading(false));
  }, [isOpen, policyId]);

  const handleRollback = async (version: number) => {
    if (!policyId) return;
    setIsRollingBack(version);
    try {
      const res = await fetch(`/api/themis/policies/${policyId}/rollback/${version}`, {
        method: "POST",
      });
      if (!res.ok) {
        const err = await res.json();
        throw new Error(err.error || "Rollback failed");
      }
      toast.success(`Successfully rolled back to version ${version}`);
      onRolledBack?.();
      onOpenChange(false);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Rollback failed");
    } finally {
      setIsRollingBack(null);
    }
  };

  return (
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
                  <div className="space-y-1 flex-1">
                    <div className="flex items-center gap-2">
                      <Badge variant="outline" className="font-mono text-[11px]">
                        <GitCommit className="w-3 h-3 mr-1" />
                        v{v.version}
                      </Badge>
                      <span className="font-semibold text-foreground">{v.name}</span>
                      <span className="text-[11px] text-muted-foreground">
                        {v.created_at ? new Date(v.created_at).toLocaleString() : ""}
                      </span>
                    </div>
                    <div className="font-mono text-[11px] bg-muted/40 p-1.5 rounded text-muted-foreground truncate max-w-md">
                      {v.expression}
                    </div>
                  </div>

                  <Button
                    variant="outline"
                    size="sm"
                    className="h-7 text-xs gap-1 shrink-0"
                    disabled={isRollingBack !== null}
                    onClick={() => handleRollback(v.version)}
                  >
                    {isRollingBack === v.version ? (
                      <Loader2 className="w-3 h-3 animate-spin" />
                    ) : (
                      <RotateCcw className="w-3 h-3" />
                    )}
                    Rollback
                  </Button>
                </div>
              ))}
            </div>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
}
