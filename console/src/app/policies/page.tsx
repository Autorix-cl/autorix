"use client";

import * as React from "react";
import {
  Scale,
  RefreshCw,
  Search,
  Code2,
  Play,
  Loader2,
  CheckCircle2,
  XCircle,
} from "lucide-react";
import { useTranslation } from "@/lib/i18n";
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Textarea } from "@/components/ui/textarea";
import type { z } from "zod";
import { useApiQuery } from "@/lib/query/use-api-query";
import { useApiMutation } from "@/lib/query/use-api-mutation";
import { fetchAndParse } from "@/lib/api/schema";
import {
  paginatedPolicyListSchema,
  evaluateResponseSchema,
  type Policy,
} from "@/lib/api/schemas/themis";
import { ErrorState } from "@/components/state/error-state";
import { NotConnectedState } from "@/components/state/not-connected-state";
import { NotConnectedEngine } from "@/components/resources/not-connected-engine";
import { useCapabilities } from "@/lib/capabilities/capability-context";
import { DataTable } from "@/components/ui/data-table";
import { getColumns } from "./columns";
import { PolicyBuilderSheet } from "./policy-builder-sheet";

type EvalResult = z.infer<typeof evaluateResponseSchema>;

export default function PoliciesPage() {
  const { t } = useTranslation();
  const { isEngineConnected } = useCapabilities();

  const [tenantId] = React.useState("default");
  const [searchQuery, setSearchQuery] = React.useState("");

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

  // Pagination state
  const limit = 10;
  const [cursorHistory, setCursorHistory] = React.useState<string[]>([]);
  const [cursorIndex, setCursorIndex] = React.useState(0);
  const currentCursor = cursorHistory[cursorIndex] || "";

  // Reset pagination on search
  React.useEffect(() => {
    setCursorHistory([]);
    setCursorIndex(0);
  }, [searchQuery]);

  const {
    data: paginatedPolicies,
    isLoading,
    isFetching,
    isError,
    error,
    refetch,
  } = useApiQuery(["policies", tenantId, currentCursor, searchQuery], () => {
    const params = new URLSearchParams({ tenant_id: tenantId, limit: String(limit) });
    if (currentCursor) params.append("cursor", currentCursor);
    if (searchQuery) params.append("search", searchQuery);
    return fetchAndParse(`/api/policies?${params.toString()}`, paginatedPolicyListSchema);
  });

  const policies: Policy[] = paginatedPolicies?.data ?? [];

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

  const evaluating = evaluatePolicies.isPending;

  const handleNextPage = () => {
    if (paginatedPolicies?.has_more && paginatedPolicies.next_cursor) {
      setCursorHistory((prev) => {
        const next = [...prev];
        next[cursorIndex + 1] = paginatedPolicies.next_cursor;
        return next;
      });
      setCursorIndex((prev) => prev + 1);
    }
  };

  const handlePreviousPage = () => {
    setCursorIndex((prev) => Math.max(0, prev - 1));
  };

  const columns = React.useMemo(() => getColumns(tenantId), [tenantId]);

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
          <PolicyBuilderSheet tenantId={tenantId} />
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

      {/* Policies Directory Table */}
      <Card className="bg-card/80">
        <CardHeader className="p-6 pb-4">
          <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <CardTitle className="text-sm font-semibold flex items-center gap-2">
                <Scale className="h-4 w-4 text-purple-400" />
                <span>{t("themis.tableTitle")}</span>
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
          {isError ? (
            error?.kind === "engine-unreachable" ? (
              <NotConnectedState engineName="Themis" onRetry={refetch} />
            ) : (
              <ErrorState error={error} onRetry={refetch} />
            )
          ) : (
            <DataTable
              columns={columns}
              data={policies}
              isLoading={isLoading}
              manualPagination={true}
              onNextPage={handleNextPage}
              onPreviousPage={handlePreviousPage}
              canNextPage={paginatedPolicies?.has_more ?? false}
              canPreviousPage={cursorIndex > 0}
            />
          )}
        </CardContent>
      </Card>
    </div>
  );
}
