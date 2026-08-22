"use client";

import * as React from "react";
import { Play, CheckCircle2, XCircle, Zap, Database, GitGraph, Plus, RefreshCw } from "lucide-react";
import { useTranslation } from "@/lib/i18n";
import { useQueryClient } from "@tanstack/react-query";
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Badge } from "@/components/ui/badge";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { CodeBlock } from "@/components/ui/code-block";
import { DataTable } from "@/components/ui/data-table";
import { useApiQuery } from "@/lib/query/use-api-query";
import { useApiMutation } from "@/lib/query/use-api-mutation";
import { fetchAndParse } from "@/lib/api/schema";
import { paginatedTupleListSchema, checkResponseSchema, deleteTuplesResponseSchema, type Tuple, type PaginatedTuples } from "@/lib/api/schemas/nexus";
import { ErrorState } from "@/components/state/error-state";
import { NotConnectedState } from "@/components/state/not-connected-state";
import { NotConnectedEngine } from "@/components/resources/not-connected-engine";
import { useCapabilities } from "@/lib/capabilities/capability-context";
import { getColumns, ZanzibarTupleItem } from "./columns";
import { TupleBuilderSheet } from "./tuple-builder-sheet";

interface CheckVars {
  namespace: string;
  object: string;
  relation: string;
  subjectId: string;
  requestContext?: Record<string, unknown>;
  startedAt: number;
}

function toZanzibarTuple(t: Tuple): ZanzibarTupleItem {
  return {
    namespace: t.namespace,
    object: t.object,
    relation: t.relation,
    subject: `${t.subject_namespace || "user"}:${t.subject_id}${t.subject_relation ? "#" + t.subject_relation : ""}`,
    caveat: t.caveat_name || undefined,
    original: t,
  };
}

export default function PermissionsPage() {
  const { t } = useTranslation();
  const { isEngineConnected } = useCapabilities();
  const queryClient = useQueryClient();

  const [namespace, setNamespace] = React.useState("document");
  const [object, setObject] = React.useState("financial_report_2026");
  const [relation, setRelation] = React.useState("viewer");
  const [subjectId, setSubjectId] = React.useState("alice");
  const [requestContext, setRequestContext] = React.useState(
    '{\n  "ip": "192.168.1.100",\n  "amount": 450,\n  "mfa_verified": true\n}',
  );

  const [result, setResult] = React.useState<{
    allowed: boolean;
    reason: string;
    latencyMs: number;
    path: string;
  } | null>(null);

  const [cursor, setCursor] = React.useState("");
  const [cursorHistory, setCursorHistory] = React.useState<string[]>([]);
  const [isBuilderOpen, setIsBuilderOpen] = React.useState(false);

  const {
    data: tuplesRaw,
    isLoading: isLoadingTuples,
    isFetching: isFetchingTuples,
    isError: isTuplesError,
    error: tuplesError,
    refetch: refetchTuples,
  } = useApiQuery(
    ["nexus-tuples", { cursor }],
    () => fetchAndParse<PaginatedTuples>(`/api/permissions?cursor=${encodeURIComponent(cursor)}`, paginatedTupleListSchema)
  );

  const tuples: ZanzibarTupleItem[] = React.useMemo(() => (tuplesRaw?.data ?? []).map(toZanzibarTuple), [tuplesRaw]);

  const deleteMutation = useApiMutation(
    (tuple: ZanzibarTupleItem) =>
      fetchAndParse("/api/permissions", deleteTuplesResponseSchema, {
        method: "DELETE",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(tuple.original),
      }),
    {
      successMessage: () => "Tuple deleted successfully.",
      onSuccess: () => {
        queryClient.invalidateQueries({ queryKey: ["nexus-tuples"] });
      },
    }
  );

  const columns = React.useMemo(() => getColumns((tuple) => deleteMutation.mutate(tuple)), [deleteMutation]);

  const checkMutation = useApiMutation(
    // eslint-disable-next-line @typescript-eslint/no-unused-vars -- startedAt is stripped from the request body on purpose
    ({ startedAt, ...body }: CheckVars) =>
      fetchAndParse("/api/permissions/check", checkResponseSchema, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
    {
      onSuccess: (data, vars) => {
        const latencyMs = Number((performance.now() - vars.startedAt).toFixed(2));
        setResult({
          allowed: Boolean(data.allowed),
          reason:
            data.reason ||
            t("permissions.directMatchReason", {
              namespace: vars.namespace,
              object: vars.object,
              relation: vars.relation,
              subject: `user:${vars.subjectId}`,
            }),
          latencyMs,
          path: `user:${vars.subjectId} -> direct relation -> ${vars.namespace}:${vars.object}#${vars.relation}`,
        });
      },
      onError: (err, vars) => {
        const latencyMs = Number((performance.now() - vars.startedAt).toFixed(2));
        setResult({
          allowed: false,
          reason: err.message || "Check request failed",
          latencyMs,
          path: `user:${vars.subjectId} -> ${vars.namespace}:${vars.object}#${vars.relation}`,
        });
      },
    },
  );

  const handleSimulate = (e: React.FormEvent) => {
    e.preventDefault();

    let parsedContext: Record<string, unknown> | undefined;
    try {
      parsedContext = requestContext ? JSON.parse(requestContext) : undefined;
    } catch {
      // Let Nexus's CEL evaluator surface the parsing/evaluation error instead.
      parsedContext = undefined;
    }

    checkMutation.mutate({
      namespace,
      object,
      relation,
      subjectId,
      requestContext: parsedContext,
      startedAt: performance.now(),
    });
  };

  const evaluating = checkMutation.isPending;

  const handleNextPage = () => {
    if (tuplesRaw?.has_more && tuplesRaw.next_cursor) {
      setCursorHistory((prev) => [...prev, cursor]);
      setCursor(tuplesRaw.next_cursor);
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

  if (!isEngineConnected("nexus")) {
    return (
      <div className="space-y-6">
        <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
          <div>
            <h1 className="text-xl font-bold tracking-tight text-foreground sm:text-2xl">{t("permissions.title")}</h1>
            <p className="text-xs text-muted-foreground mt-1">{t("permissions.subtitle")}</p>
          </div>
        </div>
        <NotConnectedEngine
          engineType="nexus"
          engineName="Autorix Nexus (ReBAC)"
          description="Google Zanzibar authorization engine."
        />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <TupleBuilderSheet isOpen={isBuilderOpen} onOpenChange={setIsBuilderOpen} />

      {/* Page Header */}
      <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
        <div>
          <h1 className="text-xl font-bold tracking-tight text-foreground sm:text-2xl">{t("permissions.title")}</h1>
          <p className="text-xs text-muted-foreground mt-1">{t("permissions.subtitle")}</p>
        </div>

        <div className="flex items-center gap-2">
          <Badge variant="purple" className="gap-1.5 py-1 px-3">
            <Zap className="h-3.5 w-3.5" />
            <span>{t("permissions.statusBadge")}</span>
          </Badge>
        </div>
      </div>

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        {/* Simulator Query Form */}
        <Card className="bg-card/80">
          <CardHeader className="p-6 pb-4">
            <div className="flex items-center gap-2">
              <Play className="h-4 w-4 text-purple-400" />
              <CardTitle className="text-sm font-semibold">{t("permissions.simulatorTitle")}</CardTitle>
            </div>
            <CardDescription className="text-xs">{t("permissions.simulatorDesc")}</CardDescription>
          </CardHeader>

          <CardContent className="p-6 pt-0">
            <form onSubmit={handleSimulate} className="space-y-4">
              <div className="grid grid-cols-2 gap-3">
                <div className="space-y-1.5">
                  <Label htmlFor="namespace">{t("permissions.namespaceLabel")}</Label>
                  <Input id="namespace" value={namespace} onChange={(e) => setNamespace(e.target.value)} required />
                </div>

                <div className="space-y-1.5">
                  <Label htmlFor="object">{t("permissions.objectLabel")}</Label>
                  <Input id="object" value={object} onChange={(e) => setObject(e.target.value)} required />
                </div>
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div className="space-y-1.5">
                  <Label htmlFor="relation">{t("permissions.relationLabel")}</Label>
                  <Select value={relation} onValueChange={setRelation}>
                    <SelectTrigger id="relation">
                      <SelectValue placeholder="Select relation" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="viewer">viewer</SelectItem>
                      <SelectItem value="editor">editor</SelectItem>
                      <SelectItem value="owner">owner</SelectItem>
                      <SelectItem value="member">member</SelectItem>
                      <SelectItem value="admin">admin</SelectItem>
                    </SelectContent>
                  </Select>
                </div>

                <div className="space-y-1.5">
                  <Label htmlFor="subject">{t("permissions.subjectLabel")}</Label>
                  <Input id="subject" value={subjectId} onChange={(e) => setSubjectId(e.target.value)} required />
                </div>
              </div>

              <div className="space-y-1.5">
                <Label htmlFor="context">{t("permissions.contextLabel")}</Label>
                <Textarea
                  id="context"
                  value={requestContext}
                  onChange={(e) => setRequestContext(e.target.value)}
                  rows={4}
                  className="font-mono text-xs"
                />
              </div>

              <Button type="submit" variant="purple" disabled={evaluating} className="w-full gap-2 mt-2">
                <Zap className="h-4 w-4" />
                <span>{evaluating ? t("permissions.evaluatingBtn") : t("permissions.evaluateBtn")}</span>
              </Button>
            </form>
          </CardContent>
        </Card>

        {/* Results Panel */}
        <Card className="bg-card/80 flex flex-col justify-between">
          <CardHeader className="p-6 pb-4">
            <div className="flex items-center gap-2">
              <GitGraph className="h-4 w-4 text-purple-400" />
              <CardTitle className="text-sm font-semibold">{t("permissions.resultTitle")}</CardTitle>
            </div>
            <CardDescription className="text-xs">
              Live Zanzibar traversal graph & CEL ABAC execution trace.
            </CardDescription>
          </CardHeader>

          <CardContent className="p-6 pt-0 space-y-4 flex-1">
            {result ? (
              <div className="space-y-4">
                {/* Result Hero Banner */}
                <div
                  className={`flex items-center gap-4 rounded-xl border p-4 transition-colors ${
                    result.allowed
                      ? "border-emerald-500/30 bg-emerald-500/10 text-emerald-400"
                      : "border-rose-500/30 bg-rose-500/10 text-rose-400"
                  }`}
                >
                  {result.allowed ? (
                    <CheckCircle2 className="h-8 w-8 text-emerald-400 flex-shrink-0" />
                  ) : (
                    <XCircle className="h-8 w-8 text-rose-400 flex-shrink-0" />
                  )}
                  <div>
                    <div className="text-lg font-bold tracking-tight">
                      {result.allowed ? t("permissions.allowed") : t("permissions.denied")}
                    </div>
                    <div className="text-xs text-muted-foreground font-mono mt-0.5">
                      {t("permissions.resolvedIn", { ms: result.latencyMs })}
                    </div>
                  </div>
                </div>

                <div className="space-y-2">
                  <Label>{t("permissions.reasonLabel")}</Label>
                  <CodeBlock code={result.reason} language="text" title="DECISION ENGINE" />
                </div>

                <div className="space-y-2">
                  <Label>{t("permissions.pathLabel")}</Label>
                  <CodeBlock code={result.path} language="text" title="ZANZIBAR RESOLUTION GRAPH" />
                </div>
              </div>
            ) : (
              <div className="flex h-48 items-center justify-center text-xs text-muted-foreground">
                Run a simulation to evaluate the ReBAC authorization tree.
              </div>
            )}
          </CardContent>
        </Card>
      </div>

      {/* Active Relation Tuples Table */}
      <Card className="bg-card/80">
        <CardHeader className="p-6 pb-4">
          <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <CardTitle className="text-sm font-semibold flex items-center gap-2">
                <Database className="h-4 w-4 text-purple-400" />
                <span>{t("permissions.tuplesTitle")}</span>
                <Badge variant="secondary" className="font-mono text-[10px]">
                  {tuples.length}
                </Badge>
              </CardTitle>
              <CardDescription className="text-xs">{t("permissions.tuplesDesc")}</CardDescription>
            </div>

            <div className="flex items-center gap-2">
              <Button
                variant="outline"
                size="sm"
                onClick={() => refetchTuples()}
                disabled={isFetchingTuples || isLoadingTuples}
                className="h-8 gap-1 text-xs"
              >
                <RefreshCw className={`h-3.5 w-3.5 ${isFetchingTuples ? "animate-spin" : ""}`} />
                <span>{t("common.refresh")}</span>
              </Button>
              <Button
                size="sm"
                onClick={() => setIsBuilderOpen(true)}
                className="h-8 gap-1 text-xs bg-purple-600 hover:bg-purple-700 text-white"
              >
                <Plus className="h-3.5 w-3.5" />
                <span>Add Tuple</span>
              </Button>
            </div>
          </div>
        </CardHeader>

        <CardContent className="p-6 pt-0">
          {isTuplesError && tuplesError?.kind === "engine-unreachable" ? (
            <NotConnectedState engineName="Nexus" onRetry={refetchTuples} />
          ) : isTuplesError ? (
            <ErrorState error={tuplesError} onRetry={refetchTuples} />
          ) : (
            <DataTable
              columns={columns}
              data={tuples}
              isLoading={isLoadingTuples || isFetchingTuples}
              manualPagination={true}
              onNextPage={handleNextPage}
              onPreviousPage={handlePrevPage}
              canNextPage={!!tuplesRaw?.has_more}
              canPreviousPage={cursorHistory.length > 0}
            />
          )}
        </CardContent>
      </Card>
    </div>
  );
}
