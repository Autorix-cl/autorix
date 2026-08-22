"use client";

import * as React from "react";
import { Users, Plus, UserCheck, Search, RefreshCw } from "lucide-react";
import { useTranslation } from "@/lib/i18n";
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { useApiQuery } from "@/lib/query/use-api-query";
import { fetchAndParse } from "@/lib/api/schema";
import { paginatedIdentityListSchema, type Identity, type PaginatedIdentities } from "@/lib/api/schemas/identity";
import { ErrorState } from "@/components/state/error-state";
import { NotConnectedState } from "@/components/state/not-connected-state";
import { NotConnectedEngine } from "@/components/resources/not-connected-engine";
import { useCapabilities } from "@/lib/capabilities/capability-context";
import { DataTable } from "@/components/ui/data-table";
import { getColumns, IdentityItem } from "./columns";
import { IdentitySheet } from "./identity-sheet";
import { IdentityBuilderSheet } from "./identity-builder-sheet";

function toIdentityItem(item: Identity): IdentityItem {
  const traits = item.traits as { email?: string; name?: { first?: string; last?: string } };
  return {
    id: item.id,
    email: traits?.email || "unknown@autorix.io",
    name: traits?.name
      ? `${traits.name.first || ""} ${traits.name.last || ""}`.trim()
      : traits?.email?.split("@")[0] || "User",
    state: item.state || "active",
    createdAt: item.created_at ? new Date(item.created_at).toLocaleString() : "Recently",
    original: item,
  };
}

export default function IdentitiesPage() {
  const { t } = useTranslation();
  const { isEngineConnected } = useCapabilities();

  const [searchQueryInput, setSearchQueryInput] = React.useState("");
  const [searchQuery, setSearchQuery] = React.useState("");
  const [cursor, setCursor] = React.useState("");
  const [cursorHistory, setCursorHistory] = React.useState<string[]>([]);
  
  const [selectedIdentity, setSelectedIdentity] = React.useState<IdentityItem | null>(null);
  const [isBuilderOpen, setIsBuilderOpen] = React.useState(false);

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
  } = useApiQuery(
    ["identities", { q: searchQuery, cursor }],
    () => fetchAndParse<PaginatedIdentities>(
      `/api/identities?q=${encodeURIComponent(searchQuery)}&cursor=${encodeURIComponent(cursor)}`, 
      paginatedIdentityListSchema
    )
  );

  const identities: IdentityItem[] = React.useMemo(() => (identitiesRaw?.data ?? []).map(toIdentityItem), [identitiesRaw]);
  
  const columns = React.useMemo(() => getColumns((identity) => setSelectedIdentity(identity)), []);

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

  const handlePrevPage = () => {
    setCursorHistory((prev) => {
      const newHistory = [...prev];
      const prevCursor = newHistory.pop() || "";
      setCursor(prevCursor);
      return newHistory;
    });
  };

  return (
    <div className="space-y-6">
      <IdentitySheet 
        identity={selectedIdentity} 
        isOpen={!!selectedIdentity} 
        onOpenChange={(open) => !open && setSelectedIdentity(null)} 
      />

      <IdentityBuilderSheet
        isOpen={isBuilderOpen}
        onOpenChange={setIsBuilderOpen}
      />

      {/* Page Header */}
      <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
        <div>
          <h1 className="text-xl font-bold tracking-tight text-foreground sm:text-2xl">{t("identities.title")}</h1>
          <p className="text-xs text-muted-foreground mt-1">{t("identities.subtitle")}</p>
        </div>

        <div className="flex items-center gap-2">
          <Badge variant="info" className="gap-1.5 py-1 px-3">
            <UserCheck className="h-3.5 w-3.5" />
            <span>{t("identities.statusBadge")}</span>
          </Badge>
          <Button
            variant="outline"
            size="sm"
            onClick={() => refetch()}
            disabled={isFetching}
            className="h-8 gap-1 text-xs"
          >
            <RefreshCw className={`h-3.5 w-3.5 ${isFetching ? "animate-spin" : ""}`} />
            <span>{t("common.refresh")}</span>
          </Button>
          <Button
            size="sm"
            onClick={() => setIsBuilderOpen(true)}
            className="h-8 gap-1 text-xs bg-blue-600 hover:bg-blue-700 text-white"
          >
            <Plus className="h-3.5 w-3.5" />
            <span>Create Identity</span>
          </Button>
        </div>
      </div>

      {/* Identities Directory Table */}
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
              onNextPage={handleNextPage}
              onPreviousPage={handlePrevPage}
              canNextPage={!!identitiesRaw?.has_more}
              canPreviousPage={cursorHistory.length > 0}
            />
          )}
        </CardContent>
      </Card>
    </div>
  );
}
