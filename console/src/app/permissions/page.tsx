"use client";

import * as React from "react";
import { Play, CheckCircle2, XCircle, Zap, Database, GitGraph } from "lucide-react";
import { useTranslation } from "@/lib/i18n";
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Badge } from "@/components/ui/badge";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from "@/components/ui/table";
import { CodeBlock } from "@/components/ui/code-block";
import { useApiQuery } from "@/lib/query/use-api-query";
import { useApiMutation } from "@/lib/query/use-api-mutation";
import { fetchAndParse } from "@/lib/api/schema";
import { tupleListSchema, checkResponseSchema, type Tuple } from "@/lib/api/schemas/nexus";
import { LoadingState } from "@/components/state/loading-state";
import { EmptyState } from "@/components/state/empty-state";
import { ErrorState } from "@/components/state/error-state";
import { NotConnectedState } from "@/components/state/not-connected-state";
import { NotConnectedEngine } from "@/components/resources/not-connected-engine";
import { useCapabilities } from "@/lib/capabilities/capability-context";

interface ZanzibarTuple {
  namespace: string;
  object: string;
  relation: string;
  subject: string;
  caveat?: string;
}

function toZanzibarTuple(t: Tuple): ZanzibarTuple {
  return {
    namespace: t.namespace,
    object: t.object,
    relation: t.relation,
    subject: `${t.subject_namespace || "user"}:${t.subject_id}${t.subject_relation ? "#" + t.subject_relation : ""}`,
    caveat: t.caveat_name || undefined,
  };
}

interface CheckVars {
  namespace: string;
  object: string;
  relation: string;
  subjectId: string;
  requestContext?: Record<string, unknown>;
  startedAt: number;
}

export default function PermissionsPage() {
  const { t } = useTranslation();
  const { isEngineConnected } = useCapabilities();

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

  const {
    data: tuplesRaw,
    isLoading: isLoadingTuples,
    isError: isTuplesError,
    error: tuplesError,
    refetch: refetchTuples,
  } = useApiQuery(["nexus-tuples"], () => fetchAndParse<Tuple[]>("/api/permissions", tupleListSchema));

  const tuples: ZanzibarTuple[] = React.useMemo(() => (tuplesRaw ?? []).map(toZanzibarTuple), [tuplesRaw]);

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
          <div className="flex items-center justify-between">
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
          </div>
        </CardHeader>

        <CardContent className="p-6 pt-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t("permissions.colNamespace")}</TableHead>
                <TableHead>{t("permissions.colObject")}</TableHead>
                <TableHead>{t("permissions.colRelation")}</TableHead>
                <TableHead>{t("permissions.colSubject")}</TableHead>
                <TableHead>{t("permissions.colCaveat")}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {isLoadingTuples ? (
                <TableRow>
                  <TableCell colSpan={5} className="p-0">
                    <LoadingState label="Loading relation tuples from Nexus…" />
                  </TableCell>
                </TableRow>
              ) : isTuplesError ? (
                <TableRow>
                  <TableCell colSpan={5} className="p-0">
                    {tuplesError?.kind === "engine-unreachable" ? (
                      <NotConnectedState engineName="Nexus" onRetry={refetchTuples} />
                    ) : (
                      <ErrorState error={tuplesError} onRetry={refetchTuples} />
                    )}
                  </TableCell>
                </TableRow>
              ) : tuples.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={5} className="p-0">
                    <EmptyState
                      title={t("permissions.tuplesTitle")}
                      description="No relation tuples yet. Write one via the Nexus REST API to see it here."
                    />
                  </TableCell>
                </TableRow>
              ) : (
                tuples.map((tup, idx) => (
                  <TableRow key={idx}>
                    <TableCell className="font-mono text-xs text-purple-400 font-medium">{tup.namespace}</TableCell>
                    <TableCell className="font-mono text-xs text-foreground font-semibold">{tup.object}</TableCell>
                    <TableCell>
                      <Badge variant="outline" className="font-mono text-[10px]">
                        #{tup.relation}
                      </Badge>
                    </TableCell>
                    <TableCell className="font-mono text-xs text-blue-400">@{tup.subject}</TableCell>
                    <TableCell>
                      {tup.caveat ? (
                        <span className="rounded bg-purple-500/10 px-2 py-0.5 text-[10px] font-mono text-purple-300 border border-purple-500/20">
                          {tup.caveat}
                        </span>
                      ) : (
                        <span className="text-xs text-muted-foreground">—</span>
                      )}
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
