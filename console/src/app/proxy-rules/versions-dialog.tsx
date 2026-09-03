"use client";

import * as React from "react";
import { History, RotateCcw } from "lucide-react";
import { useMutation, useQueryClient } from "@tanstack/react-query";

import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { toast } from "sonner";
import { useApiQuery } from "@/lib/query/use-api-query";
import { fetchAndParse } from "@/lib/api/schema";
import { ruleVersionListSchema } from "@/lib/api/schemas/aegis";
import { LoadingState } from "@/components/state/loading-state";
import { EmptyState } from "@/components/state/empty-state";

export function VersionsDialog() {
  const [open, setOpen] = React.useState(false);
  const [selectedVersion, setSelectedVersion] = React.useState<number | null>(null);

  const queryClient = useQueryClient();

  const {
    data: versions,
    isLoading,
    isError,
    refetch,
  } = useApiQuery(
    ["proxy-rules-versions"],
    () => fetchAndParse("/api/proxy-rules/versions", ruleVersionListSchema),
    { enabled: open }
  );

  const rollbackMutation = useMutation({
    mutationFn: async (version: number) => {
      const res = await fetch(`/api/proxy-rules/rollback/${version}`, {
        method: "POST",
      });
      const data = await res.json();
      if (!res.ok) {
        throw new Error(data.error || "Failed to rollback rules");
      }
      return data;
    },
    onSuccess: (data) => {
      toast.success(`Successfully rolled back to version ${data.version || selectedVersion}`);
      queryClient.invalidateQueries({ queryKey: ["proxy-rules"] });
      queryClient.invalidateQueries({ queryKey: ["proxy-rules-versions"] });
      setSelectedVersion(null);
      setOpen(false);
    },
    onError: (err) => {
      toast.error(err instanceof Error ? err.message : "Rollback failed");
    },
  });

  const handleRollback = (ver: number) => {
    if (confirm(`Are you sure you want to rollback active rules to version ${ver}?`)) {
      setSelectedVersion(ver);
      rollbackMutation.mutate(ver);
    }
  };

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button variant="outline">
          <History className="w-4 h-4 mr-2" />
          Version History
        </Button>
      </DialogTrigger>
      <DialogContent className="sm:max-w-[650px] max-h-[85vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <History className="w-5 h-5 text-primary" />
            Rule Set Snapshots & Rollback
          </DialogTitle>
          <DialogDescription>
            Inspect versioned snapshots stored in Postgres and restore prior states safely.
          </DialogDescription>
        </DialogHeader>

        <div className="py-4 space-y-4">
          {isLoading && <LoadingState label="Loading version history..." />}

          {isError && (
            <div className="p-4 border border-destructive/20 bg-destructive/10 rounded-md text-destructive text-sm">
              Failed to load rule versions.
              <Button variant="ghost" size="sm" onClick={() => refetch()} className="ml-2">
                Retry
              </Button>
            </div>
          )}

          {!isLoading && (!versions || versions.length === 0) && (
            <EmptyState
              title="No prior versions recorded"
              description="Snapshots are created automatically when rules are modified or imported."
            />
          )}

          {versions && versions.length > 0 && (
            <div className="divide-y border rounded-md">
              {versions.map((v) => (
                <div key={v.version} className="p-4 flex items-center justify-between hover:bg-muted/30 transition-colors">
                  <div className="space-y-1">
                    <div className="flex items-center gap-2">
                      <span className="font-mono font-bold text-sm">Version {v.version}</span>
                      <Badge variant="secondary" className="text-xs">
                        {v.rules.length} {v.rules.length === 1 ? "rule" : "rules"}
                      </Badge>
                    </div>
                    {v.description && (
                      <p className="text-xs text-muted-foreground">{v.description}</p>
                    )}
                    <p className="text-[11px] text-muted-foreground/70">
                      Created: {new Date(v.created_at).toLocaleString()}
                    </p>
                  </div>

                  <Button
                    size="sm"
                    variant="outline"
                    disabled={rollbackMutation.isPending && selectedVersion === v.version}
                    onClick={() => handleRollback(v.version)}
                    className="hover:bg-amber-500/10 hover:text-amber-500 hover:border-amber-500/30"
                  >
                    <RotateCcw className="w-3.5 h-3.5 mr-1.5" />
                    {rollbackMutation.isPending && selectedVersion === v.version ? "Rolling back..." : "Rollback"}
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
