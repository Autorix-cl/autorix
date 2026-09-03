"use client";

import * as React from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { DataTable } from "@/components/ui/data-table";
import { ColumnDef } from "@tanstack/react-table";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { MoreHorizontal, Trash2, RotateCw, Activity, AlertCircle } from "lucide-react";
import { toast } from "sonner";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { KeyRotateDialog } from "./key-rotate-dialog";

interface KeyRecord {
  id: string;
  name: string;
  prefix: string;
  scopes: string[];
  expires_at?: string | null;
  last_used_at?: string | null;
  call_count?: number;
  last_source_ip?: string | null;
  grace_period_expires_at?: string | null;
  revoked?: boolean;
  state?: string;
  created_at: string;
}

export function KeysTable() {
  const queryClient = useQueryClient();
  const [keyToRevoke, setKeyToRevoke] = React.useState<string | null>(null);
  const [keyToRotate, setKeyToRotate] = React.useState<KeyRecord | null>(null);

  const { data: keys = [], isLoading } = useQuery<KeyRecord[]>({
    queryKey: ["vulcan-keys"],
    queryFn: async () => {
      const res = await fetch("/api/vulcan/keys");
      if (!res.ok) throw new Error("Failed to fetch keys");
      const data = await res.json();
      return Array.isArray(data) ? data : data.data || [];
    },
  });

  const revokeMutation = useMutation({
    mutationFn: async (id: string) => {
      const res = await fetch(`/api/vulcan/keys/${id}`, { method: "DELETE" });
      if (!res.ok) throw new Error("Failed to revoke key");
      return res.json();
    },
    onSuccess: () => {
      toast.success("API Key revoked successfully");
      queryClient.invalidateQueries({ queryKey: ["vulcan-keys"] });
      setKeyToRevoke(null);
    },
    onError: () => {
      toast.error("Failed to revoke API Key");
    },
  });

  const isKeyStale = (key: KeyRecord) => {
    if (key.revoked || key.state === "revoked") return false;
    if (!key.last_used_at) {
      const createdDate = new Date(key.created_at);
      const daysSinceCreation = (Date.now() - createdDate.getTime()) / (1000 * 60 * 60 * 24);
      return daysSinceCreation > 14;
    }
    const lastUsedDate = new Date(key.last_used_at);
    const daysSinceUse = (Date.now() - lastUsedDate.getTime()) / (1000 * 60 * 60 * 24);
    return daysSinceUse > 30;
  };

  const columns: ColumnDef<KeyRecord>[] = [
    {
      accessorKey: "name",
      header: "Name",
      cell: ({ row }) => (
        <div className="space-y-0.5">
          <div className="font-medium text-xs text-foreground flex items-center gap-1.5">
            {row.getValue("name")}
            {isKeyStale(row.original) && (
              <Badge variant="outline" className="text-[10px] text-amber-500 border-amber-500/30 gap-1 px-1.5 py-0">
                <AlertCircle className="w-2.5 h-2.5" /> Stale
              </Badge>
            )}
          </div>
          {row.original.grace_period_expires_at && (
            <div className="text-[10px] text-amber-500 font-mono">
              Grace period active until {new Date(row.original.grace_period_expires_at).toLocaleDateString()}
            </div>
          )}
        </div>
      ),
    },
    {
      accessorKey: "prefix",
      header: "Key Prefix",
      cell: ({ row }) => <code className="bg-muted px-2 py-1 rounded text-xs font-mono">{row.getValue("prefix")}</code>,
    },
    {
      accessorKey: "scopes",
      header: "Scopes",
      cell: ({ row }) => {
        const scopes = row.getValue<string[]>("scopes") || [];
        return (
          <div className="flex gap-1 flex-wrap max-w-[200px]">
            {scopes.map((s) => (
              <Badge key={s} variant="secondary" className="text-[10px] font-mono">
                {s}
              </Badge>
            ))}
          </div>
        );
      },
    },
    {
      id: "usage",
      header: "Usage Telemetry",
      cell: ({ row }) => {
        const k = row.original;
        const count = k.call_count ?? 0;
        return (
          <div className="space-y-0.5 text-[11px] text-muted-foreground">
            <div className="flex items-center gap-1">
              <Activity className="w-3 h-3 text-primary" />
              <span className="font-semibold text-foreground">{count.toLocaleString()} calls</span>
            </div>
            <div>
              Last used:{" "}
              {k.last_used_at ? new Date(k.last_used_at).toLocaleDateString() : "Never"}
            </div>
          </div>
        );
      },
    },
    {
      accessorKey: "expires_at",
      header: "Expiration",
      cell: ({ row }) => {
        const exp = row.getValue<string | null>("expires_at");
        if (!exp) return <span className="text-muted-foreground text-xs">Never</span>;
        const date = new Date(exp);
        const isExpired = date < new Date();
        return (
          <span className={`text-xs ${isExpired ? "text-destructive" : ""}`}>
            {date.toLocaleDateString()} {isExpired && "(Expired)"}
          </span>
        );
      },
    },
    {
      accessorKey: "revoked",
      header: "Status",
      cell: ({ row }) => {
        const revoked = row.getValue<boolean>("revoked") || row.original.state === "revoked";
        if (revoked) return <Badge variant="destructive" className="text-xs">Revoked</Badge>;
        return <Badge variant="default" className="bg-emerald-500 hover:bg-emerald-600 text-xs">Active</Badge>;
      },
    },
    {
      id: "actions",
      cell: ({ row }) => {
        const key = row.original;
        const isRevoked = key.revoked || key.state === "revoked";

        return (
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" className="h-8 w-8 p-0">
                <span className="sr-only">Open menu</span>
                <MoreHorizontal className="h-4 w-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuLabel>Actions</DropdownMenuLabel>
              <DropdownMenuSeparator />
              <DropdownMenuItem
                className="cursor-pointer"
                disabled={isRevoked}
                onClick={() => setKeyToRotate(key)}
              >
                <RotateCw className="mr-2 h-4 w-4 text-primary" />
                Rotate Key
              </DropdownMenuItem>
              <DropdownMenuItem
                className="text-destructive focus:bg-destructive focus:text-destructive-foreground cursor-pointer"
                disabled={isRevoked}
                onClick={() => setKeyToRevoke(key.id)}
              >
                <Trash2 className="mr-2 h-4 w-4" />
                Revoke Key
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        );
      },
    },
  ];

  return (
    <>
      <DataTable columns={columns} data={keys} isLoading={isLoading} searchKey="name" />

      {/* Revocation Dialog */}
      <Dialog open={Boolean(keyToRevoke)} onOpenChange={(open) => !open && setKeyToRevoke(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Revoke API Key</DialogTitle>
            <DialogDescription>
              Are you sure you want to revoke this API key? Any applications or services using this key will immediately lose access. This action cannot be undone.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setKeyToRevoke(null)}>
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={() => keyToRevoke && revokeMutation.mutate(keyToRevoke)}
              disabled={revokeMutation.isPending}
            >
              {revokeMutation.isPending ? "Revoking..." : "Revoke Key"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Key Rotation Dialog */}
      <KeyRotateDialog
        keyId={keyToRotate?.id || null}
        keyName={keyToRotate?.name || null}
        isOpen={Boolean(keyToRotate)}
        onOpenChange={(open) => !open && setKeyToRotate(null)}
        onSuccess={() => queryClient.invalidateQueries({ queryKey: ["vulcan-keys"] })}
      />
    </>
  );
}
