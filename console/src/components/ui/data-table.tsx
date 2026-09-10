"use client";

import * as React from "react";
import {
  ColumnDef,
  ColumnFiltersState,
  SortingState,
  VisibilityState,
  flexRender,
  getCoreRowModel,
  getFilteredRowModel,
  getPaginationRowModel,
  getSortedRowModel,
  useReactTable,
  Table as TanStackTable,
} from "@tanstack/react-table";

import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Loader2, SlidersHorizontal, Rows, FolderOpen, ChevronLeft, ChevronRight } from "lucide-react";
import { cn } from "@/lib/utils";

interface DataTableProps<TData, TValue> {
  columns: ColumnDef<TData, TValue>[];
  data: TData[];
  manualPagination?: boolean;
  pageCount?: number;
  onNextPage?: () => void;
  onPreviousPage?: () => void;
  canNextPage?: boolean;
  canPreviousPage?: boolean;
  isLoading?: boolean;
  searchKey?: string;
  renderToolbar?: (table: TanStackTable<TData>) => React.ReactNode;
  emptyState?: React.ReactNode;
  defaultDensity?: "comfortable" | "compact";
  pageSizeOptions?: number[];
  defaultPageSize?: number;
  pageSize?: number;
  pageIndex?: number;
  onPageSizeChange?: (pageSize: number) => void;
}

export function DataTable<TData, TValue>({
  columns,
  data,
  manualPagination,
  pageCount,
  pageIndex: controlledPageIndex,
  onNextPage,
  onPreviousPage,
  canNextPage,
  canPreviousPage,
  isLoading,
  searchKey,
  renderToolbar,
  emptyState,
  defaultDensity = "comfortable",
  pageSizeOptions = [10, 20, 50, 100],
  defaultPageSize = 10,
  pageSize: controlledPageSize,
  onPageSizeChange,
}: DataTableProps<TData, TValue>) {
  const [density, setDensity] = React.useState<"comfortable" | "compact">(defaultDensity);
  const [sorting, setSorting] = React.useState<SortingState>([]);
  const [columnFilters, setColumnFilters] = React.useState<ColumnFiltersState>([]);
  const [columnVisibility, setColumnVisibility] = React.useState<VisibilityState>({});
  const [rowSelection, setRowSelection] = React.useState({});
  const [pagination, setPagination] = React.useState({
    pageIndex: controlledPageIndex ?? 0,
    pageSize: controlledPageSize ?? defaultPageSize,
  });

  React.useEffect(() => {
    if (controlledPageSize !== undefined) {
      setPagination((prev) =>
        controlledPageSize !== prev.pageSize
          ? { ...prev, pageSize: controlledPageSize, pageIndex: 0 }
          : prev
      );
    }
  }, [controlledPageSize]);

  React.useEffect(() => {
    if (controlledPageIndex !== undefined) {
      setPagination((prev) =>
        controlledPageIndex !== prev.pageIndex ? { ...prev, pageIndex: controlledPageIndex } : prev
      );
    }
  }, [controlledPageIndex]);

  const table = useReactTable({
    data,
    columns,
    getCoreRowModel: getCoreRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    onSortingChange: setSorting,
    onColumnFiltersChange: setColumnFilters,
    onColumnVisibilityChange: setColumnVisibility,
    onRowSelectionChange: setRowSelection,
    onPaginationChange: setPagination,
    manualPagination,
    pageCount,
    state: {
      sorting,
      columnFilters,
      columnVisibility,
      rowSelection,
      pagination,
    },
  });

  const activeRows = table.getRowModel().rows;
  const rowsToRender =
    manualPagination && activeRows.length > pagination.pageSize
      ? activeRows.slice(0, pagination.pageSize)
      : activeRows;

  return (
    <div>
      <div className="flex items-center justify-between gap-2 py-4">
        {searchKey ? (
          <Input
            placeholder="Search..."
            value={(table.getColumn(searchKey)?.getFilterValue() as string) ?? ""}
            onChange={(event) =>
              table.getColumn(searchKey)?.setFilterValue(event.target.value)
            }
            className="max-w-sm"
          />
        ) : (
          <div />
        )}
        <div className="flex items-center gap-2">
          {renderToolbar && renderToolbar(table)}
          <Button
            variant="outline"
            size="sm"
            onClick={() => setDensity(density === "comfortable" ? "compact" : "comfortable")}
            className="flex items-center gap-1.5 text-xs h-9"
            title={`Toggle table density (current: ${density})`}
          >
            <Rows className="h-3.5 w-3.5" />
            <span className="capitalize hidden sm:inline">{density}</span>
          </Button>
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button
                variant="outline"
                size="sm"
                className="ml-auto flex items-center gap-1.5 text-xs h-9"
              >
                <SlidersHorizontal className="h-3.5 w-3.5" />
                Columns
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              {table
                .getAllColumns()
                .filter((column) => column.getCanHide())
                .map((column) => {
                  return (
                    <DropdownMenuCheckboxItem
                      key={column.id}
                      className="capitalize text-xs"
                      checked={column.getIsVisible()}
                      onCheckedChange={(value) => column.toggleVisibility(!!value)}
                    >
                      {column.id}
                    </DropdownMenuCheckboxItem>
                  );
                })}
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>
      <div className="rounded-md border">
        <Table>
          <TableHeader>
            {table.getHeaderGroups().map((headerGroup) => (
              <TableRow key={headerGroup.id}>
                {headerGroup.headers.map((header) => {
                  return (
                    <TableHead
                      key={header.id}
                      className={cn(
                        density === "compact"
                          ? "h-8 px-3 text-[10px]"
                          : "h-10 px-4 text-[11px]"
                      )}
                    >
                      {header.isPlaceholder
                        ? null
                        : flexRender(
                            header.column.columnDef.header,
                            header.getContext()
                          )}
                    </TableHead>
                  );
                })}
              </TableRow>
            ))}
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <TableRow>
                <TableCell
                  colSpan={columns.length}
                  className="h-28 text-center"
                >
                  <div className="flex items-center justify-center">
                    <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
                  </div>
                </TableCell>
              </TableRow>
            ) : rowsToRender.length ? (
              rowsToRender.map((row) => (
                <TableRow
                  key={row.id}
                  data-state={row.getIsSelected() && "selected"}
                  className={cn(density === "compact" ? "h-9" : "")}
                >
                  {row.getVisibleCells().map((cell) => (
                    <TableCell
                      key={cell.id}
                      className={cn(
                        density === "compact" ? "py-1.5 px-3 text-xs" : "p-4"
                      )}
                    >
                      {flexRender(
                        cell.column.columnDef.cell,
                        cell.getContext()
                      )}
                    </TableCell>
                  ))}
                </TableRow>
              ))
            ) : (
              <TableRow>
                <TableCell
                  colSpan={columns.length}
                  className="h-32 text-center"
                >
                  {emptyState ? (
                    emptyState
                  ) : (
                    <div className="flex flex-col items-center justify-center py-6 text-center">
                      <FolderOpen className="h-8 w-8 text-muted-foreground/40 mb-2" />
                      <p className="text-sm font-medium text-foreground">No results.</p>
                      <p className="text-xs text-muted-foreground mt-1 max-w-sm">
                        No records match your active query or no data has been registered yet.
                      </p>
                      {Boolean(searchKey && table.getColumn(searchKey)?.getFilterValue()) && (
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => table.getColumn(searchKey!)?.setFilterValue("")}
                          className="mt-3 text-xs h-7 text-cyan-400 hover:text-cyan-300"
                        >
                          Clear filter
                        </Button>
                      )}
                    </div>
                  )}
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </div>
      <div className="flex flex-col sm:flex-row items-center justify-between gap-4 py-4 text-xs text-muted-foreground">
        <div className="flex items-center gap-2">
          {table.getFilteredSelectedRowModel().rows.length > 0 ? (
            <span>
              {table.getFilteredSelectedRowModel().rows.length} of{" "}
              {table.getFilteredRowModel().rows.length} row(s) selected.
            </span>
          ) : (
            <span>
              Total: {manualPagination && pageCount ? "Paginated" : table.getFilteredRowModel().rows.length} records
            </span>
          )}
        </div>

        <div className="flex flex-wrap items-center gap-4 sm:gap-6">
          <div className="flex items-center gap-2">
            <span className="text-xs font-medium text-muted-foreground whitespace-nowrap">
              Rows per page
            </span>
            <Select
              value={String(table.getState().pagination.pageSize)}
              onValueChange={(value) => {
                const newSize = Number(value);
                setPagination((prev) => ({ ...prev, pageSize: newSize, pageIndex: 0 }));
                table.setPageSize(newSize);
                if (onPageSizeChange) onPageSizeChange(newSize);
              }}
            >
              <SelectTrigger className="h-8 w-[76px] text-xs font-mono">
                <SelectValue placeholder={String(table.getState().pagination.pageSize)} />
              </SelectTrigger>
              <SelectContent side="top">
                {pageSizeOptions.map((size) => (
                  <SelectItem key={size} value={String(size)} className="text-xs font-mono">
                    {size}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="flex items-center justify-center text-xs font-medium">
            Page {(controlledPageIndex ?? table.getState().pagination.pageIndex) + 1}
            {manualPagination ? (pageCount ? ` of ${pageCount}` : "") : ` of ${table.getPageCount() || 1}`}
          </div>

          <div className="flex items-center space-x-2">
            <Button
              variant="outline"
              size="sm"
              className="h-8 gap-1 text-xs"
              onClick={() => {
                if (onPreviousPage) onPreviousPage();
                else table.previousPage();
              }}
              disabled={manualPagination ? !canPreviousPage : !table.getCanPreviousPage()}
            >
              <ChevronLeft className="h-3.5 w-3.5" />
              <span>Previous</span>
            </Button>
            <Button
              variant="outline"
              size="sm"
              className="h-8 gap-1 text-xs"
              onClick={() => {
                if (onNextPage) onNextPage();
                else table.nextPage();
              }}
              disabled={manualPagination ? !canNextPage : !table.getCanNextPage()}
            >
              <span>Next</span>
              <ChevronRight className="h-3.5 w-3.5" />
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
}
