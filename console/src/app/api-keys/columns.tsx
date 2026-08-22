"use client";

import { ColumnDef } from "@tanstack/react-table";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { MoreHorizontal, Trash } from "lucide-react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { toast } from "sonner";

export interface KeyItem {
  id: string;
  name: string;
  prefix: string;
  keyMasked: string;
  ownerId: string;
  createdAt: string;
}

export const columns: ColumnDef<KeyItem, unknown>[] = [
  {
    accessorKey: "name",
    header: "Name",
    cell: ({ row }) => {
      return <div className="font-semibold text-foreground">{row.getValue("name")}</div>;
    },
  },
  {
    accessorKey: "keyMasked",
    header: "Token",
    cell: ({ row }) => {
      return <div className="font-mono text-xs text-cyan-400 font-medium">{row.getValue("keyMasked")}</div>;
    },
  },
  {
    accessorKey: "ownerId",
    header: "Owner",
    cell: ({ row }) => {
      return <div className="text-muted-foreground">{row.getValue("ownerId")}</div>;
    },
  },
  {
    accessorKey: "prefix",
    header: "Environment",
    cell: ({ row }) => {
      const prefix = row.getValue("prefix") as string;
      const isLive = prefix.includes("live");
      return (
        <Badge variant={isLive ? "success" : "warning"} className="text-[10px]">
          {isLive ? "Production" : "Sandbox"}
        </Badge>
      );
    },
  },
  {
    accessorKey: "createdAt",
    header: "Created",
    cell: ({ row }) => {
      return <div className="text-muted-foreground text-xs font-mono">{row.getValue("createdAt")}</div>;
    },
  },
  {
    id: "actions",
    cell: ({ row }) => {
      const keyItem = row.original;

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
            <DropdownMenuItem
              className="text-destructive focus:text-destructive cursor-pointer"
              onClick={() => {
                toast.success(`Key ${keyItem.name} revoked successfully.`);
              }}
            >
              <Trash className="mr-2 h-4 w-4" />
              Revoke Key
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      );
    },
  },
];
