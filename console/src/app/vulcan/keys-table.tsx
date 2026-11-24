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
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { KeyRotateDialog } from "./key-rotate-dialog";
import { CopyableIdentifier } from "@/components/ui/masked-secret";
import { formatDateTime } from "@/lib/format/date-time";
import { usePermission } from "@/lib/auth/use-permission";

interface KeyRecord {
  id: string;
  name: string;
  prefix?: string;
  key_prefix?: string;
  key_hint?: string;
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

function readCsrfCookie(): string | undefined {
  if (typeof document === "undefined") return undefined;
  const match = document.cookie.match(/(?:^|;\s*)autorix_csrf=([^;]+)/);
  return match ? decodeURIComponent(match[1]) : undefined;
}

function isKeyStale(key: KeyRecord): boolean {
  if (key.revoked || key.state === "revoked") return false;
  if (!key.last_used_at) {
    const createdDate = new Date(key.created_at);
    const daysSinceCreation = (Date.now() - createdDate.getTime()) / (1000 * 60 * 60 * 24);
    return daysSinceCreation > 14;
  }
  const lastUsedDate = new Date(key.last_used_at);
  const daysSinceUse = (Date.now() - lastUsedDate.getTime()) / (1000 * 60 * 60 * 24);
  return daysSinceUse > 30;
}

export function KeysTable() {
  const queryClient = useQueryClient();
  const { can } = usePermission();
  const canWriteKeys = can("api-keys:write");
  const [keyToRevoke, setKeyToRevoke] = React.useState<KeyRecord | null>(null);
  const [keyToRotate, setKeyToRotate] = React.useState<KeyRecord | null>(null);

  const {
    data: keys = [],
    isLoading,
    isError,
    error,
    refetch,
  } = useQuery<KeyRecord[]>({
    queryKey: ["vulcan-keys"],
    queryFn: async () => {
      const res = await fetch("/api/vulcan/keys");
      if (!res.ok) {
        const errBody = await res.json().catch(() => ({}));
        throw new Error(errBody.error || `Failed to fetch keys (HTTP ${res.status})`);
      }
      const data = await res.json();
      return Array.isArray(data) ? data : data.data || [];
    },
  });

  const revokeMutation = useMutation({
    mutationFn: async (id: string) => {
      const csrf = readCsrfCookie();
      const res = await fetch(`/api/vulcan/keys/${encodeURIComponent(id)}`, {
        method: "DELETE",
        headers: csrf ? { "X-CSRF-Token": csrf } : undefined,
      });
      if (!res.ok) {
        const errBody = await res.json().catch(() => ({}));
        throw new Error(errBody.error || "Failed to revoke key");
      }
      return res.status === 204 ? {} : res.json().catch(() => ({}));
    },
    onSuccess: () => {
      toast.success("API Key revoked successfully");
      queryClient.invalidateQueries({ queryKey: ["vulcan-keys"] });
      setKeyToRevoke(null);
    },
    onError: (err: Error) => {
      toast.error(err.message || "Failed to revoke API Key");
    },
  });

  const columns = React.useMemo<ColumnDef<KeyRecord>[]>(
    () => [
      {
        accessorKey: "name",
        header: "Name",
        cell: ({ row }) => (
          <div className="space-y-0.5">
            <div className="font-medium text-xs text-foreground flex items-center gap-1.5">
              {row.getValue("name")}
              {isKeyStale(row.original) && (
                <Badge variant="outline" className="text-[10px] text-amber-500 border-amber-500/30 gap-1 px-1.5 py-0">
                  <AlertCircle className="w-2.5 h-2.5" aria-hidden="true" /> Stale
                </Badge>
              )}
            </div>
            {row.original.grace_period_expires_at && (
              <div className="text-[10px] text-amber-500 font-mono">
                Grace period active until {formatDateTime(row.original.grace_period_expires_at)}
              </div>
            )}
          </div>
        ),
      },
      {
        id: "prefix",
        header: "Key Prefix",
        cell: ({ row }) => {
          const prefix =
            row.original.key_prefix ||
            row.original.key_hint ||
            row.original.prefix ||
            row.original.id?.slice(0, 8) ||
            "key";
          return <CopyableIdentifier value={prefix} label="Key Prefix" />;
        },
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
                <Activity className="w-3 h-3 text-primary" aria-hidden="true" />
                <span className="font-semibold text-foreground">{count.toLocaleString()} calls</span>
              </div>
              <div>Last used: {k.last_used_at ? formatDateTime(k.last_used_at) : "Never"}</div>
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
            <span className={`text-xs font-mono ${isExpired ? "text-destructive" : ""}`}>
              {formatDateTime(exp)} {isExpired && "(Expired)"}
            </span>
          );
        },
      },
      {
        id: "status",
        header: "Status",
        cell: ({ row }) => {
          const revoked = Boolean(row.original.revoked) || row.original.state === "revoked";
          if (revoked)
            return (
              <Badge variant="destructive" className="text-xs">
                Revoked
              </Badge>
            );
          return (
            <Badge variant="default" className="bg-emerald-500 hover:bg-emerald-600 text-xs">
              Active
            </Badge>
          );
        },
      },
      {
        id: "actions",
        cell: ({ row }) => {
          const key = row.original;
          const isRevoked = Boolean(key.revoked) || key.state === "revoked";
          if (!canWriteKeys) return null;

          return (
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button
                  type="button"
                  variant="ghost"
                  className="h-8 w-8 p-0"
                  aria-label={`Actions for key ${key.name}`}
                >
                  <MoreHorizontal className="h-4 w-4" aria-hidden="true" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuLabel>Actions</DropdownMenuLabel>
                <DropdownMenuSeparator />
                <DropdownMenuItem className="cursor-pointer" disabled={isRevoked} onClick={() => setKeyToRotate(key)}>
                  <RotateCw className="mr-2 h-4 w-4 text-primary" aria-hidden="true" />
                  Rotate Key
                </DropdownMenuItem>
                <DropdownMenuItem
                  className="text-destructive focus:bg-destructive focus:text-destructive-foreground cursor-pointer"
                  disabled={isRevoked}
                  onClick={() => setKeyToRevoke(key)}
                >
                  <Trash2 className="mr-2 h-4 w-4" aria-hidden="true" />
                  Revoke Key
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          );
        },
      },
    ],
    [canWriteKeys],
  );

  if (isError) {
    return (
      <div role="alert" className="py-12 text-center space-y-2">
        <AlertCircle className="h-6 w-6 text-amber-400 mx-auto" aria-hidden="true" />
        <p className="text-xs font-medium text-foreground">
          {error instanceof Error ? error.message : "Unable to load API keys from Vulcan."}
        </p>
        <Button type="button" variant="outline" size="sm" onClick={() => refetch()} className="text-xs h-7">
          Retry
        </Button>
      </div>
    );
  }

  return (
    <>
      <DataTable columns={columns} data={keys} isLoading={isLoading} searchKey="name" />

      <ConfirmDialog
        open={Boolean(keyToRevoke)}
        onOpenChange={(open) => !open && setKeyToRevoke(null)}
        title="Revoke API Key"
        description={`Are you sure you want to revoke "${keyToRevoke?.name ?? ""}"? Any applications or services using this key or its attenuated macaroons will immediately lose access.`}
        confirmLabel="Revoke Key"
        isLoading={revokeMutation.isPending}
        onConfirm={() => {
          if (keyToRevoke) revokeMutation.mutate(keyToRevoke.id);
        }}
      />

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
