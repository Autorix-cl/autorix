"use client";

import * as React from "react";
import { KeyRound, Plus, Shield, Key, Search, Loader2, RefreshCw } from "lucide-react";
import { useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "@/lib/i18n";
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from "@/components/ui/table";
import { CodeBlock } from "@/components/ui/code-block";
import { useApiQuery } from "@/lib/query/use-api-query";
import { useApiMutation } from "@/lib/query/use-api-mutation";
import { fetchAndParse } from "@/lib/api/schema";
import { oauth2ClientListSchema, oauth2ClientSchema, jwksSchema, type OAuth2Client } from "@/lib/api/schemas/oauth2";
import { LoadingState } from "@/components/state/loading-state";
import { EmptyState } from "@/components/state/empty-state";
import { ErrorState } from "@/components/state/error-state";
import { NotConnectedState } from "@/components/state/not-connected-state";
import { NotConnectedEngine } from "@/components/resources/not-connected-engine";
import { useCapabilities } from "@/lib/capabilities/capability-context";

interface ClientApp {
  id: string;
  name: string;
  grantTypes: string[];
  scopes: string[];
  isPublic: boolean;
  createdAt: string;
}

function toClientApp(c: OAuth2Client): ClientApp {
  return {
    id: c.client_id,
    name: c.client_name || c.client_id,
    grantTypes: c.grant_types?.length ? c.grant_types : ["client_credentials"],
    scopes: c.scopes?.length ? c.scopes : ["openid"],
    isPublic: Boolean(c.is_public),
    createdAt: c.created_at ? new Date(c.created_at).toLocaleString() : "Recently",
  };
}

export default function OAuth2Page() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const { isEngineConnected } = useCapabilities();

  const [clientId, setClientId] = React.useState("");
  const [clientName, setClientName] = React.useState("");
  const [clientSecret, setClientSecret] = React.useState("");
  const [scopes, setScopes] = React.useState("openid profile email");
  const [clientType, setClientType] = React.useState("confidential");
  const [searchQuery, setSearchQuery] = React.useState("");

  const isPublic = clientType === "public";

  const {
    data: clientsRaw,
    isLoading,
    isFetching,
    isError,
    error,
    refetch,
  } = useApiQuery(["oauth2-clients"], () =>
    fetchAndParse<OAuth2Client[]>("/api/oauth2/clients", oauth2ClientListSchema),
  );

  const {
    data: jwks,
    isLoading: isJwksLoading,
    isFetching: isJwksFetching,
    isError: isJwksError,
    error: jwksError,
    refetch: refetchJwks,
  } = useApiQuery(["oauth2-jwks"], () => fetchAndParse("/api/oauth2/jwks", jwksSchema));

  const clients: ClientApp[] = React.useMemo(() => (clientsRaw ?? []).map(toClientApp), [clientsRaw]);

  const createClient = useApiMutation(
    (vars: { clientId: string; clientName: string; clientSecret: string; isPublic: boolean; scopes: string[] }) =>
      fetchAndParse("/api/oauth2/clients", oauth2ClientSchema, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(vars),
      }),
    {
      successMessage: t("oauth2.successMsg", { id: clientId }),
      onSuccess: () => {
        queryClient.invalidateQueries({ queryKey: ["oauth2-clients"] });
        setClientId("");
        setClientName("");
        setClientSecret("");
      },
    },
  );

  const handleCreate = (e: React.FormEvent) => {
    e.preventDefault();
    if (!clientId || !clientName) return;
    createClient.mutate({
      clientId,
      clientName,
      clientSecret,
      isPublic,
      scopes: scopes.trim().split(" ").filter(Boolean),
    });
  };

  const handleRefresh = () => {
    refetch();
    refetchJwks();
  };

  const isSubmitting = createClient.isPending;
  const loading = isFetching || isJwksFetching;

  const filteredClients = clients.filter(
    (c) =>
      c.id.toLowerCase().includes(searchQuery.toLowerCase()) ||
      c.name.toLowerCase().includes(searchQuery.toLowerCase()),
  );

  if (!isEngineConnected("janus")) {
    return (
      <div className="space-y-6">
        <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
          <div>
            <h1 className="text-xl font-bold tracking-tight text-foreground sm:text-2xl">{t("oauth2.title")}</h1>
            <p className="text-xs text-muted-foreground mt-1">{t("oauth2.subtitle")}</p>
          </div>
        </div>
        <NotConnectedEngine
          engineType="janus"
          engineName="Autorix Janus (OAuth2 / OIDC)"
          description="RFC 6749 OAuth 2.0 and OpenID Connect authority."
        />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
        <div>
          <h1 className="text-xl font-bold tracking-tight text-foreground sm:text-2xl">{t("oauth2.title")}</h1>
          <p className="text-xs text-muted-foreground mt-1">{t("oauth2.subtitle")}</p>
        </div>

        <div className="flex items-center gap-2">
          <Badge variant="warning" className="gap-1.5 py-1 px-3">
            <Key className="h-3.5 w-3.5" />
            <span>{t("oauth2.statusBadge")}</span>
          </Badge>
          <Button variant="outline" size="sm" onClick={handleRefresh} disabled={loading} className="h-8 gap-1 text-xs">
            <RefreshCw className={`h-3.5 w-3.5 ${loading ? "animate-spin" : ""}`} />
            <span>{t("common.refresh")}</span>
          </Button>
        </div>
      </div>

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        {/* Register Client Card */}
        <Card className="bg-card/80">
          <CardHeader className="p-6 pb-4">
            <div className="flex items-center gap-2">
              <Plus className="h-4 w-4 text-amber-400" />
              <CardTitle className="text-sm font-semibold">{t("oauth2.registerTitle")}</CardTitle>
            </div>
            <CardDescription className="text-xs">{t("oauth2.registerDesc")}</CardDescription>
          </CardHeader>

          <CardContent className="p-6 pt-0">
            <form onSubmit={handleCreate} className="space-y-4">
              <div className="grid grid-cols-2 gap-3">
                <div className="space-y-1.5">
                  <Label htmlFor="clientId">{t("oauth2.clientIdLabel")}</Label>
                  <Input
                    id="clientId"
                    placeholder={t("oauth2.clientIdPlaceholder")}
                    value={clientId}
                    onChange={(e) => setClientId(e.target.value)}
                    required
                  />
                </div>

                <div className="space-y-1.5">
                  <Label htmlFor="clientName">{t("oauth2.appNameLabel")}</Label>
                  <Input
                    id="clientName"
                    placeholder={t("oauth2.appNamePlaceholder")}
                    value={clientName}
                    onChange={(e) => setClientName(e.target.value)}
                    required
                  />
                </div>
              </div>

              <div className="space-y-1.5">
                <Label htmlFor="clientType">{t("oauth2.clientTypeLabel")}</Label>
                <Select value={clientType} onValueChange={setClientType}>
                  <SelectTrigger id="clientType">
                    <SelectValue placeholder="Select type" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="confidential">{t("oauth2.confidentialOption")}</SelectItem>
                    <SelectItem value="public">{t("oauth2.publicOption")}</SelectItem>
                  </SelectContent>
                </Select>
              </div>

              {!isPublic && (
                <div className="space-y-1.5">
                  <Label htmlFor="clientSecret">{t("oauth2.clientSecretLabel")}</Label>
                  <Input
                    id="clientSecret"
                    type="password"
                    placeholder={t("oauth2.clientSecretPlaceholder")}
                    value={clientSecret}
                    onChange={(e) => setClientSecret(e.target.value)}
                  />
                </div>
              )}

              <div className="space-y-1.5">
                <Label htmlFor="scopes">{t("oauth2.scopesLabel")}</Label>
                <Input id="scopes" value={scopes} onChange={(e) => setScopes(e.target.value)} />
              </div>

              <Button type="submit" variant="amber" disabled={isSubmitting} className="w-full gap-2 mt-2">
                {isSubmitting ? <Loader2 className="h-4 w-4 animate-spin" /> : <KeyRound className="h-4 w-4" />}
                <span>{isSubmitting ? t("common.loading") : t("oauth2.submitBtn")}</span>
              </Button>
            </form>
          </CardContent>
        </Card>

        {/* Active JWKS Keys Viewer Card */}
        <Card className="bg-card/80 flex flex-col justify-between">
          <CardHeader className="p-6 pb-4">
            <div className="flex items-center gap-2">
              <Shield className="h-4 w-4 text-emerald-400" />
              <CardTitle className="text-sm font-semibold">{t("oauth2.jwksTitle")}</CardTitle>
            </div>
            <CardDescription className="text-xs">{t("oauth2.jwksDesc")}</CardDescription>
          </CardHeader>

          <CardContent className="p-6 pt-0 flex-1">
            {isJwksLoading ? (
              <LoadingState label="Loading RS256 JWKS from Janus..." />
            ) : isJwksError ? (
              jwksError?.kind === "engine-unreachable" ? (
                <NotConnectedState engineName="Janus" onRetry={refetchJwks} />
              ) : (
                <ErrorState error={jwksError} onRetry={refetchJwks} />
              )
            ) : (
              <CodeBlock
                code={JSON.stringify(jwks, null, 2)}
                language="json"
                title="/.well-known/jwks.json (Live RS256)"
                className="h-full max-h-72"
              />
            )}
          </CardContent>
        </Card>
      </div>

      {/* Registered Clients Table */}
      <Card className="bg-card/80">
        <CardHeader className="p-6 pb-4">
          <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <CardTitle className="text-sm font-semibold flex items-center gap-2">
                <KeyRound className="h-4 w-4 text-amber-400" />
                <span>{t("oauth2.tableTitle")}</span>
                <Badge variant="secondary" className="font-mono text-[10px]">
                  {clients.length}
                </Badge>
              </CardTitle>
              <CardDescription className="text-xs">{t("oauth2.tableDesc")}</CardDescription>
            </div>

            {/* Filter */}
            <div className="flex items-center gap-2 w-full sm:w-72">
              <div className="relative w-full">
                <Search className="absolute left-2.5 top-2.5 h-3.5 w-3.5 text-muted-foreground" />
                <Input
                  placeholder={t("oauth2.searchPlaceholder")}
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
                <TableHead>{t("oauth2.colClientId")}</TableHead>
                <TableHead>{t("oauth2.colName")}</TableHead>
                <TableHead>{t("oauth2.colGrantTypes")}</TableHead>
                <TableHead>{t("oauth2.colScopes")}</TableHead>
                <TableHead>{t("oauth2.colType")}</TableHead>
                <TableHead>{t("oauth2.colCreated")}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {isLoading ? (
                <TableRow>
                  <TableCell colSpan={6} className="p-0">
                    <LoadingState label="Fetching OAuth2 clients from Janus database..." />
                  </TableCell>
                </TableRow>
              ) : isError ? (
                <TableRow>
                  <TableCell colSpan={6} className="p-0">
                    {error?.kind === "engine-unreachable" ? (
                      <NotConnectedState engineName="Janus" onRetry={refetch} />
                    ) : (
                      <ErrorState error={error} onRetry={refetch} />
                    )}
                  </TableCell>
                </TableRow>
              ) : filteredClients.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={6} className="p-0">
                    <EmptyState
                      title={t("oauth2.tableTitle")}
                      description="No OAuth2 applications registered in Janus."
                    />
                  </TableCell>
                </TableRow>
              ) : (
                filteredClients.map((c) => (
                  <TableRow key={c.id}>
                    <TableCell className="font-mono text-xs text-amber-400 font-semibold">{c.id}</TableCell>
                    <TableCell className="font-medium text-foreground">{c.name}</TableCell>
                    <TableCell>
                      <div className="flex flex-wrap gap-1">
                        {c.grantTypes.map((g) => (
                          <Badge key={g} variant="outline" className="text-[10px] font-mono">
                            {g}
                          </Badge>
                        ))}
                      </div>
                    </TableCell>
                    <TableCell className="font-mono text-[11px] text-muted-foreground">{c.scopes.join(" ")}</TableCell>
                    <TableCell>
                      <Badge variant={c.isPublic ? "info" : "purple"} className="text-[10px]">
                        {c.isPublic ? "Public (PKCE)" : "Confidential"}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-muted-foreground text-xs font-mono">{c.createdAt}</TableCell>
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
