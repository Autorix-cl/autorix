"use client";

import * as React from "react";
import {
  Building2,
  RefreshCw,
  FileCode,
  Users,
  Shield,
  Activity,
  Download,
  Copy,
  Check,
  Terminal,
} from "lucide-react";
import { ServiceHeader } from "@/components/layout/service-header";
import { CloudSection } from "@/components/layout/cloud-section";
import { useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "@/lib/i18n";
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from "@/components/ui/card";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs";
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
import { toast } from "sonner";
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

function CopyButton({ text, label }: { text: string; label: string }) {
  const [copied, setCopied] = React.useState(false);

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(text);
      setCopied(true);
      toast.success(`${label} copied to clipboard`);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      toast.error("Failed to copy");
    }
  };

  return (
    <Button
      variant="ghost"
      size="icon-sm"
      onClick={handleCopy}
      className="h-6 w-6 text-muted-foreground hover:text-foreground"
      title={`Copy ${label}`}
    >
      {copied ? <Check className="h-3.5 w-3.5 text-emerald-400" /> : <Copy className="h-3.5 w-3.5" />}
    </Button>
  );
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

  const handleDownloadXml = () => {
    if (!metadataXml) return;
    const blob = new Blob([metadataXml], { type: "application/xml;charset=utf-8" });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = "autorix-saml-sp-metadata.xml";
    link.click();
    URL.revokeObjectURL(url);
    toast.success("Downloaded autorix-saml-sp-metadata.xml");
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
    <div className="space-y-6">
      {/* Cloud Service Header & Telemetry HUD */}
      <ServiceHeader
        serviceName="Hermes Federation Engine"
        title={t("enterprise.title")}
        description={t("enterprise.subtitle")}
        icon={Building2}
        iconColor="text-rose-400"
        statusText={t("enterprise.statusBadge")}
        statusVariant="rose"
        metrics={[
          {
            label: "SAML Providers",
            value: providers.length,
            hint: "Active IdP Connections",
            icon: Building2,
          },
          {
            label: "SCIM Directory",
            value: scimUsers.length,
            hint: "Synced Corporate Users",
            icon: Users,
          },
          {
            label: "Metadata Protocol",
            value: "SAML 2.0",
            hint: "XML SP & IdP SLO",
            icon: Shield,
          },
          {
            label: "Engine Gateway",
            value: "Port 4438",
            hint: "Enterprise SSO Gateway",
            icon: Activity,
          },
        ]}
        actions={
          <Button
            variant="outline"
            size="sm"
            onClick={handleRefresh}
            disabled={loading}
            className="h-8 gap-1.5 text-xs"
          >
            <RefreshCw className={`h-3.5 w-3.5 ${loading ? "animate-spin" : ""}`} />
            <span>{t("common.refresh")}</span>
          </Button>
        }
      />

      {/* Studio Navigation Tabs */}
      <Tabs defaultValue="providers" className="space-y-4">
        <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3 border-b border-border pb-3">
          <TabsList className="bg-muted/40 p-1">
            <TabsTrigger value="providers" className="gap-2 text-xs font-medium">
              <Building2 className="h-3.5 w-3.5 text-rose-400" />
              <span>SAML Identity Providers</span>
            </TabsTrigger>
            <TabsTrigger value="scim" className="gap-2 text-xs font-medium">
              <Users className="h-3.5 w-3.5 text-emerald-400" />
              <span>SCIM Directory Sync</span>
            </TabsTrigger>
            <TabsTrigger value="metadata" className="gap-2 text-xs font-medium">
              <FileCode className="h-3.5 w-3.5 text-blue-400" />
              <span>SP Metadata & Endpoints</span>
            </TabsTrigger>
          </TabsList>
        </div>

        {/* Tab 1: SAML 2.0 Identity Providers Table with Diagnostics & Wizard */}
        <TabsContent value="providers" className="space-y-4 focus-visible:outline-none">
          <CloudSection
            title="SAML 2.0 Identity Providers"
            description="Active IdP Connections & Assertion Mappings"
            icon={Building2}
            badge="Federation Gateways"
            badgeVariant="rose"
          >
            <SAMLProvidersTable
              providers={providers}
              isLoading={providersQuery.isLoading}
              onRefresh={() => {
                providersQuery.refetch();
                queryClient.invalidateQueries({ queryKey: ["enterprise-saml-providers"] });
              }}
            />
          </CloudSection>
        </TabsContent>

        {/* Tab 2: SCIM 2.0 Directory Management & Sync Monitoring */}
        <TabsContent value="scim" className="space-y-4 focus-visible:outline-none">
          <CloudSection
            title="SCIM 2.0 Directory Management & Sync Monitor"
            description="Automated provisioning of corporate users, role groups, and real-time synchronization telemetry"
            icon={Users}
            badge="Directory Sync"
            badgeVariant="success"
          >
            <SCIMSyncMonitor users={scimUsers} onRefresh={refetchScim} />
          </CloudSection>
        </TabsContent>

        {/* Tab 3: SP Metadata Descriptor & Federation Endpoints */}
        <TabsContent value="metadata" className="space-y-4 focus-visible:outline-none">
          <CloudSection
            title="Service Provider (SP) Metadata & Federation Endpoints"
            description="SAML 2.0 SP EntityDescriptor XML, Assertion Consumer Service (ACS) endpoints, and SCIM credentials"
            icon={FileCode}
            badge="SAML 2.0 SP"
            badgeVariant="info"
          >
            {/* Endpoints Quick-Reference Grid */}
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
              <div className="p-3.5 rounded-lg border border-border/80 bg-card/60 space-y-2">
                <div className="flex items-center justify-between">
                  <span className="text-[10px] font-bold uppercase tracking-wider text-rose-400">ACS Endpoint (POST)</span>
                  <CopyButton text="http://localhost:4477/saml/acs" label="ACS URL" />
                </div>
                <div className="font-mono text-xs text-foreground font-semibold truncate select-all">
                  http://localhost:4477/saml/acs
                </div>
                <p className="text-[11px] text-muted-foreground">
                  Assertion Consumer Service receiving signed IdP assertions
                </p>
              </div>

              <div className="p-3.5 rounded-lg border border-border/80 bg-card/60 space-y-2">
                <div className="flex items-center justify-between">
                  <span className="text-[10px] font-bold uppercase tracking-wider text-blue-400">SP Entity ID / Issuer</span>
                  <CopyButton text="https://hermes.autorix.io/saml/metadata" label="Entity ID" />
                </div>
                <div className="font-mono text-xs text-foreground font-semibold truncate select-all">
                  https://hermes.autorix.io/saml/metadata
                </div>
                <p className="text-[11px] text-muted-foreground">
                  Global unique URI identifying this Autorix Service Provider
                </p>
              </div>

              <div className="p-3.5 rounded-lg border border-border/80 bg-card/60 space-y-2">
                <div className="flex items-center justify-between">
                  <span className="text-[10px] font-bold uppercase tracking-wider text-emerald-400">SCIM 2.0 Base URL</span>
                  <CopyButton text="http://localhost:4477/scim/v2" label="SCIM Base URL" />
                </div>
                <div className="font-mono text-xs text-foreground font-semibold truncate select-all">
                  http://localhost:4477/scim/v2
                </div>
                <p className="text-[11px] text-muted-foreground">
                  RFC 7644 SCIM endpoint for corporate user & group provisioning
                </p>
              </div>
            </div>

            {/* SAML SP Metadata XML Card */}
            <Card className="border-border">
              <CardHeader className="p-5 pb-3">
                <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
                  <div className="space-y-1">
                    <CardTitle className="text-sm font-semibold flex items-center gap-2">
                      <FileCode className="h-4 w-4 text-blue-400" />
                      <span>{t("enterprise.spMetadataTitle")}</span>
                    </CardTitle>
                    <CardDescription className="text-xs">
                      {t("enterprise.spMetadataDesc")}
                    </CardDescription>
                  </div>

                  <div className="flex items-center gap-2">
                    {metadataXml && (
                      <Badge variant="outline" className="font-mono text-[10px] px-2 py-0.5">
                        {metadataXml.trim().split("\n").length} lines · {(new Blob([metadataXml]).size / 1024).toFixed(1)} KB
                      </Badge>
                    )}
                    <Button
                      variant="outline"
                      size="sm"
                      disabled={!metadataXml}
                      onClick={handleDownloadXml}
                      className="h-7 text-xs gap-1.5"
                    >
                      <Download className="w-3.5 h-3.5" />
                      <span>Download XML</span>
                    </Button>
                  </div>
                </div>
              </CardHeader>

              <CardContent className="p-5 pt-0 space-y-4">
                {isMetaLoading ? (
                  <LoadingState label="Loading SAML SP metadata from Hermes..." />
                ) : isMetaError ? (
                  metaError?.kind === "engine-unreachable" ? (
                    <NotConnectedState engineName="Hermes" onRetry={refetchMetadata} />
                  ) : (
                    <ErrorState error={metaError} onRetry={refetchMetadata} />
                  )
                ) : (
                  <div className="space-y-3">
                    <CodeBlock
                      code={metadataXml || ""}
                      language="xml"
                      title="LIVE SAML SP METADATA DESCRIPTOR (XML)"
                      showLineNumbers={true}
                      maxHeight="max-h-[520px]"
                    />
                    <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-2 text-[11px] text-muted-foreground px-1">
                      <span>Public SP EntityDescriptor generated dynamically by Autorix Hermes federation engine</span>
                      <span className="font-mono">Audience: https://hermes.autorix.io/saml/metadata</span>
                    </div>
                  </div>
                )}
              </CardContent>
            </Card>

            {/* Automation & CLI Card */}
            <Card className="border-border">
              <CardHeader className="p-4 pb-2">
                <CardTitle className="text-xs font-semibold flex items-center gap-2">
                  <Terminal className="h-3.5 w-3.5 text-primary" />
                  <span>Programmatic Federation Retrieval (cURL)</span>
                </CardTitle>
                <CardDescription className="text-[11px]">
                  Retrieve the live SAML SP descriptor in CI/CD or identity configuration scripts
                </CardDescription>
              </CardHeader>
              <CardContent className="p-4 pt-0">
                <CodeBlock
                  code="curl -s http://localhost:4477/saml/metadata | xmllint --format -"
                  language="bash"
                  title="BASH"
                  maxHeight="none"
                />
              </CardContent>
            </Card>
          </CloudSection>
        </TabsContent>
      </Tabs>
    </div>
  );
}
