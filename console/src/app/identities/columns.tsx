"use client";

import { ColumnDef } from "@tanstack/react-table";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { MoreHorizontal } from "lucide-react";
import { Identity } from "@/lib/api/schemas/identity";

export interface IdentityItem {
  id: string;
  email: string;
  name: string;
  state: string;
  createdAt: string;
  original: Identity;
}

export const getColumns = (onSelect: (identity: IdentityItem) => void): ColumnDef<IdentityItem, unknown>[] => [
  {
    id: "select",
    header: ({ table }) => (
      <Checkbox
        checked={table.getIsAllPageRowsSelected() || (table.getIsSomePageRowsSelected() && "indeterminate")}
        onCheckedChange={(value) => table.toggleAllPageRowsSelected(!!value)}
        aria-label="Select all identities on this page"
      />
    ),
    cell: ({ row }) => (
      <Checkbox
        checked={row.getIsSelected()}
        onCheckedChange={(value) => row.toggleSelected(!!value)}
        aria-label={`Select identity ${row.original.email}`}
      />
    ),
    enableSorting: false,
    enableHiding: false,
  },
  {
    accessorKey: "id",
    header: "ID",
    cell: ({ row }) => {
      const id = row.original.id;
      return <span className="font-mono text-xs text-blue-400 font-medium">{id.slice(0, 8)}...</span>;
    },
  },
  {
    accessorKey: "email",
    header: "Email",
    cell: ({ row }) => <span className="font-semibold text-foreground">{row.original.email}</span>,
  },
  {
    accessorKey: "name",
    header: "Name",
    cell: ({ row }) => <span className="text-muted-foreground">{row.original.name}</span>,
  },
  {
    accessorKey: "state",
    header: "State",
    cell: ({ row }) => {
      const state = (row.original.state || "inactive").toLowerCase();
      return (
        <Badge
          variant={state === "active" ? "success" : state === "suspended" ? "destructive" : "secondary"}
          className="text-[10px] capitalize"
        >
          {state}
        </Badge>
      );
    },
  },
  {
    accessorKey: "createdAt",
    header: "Created At",
    cell: ({ row }) => <span className="text-muted-foreground text-xs font-mono">{row.original.createdAt}</span>,
  },
  {
    id: "actions",
    cell: ({ row }) => {
      const identity = row.original;
      return (
        <Button
          type="button"
          variant="ghost"
          className="h-8 w-8 p-0"
          onClick={() => onSelect(identity)}
          aria-label={`Inspect identity ${identity.email}`}
        >
          <MoreHorizontal className="h-4 w-4" aria-hidden="true" />
        </Button>
      );
    },
  },
];
