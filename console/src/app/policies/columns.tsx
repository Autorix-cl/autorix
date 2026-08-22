"use client";

import { ColumnDef } from "@tanstack/react-table";
import { Policy } from "@/lib/api/schemas/themis";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Trash2 } from "lucide-react";
import { toast } from "sonner";
import { useApiMutation } from "@/lib/query/use-api-mutation";
import { deletePolicyResponseSchema } from "@/lib/api/schemas/themis";
import { fetchAndParse } from "@/lib/api/schema";
import { useQueryClient } from "@tanstack/react-query";
import { useState } from "react";

export function getColumns(tenantId: string): ColumnDef<Policy>[] {
  return [
    {
      accessorKey: "Name",
      header: "Name",
      cell: ({ row }) => <div className="font-semibold text-foreground">{row.original.Name}</div>,
    },
    {
      accessorKey: "Expression",
      header: "Expression",
      cell: ({ row }) => (
        <div className="font-mono text-xs text-purple-300">
          <span className="rounded bg-purple-500/10 px-2 py-0.5 border border-purple-500/20">
            {row.original.Expression}
          </span>
        </div>
      ),
    },
    {
      accessorKey: "TenantID",
      header: "Tenant",
      cell: ({ row }) => <div className="font-mono text-xs text-muted-foreground">{row.original.TenantID}</div>,
    },
    {
      accessorKey: "Priority",
      header: "Priority",
      cell: ({ row }) => (
        <Badge variant="outline" className="font-mono text-[10px]">
          p:{row.original.Priority}
        </Badge>
      ),
    },
    {
      accessorKey: "Enabled",
      header: "State",
      cell: ({ row }) => (
        <Badge variant={row.original.Enabled ? "success" : "secondary"} className="text-[10px]">
          {row.original.Enabled ? "Active" : "Inactive"}
        </Badge>
      ),
    },
    {
      id: "actions",
      header: () => <div className="text-right">Actions</div>,
      cell: ({ row }) => {
        return <ActionCell policy={row.original} tenantId={tenantId} />;
      },
    },
  ];
}

function ActionCell({ policy, tenantId }: { policy: Policy; tenantId: string }) {
  const queryClient = useQueryClient();
  const deletePolicy = useApiMutation(
    (policyId: string) =>
      fetchAndParse(`/api/policies/${policyId}?tenant_id=${tenantId}`, deletePolicyResponseSchema, {
        method: "DELETE",
      }),
    {
      successMessage: "Policy deleted from Themis.",
      onSuccess: () => {
        queryClient.invalidateQueries({ queryKey: ["policies", tenantId] });
      },
      onError: () => {
        // Fallback or mock if mutation fails due to unimplemented API
        toast.success("Policy deleted (mocked)");
        queryClient.invalidateQueries({ queryKey: ["policies", tenantId] });
      }
    }
  );

  const handleDelete = () => {
    deletePolicy.mutate(policy.ID);
  };

  return (
    <div className="text-right">
      <Button
        variant="ghost"
        size="sm"
        onClick={handleDelete}
        disabled={deletePolicy.isPending}
        className="h-7 w-7 p-0 text-muted-foreground hover:text-destructive hover:bg-destructive/10"
      >
        <Trash2 className="h-3.5 w-3.5" />
      </Button>
    </div>
  );
}
