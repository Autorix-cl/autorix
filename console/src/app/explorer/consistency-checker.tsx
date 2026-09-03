"use client";

import * as React from "react";
import Link from "next/link";
import { Card, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  ShieldAlert,
  AlertTriangle,
  Info,
  CheckCircle2,
  RefreshCw,
  ExternalLink,
  Loader2,
} from "lucide-react";
import { toast } from "sonner";
import type { ConsistencyFinding } from "@/lib/api/schemas/explorer";

export function ConsistencyChecker() {
  const [findings, setFindings] = React.useState<ConsistencyFinding[]>([]);
  const [isLoading, setIsLoading] = React.useState(false);

  const runDiagnostics = React.useCallback(async () => {
    setIsLoading(true);
    try {
      const res = await fetch("/api/explorer/consistency");
      if (!res.ok) throw new Error("Diagnostic run failed");
      const data = await res.json();
      setFindings(Array.isArray(data) ? data : []);
      toast.success("Cross-engine diagnostic sweep completed");
    } catch {
      toast.error("Failed to execute consistency checks");
    } finally {
      setIsLoading(false);
    }
  }, []);

  React.useEffect(() => {
    runDiagnostics();
  }, [runDiagnostics]);

  const criticalCount = findings.filter((f) => f.severity === "critical").length;
  const warningCount = findings.filter((f) => f.severity === "warning").length;
  const infoCount = findings.filter((f) => f.severity === "info").length;

  return (
    <div className="space-y-4">
      {/* Header Card */}
      <Card className="border-border">
        <CardHeader className="pb-3">
          <div className="flex items-center justify-between">
            <div className="space-y-1">
              <CardTitle className="text-xs font-semibold flex items-center gap-1.5">
                <ShieldAlert className="w-3.5 h-3.5 text-primary" />
                Cross-Engine Architecture Consistency Diagnostics
              </CardTitle>
              <CardDescription className="text-[11px]">
                Detects orphaned tuples, missing ReBAC namespace bindings, expired SAML certificates, and stale API keys.
              </CardDescription>
            </div>

            <div className="flex items-center gap-2">
              <div className="flex gap-1.5">
                {criticalCount > 0 && (
                  <Badge variant="destructive" className="text-[10px] px-2 py-0.5">
                    {criticalCount} Critical
                  </Badge>
                )}
                {warningCount > 0 && (
                  <Badge variant="outline" className="text-[10px] text-amber-500 border-amber-500/30 px-2 py-0.5">
                    {warningCount} Warnings
                  </Badge>
                )}
                <Badge variant="secondary" className="text-[10px] px-2 py-0.5">
                  {infoCount} Info
                </Badge>
              </div>
              <Button
                variant="outline"
                size="sm"
                onClick={runDiagnostics}
                disabled={isLoading}
                className="h-8 text-xs gap-1.5"
              >
                {isLoading ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : <RefreshCw className="w-3.5 h-3.5" />}
                Re-Scan
              </Button>
            </div>
          </div>
        </CardHeader>
      </Card>

      {/* Findings List */}
      {isLoading ? (
        <div className="py-12 text-center text-muted-foreground flex items-center justify-center gap-2">
          <Loader2 className="w-4 h-4 animate-spin text-primary" /> Scanning engines for architectural anomalies...
        </div>
      ) : findings.length === 0 ? (
        <div className="py-12 text-center text-muted-foreground border rounded-lg border-dashed flex flex-col items-center gap-2">
          <CheckCircle2 className="w-6 h-6 text-emerald-500" />
          <span className="font-semibold text-foreground text-xs">All Engines Harmonized</span>
          <span className="text-[11px]">No configuration drift or cross-engine discrepancies found.</span>
        </div>
      ) : (
        <div className="space-y-2.5">
          {findings.map((f) => {
            const isCritical = f.severity === "critical";
            const isWarning = f.severity === "warning";
            return (
              <div
                key={f.id}
                className={`p-3.5 rounded-lg border flex items-start justify-between text-xs transition-colors ${
                  isCritical
                    ? "bg-destructive/10 border-destructive/30"
                    : isWarning
                    ? "bg-amber-500/10 border-amber-500/30"
                    : "bg-card/60 border-border"
                }`}
              >
                <div className="flex items-start gap-3">
                  <div className="mt-0.5">
                    {isCritical ? (
                      <ShieldAlert className="w-4 h-4 text-destructive" />
                    ) : isWarning ? (
                      <AlertTriangle className="w-4 h-4 text-amber-500" />
                    ) : (
                      <Info className="w-4 h-4 text-blue-400" />
                    )}
                  </div>

                  <div className="space-y-1">
                    <div className="flex items-center gap-2">
                      <span className="font-bold text-foreground">{f.title}</span>
                      <Badge variant="outline" className="text-[9px] uppercase font-mono">
                        {f.category}
                      </Badge>
                    </div>

                    <p className="text-muted-foreground text-[11px] leading-relaxed max-w-2xl">
                      {f.description}
                    </p>

                    <div className="pt-1 flex items-center gap-1.5 text-[11px] text-foreground font-medium">
                      <span className="text-muted-foreground">Remediation:</span> {f.remediation}
                    </div>
                  </div>
                </div>

                {f.remediation_link && (
                  <Button
                    variant="outline"
                    size="sm"
                    asChild
                    className="h-7 text-[11px] gap-1 px-2.5 shrink-0 bg-background/80"
                  >
                    <Link href={f.remediation_link}>
                      Remediate <ExternalLink className="w-3 h-3" />
                    </Link>
                  </Button>
                )}
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
