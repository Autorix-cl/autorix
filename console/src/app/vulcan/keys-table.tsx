"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { DataTable } from "@/components/ui/data-table";
import { ColumnDef } from "@tanstack/react-table";
import { KeyMetadata } from "@/lib/schemas/vulcan";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuLabel, DropdownMenuSeparator, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";
import { MoreHorizontal, Trash2 } from "lucide-react";
import { toast } from "sonner";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { useState } from "react";

export function KeysTable() {
  const queryClient = useQueryClient();
  const [keyToRevoke, setKeyToRevoke] = useState<string | null>(null);

  const { data: keys = [], isLoading } = useQuery<KeyMetadata[]>({
    queryKey: ["vulcan-keys"],
    queryFn: async () => {
      const res = await fetch("/api/vulcan/keys");
      if (!res.ok) throw new Error("Failed to fetch keys");
      return res.json();
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
    }
  });

  const columns: ColumnDef<KeyMetadata>[] = [
    {
      accessorKey: "name",
      header: "Name",
      cell: ({ row }) => <div className="font-medium">{row.getValue("name")}</div>,
    },
    {
      accessorKey: "prefix",
      header: "Key Prefix",
      cell: ({ row }) => <code className="bg-muted px-2 py-1 rounded text-xs">{row.getValue("prefix")}</code>,
    },
    {
      accessorKey: "scopes",
      header: "Scopes",
      cell: ({ row }) => {
        const scopes = row.getValue<string[]>("scopes") || [];
        return (
          <div className="flex gap-1 flex-wrap max-w-[200px]">
            {scopes.map(s => <Badge key={s} variant="secondary" className="text-xs">{s}</Badge>)}
          </div>
        );
      },
    },
    {
      accessorKey: "expires_at",
      header: "Expiration",
      cell: ({ row }) => {
        const exp = row.getValue<string | null>("expires_at");
        if (!exp) return <span className="text-muted-foreground">Never</span>;
        const date = new Date(exp);
        const isExpired = date < new Date();
        return (
          <span className={isExpired ? "text-destructive" : ""}>
            {date.toLocaleDateString()} {isExpired && "(Expired)"}
          </span>
        );
      },
    },
    {
      accessorKey: "revoked",
      header: "Status",
      cell: ({ row }) => {
        const revoked = row.getValue<boolean>("revoked");
        if (revoked) return <Badge variant="destructive">Revoked</Badge>;
        return <Badge variant="default" className="bg-emerald-500 hover:bg-emerald-600">Active</Badge>;
      },
    },
    {
      id: "actions",
      cell: ({ row }) => {
        const key = row.original;
        
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
                className="text-destructive focus:bg-destructive focus:text-destructive-foreground cursor-pointer"
                disabled={key.revoked}
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
      <DataTable 
        columns={columns} 
        data={keys} 
        isLoading={isLoading} 
        searchKey="name"
      />

      <Dialog open={!!keyToRevoke} onOpenChange={(open) => !open && setKeyToRevoke(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Revoke API Key</DialogTitle>
            <DialogDescription>
              Are you sure you want to revoke this API key? Any applications or services using this key will immediately lose access. This action cannot be undone.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setKeyToRevoke(null)}>Cancel</Button>
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
    </>
  );
}
