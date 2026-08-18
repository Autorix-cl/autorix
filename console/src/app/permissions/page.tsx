"use client";

import * as React from "react";
import { 
  Network, 
  Play, 
  CheckCircle2, 
  XCircle, 
  Zap, 
  ShieldAlert,
  Database,
  Layers,
  Sparkles,
  GitGraph
} from "lucide-react";
import { useTranslation } from "@/lib/i18n";
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Badge } from "@/components/ui/badge";
import { 
  Select, 
  SelectContent, 
  SelectItem, 
  SelectTrigger, 
  SelectValue 
} from "@/components/ui/select";
import { 
  Table, 
  TableHeader, 
  TableBody, 
  TableRow, 
  TableHead, 
  TableCell 
} from "@/components/ui/table";
import { CodeBlock } from "@/components/ui/code-block";

interface ZanzibarTuple {
  namespace: string;
  object: string;
  relation: string;
  subject: string;
  caveat?: string;
}

export default function PermissionsPage() {
  const { t } = useTranslation();

  const [namespace, setNamespace] = React.useState("document");
  const [object, setObject] = React.useState("financial_report_2026");
  const [relation, setRelation] = React.useState("viewer");
  const [subjectId, setSubjectId] = React.useState("alice");
  const [requestContext, setRequestContext] = React.useState(
    '{\n  "ip": "192.168.1.100",\n  "amount": 450,\n  "mfa_verified": true\n}'
  );

  const [evaluating, setEvaluating] = React.useState(false);
  const [result, setResult] = React.useState<{
    allowed: boolean;
    reason: string;
    latencyMs: number;
    path: string;
  } | null>(null);

  const [tuples, setTuples] = React.useState<ZanzibarTuple[]>([]);
  const [loadingTuples, setLoadingTuples] = React.useState(true);

  const fetchTuples = React.useCallback(async () => {
    setLoadingTuples(true);
    try {
      const res = await fetch("/api/permissions");
      const data = await res.json();
      const mapped: ZanzibarTuple[] = (Array.isArray(data) ? data : []).map((t: any) => ({
        namespace: t.namespace,
        object: t.object,
        relation: t.relation,
        subject: `${t.subject_namespace || "user"}:${t.subject_id}${
          t.subject_relation ? "#" + t.subject_relation : ""
        }`,
        caveat: t.caveat_name || undefined,
      }));
      setTuples(mapped);
    } catch (err) {
      setTuples([]);
    } finally {
      setLoadingTuples(false);
    }
  }, []);

  React.useEffect(() => {
    fetchTuples();
  }, [fetchTuples]);

  const handleSimulate = async (e: React.FormEvent) => {
    e.preventDefault();
    setEvaluating(true);

    const startedAt = performance.now();
    let parsedContext: Record<string, unknown> | undefined;
    try {
      parsedContext = requestContext ? JSON.parse(requestContext) : undefined;
    } catch (err) {
      // Let Nexus's CEL evaluator surface the parsing/evaluation error instead.
      parsedContext = undefined;
    }

    try {
      const res = await fetch("/api/permissions/check", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          namespace,
          object,
          relation,
          subjectId,
          requestContext: parsedContext,
        }),
      });
      const data = await res.json();
      const latencyMs = Number((performance.now() - startedAt).toFixed(2));

      if (!res.ok) {
        setResult({
          allowed: false,
          reason: data.error || "Check request failed",
          latencyMs,
          path: `user:${subjectId} -> ${namespace}:${object}#${relation}`,
        });
        return;
      }

      setResult({
        allowed: Boolean(data.allowed),
        reason:
          data.reason ||
          t("permissions.directMatchReason", {
            namespace,
            object,
            relation,
            subject: `user:${subjectId}`,
          }),
        latencyMs,
        path: `user:${subjectId} -> direct relation -> ${namespace}:${object}#${relation}`,
      });
    } catch (err: any) {
      setResult({
        allowed: false,
        reason: err.message || "Failed to reach Nexus",
        latencyMs: Number((performance.now() - startedAt).toFixed(2)),
        path: `user:${subjectId} -> ${namespace}:${object}#${relation}`,
      });
    } finally {
      setEvaluating(false);
    }
  };

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
        <div>
          <h1 className="text-xl font-bold tracking-tight text-foreground sm:text-2xl">
            {t("permissions.title")}
          </h1>
          <p className="text-xs text-muted-foreground mt-1">
            {t("permissions.subtitle")}
          </p>
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
              <CardTitle className="text-sm font-semibold">
                {t("permissions.simulatorTitle")}
              </CardTitle>
            </div>
            <CardDescription className="text-xs">
              {t("permissions.simulatorDesc")}
            </CardDescription>
          </CardHeader>

          <CardContent className="p-6 pt-0">
            <form onSubmit={handleSimulate} className="space-y-4">
              <div className="grid grid-cols-2 gap-3">
                <div className="space-y-1.5">
                  <Label htmlFor="namespace">{t("permissions.namespaceLabel")}</Label>
                  <Input
                    id="namespace"
                    value={namespace}
                    onChange={(e) => setNamespace(e.target.value)}
                    required
                  />
                </div>

                <div className="space-y-1.5">
                  <Label htmlFor="object">{t("permissions.objectLabel")}</Label>
                  <Input
                    id="object"
                    value={object}
                    onChange={(e) => setObject(e.target.value)}
                    required
                  />
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
                  <Input
                    id="subject"
                    value={subjectId}
                    onChange={(e) => setSubjectId(e.target.value)}
                    required
                  />
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

              <Button
                type="submit"
                variant="purple"
                disabled={evaluating}
                className="w-full gap-2 mt-2"
              >
                <Zap className="h-4 w-4" />
                <span>
                  {evaluating
                    ? t("permissions.evaluatingBtn")
                    : t("permissions.evaluateBtn")}
                </span>
              </Button>
            </form>
          </CardContent>
        </Card>

        {/* Results Panel */}
        <Card className="bg-card/80 flex flex-col justify-between">
          <CardHeader className="p-6 pb-4">
            <div className="flex items-center gap-2">
              <GitGraph className="h-4 w-4 text-purple-400" />
              <CardTitle className="text-sm font-semibold">
                {t("permissions.resultTitle")}
              </CardTitle>
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
                  <CodeBlock
                    code={result.reason}
                    language="text"
                    title="DECISION ENGINE"
                  />
                </div>

                <div className="space-y-2">
                  <Label>{t("permissions.pathLabel")}</Label>
                  <CodeBlock
                    code={result.path}
                    language="text"
                    title="ZANZIBAR RESOLUTION GRAPH"
                  />
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
              <CardDescription className="text-xs">
                {t("permissions.tuplesDesc")}
              </CardDescription>
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
              {loadingTuples ? (
                <TableRow>
                  <TableCell colSpan={5} className="text-center text-xs text-muted-foreground py-8">
                    Loading relation tuples from Nexus…
                  </TableCell>
                </TableRow>
              ) : tuples.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={5} className="text-center text-xs text-muted-foreground py-8">
                    No relation tuples yet. Write one via the Nexus REST API to see it here.
                  </TableCell>
                </TableRow>
              ) : (
                tuples.map((tup, idx) => (
                  <TableRow key={idx}>
                    <TableCell className="font-mono text-xs text-purple-400 font-medium">
                      {tup.namespace}
                    </TableCell>
                    <TableCell className="font-mono text-xs text-foreground font-semibold">
                      {tup.object}
                    </TableCell>
                    <TableCell>
                      <Badge variant="outline" className="font-mono text-[10px]">
                        #{tup.relation}
                      </Badge>
                    </TableCell>
                    <TableCell className="font-mono text-xs text-blue-400">
                      @{tup.subject}
                    </TableCell>
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
