"use client";

import * as React from "react";
import { Building2, Plus, ShieldCheck, RefreshCw, Search, FileCode, Users, Loader2 } from "lucide-react";
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
import type { ApiResult } from "@/lib/api/client";
import {
  samlProviderListSchema,
  samlProviderSchema,
  scimListResponseSchema,
  type SCIMUser,
} from "@/lib/api/schemas/hermes";
import { LoadingState } from "@/components/state/loading-state";
import { EmptyState } from "@/components/state/empty-state";
import { ErrorState } from "@/components/state/error-state";
import { NotConnectedState } from "@/components/state/not-connected-state";
import { NotConnectedEngine } from "@/components/resources/not-connected-engine";
import { useCapabilities } from "@/lib/capabilities/capability-context";

interface SCIMUserItem {
  id: string;
  externalId: string;
  userName: string;
  email: string;
  active: boolean;
}

function toScimUserItem(u: SCIMUser): SCIMUserItem {
  return {
    id: u.id,
    externalId: u.externalId || "ext_id",
    userName: u.userName || "user",
    email: u.emails?.[0]?.value || "user@enterprise.io",
    active: u.active ?? true,
  };
}

// Hermes's /saml/metadata returns raw XML, not JSON, so this stays on plain
// fetch instead of fetchJSON/fetchAndParse — only the resulting loading /
// error states are folded into useApiQuery for a consistent section UX.
async function fetchMetadataXml(): Promise<ApiResult<string>> {
  try {
    const res = await fetch("/api/enterprise/metadata");
    const xml = await res.text();
    if (!res.ok) {
      return {
        ok: false,
        error: {
          kind: res.status >= 500 ? "engine-error" : "unknown",
          message: xml || `request failed with status ${res.status}`,
          status: res.status,
        },
      };
    }
    return { ok: true, data: xml };
  } catch (err) {
    return {
      ok: false,
      error: {
        kind: "engine-unreachable",
        message: err instanceof Error ? err.message : "network error",
        cause: err,
      },
    };
  }
}

export default function EnterprisePage() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const { isEngineConnected } = useCapabilities();

  const [providerId, setProviderId] = React.useState("");
  const [providerName, setProviderName] = React.useState("");
  const [idpEntityId, setIdpEntityId] = React.useState("");
  const [idpSsoUrl, setIdpSsoUrl] = React.useState("");
  const [searchQuery, setSearchQuery] = React.useState("");

  // Kept fetched (and invalidated on create) even though the provider list
  // itself isn't rendered as a table today, matching prior behavior.
  const providersQuery = useApiQuery(["enterprise-saml-providers"], () =>
    fetchAndParse("/api/enterprise/saml", samlProviderListSchema),
  );

  const {
    data: scimData,
    isLoading,
    isFetching,
    isError,
    error,
    refetch,
  } = useApiQuery(["enterprise-scim"], () => fetchAndParse("/api/enterprise/scim", scimListResponseSchema));

  const {
    data: metadataXml,
    isLoading: isMetaLoading,
    isFetching: isMetaFetching,
    isError: isMetaError,
    error: metaError,
    refetch: refetchMetadata,
  } = useApiQuery(["enterprise-metadata"], fetchMetadataXml);

  const scimUsers: SCIMUserItem[] = React.useMemo(() => (scimData?.Resources ?? []).map(toScimUserItem), [scimData]);

  const createProvider = useApiMutation(
    (vars: { id: string; name: string; idpEntityId: string; ssoUrl: string }) =>
      fetchAndParse("/api/enterprise/saml", samlProviderSchema, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(vars),
      }),
    {
      successMessage: t("enterprise.successMsg", { id: providerId }),
      onSuccess: () => {
        queryClient.invalidateQueries({ queryKey: ["enterprise-saml-providers"] });
        setProviderId("");
        setProviderName("");
        setIdpEntityId("");
        setIdpSsoUrl("");
      },
    },
  );

  const handleCreateProvider = (e: React.FormEvent) => {
    e.preventDefault();
    if (!providerId || !idpSsoUrl) return;
    createProvider.mutate({
      id: providerId,
      name: providerName || providerId,
      idpEntityId,
      ssoUrl: idpSsoUrl,
    });
  };

  const handleRefresh = () => {
    providersQuery.refetch();
    refetch();
    refetchMetadata();
  };

  const isSubmitting = createProvider.isPending;
  const loading = isFetching || isMetaFetching || providersQuery.isFetching;

  const filteredScimUsers = scimUsers.filter(
    (u) =>
      u.userName.toLowerCase().includes(searchQuery.toLowerCase()) ||
      u.email.toLowerCase().includes(searchQuery.toLowerCase()) ||
      u.externalId.toLowerCase().includes(searchQuery.toLowerCase()),
  );

  if (!isEngineConnected("hermes")) {
    return (
      <div className="space-y-6">
        <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
          <div>
            <h1 className="text-xl font-bold tracking-tight text-foreground sm:text-2xl">{t("enterprise.title")}</h1>
            <p className="text-xs text-muted-foreground mt-1">{t("enterprise.subtitle")}</p>
          </div>
        </div>
        <NotConnectedEngine
          engineType="hermes"
          engineName="Autorix Hermes (Enterprise SSO & SCIM)"
          description="SAML 2.0 identity federation and SCIM 2.0 directory synchronization."
        />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
        <div>
          <h1 className="text-xl font-bold tracking-tight text-foreground sm:text-2xl">{t("enterprise.title")}</h1>
          <p className="text-xs text-muted-foreground mt-1">{t("enterprise.subtitle")}</p>
        </div>

        <div className="flex items-center gap-2">
          <Badge variant="rose" className="gap-1.5 py-1 px-3">
            <Building2 className="h-3.5 w-3.5" />
            <span>{t("enterprise.statusBadge")}</span>
          </Badge>
          <Button variant="outline" size="sm" onClick={handleRefresh} disabled={loading} className="h-8 gap-1 text-xs">
            <RefreshCw className={`h-3.5 w-3.5 ${loading ? "animate-spin" : ""}`} />
            <span>{t("common.refresh")}</span>
          </Button>
        </div>
      </div>

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        {/* Register SAML 2.0 Provider Card */}
        <Card className="bg-card/80">
          <CardHeader className="p-6 pb-4">
            <div className="flex items-center gap-2">
              <Plus className="h-4 w-4 text-rose-400" />
              <CardTitle className="text-sm font-semibold">{t("enterprise.registerTitle")}</CardTitle>
            </div>
            <CardDescription className="text-xs">{t("enterprise.registerDesc")}</CardDescription>
          </CardHeader>

          <CardContent className="p-6 pt-0">
            <form onSubmit={handleCreateProvider} className="space-y-4">
              <div className="grid grid-cols-2 gap-3">
                <div className="space-y-1.5">
                  <Label htmlFor="providerId">{t("enterprise.providerIdLabel")}</Label>
                  <Input
                    id="providerId"
                    placeholder={t("enterprise.providerIdPlaceholder")}
                    value={providerId}
                    onChange={(e) => setProviderId(e.target.value)}
                    required
                  />
                </div>

                <div className="space-y-1.5">
                  <Label htmlFor="providerName">{t("enterprise.displayNameLabel")}</Label>
                  <Input
                    id="providerName"
                    placeholder={t("enterprise.displayNamePlaceholder")}
                    value={providerName}
                    onChange={(e) => setProviderName(e.target.value)}
                  />
                </div>
              </div>

              <div className="space-y-1.5">
                <Label htmlFor="idpEntityId">{t("enterprise.entityIdLabel")}</Label>
                <Input
                  id="idpEntityId"
                  placeholder={t("enterprise.entityIdPlaceholder")}
                  value={idpEntityId}
                  onChange={(e) => setIdpEntityId(e.target.value)}
                />
              </div>

              <div className="space-y-1.5">
                <Label htmlFor="idpSsoUrl">{t("enterprise.ssoUrlLabel")}</Label>
                <Input
                  id="idpSsoUrl"
                  type="url"
                  placeholder={t("enterprise.ssoUrlPlaceholder")}
                  value={idpSsoUrl}
                  onChange={(e) => setIdpSsoUrl(e.target.value)}
                  required
                />
              </div>

              <Button type="submit" variant="rose" disabled={isSubmitting} className="w-full gap-2 mt-2">
                {isSubmitting ? <Loader2 className="h-4 w-4 animate-spin" /> : <ShieldCheck className="h-4 w-4" />}
                <span>{isSubmitting ? t("common.loading") : t("enterprise.submitBtn")}</span>
              </Button>
            </form>
          </CardContent>
        </Card>

        {/* SP Metadata & SCIM Endpoints Card */}
        <Card className="bg-card/80 flex flex-col justify-between">
          <CardHeader className="p-6 pb-4">
            <div className="flex items-center gap-2">
              <FileCode className="h-4 w-4 text-blue-400" />
              <CardTitle className="text-sm font-semibold">{t("enterprise.spMetadataTitle")}</CardTitle>
            </div>
            <CardDescription className="text-xs">{t("enterprise.spMetadataDesc")}</CardDescription>
          </CardHeader>

          <CardContent className="p-6 pt-0 space-y-3 flex-1">
            {isMetaLoading ? (
              <LoadingState label="Loading SAML SP metadata from Hermes..." />
            ) : isMetaError ? (
              metaError?.kind === "engine-unreachable" ? (
                <NotConnectedState engineName="Hermes" onRetry={refetchMetadata} />
              ) : (
                <ErrorState error={metaError} onRetry={refetchMetadata} />
              )
            ) : (
              <CodeBlock
                code={metadataXml || ""}
                language="xml"
                title="LIVE SAML SP METADATA DESCRIPTOR"
                className="max-h-52"
              />
            )}

            <div className="rounded-lg border border-border/70 bg-muted/30 p-3 space-y-1">
              <div className="text-[10px] font-bold uppercase text-blue-400">{t("enterprise.scimBaseLabel")}</div>
              <div className="font-mono text-xs text-foreground font-semibold">http://localhost:4477/scim/v2</div>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* SCIM Synced Directory Table */}
      <Card className="bg-card/80">
        <CardHeader className="p-6 pb-4">
          <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <CardTitle className="text-sm font-semibold flex items-center gap-2">
                <Users className="h-4 w-4 text-rose-400" />
                <span>{t("enterprise.tableTitle")}</span>
                <Badge variant="secondary" className="font-mono text-[10px]">
                  {scimUsers.length}
                </Badge>
              </CardTitle>
              <CardDescription className="text-xs">{t("enterprise.tableDesc")}</CardDescription>
            </div>

            {/* Filter Search */}
            <div className="flex items-center gap-2 w-full sm:w-72">
              <div className="relative w-full">
                <Search className="absolute left-2.5 top-2.5 h-3.5 w-3.5 text-muted-foreground" />
                <Input
                  placeholder={t("enterprise.searchPlaceholder")}
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
                <TableHead>{t("enterprise.colUuid")}</TableHead>
                <TableHead>{t("enterprise.colExtId")}</TableHead>
                <TableHead>{t("enterprise.colUsername")}</TableHead>
                <TableHead>{t("enterprise.colEmail")}</TableHead>
                <TableHead>{t("enterprise.colStatus")}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {isLoading ? (
                <TableRow>
                  <TableCell colSpan={5} className="p-0">
                    <LoadingState label="Connecting to Hermes SCIM directory..." />
                  </TableCell>
                </TableRow>
              ) : isError ? (
                <TableRow>
                  <TableCell colSpan={5} className="p-0">
                    {error?.kind === "engine-unreachable" ? (
                      <NotConnectedState engineName="Hermes" onRetry={refetch} />
                    ) : (
                      <ErrorState error={error} onRetry={refetch} />
                    )}
                  </TableCell>
                </TableRow>
              ) : filteredScimUsers.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={5} className="p-0">
                    <EmptyState
                      title={t("enterprise.tableTitle")}
                      description="No SCIM directory users found in database."
                    />
                  </TableCell>
                </TableRow>
              ) : (
                filteredScimUsers.map((u) => (
                  <TableRow key={u.id}>
                    <TableCell className="font-mono text-xs text-rose-400 font-medium">{u.id}</TableCell>
                    <TableCell className="font-mono text-xs text-muted-foreground">{u.externalId}</TableCell>
                    <TableCell className="font-semibold text-foreground">{u.userName}</TableCell>
                    <TableCell className="text-muted-foreground">{u.email}</TableCell>
                    <TableCell>
                      <Badge variant={u.active ? "success" : "secondary"} className="text-[10px]">
                        {u.active ? t("common.active") : t("common.inactive")}
                      </Badge>
                    </TableCell>
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
