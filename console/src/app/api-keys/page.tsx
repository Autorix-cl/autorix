"use client";

import * as React from "react";
import { Layers, Plus, Lock, RefreshCw, Key, Search, Loader2 } from "lucide-react";
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
import type { z } from "zod";
import { useApiQuery } from "@/lib/query/use-api-query";
import { useApiMutation } from "@/lib/query/use-api-mutation";
import { fetchAndParse } from "@/lib/api/schema";
import {
  apiKeyListSchema,
  createKeyResponseSchema,
  attenuateResponseSchema,
  macaroonSchema,
  type APIKey,
} from "@/lib/api/schemas/vulcan";

type Macaroon = z.infer<typeof macaroonSchema>;
import { LoadingState } from "@/components/state/loading-state";
import { EmptyState } from "@/components/state/empty-state";
import { ErrorState } from "@/components/state/error-state";
import { NotConnectedState } from "@/components/state/not-connected-state";

interface KeyItem {
  id: string;
  name: string;
  prefix: string;
  keyMasked: string;
  ownerId: string;
  createdAt: string;
}

function toKeyItem(k: APIKey): KeyItem {
  return {
    id: k.id,
    name: k.name,
    prefix: k.key_prefix || "av_live",
    keyMasked: `${k.key_prefix}_${k.key_hint || "xxxx"}...`,
    ownerId: k.owner_id || "system",
    createdAt: k.created_at ? new Date(k.created_at).toLocaleString() : "Recently",
  };
}

export default function ApiKeysPage() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();

  const [keyName, setKeyName] = React.useState("");
  const [ownerId, setOwnerId] = React.useState("");
  const [isLive, setIsLive] = React.useState(true);
  const [newlyCreatedKey, setNewlyCreatedKey] = React.useState<string | null>(null);
  const [searchQuery, setSearchQuery] = React.useState("");

  // Attenuation studio state
  const [selectedKeyForAttenuation, setSelectedKeyForAttenuation] = React.useState<string>("");
  const [newCaveat, setNewCaveat] = React.useState("ip = 192.168.1.50");
  const [attenuatedToken, setAttenuatedToken] = React.useState<string | null>(null);
  // Vulcan's GET /keys never returns macaroon signature material (by design,
  // to avoid leaking capability tokens at rest) — only the POST /keys create
  // response includes the real macaroon. So attenuation can only operate on
  // keys created during this browser session; we track those here instead of
  // fabricating a placeholder macaroon for keys loaded from the directory.
  const [macaroonsByKeyId, setMacaroonsByKeyId] = React.useState<Record<string, Macaroon>>({});

  const {
    data: keysRaw,
    isLoading,
    isFetching,
    isError,
    error,
    refetch,
  } = useApiQuery(["api-keys"], () => fetchAndParse<APIKey[]>("/api/keys", apiKeyListSchema));

  const keys: KeyItem[] = React.useMemo(() => (keysRaw ?? []).map(toKeyItem), [keysRaw]);

  React.useEffect(() => {
    if (keys.length > 0 && !selectedKeyForAttenuation) {
      setSelectedKeyForAttenuation(keys[0].id);
    }
  }, [keys, selectedKeyForAttenuation]);

  const createKey = useApiMutation(
    (vars: { name: string; ownerId: string; isLive: boolean }) =>
      fetchAndParse("/api/keys", createKeyResponseSchema, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(vars),
      }),
    {
      successMessage: "API key generated",
      onSuccess: (data, vars) => {
        queryClient.invalidateQueries({ queryKey: ["api-keys"] });
        setNewlyCreatedKey(data.raw_token || `av_${vars.isLive ? "live" : "test"}_${data.api_key.id}`);
        setSelectedKeyForAttenuation(data.api_key.id);
        setMacaroonsByKeyId((prev) => ({ ...prev, [data.api_key.id]: data.macaroon }));
        setKeyName("");
        setOwnerId("");
      },
    },
  );

  const handleCreate = (e: React.FormEvent) => {
    e.preventDefault();
    if (!keyName) return;
    createKey.mutate({ name: keyName, ownerId, isLive });
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
          `macaroon_v1:${data.key_id}:[${data.caveats.map((c) => c.predicate).join(", ")}]:hmac_${data.signature}`,
        );
      },
    },
  );

  const targetKeyForAttenuation = keys.find((k) => k.id === selectedKeyForAttenuation) || keys[0];
  const baseMacaroonForAttenuation = targetKeyForAttenuation ? macaroonsByKeyId[targetKeyForAttenuation.id] : undefined;

  const handleAttenuate = () => {
    if (!newCaveat || !baseMacaroonForAttenuation) return;
    attenuate.mutate({ macaroon: baseMacaroonForAttenuation, caveat: newCaveat });
  };

  const filteredKeys = keys.filter(
    (k) =>
      k.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      k.ownerId.toLowerCase().includes(searchQuery.toLowerCase()) ||
      k.keyMasked.toLowerCase().includes(searchQuery.toLowerCase()),
  );

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

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        {/* Generate API Key Card */}
        <Card className="bg-card/80">
          <CardHeader className="p-6 pb-4">
            <div className="flex items-center gap-2">
              <Plus className="h-4 w-4 text-cyan-400" />
              <CardTitle className="text-sm font-semibold">{t("apiKeys.generateTitle")}</CardTitle>
            </div>
            <CardDescription className="text-xs">{t("apiKeys.generateDesc")}</CardDescription>
          </CardHeader>

          <CardContent className="p-6 pt-0">
            <form onSubmit={handleCreate} className="space-y-4">
              <div className="space-y-1.5">
                <Label htmlFor="keyName">{t("apiKeys.keyNameLabel")}</Label>
                <Input
                  id="keyName"
                  placeholder={t("apiKeys.keyNamePlaceholder")}
                  value={keyName}
                  onChange={(e) => setKeyName(e.target.value)}
                  required
                />
              </div>

              <div className="space-y-1.5">
                <Label htmlFor="ownerId">{t("apiKeys.ownerLabel")}</Label>
                <Input
                  id="ownerId"
                  placeholder={t("apiKeys.ownerPlaceholder")}
                  value={ownerId}
                  onChange={(e) => setOwnerId(e.target.value)}
                />
              </div>

              <div className="space-y-1.5">
                <Label htmlFor="env">{t("apiKeys.envLabel")}</Label>
                <Select value={isLive ? "live" : "test"} onValueChange={(val) => setIsLive(val === "live")}>
                  <SelectTrigger id="env">
                    <SelectValue placeholder="Select Environment" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="live">{t("apiKeys.liveOption")}</SelectItem>
                    <SelectItem value="test">{t("apiKeys.testOption")}</SelectItem>
                  </SelectContent>
                </Select>
              </div>

              <Button type="submit" variant="cyan" disabled={createKey.isPending} className="w-full gap-2 mt-2">
                {createKey.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : <Lock className="h-4 w-4" />}
                <span>{createKey.isPending ? t("common.loading") : t("apiKeys.generateBtn")}</span>
              </Button>
            </form>
          </CardContent>
        </Card>

        {/* Macaroon Attenuation Studio Card */}
        <Card className="bg-card/80 flex flex-col justify-between">
          <CardHeader className="p-6 pb-4">
            <div className="flex items-center gap-2">
              <RefreshCw className="h-4 w-4 text-purple-400" />
              <CardTitle className="text-sm font-semibold">{t("apiKeys.attenuationTitle")}</CardTitle>
            </div>
            <CardDescription className="text-xs">{t("apiKeys.attenuationDesc")}</CardDescription>
          </CardHeader>

          <CardContent className="p-6 pt-0 space-y-4 flex-1">
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
      </div>

      {/* Issued Keys Table */}
      <Card className="bg-card/80">
        <CardHeader className="p-6 pb-4">
          <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <CardTitle className="text-sm font-semibold flex items-center gap-2">
                <Key className="h-4 w-4 text-cyan-400" />
                <span>{t("apiKeys.tableTitle")}</span>
                <Badge variant="secondary" className="font-mono text-[10px]">
                  {keys.length}
                </Badge>
              </CardTitle>
              <CardDescription className="text-xs">{t("apiKeys.tableDesc")}</CardDescription>
            </div>

            {/* Search Filter */}
            <div className="flex items-center gap-2 w-full sm:w-72">
              <div className="relative w-full">
                <Search className="absolute left-2.5 top-2.5 h-3.5 w-3.5 text-muted-foreground" />
                <Input
                  placeholder={t("apiKeys.searchPlaceholder")}
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
                <TableHead>{t("apiKeys.colName")}</TableHead>
                <TableHead>{t("apiKeys.colToken")}</TableHead>
                <TableHead>{t("apiKeys.colOwner")}</TableHead>
                <TableHead>{t("apiKeys.colEnv")}</TableHead>
                <TableHead>{t("apiKeys.colCreated")}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {isLoading ? (
                <TableRow>
                  <TableCell colSpan={5} className="p-0">
                    <LoadingState label="Fetching API keys from Vulcan database..." />
                  </TableCell>
                </TableRow>
              ) : isError ? (
                <TableRow>
                  <TableCell colSpan={5} className="p-0">
                    {error?.kind === "engine-unreachable" ? (
                      <NotConnectedState engineName="Vulcan" onRetry={refetch} />
                    ) : (
                      <ErrorState error={error} onRetry={refetch} />
                    )}
                  </TableCell>
                </TableRow>
              ) : filteredKeys.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={5} className="p-0">
                    <EmptyState title={t("apiKeys.tableTitle")} description="No API keys issued in Vulcan yet." />
                  </TableCell>
                </TableRow>
              ) : (
                filteredKeys.map((k) => (
                  <TableRow key={k.id}>
                    <TableCell className="font-semibold text-foreground">{k.name}</TableCell>
                    <TableCell className="font-mono text-xs text-cyan-400 font-medium">{k.keyMasked}</TableCell>
                    <TableCell className="text-muted-foreground">{k.ownerId}</TableCell>
                    <TableCell>
                      <Badge variant={k.prefix.includes("live") ? "success" : "warning"} className="text-[10px]">
                        {k.prefix.includes("live") ? t("common.production") : t("common.sandbox")}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-muted-foreground text-xs font-mono">{k.createdAt}</TableCell>
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
