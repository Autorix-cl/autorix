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
  Shield,
  Zap,
  Activity,
  Sparkles,
  Download,
  Copy,
  Check,
  Terminal,
  FileCode2,
} from "lucide-react";
import { ServiceHeader } from "@/components/layout/service-header";
import { CloudSection } from "@/components/layout/cloud-section";
import { useTranslation } from "@/lib/i18n";
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from "@/components/ui/card";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { CodeBlock } from "@/components/ui/code-block";
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
import { toast } from "sonner";
import { getColumns } from "./columns";
import { PolicyBuilderSheet } from "./policy-builder-sheet";

type EvalResult = z.infer<typeof evaluateResponseSchema>;

const SAMPLE_EVAL_PAYLOAD = JSON.stringify(
  {
    request: {
      auth: {
        claims: {
          department: "finance",
          role: "admin",
          sub: "user_123",
          mfa: true,
        },
      },
      amount: 1500,
      resource: {
        type: "invoice",
        owner_id: "user_123",
        tenant_id: "default",
      },
    },
  },
  null,
  2
);

const STANDARD_CEL_PATTERNS = [
  {
    title: "Role & Multi-Factor Auth (MFA)",
    expression: "request.auth.claims.role == 'admin' && request.auth.claims.mfa == true",
    description: "Requires administrator credentials and hardware MFA verification.",
    category: "Identity & Session",
  },
  {
    title: "Tenant & Resource Ownership",
    expression: "request.auth.claims.sub == resource.owner_id && request.auth.claims.tenant_id == resource.tenant_id",
    description: "Ensures tenant isolation and matching resource owner identity.",
    category: "Tenancy & RBAC",
  },
  {
    title: "Financial Approval Limit",
    expression: "request.amount < 10000 || (request.auth.claims.department == 'finance' && request.auth.claims.role == 'director')",
    description: "Enforces dollar threshold or delegated executive approval.",
    category: "Business Rules",
  },
  {
    title: "Temporal Access Window",
    expression: "timestamp(request.time) < timestamp('2026-12-31T23:59:59Z')",
    description: "Verifies time-bounded access against RFC 3339 timestamp.",
    category: "Time & Expiry",
  },
];

function CopySnippetButton({ text, label }: { text: string; label: string }) {
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

export default function PoliciesPage() {
  const { t } = useTranslation();
  const { isEngineConnected } = useCapabilities();

  const [tenantId] = React.useState("default");
  const [searchQuery, setSearchQuery] = React.useState("");

  // Evaluator Studio State
  const [evalPayload, setEvalPayload] = React.useState(SAMPLE_EVAL_PAYLOAD);
  const [evalResult, setEvalResult] = React.useState<EvalResult | null>(null);
  const [evalJsonError, setEvalJsonError] = React.useState("");

  // Pagination state
  const [pageSize, setPageSize] = React.useState(10);
  const [cursorHistory, setCursorHistory] = React.useState<string[]>([]);
  const [cursorIndex, setCursorIndex] = React.useState(0);
  const currentCursor = cursorHistory[cursorIndex] || "";

  // Reset pagination on search
  React.useEffect(() => {
    setCursorHistory([]);
    setCursorIndex(0);
  }, [searchQuery]);

  const handlePageSizeChange = (newSize: number) => {
    setPageSize(newSize);
    setCursorHistory([]);
    setCursorIndex(0);
  };

  const {
    data: paginatedPolicies,
    isLoading,
    isFetching,
    isError,
    error,
    refetch,
  } = useApiQuery(["policies", tenantId, currentCursor, searchQuery, pageSize], () => {
    const params = new URLSearchParams({ tenant_id: tenantId, limit: String(pageSize) });
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

  const handleDownloadManifest = () => {
    const manifestJson = JSON.stringify(policies, null, 2);
    const blob = new Blob([manifestJson], { type: "application/json;charset=utf-8" });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = "themis-policies-manifest.json";
    link.click();
    URL.revokeObjectURL(url);
    toast.success("Downloaded themis-policies-manifest.json");
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
      {/* Cloud Service Header & Telemetry HUD */}
      <ServiceHeader
        serviceName="Themis ABAC Engine"
        title={t("themis.title")}
        description={t("themis.subtitle")}
        icon={Scale}
        iconColor="text-purple-400"
        statusText={t("themis.statusBadge")}
        statusVariant="purple"
        metrics={[
          {
            label: "Registered Policies",
            value: policies.length,
            hint: "CEL Declarative rules",
            icon: Scale,
          },
          {
            label: "Default Verdict",
            value: "DENY",
            hint: "Zero-Trust Fail Closed",
            icon: Shield,
          },
          {
            label: "AST Evaluation",
            value: "CEL v0.12",
            hint: "Google Compiler",
            icon: Zap,
          },
          {
            label: "Engine Gateway",
            value: "Port 4437",
            hint: "In-Memory Evaluator",
            icon: Activity,
          },
        ]}
        actions={
          <>
            <PolicyBuilderSheet tenantId={tenantId} />
            <Button
              variant="outline"
              size="sm"
              onClick={() => refetch()}
              disabled={isFetching}
              className="h-8 gap-1.5 text-xs"
            >
              <RefreshCw className={`h-3.5 w-3.5 ${isFetching ? "animate-spin" : ""}`} />
              <span>{t("common.refresh")}</span>
            </Button>
          </>
        }
      />

      {/* Studio Navigation Tabs */}
      <Tabs defaultValue="policies" className="space-y-4">
        <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3 border-b border-border pb-3">
          <TabsList className="bg-muted/40 p-1">
            <TabsTrigger value="policies" className="gap-2 text-xs font-medium">
              <Scale className="h-3.5 w-3.5 text-purple-400" />
              <span>Policies Directory</span>
            </TabsTrigger>
            <TabsTrigger value="evaluator" className="gap-2 text-xs font-medium">
              <Play className="h-3.5 w-3.5 text-cyan-400" />
              <span>CEL Dry-Run Simulator</span>
            </TabsTrigger>
            <TabsTrigger value="manifest" className="gap-2 text-xs font-medium">
              <Code2 className="h-3.5 w-3.5 text-amber-400" />
              <span>CEL Reference & Manifest</span>
            </TabsTrigger>
          </TabsList>
        </div>

        {/* Tab 1: Policies Directory Table */}
        <TabsContent value="policies" className="space-y-4 focus-visible:outline-none">
          <CloudSection
            title="Declarative Security Policies"
            description="Registered zero-trust policies and custom CEL validation expressions"
            icon={Scale}
            badge={`${policies.length} Policies`}
            badgeVariant="purple"
          >
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
                    pageIndex={cursorIndex}
                    pageSize={pageSize}
                    defaultPageSize={pageSize}
                    onPageSizeChange={handlePageSizeChange}
                    onNextPage={handleNextPage}
                    onPreviousPage={handlePreviousPage}
                    canNextPage={paginatedPolicies?.has_more ?? false}
                    canPreviousPage={cursorIndex > 0}
                  />
                )}
              </CardContent>
            </Card>
          </CloudSection>
        </TabsContent>

        {/* Tab 2: Live CEL Evaluation Studio */}
        <TabsContent value="evaluator" className="space-y-4 focus-visible:outline-none">
          <CloudSection
            title="Evaluation & AST Compilation"
            description="Live Google CEL rule evaluator and dry-run execution environment"
            icon={Play}
            badge="Dry Run"
            badgeVariant="cyan"
          >
            <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
              {/* Left Column (col-span-7): Spacious Code Context Editor */}
              <div className="lg:col-span-7 space-y-4">
                <Card className="border-border bg-card/80">
                  <CardHeader className="p-5 pb-3">
                    <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-2">
                      <div className="space-y-0.5">
                        <CardTitle className="text-sm font-semibold flex items-center gap-2">
                          <Code2 className="h-4 w-4 text-cyan-400" />
                          <span>{t("themis.evalTitle")}</span>
                        </CardTitle>
                        <CardDescription className="text-xs">{t("themis.evalDesc")}</CardDescription>
                      </div>
                      <div className="flex items-center gap-2">
                        {evalPayload && (
                          <Button
                            type="button"
                            variant="ghost"
                            size="sm"
                            onClick={() => {
                              setEvalPayload("");
                              setEvalResult(null);
                              setEvalJsonError("");
                            }}
                            className="h-6 text-[11px] px-2 text-muted-foreground hover:text-foreground"
                          >
                            Clear
                          </Button>
                        )}
                        <Button
                          type="button"
                          variant="outline"
                          size="sm"
                          onClick={() => {
                            setEvalPayload(SAMPLE_EVAL_PAYLOAD);
                            setEvalJsonError("");
                          }}
                          className="h-6 text-[11px] px-2.5 gap-1.5"
                        >
                          <Sparkles className="w-3 h-3 text-cyan-400" />
                          <span>Load Sample Context</span>
                        </Button>
                      </div>
                    </div>
                  </CardHeader>

                  <CardContent className="p-5 pt-0 space-y-3">
                    <div className="rounded-lg border bg-muted/10 focus-within:border-cyan-500/50 transition-colors overflow-hidden">
                      <div className="flex items-center justify-between px-3 py-1.5 border-b bg-muted/20 text-xs">
                        <span className="font-mono text-[10px] text-muted-foreground uppercase tracking-wider font-semibold">
                          Evaluation Request Context (JSON)
                        </span>
                        <Badge variant="outline" className="text-[10px] font-normal px-1.5 py-0">
                          RFC 8259
                        </Badge>
                      </div>
                      <textarea
                        rows={11}
                        value={evalPayload}
                        onChange={(e) => {
                          setEvalPayload(e.target.value);
                          if (evalJsonError) setEvalJsonError("");
                        }}
                        placeholder='Enter evaluation context object, e.g. { "request": { "auth": { ... } } }'
                        className="w-full text-xs font-mono p-3.5 bg-transparent border-0 focus:outline-none resize-y min-h-[220px] leading-relaxed block"
                      />
                      <div className="flex items-center justify-between px-3.5 py-2.5 border-t bg-muted/20 text-xs">
                        <span className="text-[11px] text-muted-foreground">
                          {evalPayload.trim()
                            ? `${evalPayload.trim().split("\n").length} lines · ${evalPayload.trim().length} characters`
                            : "Enter JSON payload"}
                        </span>
                        <Button
                          type="button"
                          variant="outline"
                          onClick={handleEvaluate}
                          disabled={evaluating}
                          className="h-7 px-3.5 text-xs font-medium gap-1.5 border-cyan-500/30 hover:border-cyan-500/60 hover:bg-cyan-500/10 shadow-sm"
                        >
                          {evaluating ? (
                            <Loader2 className="h-3.5 w-3.5 animate-spin text-cyan-400" />
                          ) : (
                            <Play className="h-3.5 w-3.5 text-cyan-400 fill-cyan-400" />
                          )}
                          <span>{t("themis.evalBtn")}</span>
                        </Button>
                      </div>
                    </div>

                    {evalJsonError && <p className="text-xs font-medium text-destructive">{evalJsonError}</p>}
                  </CardContent>
                </Card>
              </div>

              {/* Right Column (col-span-5): Live Verdict & AST Execution Traces */}
              <div className="lg:col-span-5 space-y-4">
                <Card className="border-border bg-card/80 h-full flex flex-col justify-between">
                  <CardHeader className="p-5 pb-3">
                    <div className="flex items-center justify-between">
                      <div className="font-semibold text-sm flex items-center gap-2">
                        <Activity className="h-4 w-4 text-cyan-400" />
                        <span>Evaluation Outcome</span>
                      </div>
                      {evalResult && (
                        <Badge
                          variant={evalResult.AllPassed ? "success" : "destructive"}
                          className="gap-1 text-[10px]"
                        >
                          {evalResult.AllPassed ? (
                            <CheckCircle2 className="h-3 w-3" />
                          ) : (
                            <XCircle className="h-3 w-3" />
                          )}
                          <span>{evalResult.AllPassed ? "ALLOW" : "DENY"}</span>
                        </Badge>
                      )}
                    </div>
                    <CardDescription className="text-xs">
                      AST compilation decisions across registered tenant policies
                    </CardDescription>
                  </CardHeader>

                  <CardContent className="p-5 pt-0 flex-1 flex flex-col justify-between space-y-4">
                    {evalResult ? (
                      <div className="space-y-3">
                        <div
                          className={`p-3.5 rounded-lg border flex items-center justify-between text-xs ${
                            evalResult.AllPassed
                              ? "bg-emerald-500/10 border-emerald-500/30 text-emerald-400"
                              : "bg-destructive/10 border-destructive/30 text-destructive"
                          }`}
                        >
                          <div className="flex items-center gap-2 font-medium">
                            {evalResult.AllPassed ? (
                              <CheckCircle2 className="w-4 h-4 shrink-0" />
                            ) : (
                              <XCircle className="w-4 h-4 shrink-0" />
                            )}
                            <span>
                              {evalResult.AllPassed
                                ? "All Active Policies Passed (Verdict: ALLOW)"
                                : "Zero-Trust Policy Blocked (Verdict: DENY)"}
                            </span>
                          </div>
                        </div>

                        <div className="space-y-1.5">
                          <div className="flex items-center justify-between text-[11px] font-semibold text-muted-foreground uppercase tracking-wider pb-1">
                            <span>Evaluated Policies ({(evalResult.Results || []).length})</span>
                            <span>Result</span>
                          </div>

                          <div className="space-y-1.5 max-h-72 overflow-y-auto pr-1">
                            {(evalResult.Results || []).map((r) => (
                              <div
                                key={r.PolicyID}
                                className="flex items-center justify-between rounded-md border border-border/60 bg-muted/20 p-2 text-xs font-mono"
                              >
                                <span className="truncate max-w-[220px] text-foreground font-medium">
                                  {r.PolicyName}
                                </span>
                                <Badge
                                  variant={r.Passed ? "success" : "destructive"}
                                  className="text-[9px] font-sans font-normal py-0 px-1.5"
                                >
                                  {r.Passed ? "PASSED" : r.Error ? "ERROR" : "FAILED"}
                                </Badge>
                              </div>
                            ))}
                          </div>
                        </div>
                      </div>
                    ) : (
                      <div className="p-6 rounded-lg border border-dashed border-border/80 text-center space-y-2 my-auto">
                        <Code2 className="w-8 h-8 text-muted-foreground/50 mx-auto" />
                        <p className="text-xs font-medium text-foreground">No Evaluation Executed Yet</p>
                        <p className="text-[11px] text-muted-foreground max-w-xs mx-auto">
                          Click &ldquo;{t("themis.evalBtn")}&rdquo; to evaluate the JSON context against all active CEL rules.
                        </p>
                      </div>
                    )}

                    <div className="p-2.5 rounded border border-border/60 bg-muted/10 flex items-center justify-between text-[11px] text-muted-foreground">
                      <span>Evaluator: Google CEL AST</span>
                      <span className="font-mono">Fail Closed (Zero-Trust)</span>
                    </div>
                  </CardContent>
                </Card>
              </div>
            </div>
          </CloudSection>
        </TabsContent>

        {/* Tab 3: CEL Reference & Standard Manifest */}
        <TabsContent value="manifest" className="space-y-4 focus-visible:outline-none">
          <CloudSection
            title="CEL Expression Library & Declarative Manifest"
            description="Common CEL access patterns, syntactical reference, and exported policy definitions"
            icon={Code2}
            badge="CEL Reference"
            badgeVariant="purple"
          >
            {/* Quick-Copy CEL Expression Library */}
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              {STANDARD_CEL_PATTERNS.map((p, idx) => (
                <div key={idx} className="p-3.5 rounded-lg border border-border/80 bg-card/60 space-y-2">
                  <div className="flex items-center justify-between">
                    <span className="text-[10px] font-bold uppercase tracking-wider text-purple-400">
                      {p.category}
                    </span>
                    <CopySnippetButton text={p.expression} label={p.title} />
                  </div>
                  <div className="font-semibold text-xs text-foreground">{p.title}</div>
                  <div className="font-mono text-[11px] text-purple-300 p-2 rounded bg-purple-500/10 border border-purple-500/20 break-all select-all">
                    {p.expression}
                  </div>
                  <p className="text-[11px] text-muted-foreground">{p.description}</p>
                </div>
              ))}
            </div>

            {/* Live Active Policies Manifest */}
            <Card className="border-border">
              <CardHeader className="p-5 pb-3">
                <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
                  <div className="space-y-1">
                    <CardTitle className="text-sm font-semibold flex items-center gap-2">
                      <FileCode2 className="h-4 w-4 text-purple-400" />
                      <span>Active Policies Declarative Manifest</span>
                    </CardTitle>
                    <CardDescription className="text-xs">
                      Exported zero-trust rules currently loaded into Themis engine memory
                    </CardDescription>
                  </div>

                  <div className="flex items-center gap-2">
                    <Badge variant="outline" className="font-mono text-[10px] px-2 py-0.5">
                      {policies.length} {policies.length === 1 ? "Policy" : "Policies"}
                    </Badge>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={handleDownloadManifest}
                      className="h-7 text-xs gap-1.5"
                    >
                      <Download className="w-3.5 h-3.5" />
                      <span>Download Manifest</span>
                    </Button>
                  </div>
                </div>
              </CardHeader>

              <CardContent className="p-5 pt-0 space-y-4">
                <CodeBlock
                  code={JSON.stringify(policies, null, 2)}
                  language="json"
                  title="THEMIS_POLICIES_MANIFEST.JSON"
                  showLineNumbers={true}
                  maxHeight="max-h-[480px]"
                />
              </CardContent>
            </Card>

            {/* Automation & Evaluation API snippet */}
            <Card className="border-border">
              <CardHeader className="p-4 pb-2">
                <CardTitle className="text-xs font-semibold flex items-center gap-2">
                  <Terminal className="h-3.5 w-3.5 text-primary" />
                  <span>Programmatic Evaluation via REST (cURL)</span>
                </CardTitle>
                <CardDescription className="text-[11px]">
                  Evaluate requests against Themis engine from upstream microservices or gateways
                </CardDescription>
              </CardHeader>
              <CardContent className="p-4 pt-0">
                <CodeBlock
                  code='curl -X POST http://localhost:4437/policies/evaluate \
  -H "Content-Type: application/json" \
  -d "{\"tenant_id\":\"default\",\"payload\":{\"request\":{\"auth\":{\"claims\":{\"role\":\"admin\"}}}}}"'
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
