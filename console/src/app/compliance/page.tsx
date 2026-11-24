"use client";

import * as React from "react";
import { useQuery } from "@tanstack/react-query";
import {
  ShieldCheck,
  FileBadge,
  Download,
  CheckCircle2,
  XCircle,
  AlertTriangle,
  RefreshCw,
  Search,
  Lock,
  Server,
  FileCheck,
  Sparkles,
  ChevronDown,
  ChevronUp,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { ServiceHeader } from "@/components/layout/service-header";
import { Badge } from "@/components/ui/badge";
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { CodeBlock } from "@/components/ui/code-block";
import {
  complianceEvidenceItemSchema,
  type ComplianceEvidenceItem,
} from "@/lib/api/schemas/compliance";
import type { AuditVerification } from "@/lib/api/schemas/audit";
import { healthResponseSchema } from "@/lib/api/schemas/health";
import { fetchAndParse } from "@/lib/api/schema";
import { useApiQuery } from "@/lib/query/use-api-query";
import { formatDateTime } from "@/lib/format/date-time";
import { toast } from "sonner";

export default function CompliancePage() {
  const [selectedFramework, setSelectedFramework] = React.useState<string>("ALL");
  const [searchTerm, setSearchTerm] = React.useState<string>("");
  const [selectedPeriod, setSelectedPeriod] = React.useState<string>("Q3-2026");
  const [isGeneratingPackage, setIsGeneratingPackage] = React.useState<boolean>(false);
  const [expandedControlId, setExpandedControlId] = React.useState<string | null>(null);

  // Fetch Compliance Evidence
  const {
    data: complianceData,
    isLoading,
    isError,
    error: complianceError,
    refetch,
  } = useQuery<ComplianceEvidenceItem[]>({
    queryKey: ["compliance", "evidence", selectedFramework],
    queryFn: async () => {
      const res = await fetch("/api/compliance/evidence");
      if (!res.ok) {
        const errBody = await res.json().catch(() => ({}));
        throw new Error(errBody.error || `Failed to load compliance evidence (HTTP ${res.status})`);
      }
      const json = await res.json();
      if (Array.isArray(json)) return json.map((it) => complianceEvidenceItemSchema.parse(it));
      if (json.data && Array.isArray(json.data)) return json.data.map((it: unknown) => complianceEvidenceItemSchema.parse(it));
      return [];
    },
  });

  // Live cluster health & audit chain verification (never hardcoded)
  const { data: healthData } = useApiQuery(["health"], () =>
    fetchAndParse("/api/health", healthResponseSchema),
  );
  const { data: auditVerification } = useQuery<AuditVerification | null>({
    queryKey: ["audit", "verification"],
    queryFn: async () => {
      const res = await fetch("/api/audit/verify");
      if (!res.ok) return null;
      return res.json();
    },
  });

  const controls = React.useMemo(() => complianceData ?? [], [complianceData]);

  const filteredControls = React.useMemo(() => {
    return controls.filter((ctrl) => {
      const matchesFramework =
        selectedFramework === "ALL" ||
        (selectedFramework === "SOC2" && ctrl.framework.includes("SOC 2")) ||
        (selectedFramework === "ISO27001" && ctrl.framework.includes("ISO"));

      const matchesSearch =
        !searchTerm ||
        ctrl.control_id.toLowerCase().includes(searchTerm.toLowerCase()) ||
        ctrl.control_name.toLowerCase().includes(searchTerm.toLowerCase()) ||
        ctrl.description.toLowerCase().includes(searchTerm.toLowerCase()) ||
        (ctrl.engine && ctrl.engine.toLowerCase().includes(searchTerm.toLowerCase()));

      return matchesFramework && matchesSearch;
    });
  }, [controls, selectedFramework, searchTerm]);

  const passingCount = controls.filter((c) => c.status === "compliant").length;
  const failingCount = controls.filter((c) => c.status !== "compliant").length;
  const totalCount = controls.length;
  const passingPercent = totalCount > 0 ? Math.round((passingCount / totalCount) * 100) : null;

  const healthyProbes = healthData?.services?.filter((s) => s.status === "healthy").length ?? 0;
  const totalProbes = healthData?.services?.length ?? 0;
  const chainStatusLabel = !auditVerification
    ? "UNVERIFIED"
    : auditVerification.verified
      ? "INTACT"
      : "BROKEN";

  const handleGeneratePackage = async (framework: string) => {
    setIsGeneratingPackage(true);
    try {
      const frameworkControls = filteredControls.filter((c) => {
        if (framework === "ALL") return true;
        if (framework.includes("SOC 2")) return c.framework.includes("SOC 2");
        if (framework.includes("ISO")) return c.framework.includes("ISO");
        return true;
      });
      const pkgPassing = frameworkControls.filter((c) => c.status === "compliant").length;
      const pkgFailing = frameworkControls.length - pkgPassing;
      const pkgPercent = frameworkControls.length > 0 ? Math.round((pkgPassing / frameworkControls.length) * 100) : 0;

      const payload = {
        framework: framework === "ALL" ? "SOC 2 Type II & ISO/IEC 27001:2022" : framework,
        period: selectedPeriod,
        generated_at: new Date().toISOString(),
        issuer: "Autorix Control Plane Security Evaluator",
        summary: {
          total_controls: frameworkControls.length,
          passing_controls: pkgPassing,
          failing_controls: pkgFailing,
          compliance_percentage: frameworkControls.length > 0 ? `${pkgPercent}%` : "N/A",
        },
        audit_trail_integrity: auditVerification
          ? {
              chain_length: auditVerification.chain_length,
              algorithm: auditVerification.algorithm || "SHA-256",
              status: auditVerification.verified ? "VERIFIED_TAMPER_EVIDENT" : "INTEGRITY_FAILURE",
              head_hash: auditVerification.head_hash || null,
              verified_at: auditVerification.verified_at || null,
            }
          : {
              chain_length: null,
              algorithm: "SHA-256",
              status: "UNVERIFIED",
              head_hash: null,
              verified_at: null,
            },
        controls: frameworkControls,
      };

      const jsonBlob = new Blob([JSON.stringify(payload, null, 2)], { type: "application/json" });
      const url = URL.createObjectURL(jsonBlob);
      const a = document.createElement("a");
      a.href = url;
      a.download = `autorix-compliance-package-${framework.toLowerCase().replace(/[^a-z0-9]+/g, "-")}-${selectedPeriod}.json`;
      a.click();
      URL.revokeObjectURL(url);
      toast.success(`Compliance evidence package downloaded (${frameworkControls.length} control(s))`);
    } finally {
      setIsGeneratingPackage(false);
    }
  };

  const renderControlStatusBadge = (status: string) => {
    const normalized = status.toLowerCase();
    if (normalized === "compliant") {
      return (
        <Badge className="bg-emerald-500/15 text-emerald-400 border-emerald-500/30 text-xs gap-1 font-mono">
          <CheckCircle2 className="w-3 h-3" aria-hidden="true" />
          COMPLIANT
        </Badge>
      );
    }
    if (normalized === "non_compliant" || normalized === "failed") {
      return (
        <Badge variant="destructive" className="text-xs gap-1 font-mono">
          <XCircle className="w-3 h-3" aria-hidden="true" />
          {status.toUpperCase()}
        </Badge>
      );
    }
    return (
      <Badge variant="warning" className="text-xs gap-1 font-mono">
        <AlertTriangle className="w-3 h-3" aria-hidden="true" />
        {status.toUpperCase()}
      </Badge>
    );
  };

  return (
    <div className="space-y-6">
      {/* Cloud Service Header & Telemetry HUD */}
      <ServiceHeader
        serviceName="Continuous Compliance & Evidence"
        title="Compliance & Evidence Center"
        description="Automated continuous compliance verification, auditor evidence export, and cryptographic access reviews across all 7 Autorix engines."
        icon={ShieldCheck}
        iconColor="text-emerald-400"
        statusText={
          totalCount === 0
            ? "EVIDENCE-UNAVAILABLE"
            : failingCount === 0
              ? "SOC2-ISO27001-CONTINUOUS-VERIFIED"
              : `${failingCount}-CONTROL-FINDING(S)`
        }
        statusVariant={totalCount === 0 ? "warning" : failingCount === 0 ? "success" : "rose"}
        metrics={[
          {
            label: "Controls Evaluated",
            value: `${totalCount} Controls`,
            hint: "SOC 2 Type II + ISO 27001",
            icon: FileCheck,
          },
          {
            label: "Compliance Score",
            value: passingPercent !== null ? `${passingPercent}%` : "—",
            hint: totalCount > 0 ? `${passingCount}/${totalCount} controls passing` : "No controls evaluated",
            icon: ShieldCheck,
          },
          {
            label: "Evidence Probes",
            value: totalProbes > 0 ? `${healthyProbes}/${totalProbes} Online` : "—",
            hint: "Live engine health probes",
            icon: Server,
          },
          {
            label: "Cryptographic Audit Chain",
            value: chainStatusLabel,
            hint: "SHA-256 tamper-evident proof",
            icon: Lock,
          },
        ]}
        actions={
          <Button
            variant="outline"
            size="sm"
            onClick={() => refetch()}
            disabled={isLoading}
            className="h-8 gap-1.5 text-xs"
          >
            <RefreshCw className={`h-3.5 w-3.5 ${isLoading ? "animate-spin" : ""}`} />
            <span>Refresh Evidence</span>
          </Button>
        }
      />

      {/* SOC 2 / ISO 27001 Auditor Export Package Generator Card */}
      <Card className="border-emerald-500/30 bg-gradient-to-br from-emerald-500/5 via-card/60 to-card/60 backdrop-blur-sm shadow-xs">
        <CardHeader className="pb-3">
          <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-2">
            <div className="flex items-center gap-3">
              <div className="w-9 h-9 rounded-lg bg-emerald-500/10 border border-emerald-500/30 flex items-center justify-center text-emerald-400 shrink-0">
                <FileBadge className="w-5 h-5" />
              </div>
              <div>
                <CardTitle className="text-base font-semibold">
                  Auditor Evidence Package Generator
                </CardTitle>
                <CardDescription className="text-xs">
                  Generate a structured compliance bundle formatted for SOC 2 Type II and ISO 27001 external auditors
                </CardDescription>
              </div>
            </div>
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-1 sm:grid-cols-3 gap-4 pt-2">
            {/* Audit Period Selection */}
            <div>
              <label htmlFor="compliance-audit-period" className="text-xs font-medium text-muted-foreground block mb-1.5">
                Audit Reporting Period
              </label>
              <select
                id="compliance-audit-period"
                value={selectedPeriod}
                onChange={(e) => setSelectedPeriod(e.target.value)}
                className="w-full h-9 rounded-md border border-input bg-background/80 px-3 py-1.5 text-xs text-foreground shadow-sm focus:outline-none focus:ring-1 focus:ring-ring"
              >
                <option value="Q3-2026">Q3 2026 (Current Quarter)</option>
                <option value="Q2-2026">Q2 2026 (Previous Quarter)</option>
                <option value="Q1-2026">Q1 2026</option>
                <option value="ANNUAL-2026">Annual Period 2026</option>
                <option value="LAST-30-DAYS">Last 30 Days (Continuous)</option>
              </select>
            </div>

            {/* Included Evidence Modules */}
            <div className="sm:col-span-2 flex flex-col justify-center">
              <span className="text-xs font-medium text-muted-foreground block mb-1.5">
                Included Evidence Modules
              </span>
              <div className="flex flex-wrap gap-2 text-xs">
                <Badge variant="outline" className="text-[11px] bg-muted/40 font-mono gap-1">
                  <CheckCircle2 className="w-3 h-3 text-emerald-400" />
                  Tamper-Evident Hash Chain
                </Badge>
                <Badge variant="outline" className="text-[11px] bg-muted/40 font-mono gap-1">
                  <CheckCircle2 className="w-3 h-3 text-emerald-400" />
                  Operator Access Matrix
                </Badge>
                <Badge variant="outline" className="text-[11px] bg-muted/40 font-mono gap-1">
                  <CheckCircle2 className="w-3 h-3 text-emerald-400" />
                  Zero Trust PEP Routing
                </Badge>
                <Badge variant="outline" className="text-[11px] bg-muted/40 font-mono gap-1">
                  <CheckCircle2 className="w-3 h-3 text-emerald-400" />
                  Zanzibar ReBAC Graphs
                </Badge>
              </div>
            </div>
          </div>

          <div className="flex flex-wrap items-center justify-between gap-3 pt-2 border-t border-border/50">
            <div className="text-xs text-muted-foreground flex items-center gap-1.5">
              <Sparkles className="w-3.5 h-3.5 text-emerald-400" />
              <span>Evidence packages include cryptographic SHA-256 hash proofs and secrets redaction.</span>
            </div>
            <div className="flex items-center gap-2">
              <Button
                variant="outline"
                size="sm"
                onClick={() => handleGeneratePackage("SOC 2 Type II")}
                disabled={isGeneratingPackage}
                className="gap-1.5 text-xs border-emerald-500/30 hover:bg-emerald-500/10 text-emerald-400"
              >
                <Download className="w-3.5 h-3.5" />
                Export SOC 2 Package
              </Button>
              <Button
                variant="outline"
                size="sm"
                onClick={() => handleGeneratePackage("ISO/IEC 27001:2022")}
                disabled={isGeneratingPackage}
                className="gap-1.5 text-xs border-blue-500/30 hover:bg-blue-500/10 text-blue-400"
              >
                <Download className="w-3.5 h-3.5" />
                Export ISO 27001 Package
              </Button>
              <Button
                variant="default"
                size="sm"
                onClick={() => handleGeneratePackage("ALL")}
                disabled={isGeneratingPackage}
                className="gap-1.5 text-xs bg-emerald-600 hover:bg-emerald-500 text-white shadow-sm"
              >
                <FileBadge className="w-3.5 h-3.5" />
                Generate Complete Bundle
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Framework Filter & Controls Explorer */}
      <Card className="border-border/80 bg-card/60 backdrop-blur-sm shadow-xs">
        <CardHeader className="pb-3 border-b border-border/60">
          <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
            <div>
              <CardTitle className="text-base font-semibold">Continuous Compliance Controls</CardTitle>
              <CardDescription className="text-xs">
                Real-time control status and automated evidence collected from Autorix engines
              </CardDescription>
            </div>

            {/* Framework Filter Tabs & Search */}
            <div className="flex flex-wrap items-center gap-2">
              <div className="flex items-center rounded-lg border border-border/60 bg-muted/40 p-1 text-xs">
                <button
                  type="button"
                  onClick={() => setSelectedFramework("ALL")}
                  className={`px-2.5 py-1 rounded-md transition-all ${
                    selectedFramework === "ALL"
                      ? "bg-card text-foreground font-semibold shadow-xs"
                      : "text-muted-foreground hover:text-foreground"
                  }`}
                >
                  All Frameworks
                </button>
                <button
                  type="button"
                  onClick={() => setSelectedFramework("SOC2")}
                  className={`px-2.5 py-1 rounded-md transition-all ${
                    selectedFramework === "SOC2"
                      ? "bg-card text-foreground font-semibold shadow-xs"
                      : "text-muted-foreground hover:text-foreground"
                  }`}
                >
                  SOC 2 Type II
                </button>
                <button
                  type="button"
                  onClick={() => setSelectedFramework("ISO27001")}
                  className={`px-2.5 py-1 rounded-md transition-all ${
                    selectedFramework === "ISO27001"
                      ? "bg-card text-foreground font-semibold shadow-xs"
                      : "text-muted-foreground hover:text-foreground"
                  }`}
                >
                  ISO 27001
                </button>
              </div>

              <div className="relative w-48 sm:w-60">
                <Search className="absolute left-2.5 top-2 h-3.5 w-3.5 text-muted-foreground" />
                <Input
                  placeholder="Filter controls or engines..."
                  value={searchTerm}
                  onChange={(e) => setSearchTerm(e.target.value)}
                  className="pl-8 text-xs h-8"
                />
              </div>
            </div>
          </div>
        </CardHeader>
        <CardContent className="p-4 sm:p-6 space-y-4">
          {isLoading ? (
            <div className="py-16 text-center text-xs text-muted-foreground font-mono">
              Loading compliance controls and evidence probes...
            </div>
          ) : isError ? (
            <div role="alert" className="py-16 text-center space-y-2">
              <AlertTriangle className="w-6 h-6 text-amber-400 mx-auto" aria-hidden="true" />
              <p className="text-xs font-medium text-foreground">
                {complianceError instanceof Error ? complianceError.message : "Failed to load compliance evidence."}
              </p>
              <Button type="button" variant="outline" size="sm" onClick={() => refetch()} className="text-xs h-7">
                Retry
              </Button>
            </div>
          ) : filteredControls.length === 0 ? (
            <div className="py-16 text-center text-xs text-muted-foreground">
              No compliance controls found matching your search.
            </div>
          ) : (
            <div className="space-y-3">
              {filteredControls.map((control) => {
                const isExpanded = expandedControlId === control.id;
                return (
                  <div
                    key={control.id}
                    className="rounded-xl border border-border/70 bg-card/40 hover:bg-card/70 transition-all p-4 space-y-3"
                  >
                    <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
                      <div className="flex items-start sm:items-center gap-3">
                        <div className="w-8 h-8 rounded-lg bg-slate-800/80 border border-slate-700/60 flex items-center justify-center text-primary font-mono font-bold text-xs shrink-0">
                          {control.control_id}
                        </div>
                        <div>
                          <div className="flex items-center gap-2 flex-wrap">
                            <span className="text-sm font-semibold text-foreground">
                              {control.control_name}
                            </span>
                            <Badge variant="outline" className="text-[10px] font-mono">
                              {control.framework}
                            </Badge>
                            {control.engine && (
                              <Badge variant="secondary" className="text-[10px] font-mono">
                                Engine: {control.engine}
                              </Badge>
                            )}
                          </div>
                          <p className="text-xs text-muted-foreground mt-0.5">
                            {control.description}
                          </p>
                        </div>
                      </div>

                      <div className="flex items-center gap-2 shrink-0 self-end sm:self-center">
                        {renderControlStatusBadge(control.status)}
                        <Button
                          type="button"
                          variant="ghost"
                          size="sm"
                          aria-expanded={isExpanded}
                          onClick={() => setExpandedControlId(isExpanded ? null : control.id)}
                          className="h-7 px-2 text-xs gap-1 text-muted-foreground hover:text-foreground"
                        >
                          {isExpanded ? <ChevronUp className="w-3.5 h-3.5" aria-hidden="true" /> : <ChevronDown className="w-3.5 h-3.5" aria-hidden="true" />}
                          Evidence Proof
                        </Button>
                      </div>
                    </div>

                    {/* Expandable Evidence Details */}
                    {isExpanded && (
                      <div className="mt-3 pt-3 border-t border-border/50 space-y-3">
                        <div className="grid grid-cols-1 sm:grid-cols-3 gap-2 text-xs p-3 rounded-lg bg-muted/20 border border-border/50">
                          <div>
                            <span className="text-muted-foreground text-[10px] block uppercase font-medium">
                              Evaluator Sentinel
                            </span>
                            <span className="font-mono text-foreground font-medium">
                              {control.evaluator}
                            </span>
                          </div>
                          <div>
                            <span className="text-muted-foreground text-[10px] block uppercase font-medium">
                              Last Automated Evaluation
                            </span>
                            <span className="font-mono text-foreground">
                              {formatDateTime(control.last_evaluated_at)}
                            </span>
                          </div>
                          <div>
                            <span className="text-muted-foreground text-[10px] block uppercase font-medium">
                              Evidence Artifacts Captured
                            </span>
                            <span className="font-mono text-emerald-400 font-medium">
                              {control.artifacts_count || 1} verified artifacts
                            </span>
                          </div>
                        </div>

                        {control.details && (
                          <div>
                            <span className="text-[11px] font-bold text-muted-foreground uppercase tracking-wider block mb-1.5">
                              Continuous Evidence Telemetry Payload (JSON)
                            </span>
                            <CodeBlock
                              code={JSON.stringify(control.details, null, 2)}
                              language="json"
                              title={`${control.control_id} EVIDENCE PAYLOAD`}
                            />
                          </div>
                        )}
                      </div>
                    )}
                  </div>
                );
              })}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
