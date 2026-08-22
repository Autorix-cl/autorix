"use client";

import * as React from "react";
import { Users, Plus, ShieldCheck, UserCheck, Search, Fingerprint, RefreshCw, Loader2 } from "lucide-react";
import { useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "@/lib/i18n";
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { CodeBlock } from "@/components/ui/code-block";
import { useApiQuery } from "@/lib/query/use-api-query";
import { useApiMutation } from "@/lib/query/use-api-mutation";
import { fetchAndParse } from "@/lib/api/schema";
import { paginatedIdentityListSchema, registrationResponseSchema, type Identity, type PaginatedIdentities } from "@/lib/api/schemas/identity";
import { ErrorState } from "@/components/state/error-state";
import { NotConnectedState } from "@/components/state/not-connected-state";
import { NotConnectedEngine } from "@/components/resources/not-connected-engine";
import { useCapabilities } from "@/lib/capabilities/capability-context";
import { DataTable } from "@/components/ui/data-table";
import { getColumns, IdentityItem } from "./columns";
import { IdentitySheet } from "./identity-sheet";

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
  const queryClient = useQueryClient();
  const { isEngineConnected } = useCapabilities();

  const [email, setEmail] = React.useState("");
  const [firstName, setFirstName] = React.useState("");
  const [lastName, setLastName] = React.useState("");
  const [password, setPassword] = React.useState("");
  
  const [searchQueryInput, setSearchQueryInput] = React.useState("");
  // Simple custom hook alternative or just standard useEffect for debounce
  const [searchQuery, setSearchQuery] = React.useState("");

  React.useEffect(() => {
    const handler = setTimeout(() => {
      setSearchQuery(searchQueryInput);
      setCursor("");
      setCursorHistory([]);
    }, 300);
    return () => clearTimeout(handler);
  }, [searchQueryInput]);

  const [cursor, setCursor] = React.useState("");
  const [cursorHistory, setCursorHistory] = React.useState<string[]>([]);
  
  const [selectedIdentity, setSelectedIdentity] = React.useState<IdentityItem | null>(null);

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

  const createIdentity = useApiMutation(
    (vars: { email: string; firstName: string; lastName: string; password: string }) =>
      fetchAndParse("/api/identities", registrationResponseSchema, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(vars),
      }),
    {
      successMessage: (data) => t("identities.successMsg", { email: (data.identity.traits?.email as string) ?? email }),
      onSuccess: () => {
        queryClient.invalidateQueries({ queryKey: ["identities"] });
        setEmail("");
        setFirstName("");
        setLastName("");
        setPassword("");
      },
    },
  );

  const handleCreate = (e: React.FormEvent) => {
    e.preventDefault();
    if (!email || !password) return;
    createIdentity.mutate({ email, firstName, lastName, password });
  };

  const schemaJson = `{
  "$id": "https://schemas.autorix.io/default.identity.schema.json",
  "title": "Enterprise User Identity",
  "properties": {
    "traits": {
      "properties": {
        "email": { "type": "string", "format": "email" },
        "name": {
          "properties": {
            "first": { "type": "string" },
            "last": { "type": "string" }
          }
        },
        "department": { "type": "string" }
      },
      "required": ["email"]
    }
  }
}`;

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
        </div>
      </div>

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        {/* Register Identity Card */}
        <Card className="bg-card/80">
          <CardHeader className="p-6 pb-4">
            <div className="flex items-center gap-2">
              <Plus className="h-4 w-4 text-blue-400" />
              <CardTitle className="text-sm font-semibold">{t("identities.registerTitle")}</CardTitle>
            </div>
            <CardDescription className="text-xs">{t("identities.registerDesc")}</CardDescription>
          </CardHeader>

          <CardContent className="p-6 pt-0">
            <form onSubmit={handleCreate} className="space-y-4">
              <div className="space-y-1.5">
                <Label htmlFor="email">{t("identities.emailLabel")}</Label>
                <Input
                  id="email"
                  type="email"
                  placeholder={t("identities.emailPlaceholder")}
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  required
                />
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div className="space-y-1.5">
                  <Label htmlFor="firstName">{t("identities.firstNameLabel")}</Label>
                  <Input
                    id="firstName"
                    type="text"
                    placeholder={t("identities.firstNamePlaceholder")}
                    value={firstName}
                    onChange={(e) => setFirstName(e.target.value)}
                  />
                </div>

                <div className="space-y-1.5">
                  <Label htmlFor="lastName">{t("identities.lastNameLabel")}</Label>
                  <Input
                    id="lastName"
                    type="text"
                    placeholder={t("identities.lastNamePlaceholder")}
                    value={lastName}
                    onChange={(e) => setLastName(e.target.value)}
                  />
                </div>
              </div>

              <div className="space-y-1.5">
                <Label htmlFor="password">{t("identities.passwordLabel")}</Label>
                <Input
                  id="password"
                  type="password"
                  placeholder={t("identities.passwordPlaceholder")}
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  required
                  minLength={8}
                />
              </div>

              <Button
                type="submit"
                variant="default"
                disabled={createIdentity.isPending}
                className="w-full gap-2 mt-2 bg-blue-600 hover:bg-blue-700 text-white"
              >
                {createIdentity.isPending ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <Fingerprint className="h-4 w-4" />
                )}
                <span>{createIdentity.isPending ? t("common.loading") : t("identities.submitBtn")}</span>
              </Button>
            </form>
          </CardContent>
        </Card>

        {/* Dynamic Schema Preview Card */}
        <Card className="bg-card/80 flex flex-col justify-between">
          <CardHeader className="p-6 pb-4">
            <div className="flex items-center gap-2">
              <ShieldCheck className="h-4 w-4 text-purple-400" />
              <CardTitle className="text-sm font-semibold">{t("identities.schemaTitle")}</CardTitle>
            </div>
            <CardDescription className="text-xs">{t("identities.schemaDesc")}</CardDescription>
          </CardHeader>

          <CardContent className="p-6 pt-0 flex-1">
            <CodeBlock
              code={schemaJson}
              language="json"
              title="default.identity.schema.json"
              className="h-full max-h-72"
            />
          </CardContent>
        </Card>
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
