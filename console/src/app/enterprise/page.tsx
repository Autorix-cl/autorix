"use client";

import * as React from "react";
import { Building2, RefreshCw, FileCode } from "lucide-react";
import { useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "@/lib/i18n";
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { CodeBlock } from "@/components/ui/code-block";
import { useApiQuery } from "@/lib/query/use-api-query";
import { fetchAndParse } from "@/lib/api/schema";
import type { ApiResult } from "@/lib/api/client";
import {
  samlProviderListSchema,
  scimListResponseSchema,
  type SAMLProvider,
  type SCIMUser,
} from "@/lib/api/schemas/hermes";
import { LoadingState } from "@/components/state/loading-state";
import { ErrorState } from "@/components/state/error-state";
import { NotConnectedState } from "@/components/state/not-connected-state";
import { NotConnectedEngine } from "@/components/resources/not-connected-engine";
import { useCapabilities } from "@/lib/capabilities/capability-context";
import { SAMLProvidersTable } from "./saml-providers-table";
import { SCIMSyncMonitor } from "./scim-sync-monitor";

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

  const providersQuery = useApiQuery(["enterprise-saml-providers"], () =>
    fetchAndParse("/api/enterprise/saml", samlProviderListSchema)
  );

  const {
    data: scimData,
    isFetching: isScimFetching,
    refetch: refetchScim,
  } = useApiQuery(["enterprise-scim"], () => fetchAndParse("/api/enterprise/scim", scimListResponseSchema));

  const {
    data: metadataXml,
    isLoading: isMetaLoading,
    isFetching: isMetaFetching,
    isError: isMetaError,
    error: metaError,
    refetch: refetchMetadata,
  } = useApiQuery(["enterprise-metadata"], fetchMetadataXml);

  const providers: SAMLProvider[] = React.useMemo(() => {
    const res = providersQuery.data;
    if (!res) return [];
    return Array.isArray(res) ? res : [];
  }, [providersQuery.data]);

  const scimUsers: SCIMUser[] = React.useMemo(() => {
    return scimData?.Resources || [];
  }, [scimData]);

  const handleRefresh = () => {
    providersQuery.refetch();
    refetchScim();
    refetchMetadata();
  };

  const loading = isScimFetching || isMetaFetching || providersQuery.isFetching;

  if (!isEngineConnected("hermes")) {
    return (
      <div className="space-y-6 p-8">
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
    <div className="space-y-6 p-8">
      {/* Page Header */}
      <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
        <div>
          <div className="flex items-center gap-2.5 flex-wrap">
            <h1 className="text-xl font-bold tracking-tight text-foreground sm:text-2xl">{t("enterprise.title")}</h1>
            <Badge variant="rose" className="gap-1.5 py-0.5 px-2.5 text-[11px] font-mono whitespace-nowrap shrink-0">
              <Building2 className="h-3.5 w-3.5" />
              <span>{t("enterprise.statusBadge")}</span>
            </Badge>
          </div>
          <p className="text-xs text-muted-foreground mt-1">{t("enterprise.subtitle")}</p>
        </div>

        <div className="flex flex-wrap items-center gap-2">
          <Button variant="outline" size="sm" onClick={handleRefresh} disabled={loading} className="h-8 gap-1 text-xs">
            <RefreshCw className={`h-3.5 w-3.5 ${loading ? "animate-spin" : ""}`} />
            <span>{t("common.refresh")}</span>
          </Button>
        </div>
      </div>

      {/* SAML 2.0 Identity Providers Table with Diagnostics & Wizard */}
      <SAMLProvidersTable
        providers={providers}
        isLoading={providersQuery.isLoading}
        onRefresh={() => {
          providersQuery.refetch();
          queryClient.invalidateQueries({ queryKey: ["enterprise-saml-providers"] });
        }}
      />

      {/* Grid: SP Metadata Descriptor & Live SCIM Endpoint */}
      <Card className="bg-card/80">
        <CardHeader className="p-5 pb-3">
          <div className="flex items-center gap-2">
            <FileCode className="h-4 w-4 text-blue-400" />
            <CardTitle className="text-sm font-semibold">{t("enterprise.spMetadataTitle")}</CardTitle>
          </div>
          <CardDescription className="text-xs">{t("enterprise.spMetadataDesc")}</CardDescription>
        </CardHeader>

        <CardContent className="p-5 pt-0 space-y-3">
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
              className="max-h-44"
            />
          )}

          <div className="rounded-lg border border-border/70 bg-muted/30 p-2.5 flex items-center justify-between text-xs">
            <span className="text-[10px] font-bold uppercase text-blue-400">{t("enterprise.scimBaseLabel")}</span>
            <span className="font-mono text-xs text-foreground font-semibold">http://localhost:4477/scim/v2</span>
          </div>
        </CardContent>
      </Card>

      {/* SCIM 2.0 Directory Management & Sync Monitoring */}
      <SCIMSyncMonitor users={scimUsers} onRefresh={refetchScim} />
    </div>
  );
}
