"use client";

import * as React from "react";
import { Suspense } from "react";
import { useSearchParams } from "next/navigation";
import {
  Users,
  Plus,
  UserCheck,
  Search,
  RefreshCw,
  UploadCloud,
  Download,
  FileCode,
  Shield,
  BookOpen,
} from "lucide-react";
import { ServiceHeader } from "@/components/layout/service-header";
import { CloudSection } from "@/components/layout/cloud-section";

import { useTranslation } from "@/lib/i18n";
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { Can } from "@/lib/auth/use-permission";
import { useApiQuery } from "@/lib/query/use-api-query";
import { fetchAndParse } from "@/lib/api/schema";
import { paginatedIdentityListSchema, type Identity, type PaginatedIdentities } from "@/lib/api/schemas/identity";
import { formatDateTime } from "@/lib/format/date-time";
import { ErrorState } from "@/components/state/error-state";
import { NotConnectedState } from "@/components/state/not-connected-state";
import { NotConnectedEngine } from "@/components/resources/not-connected-engine";
import { useCapabilities } from "@/lib/capabilities/capability-context";
import { DataTable } from "@/components/ui/data-table";
import { getColumns, IdentityItem } from "./columns";
import { IdentitySheet } from "./identity-sheet";
import { IdentityBuilderSheet } from "./identity-builder-sheet";
import { SchemaStudio } from "./schema-studio";
import { TraitsGuide } from "./traits-guide";
import { BulkActionBar } from "./bulk-action-bar";
import { BulkImportDialog } from "./bulk-import-dialog";
import { toast } from "sonner";

function escapeCsvCell(value: unknown): string {
  const str = String(value ?? "");
  const neutralized = /^[=+\-@\t\r]/.test(str) ? `'${str}` : str;
  return `"${neutralized.replace(/"/g, '""')}"`;
}

function readCsrfCookie(): string | undefined {
  if (typeof document === "undefined") return undefined;
  const match = document.cookie.match(/(?:^|;\s*)autorix_csrf=([^;]+)/);
  return match ? decodeURIComponent(match[1]) : undefined;
}

function toIdentityItem(item: Identity): IdentityItem {
  const traits = item.traits as { email?: string; name?: { first?: string; last?: string } };
  return {
    id: item.id,
    email: traits?.email || "unknown@autorix.io",
    name: traits?.name
      ? `${traits.name.first || ""} ${traits.name.last || ""}`.trim()
      : traits?.email?.split("@")[0] || "User",
    state: item.state || "active",
    createdAt: item.created_at ? formatDateTime(item.created_at) : "Recently",
    original: item,
  };
}

function IdentitiesPageContent() {
  const { t } = useTranslation();
  const { isEngineConnected } = useCapabilities();
  const searchParams = useSearchParams();
  const requestedId = searchParams.get("id");

  const [searchQueryInput, setSearchQueryInput] = React.useState("");
  const [searchQuery, setSearchQuery] = React.useState("");
  const [pageSize, setPageSize] = React.useState(10);
  const [cursor, setCursor] = React.useState("");
  const [cursorHistory, setCursorHistory] = React.useState<string[]>([]);

  const [selectedIdentity, setSelectedIdentity] = React.useState<IdentityItem | null>(null);
  const [isBuilderOpen, setIsBuilderOpen] = React.useState(false);
  const [isBulkImportOpen, setIsBulkImportOpen] = React.useState(false);
  const [pendingSuspend, setPendingSuspend] = React.useState<{
    ids: string[];
    clear: () => void;
  } | null>(null);
  const [isSuspending, setIsSuspending] = React.useState(false);

  React.useEffect(() => {
    const handler = setTimeout(() => {
      setSearchQuery(searchQueryInput);
      setCursor("");
      setCursorHistory([]);
    }, 300);
    return () => clearTimeout(handler);
  }, [searchQueryInput]);

  const {
    data: identitiesRaw,
    isLoading,
    isFetching,
    isError,
    error,
    refetch,
  } = useApiQuery(["identities", { q: searchQuery, cursor, limit: pageSize }], () =>
    fetchAndParse<PaginatedIdentities>(
      `/api/identities?q=${encodeURIComponent(searchQuery)}&cursor=${encodeURIComponent(cursor)}&limit=${pageSize}`,
      paginatedIdentityListSchema,
    ),
  );

  const identities: IdentityItem[] = React.useMemo(
    () => (identitiesRaw?.data ?? []).map(toIdentityItem),
    [identitiesRaw],
  );

  React.useEffect(() => {
    if (!requestedId || selectedIdentity?.id === requestedId) return;
    const match = identities.find((i) => i.id === requestedId);
    if (match) {
      setSelectedIdentity(match);
    }
  }, [requestedId, identities, selectedIdentity?.id]);

  const columns = React.useMemo(() => getColumns((identity) => setSelectedIdentity(identity)), []);

  const handleConfirmBulkSuspend = async () => {
    if (!pendingSuspend || pendingSuspend.ids.length === 0) return;
    setIsSuspending(true);
    try {
      const csrf = readCsrfCookie();
      const results = await Promise.allSettled(
        pendingSuspend.ids.map(async (id) => {
          const res = await fetch(`/api/identities/${encodeURIComponent(id)}`, {
            method: "PATCH",
            headers: {
              "Content-Type": "application/json",
              ...(csrf ? { "X-CSRF-Token": csrf } : {}),
            },
            body: JSON.stringify({ state: "suspended" }),
          });
          if (!res.ok) throw new Error(`HTTP ${res.status}`);
        }),
      );
      const succeeded = results.filter((r) => r.status === "fulfilled").length;
      const failed = results.length - succeeded;
      if (succeeded > 0) {
        toast.success(`Suspended ${succeeded} identit${succeeded === 1 ? "y" : "ies"}`);
      }
      if (failed > 0) {
        toast.error(`Failed to suspend ${failed} identit${failed === 1 ? "y" : "ies"}`);
      }
      pendingSuspend.clear();
      setPendingSuspend(null);
      refetch();
    } finally {
      setIsSuspending(false);
    }
  };

  const handleExportCsv = () => {
    if (identities.length === 0) {
      toast.info("No identities to export");
      return;
    }
    const headers = ["id", "email", "name", "state", "createdAt"].map((h) => escapeCsvCell(h)).join(",") + "\n";
    const csvContent = identities
      .map((i) => [i.id, i.email, i.name, i.state, i.createdAt].map((c) => escapeCsvCell(c)).join(","))
      .join("\n");
    const blob = new Blob([headers + csvContent], { type: "text/csv;charset=utf-8;" });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.setAttribute("href", url);
    link.setAttribute("download", `autorix_identities_${Date.now()}.csv`);
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
    toast.success(`Exported ${identities.length} identities`);
  };

  if (!isEngineConnected("ego")) {
    return (
      <div className="space-y-6">
        <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
          <div>
            <h1 className="text-xl font-bold tracking-tight text-foreground sm:text-2xl">{t("identities.title")}</h1>
            <p className="text-xs text-muted-foreground mt-1">{t("identities.subtitle")}</p>
          </div>
        </div>
        <NotConnectedEngine
          engineType="ego"
          engineName="Autorix Ego (Identity)"
          description="Identity lifecycle, passwordless authentication, and session vault."
        />
      </div>
    );
  }

  const handleNextPage = () => {
    if (identitiesRaw?.has_more && identitiesRaw.next_cursor) {
      setCursorHistory((prev) => [...prev, cursor]);
      setCursor(identitiesRaw.next_cursor);
    }
  };

  const handlePreviousPage = () => {
    if (cursorHistory.length > 0) {
      const prevCursor = cursorHistory[cursorHistory.length - 1];
      setCursorHistory((prev) => prev.slice(0, -1));
      setCursor(prevCursor);
    }
  };

  const handlePageSizeChange = (newSize: number) => {
    setPageSize(newSize);
    setCursor("");
    setCursorHistory([]);
  };

  return (
    <div className="space-y-6">
      <IdentitySheet
        identity={selectedIdentity}
        isOpen={!!selectedIdentity}
        onOpenChange={(open) => !open && setSelectedIdentity(null)}
        onIdentityUpdated={() => refetch()}
      />

      <IdentityBuilderSheet isOpen={isBuilderOpen} onOpenChange={setIsBuilderOpen} onSuccess={() => refetch()} />

      <BulkImportDialog isOpen={isBulkImportOpen} onOpenChange={setIsBulkImportOpen} onSuccess={() => refetch()} />

      {/* Cloud Service Header & Telemetry HUD */}
      <ServiceHeader
        serviceName="Ego Identity Engine"
        title={t("identities.title")}
        description={t("identities.subtitle")}
        icon={Users}
        iconColor="text-cyan-400"
        statusText={t("identities.statusBadge")}
        statusVariant="cyan"
        metrics={[
          {
            label: "Total Identities",
            value: identities.length,
            hint: "Registered subjects",
            icon: Users,
          },
          {
            label: "Active Accounts",
            value: identities.filter((i) => i.state === "active").length,
            hint: "Fully verified",
            icon: UserCheck,
          },
          {
            label: "Identity Schema",
            value: "Draft-07",
            hint: "Ory Trait Model",
            icon: FileCode,
          },
          {
            label: "Engine Gateway",
            value: "Port 4433",
            hint: "mTLS SPIFFE Mesh",
            icon: Shield,
          },
        ]}
        actions={
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={handleExportCsv}
              className="h-8 gap-1.5 text-xs shrink-0"
              title="Export identities as CSV"
            >
              <Download className="h-3.5 w-3.5" />
              <span>Export CSV</span>
            </Button>
            <Can perform="identities:write">
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => setIsBulkImportOpen(true)}
                className="h-8 gap-1.5 text-xs shrink-0"
                title="Bulk import CSV or JSON identities"
              >
                <UploadCloud className="h-3.5 w-3.5" aria-hidden="true" />
                <span>Bulk Import</span>
              </Button>
            </Can>
            <Button
              type="button"
              variant="outline"
              size="icon-sm"
              onClick={() => refetch()}
              disabled={isFetching}
              className="h-8 w-8 shrink-0"
              title={t("common.refresh")}
              aria-label={t("common.refresh")}
            >
              <RefreshCw className={`h-3.5 w-3.5 ${isFetching ? "animate-spin" : ""}`} aria-hidden="true" />
            </Button>
            <Can perform="identities:write">
              <Button
                type="button"
                size="sm"
                onClick={() => setIsBuilderOpen(true)}
                className="h-8 gap-1.5 text-xs bg-cyan-600 hover:bg-cyan-700 text-white font-medium shadow-xs shrink-0"
              >
                <Plus className="h-3.5 w-3.5" aria-hidden="true" />
                <span>{t("identities.createButton") || "Create Identity"}</span>
              </Button>
            </Can>
          </div>
        }
      />

      {/* Studio Navigation Tabs */}
      <Tabs defaultValue="directory" className="space-y-4">
        <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3 border-b border-border pb-3">
          <TabsList className="bg-muted/40 p-1">
            <TabsTrigger value="directory" className="gap-2 text-xs font-medium">
              <Users className="h-3.5 w-3.5 text-cyan-400" />
              <span>Identities Directory</span>
            </TabsTrigger>
            <TabsTrigger value="schemas" className="gap-2 text-xs font-medium">
              <FileCode className="h-3.5 w-3.5 text-blue-400" />
              <span>Schema Studio & Builder</span>
            </TabsTrigger>
            <TabsTrigger value="guide" className="gap-2 text-xs font-medium">
              <BookOpen className="h-3.5 w-3.5 text-purple-400" />
              <span>Traits Architecture Guide</span>
            </TabsTrigger>
          </TabsList>
        </div>

        {/* Tab 1: Identities Directory Section */}
        <TabsContent value="directory" className="space-y-4 focus-visible:outline-none">
          <CloudSection
            title="Identity Directory & Trait Profiles"
            description="Cryptographically authenticated subjects and managed SPIFFE identities"
            icon={Users}
            badge={`${identities.length} Identities`}
            badgeVariant="cyan"
          >
            <Card className="bg-card/80">
              <CardHeader className="p-6 pb-4">
                <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                  <div>
                    <CardTitle className="text-sm font-semibold flex items-center gap-2">
                      <Users className="h-4 w-4 text-blue-400" />
                      <span>{t("identities.tableTitle")}</span>
                    </CardTitle>
                    <CardDescription className="text-xs">{t("identities.tableDesc")}</CardDescription>
                  </div>

                  {/* Filter Search */}
                  <div className="flex items-center gap-2 w-full sm:w-72">
                    <div className="relative w-full">
                      <Search className="absolute left-2.5 top-2.5 h-3.5 w-3.5 text-muted-foreground" />
                      <Input
                        id="searchQuery"
                        placeholder={t("identities.searchPlaceholder")}
                        value={searchQueryInput}
                        onChange={(e) => setSearchQueryInput(e.target.value)}
                        className="pl-8 h-8 text-xs bg-muted/30"
                      />
                    </div>
                  </div>
                </div>
              </CardHeader>

              <CardContent className="p-6 pt-0">
                {isError && error?.kind === "engine-unreachable" ? (
                  <NotConnectedState engineName="Ego" onRetry={refetch} />
                ) : isError ? (
                  <ErrorState error={error} onRetry={refetch} />
                ) : (
                  <DataTable
                    columns={columns}
                    data={identities}
                    isLoading={isLoading || isFetching}
                    manualPagination={true}
                    pageIndex={cursorHistory.length}
                    pageSize={pageSize}
                    defaultPageSize={pageSize}
                    onPageSizeChange={handlePageSizeChange}
                    onNextPage={handleNextPage}
                    onPreviousPage={handlePreviousPage}
                    canNextPage={!!identitiesRaw?.has_more}
                    canPreviousPage={cursorHistory.length > 0}
                    renderToolbar={(table) => {
                      const selectedRows = table.getFilteredSelectedRowModel().rows;
                      const selectedCount = selectedRows.length;
                      return (
                        <Can perform="identities:write">
                          <BulkActionBar
                            selectedCount={selectedCount}
                            onSuspend={() => {
                              setPendingSuspend({
                                ids: selectedRows.map((r) => r.original.id),
                                clear: () => table.resetRowSelection(),
                              });
                            }}
                            onClearSelection={() => table.resetRowSelection()}
                          />
                        </Can>
                      );
                    }}
                  />
                )}
              </CardContent>
            </Card>
          </CloudSection>
        </TabsContent>

        {/* Tab 2: Schema Studio & Builder */}
        <TabsContent value="schemas" className="space-y-4 focus-visible:outline-none">
          <CloudSection
            title="Identity Schema Studio & Trait Architect"
            description="Author, validate Draft-07 JSON schemas, define credential identifiers, and preview interactive forms"
            icon={FileCode}
            badge="Ory Kratos Trait Model"
            badgeVariant="cyan"
          >
            <SchemaStudio onSchemaCreated={() => refetch()} />
          </CloudSection>
        </TabsContent>

        {/* Tab 3: Traits Architecture Guide */}
        <TabsContent value="guide" className="space-y-4 focus-visible:outline-none">
          <CloudSection
            title="Traits & Declarative Identity Architecture"
            description="Comprehensive guide on dynamic trait models, credential mapping, and API integration"
            icon={BookOpen}
            badge="Documentation"
            badgeVariant="purple"
          >
            <TraitsGuide />
          </CloudSection>
        </TabsContent>
      </Tabs>

      <ConfirmDialog
        open={Boolean(pendingSuspend)}
        onOpenChange={(open) => !open && setPendingSuspend(null)}
        title="Suspend Selected Identities"
        description={`Are you sure you want to suspend ${pendingSuspend?.ids.length ?? 0} selected identit${(pendingSuspend?.ids.length ?? 0) === 1 ? "y" : "ies"}? Active sessions will be invalidated and logins blocked until reactivated.`}
        confirmLabel="Suspend Selected"
        isLoading={isSuspending}
        onConfirm={handleConfirmBulkSuspend}
      />
    </div>
  );
}

export default function IdentitiesPage() {
  return (
    <Suspense fallback={<div className="p-8 text-xs text-muted-foreground font-mono">Loading identities...</div>}>
      <IdentitiesPageContent />
    </Suspense>
  );
}
