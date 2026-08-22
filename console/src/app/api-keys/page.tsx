"use client";

import * as React from "react";
import { Layers, RefreshCw, Lock, Key } from "lucide-react";
import { useTranslation } from "@/lib/i18n";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from "@/components/ui/card";
import { CodeBlock } from "@/components/ui/code-block";
import { useApiQuery } from "@/lib/query/use-api-query";
import { useApiMutation } from "@/lib/query/use-api-mutation";
import { fetchAndParse } from "@/lib/api/schema";
import {
  paginatedApiKeyListSchema,
  attenuateResponseSchema,
  macaroonSchema,
} from "@/lib/api/schemas/vulcan";
import type { z } from "zod";

import { DataTable } from "@/components/ui/data-table";
import { columns, type KeyItem } from "./columns";
import { KeyBuilderSheet } from "./key-builder-sheet";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Loader2 } from "lucide-react";

type Macaroon = z.infer<typeof macaroonSchema>;

export default function ApiKeysPage() {
  const { t } = useTranslation();

  const [limit] = React.useState(10);
  const [cursor, setCursor] = React.useState<string | undefined>(undefined);
  const [history, setHistory] = React.useState<string[]>([]);

  const [newlyCreatedKey, setNewlyCreatedKey] = React.useState<string | null>(null);

  // Attenuation studio state
  const [selectedKeyForAttenuation, setSelectedKeyForAttenuation] = React.useState<string>("");
  const [newCaveat, setNewCaveat] = React.useState("ip = 192.168.1.50");
  const [attenuatedToken, setAttenuatedToken] = React.useState<string | null>(null);
  const [macaroonsByKeyId, setMacaroonsByKeyId] = React.useState<Record<string, Macaroon>>({});

  const {
    data: paginatedData,
    isLoading,
    isFetching,
    refetch,
  } = useApiQuery(
    ["api-keys", { limit, cursor }],
    () => {
      const url = new URL("/api/keys", window.location.origin);
      url.searchParams.set("limit", limit.toString());
      if (cursor) {
        url.searchParams.set("cursor", cursor);
      }
      return fetchAndParse(url.pathname + url.search, paginatedApiKeyListSchema);
    },
    {
      placeholderData: (prev) => prev,
    }
  );

  const hasMore = paginatedData?.has_more || false;
  const nextCursor = paginatedData?.next_cursor;

  const keys: KeyItem[] = React.useMemo(() => {
    const data = paginatedData?.data || [];
    return data.map((k) => ({
      id: k.id,
      name: k.name,
      prefix: k.key_prefix || "av_live",
      keyMasked: `${k.key_prefix}_${k.key_hint || "xxxx"}...`,
      ownerId: k.owner_id || "system",
      createdAt: k.created_at ? new Date(k.created_at).toLocaleString() : "Recently",
    }));
  }, [paginatedData?.data]);

  React.useEffect(() => {
    if (keys.length > 0 && !selectedKeyForAttenuation) {
      setSelectedKeyForAttenuation(keys[0].id);
    }
  }, [keys, selectedKeyForAttenuation]);

  const handleNextPage = () => {
    if (hasMore && nextCursor) {
      setHistory((prev) => [...prev, cursor || ""]);
      setCursor(nextCursor);
    }
  };

  const handlePreviousPage = () => {
    if (history.length > 0) {
      const newHistory = [...history];
      const prevCursor = newHistory.pop();
      setHistory(newHistory);
      setCursor(prevCursor || undefined);
    }
  };

  const attenuate = useApiMutation(
    (vars: { macaroon: Macaroon; caveat: string }) =>
      fetchAndParse("/api/keys/attenuate", attenuateResponseSchema, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(vars),
      }),
    {
      successMessage: "Macaroon attenuated",
      onSuccess: (data) => {
        setAttenuatedToken(
          `macaroon_v1:${data.key_id}:[${data.caveats.map((c) => c.predicate).join(", ")}]:hmac_${data.signature}`
        );
      },
    }
  );

  const targetKeyForAttenuation = keys.find((k) => k.id === selectedKeyForAttenuation) || keys[0];
  const baseMacaroonForAttenuation = targetKeyForAttenuation ? macaroonsByKeyId[targetKeyForAttenuation.id] : undefined;

  const handleAttenuate = () => {
    if (!newCaveat || !baseMacaroonForAttenuation) return;
    attenuate.mutate({ macaroon: baseMacaroonForAttenuation, caveat: newCaveat });
  };

  const handleKeyCreated = (data: { api_key: { id: string }; raw_token?: string; macaroon: Macaroon; isLive: boolean }) => {
    setNewlyCreatedKey(data.raw_token || `av_${data.isLive ? "live" : "test"}_${data.api_key.id}`);
    setSelectedKeyForAttenuation(data.api_key.id);
    setMacaroonsByKeyId((prev) => ({ ...prev, [data.api_key.id]: data.macaroon }));
  };

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
        <div>
          <h1 className="text-xl font-bold tracking-tight text-foreground sm:text-2xl">{t("apiKeys.title")}</h1>
          <p className="text-xs text-muted-foreground mt-1">{t("apiKeys.subtitle")}</p>
        </div>

        <div className="flex items-center gap-2">
          <Badge variant="cyan" className="gap-1.5 py-1 px-3">
            <Layers className="h-3.5 w-3.5" />
            <span>{t("apiKeys.statusBadge")}</span>
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
          <KeyBuilderSheet onKeyCreated={handleKeyCreated} />
        </div>
      </div>

      {/* Newly Created Key Alert Banner */}
      {newlyCreatedKey && (
        <div className="rounded-xl border border-emerald-500/40 bg-emerald-500/10 p-5 space-y-3 animate-in fade-in-50">
          <div className="flex items-center gap-2 text-emerald-400 font-semibold text-xs">
            <Lock className="h-4 w-4" />
            <span>{t("apiKeys.keyGeneratedBanner")}</span>
          </div>

          <CodeBlock
            code={newlyCreatedKey}
            language="text"
            title="PLAINTEXT VULCAN API KEY"
            className="bg-black/90 text-emerald-300 font-mono text-sm"
          />

          <p className="text-[11px] text-muted-foreground">
            Key successfully persisted in Vulcan PostgreSQL with SHA-256 hash. Copy it now as it will not be displayed
            again.
          </p>
        </div>
      )}

      {/* Macaroon Attenuation Studio Card - Placed above the table */}
      <Card className="bg-card/80">
        <CardHeader className="p-6 pb-4">
          <div className="flex items-center gap-2">
            <RefreshCw className="h-4 w-4 text-purple-400" />
            <CardTitle className="text-sm font-semibold">{t("apiKeys.attenuationTitle")}</CardTitle>
          </div>
          <CardDescription className="text-xs">{t("apiKeys.attenuationDesc")}</CardDescription>
        </CardHeader>

        <CardContent className="p-6 pt-0 space-y-4">
          <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
            <div className="space-y-1.5">
              <Label>{t("apiKeys.baseKeyLabel")}</Label>
              <Select value={selectedKeyForAttenuation} onValueChange={setSelectedKeyForAttenuation}>
                <SelectTrigger>
                  <SelectValue placeholder="Select base key" />
                </SelectTrigger>
                <SelectContent>
                  {keys.length === 0 ? (
                    <SelectItem value="none">No keys available</SelectItem>
                  ) : (
                    keys.map((k) => (
                      <SelectItem key={k.id} value={k.id}>
                        {k.name} ({k.keyMasked})
                      </SelectItem>
                    ))
                  )}
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-1.5">
              <Label>{t("apiKeys.caveatLabel")}</Label>
              <Select value={newCaveat} onValueChange={setNewCaveat}>
                <SelectTrigger>
                  <SelectValue placeholder="Select caveat" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="time_before = 2026-08-17T00:00:00Z">
                    Time Expiry (time_before = 2026-08-17T00:00:00Z)
                  </SelectItem>
                  <SelectItem value="ip = 192.168.1.50">IP Restriction (ip = 192.168.1.50)</SelectItem>
                  <SelectItem value="method = GET">Read-Only Method (method = GET)</SelectItem>
                  <SelectItem value="path_prefix = /api/v1/public">
                    Scope Limiter (path_prefix = /api/v1/public)
                  </SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>

          {!baseMacaroonForAttenuation && (
            <p className="text-[11px] text-amber-400">{t("apiKeys.macaroonUnavailable")}</p>
          )}

          <Button
            type="button"
            variant="outline"
            onClick={handleAttenuate}
            disabled={attenuate.isPending || !baseMacaroonForAttenuation}
            className="w-full gap-2 border-purple-500/30 hover:border-purple-500/60 hover:bg-purple-500/10 text-xs"
          >
            {attenuate.isPending ? (
              <Loader2 className="h-4 w-4 animate-spin text-purple-400" />
            ) : (
              <RefreshCw className="h-4 w-4 text-purple-400" />
            )}
            <span>{t("apiKeys.computeBtn")}</span>
          </Button>

          {attenuatedToken && (
            <CodeBlock
              code={attenuatedToken}
              language="text"
              title="ATTENUATED MACAROON CAPABILITY TOKEN (HMAC)"
              className="text-emerald-300"
            />
          )}
        </CardContent>
      </Card>

      {/* Issued Keys Table using generic DataTable */}
      <Card className="bg-card/80">
        <CardHeader className="p-6 pb-4">
          <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <CardTitle className="text-sm font-semibold flex items-center gap-2">
                <Key className="h-4 w-4 text-cyan-400" />
                <span>{t("apiKeys.tableTitle")}</span>
              </CardTitle>
              <CardDescription className="text-xs">{t("apiKeys.tableDesc")}</CardDescription>
            </div>
          </div>
        </CardHeader>

        <CardContent className="p-6 pt-0">
          <DataTable
            columns={columns}
            data={keys}
            manualPagination
            canNextPage={hasMore}
            canPreviousPage={history.length > 0}
            onNextPage={handleNextPage}
            onPreviousPage={handlePreviousPage}
            isLoading={isLoading}
          />
        </CardContent>
      </Card>
    </div>
  );
}
