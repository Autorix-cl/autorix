"use client";

import * as React from "react";
import {
  Shield,
  Play,
  CheckCircle2,
  FileCode,
  Server,
  ArrowUp,
  ArrowDown,
  Trash2,
  AlertTriangle,
  Layers,
  ArrowRight,
} from "lucide-react";
import { useTranslation } from "@/lib/i18n";
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { CodeBlock } from "@/components/ui/code-block";
import { useApiQuery } from "@/lib/query/use-api-query";
import { useApiMutation } from "@/lib/query/use-api-mutation";
import { fetchAndParse } from "@/lib/api/schema";
import {
  ruleListSchema,
  testMatchResponseSchema,
  type Rule,
  type PipelineTrace,
} from "@/lib/api/schemas/aegis";
import { LoadingState } from "@/components/state/loading-state";
import { EmptyState } from "@/components/state/empty-state";
import { ErrorState } from "@/components/state/error-state";
import { NotConnectedEngine } from "@/components/resources/not-connected-engine";
import { useCapabilities } from "@/lib/capabilities/capability-context";
import { RuleBuilderSheet } from "./rule-builder-sheet";
import { VersionsDialog } from "./versions-dialog";
import { detectShadowedRules } from "./shadowing-detector";
import { toast } from "sonner";
import { useMutation, useQueryClient } from "@tanstack/react-query";

interface RuleItem {
  id: string;
  match: {
    methods: string[];
    url: string;
  };
  authenticator: string;
  authorizer: string;
  mutator: string;
  upstream: string;
}

type ApiRule = Rule;

function toRuleItem(r: ApiRule): RuleItem {
  return {
    id: r.id,
    match: { methods: r.match.methods, url: r.match.url },
    authenticator: r.authenticators.map((a) => a.handler).join(", ") || "—",
    authorizer: r.authorizer?.handler || "—",
    mutator: r.mutators.map((m) => m.handler).join(", ") || "—",
    upstream: r.upstream.url,
  };
}

export default function ProxyRulesPage() {
  const { t } = useTranslation();
  const { isEngineConnected } = useCapabilities();
  const queryClient = useQueryClient();

  const {
    data: apiRulesRaw,
    isLoading: loadingRules,
    isError: rulesError,
    error: rulesErrorObj,
    refetch: refetchRules,
  } = useApiQuery(["proxy-rules"], () => fetchAndParse("/api/proxy-rules", ruleListSchema));

  const apiRules: ApiRule[] = React.useMemo(() => {
    if (!apiRulesRaw) return [];
    return [...apiRulesRaw].sort((a, b) => (a.order_idx ?? 0) - (b.order_idx ?? 0));
  }, [apiRulesRaw]);

  // Shadowing detection warnings
  const shadowWarnings = React.useMemo(() => {
    return detectShadowedRules(apiRules);
  }, [apiRules]);

  const shadowMap = React.useMemo(() => {
    const map = new Map<string, string>();
    for (const w of shadowWarnings) {
      map.set(w.ruleId, w.reason);
    }
    return map;
  }, [shadowWarnings]);

  const [testPath, setTestPath] = React.useState("/api/v1/documents/financial_report_2026");
  const [testMethod, setTestMethod] = React.useState("GET");
  const [matchedRule, setMatchedRule] = React.useState<RuleItem | null>(null);
  const [pipelineTrace, setPipelineTrace] = React.useState<PipelineTrace | null>(null);

  const testMatch = useApiMutation(
    (vars: { method: string; path: string }) =>
      fetchAndParse("/api/proxy-rules/test-match", testMatchResponseSchema, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(vars),
      }),
    {
      onSuccess: (data) => {
        setMatchedRule(data.matched && data.rule ? toRuleItem(data.rule) : null);
        setPipelineTrace(data.trace ?? null);
      },
    }
  );

  const handleTestMatch = () => {
    testMatch.mutate({ method: testMethod, path: testPath });
  };

  // Reordering Mutation
  const reorderMutation = useMutation({
    mutationFn: async (newOrderIds: string[]) => {
      const res = await fetch("/api/proxy-rules/reorder", {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ ids: newOrderIds }),
      });
      const data = await res.json();
      if (!res.ok) throw new Error(data.error || "Failed to reorder rules");
      return data;
    },
    onSuccess: () => {
      toast.success("Rule order updated successfully");
      queryClient.invalidateQueries({ queryKey: ["proxy-rules"] });
    },
    onError: (err) => {
      toast.error(err instanceof Error ? err.message : "Reorder failed");
    },
  });

  const moveRule = (index: number, direction: "up" | "down") => {
    const newRules = [...apiRules];
    const targetIndex = direction === "up" ? index - 1 : index + 1;
    if (targetIndex < 0 || targetIndex >= newRules.length) return;

    const temp = newRules[index];
    newRules[index] = newRules[targetIndex];
    newRules[targetIndex] = temp;

    const ids = newRules.map((r) => r.id);
    reorderMutation.mutate(ids);
  };

  // Delete Rule Mutation
  const deleteMutation = useMutation({
    mutationFn: async (id: string) => {
      const res = await fetch(`/api/proxy-rules/${id}`, { method: "DELETE" });
      const data = await res.json();
      if (!res.ok) throw new Error(data.error || "Failed to delete rule");
      return data;
    },
    onSuccess: () => {
      toast.success("Rule deleted successfully");
      queryClient.invalidateQueries({ queryKey: ["proxy-rules"] });
    },
    onError: (err) => {
      toast.error(err instanceof Error ? err.message : "Delete failed");
    },
  });

  const handleDelete = (id: string) => {
    if (confirm(`Are you sure you want to delete rule '${id}'?`)) {
      deleteMutation.mutate(id);
    }
  };

  const testing = testMatch.isPending;
  const rulesJSON = JSON.stringify(apiRules, null, 2);

  if (!isEngineConnected("aegis")) {
    return (
      <div className="space-y-6">
        <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
          <div>
            <h1 className="text-xl font-bold tracking-tight text-foreground sm:text-2xl">
              {t("proxyRules.title")}
            </h1>
            <p className="text-xs text-muted-foreground mt-1">{t("proxyRules.subtitle")}</p>
          </div>
        </div>
        <NotConnectedEngine
          engineType="aegis"
          engineName="Autorix Aegis (Zero-Trust Proxy)"
          description="Zero-trust identity-aware reverse proxy and traffic guard."
        />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
        <div>
          <h1 className="text-xl font-bold tracking-tight text-foreground sm:text-2xl">
            {t("proxyRules.title")}
          </h1>
          <p className="text-xs text-muted-foreground mt-1">{t("proxyRules.subtitle")}</p>
        </div>

        <div className="flex items-center gap-2">
          <VersionsDialog />
          <RuleBuilderSheet onSuccess={refetchRules} />
          <Badge variant="success" className="gap-1.5 py-1 px-3">
            <Shield className="h-3.5 w-3.5" />
            <span>{t("proxyRules.statusBadge")}</span>
          </Badge>
        </div>
      </div>

      {/* Simulator Card */}
      <Card className="bg-card/80">
        <CardHeader className="p-6 pb-4">
          <div className="flex items-center gap-2">
            <Play className="h-4 w-4 text-emerald-400" />
            <CardTitle className="text-sm font-semibold">{t("proxyRules.simulatorTitle")}</CardTitle>
          </div>
          <CardDescription className="text-xs">{t("proxyRules.simulatorDesc")}</CardDescription>
        </CardHeader>

        <CardContent className="p-6 pt-0 space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-12 gap-3 items-end">
            <div className="md:col-span-3 space-y-1.5">
              <Label>{t("proxyRules.methodLabel")}</Label>
              <Select value={testMethod} onValueChange={setTestMethod}>
                <SelectTrigger>
                  <SelectValue placeholder="Select method" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="GET">GET</SelectItem>
                  <SelectItem value="POST">POST</SelectItem>
                  <SelectItem value="PUT">PUT</SelectItem>
                  <SelectItem value="DELETE">DELETE</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div className="md:col-span-6 space-y-1.5">
              <Label>{t("proxyRules.pathLabel")}</Label>
              <Input
                value={testPath}
                onChange={(e) => setTestPath(e.target.value)}
                placeholder="/api/v1/resource"
                className="font-mono text-xs"
              />
            </div>

            <div className="md:col-span-3">
              <Button
                type="button"
                variant="success"
                onClick={handleTestMatch}
                disabled={testing}
                className="w-full gap-2"
              >
                <Play className="h-4 w-4" />
                <span>{testing ? t("proxyRules.testingBtn") : t("proxyRules.simulateBtn")}</span>
              </Button>
            </div>
          </div>

          {/* Matched Rule Result Visualization */}
          {matchedRule ? (
            <div className="rounded-xl border border-border/80 bg-muted/20 p-4 space-y-4">
              <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-2 border-b border-border/50 pb-3">
                <Badge variant="success" className="gap-1.5 py-0.5 font-mono text-xs">
                  <CheckCircle2 className="h-3.5 w-3.5" />
                  <span>{t("proxyRules.matchedBadge", { id: matchedRule.id })}</span>
                </Badge>
                <div className="flex items-center gap-2 font-mono text-xs text-muted-foreground">
                  <Server className="h-3.5 w-3.5 text-blue-400" />
                  <span>
                    {t("proxyRules.upstreamLabel")}: <strong className="text-foreground">{matchedRule.upstream}</strong>
                  </span>
                </div>
              </div>

              {/* 3 Core Pipeline Stages */}
              <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
                <div className="rounded-lg border border-border/60 bg-card p-3 space-y-1.5">
                  <div className="text-[10px] font-bold uppercase text-blue-400">{t("proxyRules.step1")}</div>
                  <div className="text-xs font-semibold text-foreground font-mono">{matchedRule.authenticator}</div>
                  <div className="text-[11px] text-muted-foreground">
                    Verifies cryptographic claims & token validity
                  </div>
                </div>

                <div className="rounded-lg border border-border/60 bg-card p-3 space-y-1.5">
                  <div className="text-[10px] font-bold uppercase text-purple-400">{t("proxyRules.step2")}</div>
                  <div className="text-xs font-semibold text-foreground font-mono">{matchedRule.authorizer}</div>
                  <div className="text-[11px] text-muted-foreground">
                    Zero-latency Zanzibar relation & CEL condition
                  </div>
                </div>

                <div className="rounded-lg border border-border/60 bg-card p-3 space-y-1.5">
                  <div className="text-[10px] font-bold uppercase text-emerald-400">{t("proxyRules.step3")}</div>
                  <div className="text-xs font-semibold text-foreground font-mono">{matchedRule.mutator}</div>
                  <div className="text-[11px] text-muted-foreground">
                    Injects verified identity claims into upstream headers
                  </div>
                </div>
              </div>

              {/* Enhanced Step-by-Step Execution Trace (P6-S5-T4) */}
              {pipelineTrace && pipelineTrace.steps && pipelineTrace.steps.length > 0 && (
                <div className="border-t border-border/60 pt-3 space-y-2">
                  <div className="flex items-center justify-between">
                    <span className="text-xs font-semibold flex items-center gap-1.5 text-foreground">
                      <Layers className="w-3.5 h-3.5 text-primary" />
                      Dry-Run Execution Trace
                    </span>
                    <Badge
                      variant={pipelineTrace.final_verdict === "allow" ? "success" : "destructive"}
                      className="text-[10px] uppercase tracking-wider font-mono font-bold"
                    >
                      Verdict: {pipelineTrace.final_verdict}
                    </Badge>
                  </div>
                  <div className="space-y-1.5">
                    {pipelineTrace.steps.map((step, idx) => (
                      <div
                        key={idx}
                        className="text-xs font-mono p-2 rounded bg-background/80 border border-border/40 flex items-center justify-between"
                      >
                        <div className="flex items-center gap-2">
                          <span className="text-muted-foreground">{idx + 1}.</span>
                          <span className="font-semibold uppercase text-[11px] text-primary">
                            {step.stage}
                          </span>
                          {step.handler && (
                            <span className="text-muted-foreground">({step.handler})</span>
                          )}
                          {step.details && (
                            <span className="text-xs text-foreground/80">{step.details}</span>
                          )}
                          {step.target_url && (
                            <span className="text-xs text-blue-400 flex items-center gap-1">
                              <ArrowRight className="w-3 h-3" />
                              {step.target_url}
                            </span>
                          )}
                        </div>
                        <Badge
                          variant={step.status === "success" ? "secondary" : "destructive"}
                          className="text-[10px]"
                        >
                          {step.status}
                        </Badge>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          ) : (
            <div className="rounded-xl border border-rose-500/30 bg-rose-500/10 p-4 text-xs text-rose-400 font-medium">
              {t("proxyRules.noMatch")}
            </div>
          )}
        </CardContent>
      </Card>

      {/* Ordered Rules Management Table (P6-S5-T2, T7) */}
      <Card className="bg-card/80">
        <CardHeader className="p-6 pb-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <Shield className="h-4 w-4 text-primary" />
              <CardTitle className="text-sm font-semibold">Active Proxy Rules & Order</CardTitle>
            </div>
            <span className="text-xs text-muted-foreground font-mono">
              {apiRules.length} {apiRules.length === 1 ? "rule configured" : "rules configured"}
            </span>
          </div>
          <CardDescription className="text-xs">
            Evaluation is first-match wins. Reorder rules using controls to prevent shadowing.
          </CardDescription>
        </CardHeader>

        <CardContent className="p-6 pt-0">
          {loadingRules ? (
            <LoadingState label="Loading rules from Aegis…" />
          ) : rulesError ? (
            <ErrorState error={rulesErrorObj} onRetry={refetchRules} />
          ) : apiRules.length === 0 ? (
            <EmptyState title="No Rules Configured" description="Use the 'New Proxy Rule' wizard to register your first routing rule." />
          ) : (
            <div className="border rounded-md divide-y overflow-hidden">
              {apiRules.map((rule, idx) => {
                const isShadowed = shadowMap.has(rule.id);
                const shadowReason = shadowMap.get(rule.id);

                return (
                  <div
                    key={rule.id}
                    className={`p-4 flex flex-col md:flex-row md:items-center justify-between gap-4 transition-colors ${
                      isShadowed ? "bg-amber-500/5 hover:bg-amber-500/10" : "hover:bg-muted/30"
                    }`}
                  >
                    <div className="flex items-start gap-3">
                      {/* Priority Controls */}
                      <div className="flex flex-col items-center gap-1 pr-2 border-r">
                        <Button
                          size="icon"
                          variant="ghost"
                          className="h-6 w-6"
                          disabled={idx === 0 || reorderMutation.isPending}
                          onClick={() => moveRule(idx, "up")}
                          title="Move up in evaluation priority"
                        >
                          <ArrowUp className="w-3.5 h-3.5" />
                        </Button>
                        <span className="text-[11px] font-mono font-bold text-muted-foreground">
                          #{idx + 1}
                        </span>
                        <Button
                          size="icon"
                          variant="ghost"
                          className="h-6 w-6"
                          disabled={idx === apiRules.length - 1 || reorderMutation.isPending}
                          onClick={() => moveRule(idx, "down")}
                          title="Move down in evaluation priority"
                        >
                          <ArrowDown className="w-3.5 h-3.5" />
                        </Button>
                      </div>

                      {/* Rule Details */}
                      <div className="space-y-1.5">
                        <div className="flex flex-wrap items-center gap-2">
                          <span className="font-mono font-semibold text-sm text-foreground">{rule.id}</span>
                          {rule.match.methods.map((m) => (
                            <Badge key={m} variant="secondary" className="text-[10px] py-0 font-mono">
                              {m}
                            </Badge>
                          ))}
                          {isShadowed && (
                            <Badge
                              variant="outline"
                              className="bg-amber-500/10 text-amber-500 border-amber-500/30 text-[10px] gap-1 py-0"
                              title={shadowReason}
                            >
                              <AlertTriangle className="w-3 h-3 text-amber-500" />
                              Shadowed Rule
                            </Badge>
                          )}
                        </div>

                        {rule.description && (
                          <p className="text-xs text-muted-foreground">{rule.description}</p>
                        )}

                        <div className="flex flex-wrap items-center gap-4 text-xs font-mono text-muted-foreground pt-1">
                          <span>
                            Path: <strong className="text-foreground">{rule.match.url}</strong>
                          </span>
                          <span>
                            Upstream: <strong className="text-foreground">{rule.upstream.url}</strong>
                          </span>
                          {rule.upstream.strip_prefix && (
                            <Badge variant="outline" className="text-[10px] py-0">
                              strip: {rule.upstream.strip_prefix}
                            </Badge>
                          )}
                          {rule.upstream.rewrite && (
                            <Badge variant="outline" className="text-[10px] py-0">
                              rewrite: {rule.upstream.rewrite}
                            </Badge>
                          )}
                        </div>

                        {isShadowed && (
                          <p className="text-xs text-amber-500/90 italic pt-1">
                            Warning: {shadowReason}
                          </p>
                        )}
                      </div>
                    </div>

                    {/* Actions */}
                    <div className="flex items-center gap-2 self-end md:self-center">
                      <Button
                        size="icon"
                        variant="ghost"
                        className="h-8 w-8 text-muted-foreground hover:text-destructive hover:bg-destructive/10"
                        onClick={() => handleDelete(rule.id)}
                        disabled={deleteMutation.isPending}
                        title="Delete rule"
                      >
                        <Trash2 className="w-4 h-4" />
                      </Button>
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </CardContent>
      </Card>

      {/* Declarative Rules Live JSON Viewer */}
      <Card className="bg-card/80">
        <CardHeader className="p-6 pb-4">
          <div className="flex items-center gap-2">
            <FileCode className="h-4 w-4 text-blue-400" />
            <CardTitle className="text-sm font-semibold">{t("proxyRules.yamlTitle")}</CardTitle>
          </div>
          <CardDescription className="text-xs">{t("proxyRules.yamlDesc")}</CardDescription>
        </CardHeader>

        <CardContent className="p-6 pt-0">
          {loadingRules ? (
            <LoadingState label="Loading rules from Aegis…" />
          ) : rulesError ? (
            <ErrorState error={rulesErrorObj} onRetry={refetchRules} />
          ) : apiRules.length === 0 ? (
            <EmptyState title={t("proxyRules.yamlTitle")} description="No proxy rules registered in Aegis yet." />
          ) : (
            <CodeBlock code={rulesJSON} language="json" title="GET /rules (Aegis admin API, live)" showLineNumbers />
          )}
        </CardContent>
      </Card>
    </div>
  );
}
