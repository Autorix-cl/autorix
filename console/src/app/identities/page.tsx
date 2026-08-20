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
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from "@/components/ui/table";
import { CodeBlock } from "@/components/ui/code-block";
import { useApiQuery } from "@/lib/query/use-api-query";
import { useApiMutation } from "@/lib/query/use-api-mutation";
import { fetchAndParse } from "@/lib/api/schema";
import { identityListSchema, registrationResponseSchema, type Identity } from "@/lib/api/schemas/identity";
import { LoadingState } from "@/components/state/loading-state";
import { EmptyState } from "@/components/state/empty-state";
import { ErrorState } from "@/components/state/error-state";
import { NotConnectedState } from "@/components/state/not-connected-state";
import { NotConnectedEngine } from "@/components/resources/not-connected-engine";
import { useCapabilities } from "@/lib/capabilities/capability-context";

interface IdentityItem {
  id: string;
  email: string;
  name: string;
  state: string;
  createdAt: string;
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
    createdAt: item.created_at ? new Date(item.created_at).toLocaleString() : "Recently",
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
  const [searchQuery, setSearchQuery] = React.useState("");

  const {
    data: identitiesRaw,
    isLoading,
    isFetching,
    isError,
    error,
    refetch,
  } = useApiQuery(["identities"], () => fetchAndParse<Identity[]>("/api/identities", identityListSchema));

  const identities: IdentityItem[] = React.useMemo(() => (identitiesRaw ?? []).map(toIdentityItem), [identitiesRaw]);

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

  const filteredIdentities = identities.filter(
    (item) =>
      item.email.toLowerCase().includes(searchQuery.toLowerCase()) ||
      item.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      item.id.toLowerCase().includes(searchQuery.toLowerCase()),
  );

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

  return (
    <div className="space-y-6">
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
                <Badge variant="secondary" className="font-mono text-[10px]">
                  {identities.length}
                </Badge>
              </CardTitle>
              <CardDescription className="text-xs">{t("identities.tableDesc")}</CardDescription>
            </div>

            {/* Filter Search */}
            <div className="flex items-center gap-2 w-full sm:w-72">
              <div className="relative w-full">
                <Search className="absolute left-2.5 top-2.5 h-3.5 w-3.5 text-muted-foreground" />
                <Input
                  placeholder={t("identities.searchPlaceholder")}
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  className="pl-8 h-8 text-xs bg-muted/30"
                />
              </div>
            </div>
          </div>
        </CardHeader>

        <CardContent className="p-6 pt-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t("identities.colUuid")}</TableHead>
                <TableHead>{t("identities.colEmail")}</TableHead>
                <TableHead>{t("identities.colName")}</TableHead>
                <TableHead>{t("identities.colState")}</TableHead>
                <TableHead>{t("identities.colCreated")}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {isLoading ? (
                <TableRow>
                  <TableCell colSpan={5} className="p-0">
                    <LoadingState label="Connecting to Ego PostgreSQL storage..." />
                  </TableCell>
                </TableRow>
              ) : isError ? (
                <TableRow>
                  <TableCell colSpan={5} className="p-0">
                    {error?.kind === "engine-unreachable" ? (
                      <NotConnectedState engineName="Ego" onRetry={refetch} />
                    ) : (
                      <ErrorState error={error} onRetry={refetch} />
                    )}
                  </TableCell>
                </TableRow>
              ) : filteredIdentities.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={5} className="p-0">
                    <EmptyState
                      title={t("identities.tableTitle")}
                      description="No registered identities found in PostgreSQL database."
                    />
                  </TableCell>
                </TableRow>
              ) : (
                filteredIdentities.map((item) => (
                  <TableRow key={item.id}>
                    <TableCell className="font-mono text-xs text-blue-400 font-medium">{item.id}</TableCell>
                    <TableCell className="font-semibold text-foreground">{item.email}</TableCell>
                    <TableCell className="text-muted-foreground">{item.name}</TableCell>
                    <TableCell>
                      <Badge variant={item.state === "active" ? "success" : "secondary"} className="text-[10px]">
                        {item.state === "active" ? t("common.active") : t("common.inactive")}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-muted-foreground text-xs font-mono">{item.createdAt}</TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </div>
  );
}
