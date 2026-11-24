import * as React from "react";
import { ColumnDef } from "@tanstack/react-table";
import { Policy } from "@/lib/api/schemas/themis";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { Trash2, Copy, Check, FlaskConical, History } from "lucide-react";
import { toast } from "sonner";
import { useApiMutation } from "@/lib/query/use-api-mutation";
import { deletePolicyResponseSchema } from "@/lib/api/schemas/themis";
import { fetchAndParse } from "@/lib/api/schema";
import { Can } from "@/lib/auth/use-permission";
import { useQueryClient } from "@tanstack/react-query";

function ExpressionCell({ expression }: { expression: string }) {
  const [copied, setCopied] = React.useState(false);

  const handleCopy = async (e: React.MouseEvent) => {
    e.stopPropagation();
    try {
      await navigator.clipboard.writeText(expression);
      setCopied(true);
      toast.success("CEL expression copied to clipboard");
      setTimeout(() => setCopied(false), 2000);
    } catch {
      toast.error("Failed to copy");
    }
  };

  return (
    <div className="flex items-center gap-1.5 font-mono text-xs text-purple-300 group max-w-md">
      <span className="rounded bg-purple-500/10 px-2 py-0.5 border border-purple-500/20 truncate" title={expression}>
        {expression}
      </span>
      <Button
        type="button"
        variant="ghost"
        size="icon-sm"
        onClick={handleCopy}
        aria-label="Copy CEL expression"
        className="h-5 w-5 opacity-0 group-hover:opacity-100 focus-visible:opacity-100 transition-opacity text-muted-foreground hover:text-foreground shrink-0"
        title="Copy CEL expression"
      >
        {copied ? <Check className="h-3 w-3 text-emerald-400" /> : <Copy className="h-3 w-3" />}
      </Button>
    </div>
  );
}

export interface PolicyColumnActions {
  onOpenTests?: (policy: Policy) => void;
  onOpenVersions?: (policy: Policy) => void;
}

export function getColumns(tenantId: string, actions?: PolicyColumnActions): ColumnDef<Policy, unknown>[] {
  return [
    {
      accessorKey: "Name",
      header: "Name",
      cell: ({ row }) => <div className="font-semibold text-foreground">{row.original.Name}</div>,
    },
    {
      accessorKey: "Expression",
      header: "Expression",
      cell: ({ row }) => <ExpressionCell expression={row.original.Expression} />,
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
        return (
          <ActionCell
            policy={row.original}
            tenantId={tenantId}
            onOpenTests={actions?.onOpenTests}
            onOpenVersions={actions?.onOpenVersions}
          />
        );
      },
    },
  ];
}

function ActionCell({
  policy,
  tenantId,
  onOpenTests,
  onOpenVersions,
}: {
  policy: Policy;
  tenantId: string;
  onOpenTests?: (policy: Policy) => void;
  onOpenVersions?: (policy: Policy) => void;
}) {
  const queryClient = useQueryClient();
  const [confirmOpen, setConfirmOpen] = React.useState(false);

  const deletePolicy = useApiMutation(
    (policyId: string) =>
      fetchAndParse(
        `/api/policies/${encodeURIComponent(policyId)}?tenant_id=${encodeURIComponent(tenantId)}`,
        deletePolicyResponseSchema,
        {
          method: "DELETE",
        },
      ),
    {
      successMessage: "Policy deleted from Themis.",
      onSuccess: () => {
        setConfirmOpen(false);
        queryClient.invalidateQueries({ queryKey: ["policies", tenantId] });
      },
    },
  );

  return (
    <div className="flex items-center justify-end gap-1">
      {onOpenTests && (
        <Button
          type="button"
          variant="ghost"
          size="sm"
          onClick={() => onOpenTests(policy)}
          aria-label={`Open test suite for ${policy.Name}`}
          title="Test Suite"
          className="h-7 px-2 text-[11px] gap-1 text-muted-foreground hover:text-foreground"
        >
          <FlaskConical className="h-3.5 w-3.5 text-purple-400" />
          <span className="hidden sm:inline">Tests</span>
        </Button>
      )}
      {onOpenVersions && (
        <Button
          type="button"
          variant="ghost"
          size="sm"
          onClick={() => onOpenVersions(policy)}
          aria-label={`Open version history for ${policy.Name}`}
          title="Version History"
          className="h-7 px-2 text-[11px] gap-1 text-muted-foreground hover:text-foreground"
        >
          <History className="h-3.5 w-3.5" />
          <span className="hidden sm:inline">History</span>
        </Button>
      )}
      <Can perform="policies:write">
        <Button
          type="button"
          variant="ghost"
          size="sm"
          onClick={() => setConfirmOpen(true)}
          disabled={deletePolicy.isPending}
          aria-label={`Delete policy ${policy.Name}`}
          title={`Delete policy ${policy.Name}`}
          className="h-7 w-7 p-0 text-muted-foreground hover:text-destructive hover:bg-destructive/10"
        >
          <Trash2 className="h-3.5 w-3.5" />
        </Button>
      </Can>

      <ConfirmDialog
        open={confirmOpen}
        onOpenChange={setConfirmOpen}
        title="Delete ABAC Policy"
        description={
          <>
            Are you sure you want to delete policy <strong className="font-mono">{policy.Name}</strong>? Requests
            evaluated by Themis will immediately stop matching this rule.
          </>
        }
        confirmLabel="Delete Policy"
        variant="destructive"
        isLoading={deletePolicy.isPending}
        onConfirm={() => deletePolicy.mutate(policy.ID)}
      />
    </div>
  );
}
