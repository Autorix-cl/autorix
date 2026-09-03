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
  onBulkAdd?: (tuples: import("./tuple-builder").TupleData[]) => Promise<void>;
}

export function TupleGrid({ tuples, onDelete, onAdd, onBulkAdd }: TupleGridProps) {
  const [isAddOpen, setIsAddOpen] = React.useState(false);
  const [isImportOpen, setIsImportOpen] = React.useState(false);
  const [importText, setImportText] = React.useState("");
  const [importFormat, setImportFormat] = React.useState<"csv" | "json">("csv");
  const [isImporting, setIsImporting] = React.useState(false);
  const [namespaceFilter, setNamespaceFilter] = React.useState("all");

  const filteredTuples = React.useMemo(() => {
    if (namespaceFilter === "all") return tuples;
    return tuples.filter(t => t.objectType === namespaceFilter);
  }, [tuples, namespaceFilter]);

  const uniqueNamespaces = React.useMemo(() => {
    const set = new Set<string>();
    tuples.forEach(t => {
      if (t.objectType) set.add(t.objectType);
    });
    return Array.from(set);
  }, [tuples]);

  const handleBulkImport = async () => {
    if (!importText.trim()) return;
    setIsImporting(true);
    try {
      const parsed: import("./tuple-builder").TupleData[] = [];
      if (importFormat === "json") {
        const raw = JSON.parse(importText);
        const list = Array.isArray(raw) ? raw : [raw];
        for (const item of list) {
          parsed.push({
            objectType: item.objectType || item.namespace || "",
            objectId: item.objectId || item.object || "",
            relation: item.relation || "",
            subjectType: item.subjectType || item.subject_namespace || "user",
            subjectId: item.subjectId || item.subject_id || "",
            subjectRelation: item.subjectRelation || item.subject_relation || "",
          });
        }
      } else {
        // CSV: objectType,objectId,relation,subjectType,subjectId[,subjectRelation]
        const lines = importText.trim().split("\n");
        for (const line of lines) {
          const parts = line.split(",").map(s => s.trim());
          if (parts.length >= 5) {
            parsed.push({
              objectType: parts[0],
              objectId: parts[1],
              relation: parts[2],
              subjectType: parts[3],
              subjectId: parts[4],
              subjectRelation: parts[5] || "",
            });
          }
        }
      }

      if (onBulkAdd) {
        await onBulkAdd(parsed);
      } else {
        for (const t of parsed) {
          await onAdd(t);
        }
      }

      setImportText("");
      setIsImportOpen(false);
    } catch (err) {
      console.error("Bulk import failed", err);
    } finally {
      setIsImporting(false);
    }
  };

  return (
    <div className="flex flex-col h-full bg-card rounded-md border text-card-foreground shadow-sm">
      <div className="p-4 border-b flex flex-wrap items-center justify-between gap-3">
        <div className="flex items-center gap-3">
          <h3 className="font-semibold text-sm">Relation Tuples</h3>
          {uniqueNamespaces.length > 0 && (
            <select
              value={namespaceFilter}
              onChange={(e) => setNamespaceFilter(e.target.value)}
              className="text-xs bg-muted/40 border border-input rounded px-2 py-1"
            >
              <option value="all">All Namespaces</option>
              {uniqueNamespaces.map(ns => (
                <option key={ns} value={ns}>{ns}</option>
              ))}
            </select>
          )}
        </div>

        <div className="flex items-center gap-2">
          {/* Bulk Import Modal */}
          <Dialog open={isImportOpen} onOpenChange={setIsImportOpen}>
            <DialogTrigger asChild>
              <Button variant="outline" size="sm">Bulk Import</Button>
            </DialogTrigger>
            <DialogContent className="max-w-xl">
              <DialogHeader>
                <DialogTitle>Bulk Import Relation Tuples</DialogTitle>
              </DialogHeader>
              <div className="space-y-4 pt-2">
                <div className="flex items-center gap-4 text-xs">
                  <label className="flex items-center gap-1.5 cursor-pointer">
                    <input
                      type="radio"
                      name="format"
                      checked={importFormat === "csv"}
                      onChange={() => setImportFormat("csv")}
                    />
                    CSV (namespace,object,relation,subject_namespace,subject_id)
                  </label>
                  <label className="flex items-center gap-1.5 cursor-pointer">
                    <input
                      type="radio"
                      name="format"
                      checked={importFormat === "json"}
                      onChange={() => setImportFormat("json")}
                    />
                    JSON Array
                  </label>
                </div>

                <textarea
                  value={importText}
                  onChange={(e) => setImportText(e.target.value)}
                  placeholder={
                    importFormat === "csv"
                      ? "document,doc_1,viewer,user,alice\ndocument,doc_1,editor,user,bob"
                      : '[{"objectType":"document","objectId":"doc_1","relation":"viewer","subjectType":"user","subjectId":"alice"}]'
                  }
                  rows={6}
                  className="w-full text-xs font-mono p-3 rounded border bg-muted/20"
                />

                <div className="flex justify-end gap-2">
                  <Button variant="ghost" size="sm" onClick={() => setIsImportOpen(false)}>
                    Cancel
                  </Button>
                  <Button size="sm" onClick={handleBulkImport} disabled={isImporting || !importText.trim()}>
                    {isImporting ? "Importing..." : "Validate & Commit"}
                  </Button>
                </div>
              </div>
            </DialogContent>
          </Dialog>

          {/* Add Tuple Modal */}
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
      </div>
      <div className="flex-1 p-4 overflow-y-auto">
        <DataTable 
          columns={tupleColumns} 
          data={filteredTuples} 
          searchKey="relation"
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

