"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { DataTable } from "@/components/ui/data-table";
import { ColumnDef } from "@tanstack/react-table";
import { Switch } from "@/components/ui/switch";
import { Badge } from "@/components/ui/badge";
import { toast } from "sonner";

interface Policy {
  id: string;
  name: string;
  expression: string;
  priority: number;
  enabled: boolean;
}

export function ThemisPoliciesTable() {
  const queryClient = useQueryClient();

  const { data: policies = [], isLoading } = useQuery<Policy[]>({
    queryKey: ["themis-policies"],
    queryFn: async () => {
      const res = await fetch("/api/themis/policies");
      if (!res.ok) throw new Error("Failed to fetch policies");
      return res.json();
    },
  });

  const toggleMutation = useMutation({
    mutationFn: async ({ id, enabled }: { id: string; enabled: boolean }) => {
      const res = await fetch("/api/themis/policies/" + id + "/toggle", {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ enabled })
      });
      if (!res.ok) throw new Error("Failed to toggle policy");
      return res.json();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["themis-policies"] });
    },
    onError: () => {
      toast.error("Failed to update policy status");
    }
  });

  const columns: ColumnDef<Policy>[] = [
    {
      accessorKey: "priority",
      header: "Priority",
      cell: ({ row }) => <Badge variant="outline">{row.getValue("priority")}</Badge>,
    },
    {
      accessorKey: "name",
      header: "Name",
      cell: ({ row }) => <div className="font-medium">{row.getValue("name")}</div>,
    },
    {
      accessorKey: "expression",
      header: "CEL Expression",
      cell: ({ row }) => (
        <code className="text-xs bg-muted p-1 rounded font-mono text-muted-foreground">
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
              onCheckedChange={(c) => toggleMutation.mutate({ id: policy.id, enabled: c })}
              disabled={toggleMutation.isPending}
            />
            <span className="text-sm text-muted-foreground">
              {policy.enabled ? "Enabled" : "Disabled"}
            </span>
          </div>
        );
      },
    }
  ];

  return (
    <DataTable 
      columns={columns} 
      data={policies} 
      isLoading={isLoading} 
      searchKey="name"
    />
  );
}
