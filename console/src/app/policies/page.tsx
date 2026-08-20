"use client";

import * as React from "react";
import {
  Scale,
  Plus,
  RefreshCw,
  Search,
  Code2,
  Play,
  Trash2,
  Loader2,
  CheckCircle2,
  XCircle,
  Sparkles,
} from "lucide-react";
import { useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "@/lib/i18n";
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Textarea } from "@/components/ui/textarea";
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from "@/components/ui/table";
import type { z } from "zod";
import { useApiQuery } from "@/lib/query/use-api-query";
import { useApiMutation } from "@/lib/query/use-api-mutation";
import { fetchAndParse } from "@/lib/api/schema";
import {
  policyListSchema,
  evaluateResponseSchema,
  deletePolicyResponseSchema,
  policySchema,
  type Policy,
} from "@/lib/api/schemas/themis";
import { LoadingState } from "@/components/state/loading-state";
import { EmptyState } from "@/components/state/empty-state";
import { ErrorState } from "@/components/state/error-state";
import { NotConnectedState } from "@/components/state/not-connected-state";
import { NotConnectedEngine } from "@/components/resources/not-connected-engine";
import { useCapabilities } from "@/lib/capabilities/capability-context";

type PolicyItem = Policy;
type EvalResult = z.infer<typeof evaluateResponseSchema>;

export default function PoliciesPage() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const { isEngineConnected } = useCapabilities();

  const [searchQuery, setSearchQuery] = React.useState("");

  // Form State
  const [tenantId] = React.useState("default");
  const [name, setName] = React.useState("");
  const [description, setDescription] = React.useState("");
  const [expression, setExpression] = React.useState('request.auth.claims.department == "finance"');
  const [priority, setPriority] = React.useState("1");

  // Evaluator Studio State
  const [evalPayload, setEvalPayload] = React.useState(`{
  "request": {
    "auth": {
      "claims": {
        "department": "finance",
        "role": "admin",
        "sub": "user_123"
      }
    },
    "amount": 1500,
    "resource": {
      "type": "invoice",
      "owner_id": "user_123"
    }
  }
}`);
  const [evalResult, setEvalResult] = React.useState<EvalResult | null>(null);
  const [evalJsonError, setEvalJsonError] = React.useState("");

  const {
    data: policiesRaw,
    isLoading,
    isFetching,
    isError,
    error,
    refetch,
  } = useApiQuery(["policies", tenantId], () => fetchAndParse(`/api/policies?tenant_id=${tenantId}`, policyListSchema));

  const policies: PolicyItem[] = policiesRaw ?? [];

  const createPolicy = useApiMutation(
    (vars: { tenantId: string; name: string; description: string; expression: string; priority: number }) =>
      fetchAndParse("/api/policies", policySchema, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ ...vars, enabled: true }),
      }),
    {
      successMessage: (data) => t("themis.successMsg", { name: data.Name }),
      onSuccess: () => {
        queryClient.invalidateQueries({ queryKey: ["policies", tenantId] });
        setName("");
        setDescription("");
      },
      errorFix: "Fix the CEL expression syntax and try again.",
    },
  );

  const deletePolicy = useApiMutation(
    (policyId: string) =>
      fetchAndParse(`/api/policies/${policyId}?tenant_id=${tenantId}`, deletePolicyResponseSchema, {
        method: "DELETE",
      }),
    {
      successMessage: "Policy deleted from Themis.",
      onSuccess: () => {
        queryClient.invalidateQueries({ queryKey: ["policies", tenantId] });
      },
    },
  );

  const evaluatePolicies = useApiMutation(
    (vars: { tenantId: string; payload: unknown }) =>
      fetchAndParse("/api/policies/evaluate", evaluateResponseSchema, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(vars),
      }),
    {
      onSuccess: (data) => {
        setEvalResult(data);
      },
    },
  );

  const handleCreatePolicy = (e: React.FormEvent) => {
    e.preventDefault();
    if (!name || !expression) return;
    createPolicy.mutate({ tenantId, name, description, expression, priority: Number(priority) || 1 });
  };

  const handleDeletePolicy = (policyId: string) => {
    deletePolicy.mutate(policyId);
  };

  const handleEvaluate = () => {
    let parsedPayload: unknown = {};
    try {
      parsedPayload = JSON.parse(evalPayload);
      setEvalJsonError("");
    } catch {
      setEvalJsonError("Invalid JSON in evaluation payload");
      return;
    }
    evaluatePolicies.mutate({ tenantId, payload: parsedPayload });
  };

  const isSubmitting = createPolicy.isPending;
  const evaluating = evaluatePolicies.isPending;

  const setTemplate = (expr: string, templateName: string) => {
    setExpression(expr);
    if (!name) setName(templateName);
  };

  const filteredPolicies = policies.filter(
    (p) =>
      p.Name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      p.Expression.toLowerCase().includes(searchQuery.toLowerCase()) ||
      p.TenantID.toLowerCase().includes(searchQuery.toLowerCase()),
  );

  if (!isEngineConnected("themis")) {
    return (
      <div className="space-y-6">
        <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
          <div>
            <h1 className="text-xl font-bold tracking-tight text-foreground sm:text-2xl">{t("themis.title")}</h1>
            <p className="text-xs text-muted-foreground mt-1">{t("themis.subtitle")}</p>
          </div>
        </div>
        <NotConnectedEngine
          engineType="themis"
          engineName="Autorix Themis (ABAC / CEL)"
          description="Google Common Expression Language (CEL) policy evaluator."
        />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
        <div>
          <h1 className="text-xl font-bold tracking-tight text-foreground sm:text-2xl">{t("themis.title")}</h1>
          <p className="text-xs text-muted-foreground mt-1">{t("themis.subtitle")}</p>
        </div>

        <div className="flex items-center gap-2">
          <Badge variant="purple" className="gap-1.5 py-1 px-3">
            <Scale className="h-3.5 w-3.5" />
            <span>{t("themis.statusBadge")}</span>
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

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        {/* Create Policy Card */}
        <Card className="bg-card/80">
          <CardHeader className="p-6 pb-4">
            <div className="flex items-center gap-2">
              <Plus className="h-4 w-4 text-purple-400" />
              <CardTitle className="text-sm font-semibold">{t("themis.createTitle")}</CardTitle>
            </div>
            <CardDescription className="text-xs">{t("themis.createDesc")}</CardDescription>
          </CardHeader>

          <CardContent className="p-6 pt-0">
            <form onSubmit={handleCreatePolicy} className="space-y-3.5">
              <div className="grid grid-cols-3 gap-3">
                <div className="col-span-2 space-y-1.5">
                  <Label htmlFor="name">{t("themis.policyNameLabel")}</Label>
                  <Input
                    id="name"
                    placeholder={t("themis.policyNamePlaceholder")}
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                    required
                  />
                </div>

                <div className="space-y-1.5">
                  <Label htmlFor="priority">{t("themis.priorityLabel")}</Label>
                  <Input
                    id="priority"
                    type="number"
                    min="1"
                    max="100"
                    value={priority}
                    onChange={(e) => setPriority(e.target.value)}
                  />
                </div>
              </div>

              <div className="space-y-1.5">
                <div className="flex items-center justify-between">
                  <Label htmlFor="expression">{t("themis.expressionLabel")}</Label>
                  <span className="text-[10px] font-mono text-purple-400">Google CEL AST</span>
                </div>
                <Input
                  id="expression"
                  className="font-mono text-xs text-purple-300"
                  placeholder={t("themis.expressionPlaceholder")}
                  value={expression}
                  onChange={(e) => setExpression(e.target.value)}
                  required
                />
              </div>

              {/* Template Chips */}
              <div className="space-y-1.5">
                <span className="text-[11px] font-medium text-muted-foreground">Quick CEL Templates:</span>
                <div className="flex flex-wrap gap-1.5">
                  <button
                    type="button"
                    onClick={() => setTemplate('request.auth.claims.department == "finance"', "Finance Dept Only")}
                    className="rounded border border-border/80 bg-muted/40 px-2 py-0.5 text-[10px] font-mono text-muted-foreground hover:text-foreground hover:border-purple-500/50 transition-colors"
                  >
                    Department Match
                  </button>
                  <button
                    type="button"
                    onClick={() => setTemplate("resource.owner_id == auth.claims.sub", "Resource Owner Enforce")}
                    className="rounded border border-border/80 bg-muted/40 px-2 py-0.5 text-[10px] font-mono text-muted-foreground hover:text-foreground hover:border-purple-500/50 transition-colors"
                  >
                    Owner Check
                  </button>
                  <button
                    type="button"
                    onClick={() =>
                      setTemplate('request.amount < 10000 || auth.claims.role == "vp"', "High Value Approver")
                    }
                    className="rounded border border-border/80 bg-muted/40 px-2 py-0.5 text-[10px] font-mono text-muted-foreground hover:text-foreground hover:border-purple-500/50 transition-colors"
                  >
                    Threshold & Role
                  </button>
                </div>
              </div>

              <div className="space-y-1.5">
                <Label htmlFor="description">{t("themis.descriptionLabel")}</Label>
                <Input
                  id="description"
                  placeholder={t("themis.descriptionPlaceholder")}
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                />
              </div>

              <Button type="submit" variant="purple" disabled={isSubmitting} className="w-full gap-2 mt-2">
                {isSubmitting ? <Loader2 className="h-4 w-4 animate-spin" /> : <Sparkles className="h-4 w-4" />}
                <span>{isSubmitting ? t("common.loading") : t("themis.submitBtn")}</span>
              </Button>
            </form>
          </CardContent>
        </Card>

        {/* Live CEL Evaluation Studio Card */}
        <Card className="bg-card/80 flex flex-col justify-between">
          <CardHeader className="p-6 pb-4">
            <div className="flex items-center gap-2">
              <Code2 className="h-4 w-4 text-cyan-400" />
              <CardTitle className="text-sm font-semibold">{t("themis.evalTitle")}</CardTitle>
            </div>
            <CardDescription className="text-xs">{t("themis.evalDesc")}</CardDescription>
          </CardHeader>

          <CardContent className="p-6 pt-0 space-y-3 flex-1 flex flex-col justify-between">
            <div className="space-y-1.5">
              <Label>{t("themis.payloadLabel")}</Label>
              <Textarea
                rows={7}
                className="font-mono text-xs text-foreground bg-black/40 border-border/70"
                value={evalPayload}
                onChange={(e) => setEvalPayload(e.target.value)}
              />
            </div>

            <Button
              type="button"
              variant="outline"
              onClick={handleEvaluate}
              disabled={evaluating}
              className="w-full gap-2 border-cyan-500/30 hover:border-cyan-500/60 hover:bg-cyan-500/10 text-xs"
            >
              {evaluating ? (
                <Loader2 className="h-4 w-4 animate-spin text-cyan-400" />
              ) : (
                <Play className="h-4 w-4 text-cyan-400 fill-cyan-400" />
              )}
              <span>{t("themis.evalBtn")}</span>
            </Button>

            {evalJsonError && <p className="text-xs font-medium text-destructive">{evalJsonError}</p>}

            {evalResult && (
              <div className="rounded-lg border border-border/70 bg-black/50 p-3 space-y-2 animate-in fade-in-50">
                <div className="flex items-center justify-between">
                  <span className="text-[11px] font-semibold text-muted-foreground uppercase">Evaluation Verdict:</span>
                  <Badge variant={evalResult.AllPassed ? "success" : "destructive"} className="gap-1 text-[10px]">
                    {evalResult.AllPassed ? <CheckCircle2 className="h-3 w-3" /> : <XCircle className="h-3 w-3" />}
                    <span>{evalResult.AllPassed ? "ALL POLICIES PASSED (ALLOW)" : "POLICY DENIED"}</span>
                  </Badge>
                </div>

                <div className="space-y-1 text-xs font-mono">
                  {(evalResult.Results || []).map((r) => (
                    <div
                      key={r.PolicyID}
                      className="flex items-center justify-between rounded bg-muted/40 p-1.5 text-[11px]"
                    >
                      <span className="truncate max-w-[200px] text-foreground font-medium">{r.PolicyName}</span>
                      <Badge variant={r.Passed ? "success" : "destructive"} className="text-[9px]">
                        {r.Passed ? "PASSED" : r.Error ? "ERROR" : "FAILED"}
                      </Badge>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </CardContent>
        </Card>
      </div>

      {/* Policies Directory Table */}
      <Card className="bg-card/80">
        <CardHeader className="p-6 pb-4">
          <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <CardTitle className="text-sm font-semibold flex items-center gap-2">
                <Scale className="h-4 w-4 text-purple-400" />
                <span>{t("themis.tableTitle")}</span>
                <Badge variant="secondary" className="font-mono text-[10px]">
                  {policies.length}
                </Badge>
              </CardTitle>
              <CardDescription className="text-xs">{t("themis.tableDesc")}</CardDescription>
            </div>

            {/* Filter Search */}
            <div className="flex items-center gap-2 w-full sm:w-72">
              <div className="relative w-full">
                <Search className="absolute left-2.5 top-2.5 h-3.5 w-3.5 text-muted-foreground" />
                <Input
                  placeholder={t("themis.searchPlaceholder")}
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
                <TableHead>{t("themis.colName")}</TableHead>
                <TableHead>{t("themis.colExpression")}</TableHead>
                <TableHead>{t("themis.colTenant")}</TableHead>
                <TableHead>{t("themis.colPriority")}</TableHead>
                <TableHead>{t("themis.colState")}</TableHead>
                <TableHead className="text-right">{t("themis.colActions")}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {isLoading ? (
                <TableRow>
                  <TableCell colSpan={6} className="p-0">
                    <LoadingState label="Fetching policies from Themis database..." />
                  </TableCell>
                </TableRow>
              ) : isError ? (
                <TableRow>
                  <TableCell colSpan={6} className="p-0">
                    {error?.kind === "engine-unreachable" ? (
                      <NotConnectedState engineName="Themis" onRetry={refetch} />
                    ) : (
                      <ErrorState error={error} onRetry={refetch} />
                    )}
                  </TableCell>
                </TableRow>
              ) : filteredPolicies.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={6} className="p-0">
                    <EmptyState
                      title={t("themis.tableTitle")}
                      description="No CEL policies registered in Themis yet."
                    />
                  </TableCell>
                </TableRow>
              ) : (
                filteredPolicies.map((p) => (
                  <TableRow key={p.ID}>
                    <TableCell className="font-semibold text-foreground">{p.Name}</TableCell>
                    <TableCell className="font-mono text-xs text-purple-300">
                      <span className="rounded bg-purple-500/10 px-2 py-0.5 border border-purple-500/20">
                        {p.Expression}
                      </span>
                    </TableCell>
                    <TableCell className="font-mono text-xs text-muted-foreground">{p.TenantID}</TableCell>
                    <TableCell>
                      <Badge variant="outline" className="font-mono text-[10px]">
                        p:{p.Priority}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      <Badge variant={p.Enabled ? "success" : "secondary"} className="text-[10px]">
                        {p.Enabled ? t("common.active") : t("common.inactive")}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-right">
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => handleDeletePolicy(p.ID)}
                        className="h-7 w-7 p-0 text-muted-foreground hover:text-destructive hover:bg-destructive/10"
                      >
                        <Trash2 className="h-3.5 w-3.5" />
                      </Button>
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
