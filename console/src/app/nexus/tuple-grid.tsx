import * as React from "react";
import { DataTable } from "@/components/ui/data-table";
import { ColumnDef } from "@tanstack/react-table";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Trash2, X, Plus } from "lucide-react";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog";
import { TupleBuilder } from "./tuple-builder";

export interface Tuple {
  id: string;
  objectType: string;
  objectId: string;
  relation: string;
  subjectType: string;
  subjectId: string;
  subjectRelation?: string;
}

export const tupleColumns: ColumnDef<Tuple>[] = [
  {
    id: "select",
    header: ({ table }) => (
      <Checkbox
        checked={
          table.getIsAllPageRowsSelected() ||
          (table.getIsSomePageRowsSelected() && "indeterminate")
        }
        onCheckedChange={(value) => table.toggleAllPageRowsSelected(!!value)}
        aria-label="Select all"
      />
    ),
    cell: ({ row }) => (
      <Checkbox
        checked={row.getIsSelected()}
        onCheckedChange={(value) => row.toggleSelected(!!value)}
        aria-label="Select row"
      />
    ),
    enableSorting: false,
    enableHiding: false,
  },
  {
    accessorKey: "object",
    header: "Object",
    cell: ({ row }) => (
      <span className="font-mono text-xs">
        {row.original.objectType}:{row.original.objectId}
      </span>
    ),
  },
  {
    accessorKey: "relation",
    header: "Relation",
    cell: ({ row }) => (
      <span className="font-mono text-xs text-muted-foreground">
        {row.original.relation}
      </span>
    ),
  },
  {
    accessorKey: "subject",
    header: "Subject",
    cell: ({ row }) => {
      const subjectStr = `${row.original.subjectType}:${row.original.subjectId}`;
      const relStr = row.original.subjectRelation ? `#${row.original.subjectRelation}` : "";
      return (
        <span className="font-mono text-xs">
          {subjectStr}{relStr}
        </span>
      );
    },
  },
];

export interface TupleGridProps {
  tuples: Tuple[];
  onDelete: (ids: string[]) => Promise<void>;
  onAdd: (tuple: import("./tuple-builder").TupleData) => Promise<void>;
}

export function TupleGrid({ tuples, onDelete, onAdd }: TupleGridProps) {
  const [isAddOpen, setIsAddOpen] = React.useState(false);
  return (
    <div className="flex flex-col h-full bg-card rounded-md border text-card-foreground shadow-sm">
      <div className="p-4 border-b flex items-center justify-between">
        <h3 className="font-semibold text-sm">Relation Tuples</h3>
        <Dialog open={isAddOpen} onOpenChange={setIsAddOpen}>
          <DialogTrigger asChild>
            <Button size="sm"><Plus className="w-4 h-4 mr-2" /> Add Tuple</Button>
          </DialogTrigger>
          <DialogContent className="max-w-2xl">
            <DialogHeader>
              <DialogTitle>Add Relation Tuple</DialogTitle>
            </DialogHeader>
            <TupleBuilder 
              onSubmit={async (data) => {
                await onAdd(data);
                setIsAddOpen(false);
              }}
              onCancel={() => setIsAddOpen(false)}
            />
          </DialogContent>
        </Dialog>
      </div>
      <div className="flex-1 p-4 overflow-y-auto">
        <DataTable 
          columns={tupleColumns} 
          data={tuples} 
          searchKey="relation" // allow filtering by relation for now
          renderToolbar={(table) => {
            const selectedRows = table.getFilteredSelectedRowModel().rows;
            const selectedCount = selectedRows.length;
            if (selectedCount === 0) return null;

            return (
              <div className="absolute bottom-6 left-1/2 -translate-x-1/2 z-50 flex items-center gap-4 bg-foreground text-background px-4 py-3 rounded-lg shadow-xl animate-in slide-in-from-bottom-5">
                <div className="flex items-center gap-2 border-r border-background/20 pr-4">
                  <span className="text-sm font-medium">{selectedCount} selected</span>
                  <Button variant="ghost" size="icon" className="h-6 w-6 text-background hover:bg-background/20" onClick={() => table.resetRowSelection()}>
                    <X className="h-4 w-4" />
                  </Button>
                </div>
                <Button 
                  variant="destructive" 
                  size="sm" 
                  onClick={async () => {
                    const ids = selectedRows.map(r => r.original.id);
                    await onDelete(ids);
                    table.resetRowSelection();
                  }}
                >
                  <Trash2 className="h-4 w-4 mr-2" />
                  Delete
                </Button>
              </div>
            );
          }}
        />
      </div>
    </div>
  );
}
