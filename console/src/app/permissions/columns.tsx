"use client";

import { ColumnDef } from "@tanstack/react-table";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Trash2 } from "lucide-react";
import { Tuple } from "@/lib/api/schemas/nexus";

export interface ZanzibarTupleItem {
  namespace: string;
  object: string;
  relation: string;
  subject: string;
  caveat?: string;
  original: Tuple;
}

export const getColumns = (
  onDelete: (tuple: ZanzibarTupleItem) => void,
): ColumnDef<ZanzibarTupleItem>[] => [
  {
    accessorKey: "namespace",
    header: "Namespace",
    cell: ({ row }) => (
      <span className="font-mono text-xs text-purple-400 font-medium">
        {row.getValue("namespace")}
      </span>
    ),
  },
  {
    accessorKey: "object",
    header: "Object",
    cell: ({ row }) => (
      <span className="font-mono text-xs text-foreground font-semibold">
        {row.getValue("object")}
      </span>
    ),
  },
  {
    accessorKey: "relation",
    header: "Relation",
    cell: ({ row }) => (
      <Badge variant="outline" className="font-mono text-[10px]">
        #{row.getValue("relation")}
      </Badge>
    ),
  },
  {
    accessorKey: "subject",
    header: "Subject",
    cell: ({ row }) => (
      <span className="font-mono text-xs text-blue-400">
        @{row.getValue("subject")}
      </span>
    ),
  },
  {
    accessorKey: "caveat",
    header: "Caveat",
    cell: ({ row }) => {
      const caveat = row.getValue("caveat") as string | undefined;
      if (caveat) {
        return (
          <span className="rounded bg-purple-500/10 px-2 py-0.5 text-[10px] font-mono text-purple-300 border border-purple-500/20">
            {caveat}
          </span>
        );
      }
      return <span className="text-xs text-muted-foreground">—</span>;
    },
  },
  {
    id: "actions",
    cell: ({ row }) => {
      const tuple = row.original;
      return (
        <Button
          variant="ghost"
          size="icon"
          className="h-8 w-8 text-rose-400 hover:text-rose-500 hover:bg-rose-500/10"
          onClick={() => onDelete(tuple)}
        >
          <span className="sr-only">Delete</span>
          <Trash2 className="h-4 w-4" />
        </Button>
      );
    },
  },
];
