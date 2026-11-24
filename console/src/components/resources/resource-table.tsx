"use client";

import * as React from "react";
import {
  ArrowUpDown,
  ArrowUp,
  ArrowDown,
  Search,
  Download,
  Trash2,
  CheckSquare,
  Square,
  RefreshCw,
} from "lucide-react";
import {
  Table,
  TableHeader,
  TableBody,
  TableHead,
  TableRow,
  TableCell,
} from "@/components/ui/table";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import type { ResourceDescriptor } from "@/lib/resources/descriptor";
import type { TableQueryState } from "@/lib/table/url-state";

export interface ResourceTableProps<T extends Record<string, unknown>> {
  descriptor: ResourceDescriptor<T>;
  data: T[];
  total?: number;
  isLoading?: boolean;
  onRefresh?: () => void;
  onDeleteSelected?: (selectedIds: string[]) => void;
  onRowClick?: (record: T) => void;
}

export function ResourceTable<T extends Record<string, unknown> & { id: string }>({
  descriptor,
  data,
  total,
  isLoading,
  onRefresh,
  onDeleteSelected,
  onRowClick,
}: ResourceTableProps<T>) {
  const [queryState, setQueryState] = React.useState<TableQueryState>({
    page: 1,
    pageSize: 10,
    sortBy: descriptor.defaultSort?.field,
    sortOrder: descriptor.defaultSort?.order || "asc",
    search: "",
    filters: {},
  });

  const [selectedIds, setSelectedIds] = React.useState<Set<string>>(new Set());

  // Filter and sort data client-side if full collection is provided
  const filteredData = React.useMemo(() => {
    let list = [...data];

    // Search filter
    if (queryState.search.trim()) {
      const q = queryState.search.toLowerCase();
      list = list.filter((item) =>
        Object.values(item).some((val) => String(val).toLowerCase().includes(q))
      );
    }

    // Sorting
    if (queryState.sortBy) {
      const field = queryState.sortBy;
      const order = queryState.sortOrder === "desc" ? -1 : 1;
      list.sort((a, b) => {
        const valA = a[field];
        const valB = b[field];
        if (valA === valB) return 0;
        if (valA === undefined || valA === null) return 1;
        if (valB === undefined || valB === null) return -1;
        return valA > valB ? order : -order;
      });
    }

    return list;
  }, [data, queryState.search, queryState.sortBy, queryState.sortOrder]);

  const totalCount = total ?? filteredData.length;
  const totalPages = Math.max(1, Math.ceil(totalCount / queryState.pageSize));
  const paginatedData = React.useMemo(() => {
    const start = (queryState.page - 1) * queryState.pageSize;
    return filteredData.slice(start, start + queryState.pageSize);
  }, [filteredData, queryState.page, queryState.pageSize]);

  const handleSort = (field: string) => {
    setQueryState((prev) => {
      if (prev.sortBy === field) {
        return { ...prev, sortOrder: prev.sortOrder === "asc" ? "desc" : "asc" };
      }
      return { ...prev, sortBy: field, sortOrder: "asc" };
    });
  };

  const toggleSelectAll = () => {
    if (selectedIds.size === paginatedData.length) {
      setSelectedIds(new Set());
    } else {
      setSelectedIds(new Set(paginatedData.map((d) => d.id)));
    }
  };

  const toggleSelectRow = (id: string, e: React.MouseEvent) => {
    e.stopPropagation();
    setSelectedIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  };

  const handleExportJSON = () => {
    const blob = new Blob([JSON.stringify(filteredData, null, 2)], { type: "application/json" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `${descriptor.id}-export.json`;
    a.click();
    URL.revokeObjectURL(url);
  };

  const escapeCsvCell = (raw: unknown): string => {
    const str = String(raw ?? "");
    const neutralized = /^[=+\-@\t\r]/.test(str) ? `'${str}` : str;
    return `"${neutralized.replace(/"/g, '""')}"`;
  };

  const handleExportCSV = () => {
    if (filteredData.length === 0) return;
    const headers = descriptor.columns.map((col) => col.key);
    const rows = filteredData.map((row) =>
      headers.map((h) => escapeCsvCell(row[h])).join(",")
    );
    const csvContent = [headers.map((h) => escapeCsvCell(h)).join(","), ...rows].join("\n");
    const blob = new Blob([csvContent], { type: "text/csv" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `${descriptor.id}-export.csv`;
    a.click();
    URL.revokeObjectURL(url);
  };

  return (
    <div className="space-y-4">
      {/* Table Toolbar */}
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex items-center gap-2 flex-1 max-w-sm">
          <div className="relative w-full">
            <Search className="absolute left-2.5 top-2.5 h-3.5 w-3.5 text-muted-foreground" aria-hidden="true" />
            <Input
              aria-label={`Search ${descriptor.pluralName.toLowerCase()}`}
              placeholder={`Search ${descriptor.pluralName.toLowerCase()}...`}
              value={queryState.search}
              onChange={(e) => setQueryState((prev) => ({ ...prev, search: e.target.value, page: 1 }))}
              className="pl-8 h-8 text-xs bg-card"
            />
          </div>
        </div>

        <div className="flex items-center gap-2">
          {selectedIds.size > 0 && onDeleteSelected && (
            <Button
              type="button"
              variant="destructive"
              size="sm"
              onClick={() => onDeleteSelected(Array.from(selectedIds))}
              className="h-8 gap-1.5 text-xs"
            >
              <Trash2 className="h-3.5 w-3.5" aria-hidden="true" />
              Delete ({selectedIds.size})
            </Button>
          )}

          <Button type="button" variant="outline" size="sm" onClick={handleExportCSV} className="h-8 gap-1.5 text-xs">
            <Download className="h-3.5 w-3.5" aria-hidden="true" />
            CSV
          </Button>

          <Button type="button" variant="outline" size="sm" onClick={handleExportJSON} className="h-8 gap-1.5 text-xs">
            <Download className="h-3.5 w-3.5" aria-hidden="true" />
            JSON
          </Button>

          {onRefresh && (
            <Button
              type="button"
              variant="ghost"
              size="icon-sm"
              onClick={onRefresh}
              aria-label={`Refresh ${descriptor.pluralName.toLowerCase()}`}
              className="h-8 w-8"
            >
              <RefreshCw className="h-3.5 w-3.5" aria-hidden="true" />
            </Button>
          )}
        </div>
      </div>

      {/* Main Table */}
      <div className="rounded-lg border border-border/80 bg-card overflow-x-auto">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-10 px-3">
                <button
                  type="button"
                  onClick={toggleSelectAll}
                  aria-label={
                    selectedIds.size > 0 && selectedIds.size === paginatedData.length
                      ? "Deselect all visible rows"
                      : "Select all visible rows"
                  }
                  className="text-muted-foreground hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring rounded"
                >
                  {selectedIds.size > 0 && selectedIds.size === paginatedData.length ? (
                    <CheckSquare className="h-4 w-4 text-primary" aria-hidden="true" />
                  ) : (
                    <Square className="h-4 w-4" aria-hidden="true" />
                  )}
                </button>
              </TableHead>
              {descriptor.columns.map((col) => (
                <TableHead
                  key={col.key}
                  aria-sort={
                    col.sortable
                      ? queryState.sortBy === col.key
                        ? queryState.sortOrder === "desc"
                          ? "descending"
                          : "ascending"
                        : "none"
                      : undefined
                  }
                >
                  {col.sortable ? (
                    <button
                      type="button"
                      onClick={() => handleSort(col.key)}
                      className="flex items-center gap-1.5 hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring rounded"
                    >
                      <span>{col.label}</span>
                      <span className="text-muted-foreground" aria-hidden="true">
                        {queryState.sortBy === col.key ? (
                          queryState.sortOrder === "desc" ? (
                            <ArrowDown className="h-3 w-3 text-primary" />
                          ) : (
                            <ArrowUp className="h-3 w-3 text-primary" />
                          )
                        ) : (
                          <ArrowUpDown className="h-3 w-3 opacity-40" />
                        )}
                      </span>
                    </button>
                  ) : (
                    <span>{col.label}</span>
                  )}
                </TableHead>
              ))}
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <TableRow>
                <TableCell colSpan={descriptor.columns.length + 1} className="py-12 text-center text-xs text-muted-foreground font-mono">
                  Loading {descriptor.pluralName.toLowerCase()}...
                </TableCell>
              </TableRow>
            ) : paginatedData.length === 0 ? (
              <TableRow>
                <TableCell colSpan={descriptor.columns.length + 1} className="py-12 text-center text-xs text-muted-foreground">
                  No {descriptor.pluralName.toLowerCase()} found matching current filters.
                </TableCell>
              </TableRow>
            ) : (
              paginatedData.map((record) => {
                const isSelected = selectedIds.has(record.id);
                return (
                  <TableRow
                    key={record.id}
                    onClick={() => onRowClick?.(record)}
                    onKeyDown={
                      onRowClick
                        ? (e) => {
                            if (e.key === "Enter" || e.key === " ") {
                              e.preventDefault();
                              onRowClick(record);
                            }
                          }
                        : undefined
                    }
                    tabIndex={onRowClick ? 0 : undefined}
                    className={
                      onRowClick
                        ? "cursor-pointer hover:bg-muted/40 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring"
                        : ""
                    }
                    data-state={isSelected ? "selected" : undefined}
                  >
                    <TableCell className="w-10 px-3">
                      <button
                        type="button"
                        onClick={(e) => toggleSelectRow(record.id, e)}
                        aria-label={isSelected ? `Deselect ${record.id}` : `Select ${record.id}`}
                        aria-pressed={isSelected}
                        className="text-muted-foreground hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring rounded"
                      >
                        {isSelected ? (
                          <CheckSquare className="h-4 w-4 text-primary" aria-hidden="true" />
                        ) : (
                          <Square className="h-4 w-4 text-muted-foreground" aria-hidden="true" />
                        )}
                      </button>
                    </TableCell>
                    {descriptor.columns.map((col) => (
                      <TableCell key={col.key}>
                        {col.render
                          ? col.render(record[col.key], record)
                          : String(record[col.key] ?? "-")}
                      </TableCell>
                    ))}
                  </TableRow>
                );
              })
            )}
          </TableBody>
        </Table>
      </div>

      {/* Pagination Footer */}
      <div className="flex items-center justify-between text-xs text-muted-foreground px-1">
        <div>
          Showing {paginatedData.length} of {totalCount} {descriptor.pluralName.toLowerCase()}
        </div>
        <div className="flex items-center gap-2">
          <span>Page {queryState.page} of {totalPages}</span>
          <Button
            variant="outline"
            size="sm"
            disabled={queryState.page <= 1}
            onClick={() => setQueryState((prev) => ({ ...prev, page: prev.page - 1 }))}
            className="h-7 px-2 text-xs"
          >
            Previous
          </Button>
          <Button
            variant="outline"
            size="sm"
            disabled={queryState.page >= totalPages}
            onClick={() => setQueryState((prev) => ({ ...prev, page: prev.page + 1 }))}
            className="h-7 px-2 text-xs"
          >
            Next
          </Button>
        </div>
      </div>
    </div>
  );
}
