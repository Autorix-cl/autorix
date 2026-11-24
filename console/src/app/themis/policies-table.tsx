"use client";

import * as React from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { DataTable } from "@/components/ui/data-table";
import { ColumnDef } from "@tanstack/react-table";
import { Switch } from "@/components/ui/switch";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { ErrorState } from "@/components/state/error-state";
import { Can, usePermission } from "@/lib/auth/use-permission";
import { FlaskConical, History, Trash2 } from "lucide-react";
import { toast } from "sonner";
import { PolicyTestSuiteDialog } from "./policy-test-suite-dialog";
import { PolicyVersionsDialog } from "./policy-versions-dialog";

function readCsrfHeaders(): Record<string, string> {
  if (typeof document === "undefined") return {};
  const m = document.cookie.match(/(?:^|;\s*)autorix_csrf=([^;]+)/);
  return m ? { "X-CSRF-Token": decodeURIComponent(m[1]) } : {};
}

interface Policy {
  id: string;
  name: string;
  expression: string;
  priority: number;
  enabled: boolean;
}

export function ThemisPoliciesTable() {
  const queryClient = useQueryClient();
  const { can } = usePermission();
  const canWrite = can("policies:write");

  const [selectedPolicyForSuite, setSelectedPolicyForSuite] = React.useState<Policy | null>(null);
  const [selectedPolicyForVersions, setSelectedPolicyForVersions] = React.useState<Policy | null>(null);
  const [pendingDelete, setPendingDelete] = React.useState<Policy | null>(null);

  const {
    data: policies = [],
    isLoading,
    isError,
    error,
    refetch,
  } = useQuery<Policy[]>({
    queryKey: ["themis-policies"],
    queryFn: async () => {
      const res = await fetch("/api/themis/policies");
      if (!res.ok) throw new Error("Failed to fetch policies");
      return res.json();
    },
  });

  const toggleMutation = useMutation({
    mutationFn: async ({ id, enabled }: { id: string; enabled: boolean }) => {
      const res = await fetch(`/api/themis/policies/${encodeURIComponent(id)}/toggle`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json", ...readCsrfHeaders() },
        body: JSON.stringify({ enabled }),
      });
      if (!res.ok) throw new Error("Failed to toggle policy");
      return res.json();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["themis-policies"] });
    },
    onError: () => {
      toast.error("Failed to update policy status");
    },
  });

  const deleteMutation = useMutation({
    mutationFn: async (id: string) => {
      const res = await fetch(`/api/themis/policies/${encodeURIComponent(id)}`, {
        method: "DELETE",
        headers: readCsrfHeaders(),
      });
      if (!res.ok) throw new Error("Failed to delete policy");
      return res.json();
    },
    onSuccess: () => {
      toast.success("Policy deleted");
      setPendingDelete(null);
      queryClient.invalidateQueries({ queryKey: ["themis-policies"] });
    },
    onError: () => {
      toast.error("Failed to delete policy");
    },
  });

  const toggleMutateRef = React.useRef(toggleMutation.mutate);
  toggleMutateRef.current = toggleMutation.mutate;
  const togglePending = toggleMutation.isPending;

  const columns = React.useMemo<ColumnDef<Policy>[]>(
    () => [
      {
        accessorKey: "priority",
        header: "Priority",
        cell: ({ row }) => <Badge variant="outline">{row.getValue("priority")}</Badge>,
      },
      {
        accessorKey: "name",
        header: "Name",
        cell: ({ row }) => <div className="font-medium text-xs">{row.getValue("name")}</div>,
      },
      {
        accessorKey: "expression",
        header: "CEL Expression",
        cell: ({ row }) => (
          <code className="text-xs bg-muted p-1 rounded font-mono text-muted-foreground truncate max-w-xs block">
            {row.getValue("expression")}
          </code>
        ),
      },
      {
        accessorKey: "enabled",
        header: "Status",
        cell: ({ row }) => {
          const policy = row.original;
          return (
            <div className="flex items-center space-x-2">
              <Switch
                checked={policy.enabled}
                aria-label={`Toggle policy ${policy.name}`}
                onCheckedChange={(c) => toggleMutateRef.current({ id: policy.id, enabled: c })}
                disabled={!canWrite || togglePending}
              />
              <span className="text-xs text-muted-foreground">{policy.enabled ? "Enabled" : "Disabled"}</span>
            </div>
          );
        },
      },
      {
        id: "actions",
        header: "Actions",
        cell: ({ row }) => {
          const policy = row.original;
          return (
            <div className="flex items-center gap-1.5">
              <Button
                type="button"
                variant="outline"
                size="sm"
                className="h-7 text-[11px] gap-1 px-2"
                onClick={() => setSelectedPolicyForSuite(policy)}
              >
                <FlaskConical className="w-3 h-3 text-primary" />
                Tests
              </Button>
              <Button
                type="button"
                variant="outline"
                size="sm"
                className="h-7 text-[11px] gap-1 px-2"
                onClick={() => setSelectedPolicyForVersions(policy)}
              >
                <History className="w-3 h-3" />
                History
              </Button>
              <Can perform="policies:write">
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  aria-label={`Delete policy ${policy.name}`}
                  title={`Delete policy ${policy.name}`}
                  className="h-7 w-7 text-muted-foreground hover:text-destructive"
                  onClick={() => setPendingDelete(policy)}
                >
                  <Trash2 className="w-3 h-3" />
                </Button>
              </Can>
            </div>
          );
        },
      },
    ],
    [canWrite, togglePending],
  );

  if (isError) {
    return (
      <ErrorState
        error={{
          kind: "engine-error",
          message: error instanceof Error ? error.message : "Failed to load policies",
        }}
        onRetry={refetch}
      />
    );
  }

  return (
    <>
      <DataTable columns={columns} data={policies} isLoading={isLoading} searchKey="name" />

      <ConfirmDialog
        open={Boolean(pendingDelete)}
        onOpenChange={(open) => !open && setPendingDelete(null)}
        title="Delete ABAC Policy"
        description={
          <>
            Delete policy <strong className="font-mono">{pendingDelete?.name}</strong>? This immediately removes it from
            Themis evaluation.
          </>
        }
        confirmLabel="Delete Policy"
        variant="destructive"
        isLoading={deleteMutation.isPending}
        onConfirm={() => {
          if (pendingDelete) deleteMutation.mutate(pendingDelete.id);
        }}
      />

      <PolicyTestSuiteDialog
        policyId={selectedPolicyForSuite?.id || null}
        policyName={selectedPolicyForSuite?.name || null}
        isOpen={Boolean(selectedPolicyForSuite)}
        onOpenChange={(open) => !open && setSelectedPolicyForSuite(null)}
      />

      <PolicyVersionsDialog
        policyId={selectedPolicyForVersions?.id || null}
        policyName={selectedPolicyForVersions?.name || null}
        isOpen={Boolean(selectedPolicyForVersions)}
        onOpenChange={(open) => !open && setSelectedPolicyForVersions(null)}
        onRolledBack={() => queryClient.invalidateQueries({ queryKey: ["themis-policies"] })}
      />
    </>
  );
}
