"use client";

import * as React from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  RefreshCw,
  Loader2,
  AlertTriangle,
  CheckCircle2,
  TrendingDown,
} from "lucide-react";
import { toast } from "sonner";
import type { SLODefinition } from "@/lib/api/schemas/observability";

export function SLODashboard() {
  const [slos, setSlos] = React.useState<SLODefinition[]>([]);
  const [isLoading, setIsLoading] = React.useState(false);

  const fetchSLOs = React.useCallback(async () => {
    setIsLoading(true);
    try {
      const res = await fetch("/api/observability/slo");
      if (!res.ok) throw new Error("Failed to load SLOs");
      const data = await res.json();
      setSlos(data);
    } catch {
      toast.error("Error loading SLO error budgets");
    } finally {
      setIsLoading(false);
    }
  }, []);

  React.useEffect(() => {
    fetchSLOs();
  }, [fetchSLOs]);

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-sm font-semibold tracking-tight">Service Level Objectives (SLOs) &amp; Error Budgets</h2>
          <p className="text-xs text-muted-foreground">
            Multi-window error-budget tracking and burn-rate alerting across critical Zero-Trust engines
          </p>
        </div>
        <Button
          variant="outline"
          size="sm"
          onClick={() => fetchSLOs()}
          disabled={isLoading}
          className="h-8 gap-1 text-xs"
        >
          {isLoading ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <RefreshCw className="h-3.5 w-3.5" />}
          Refresh
        </Button>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
        {slos.map((slo) => {
          const isAtRisk = slo.error_budget_remaining_percent < 20 || slo.burn_rate >= 1.0;
          return (
            <Card key={slo.id} className="border-border">
              <CardHeader className="p-3.5 pb-2">
                <div className="flex items-center justify-between mb-1">
                  <Badge variant="outline" className="text-[9px] uppercase px-1 py-0 font-mono">
                    {slo.engine_type}
                  </Badge>
                  <span className="text-[10px] text-muted-foreground">{slo.window} rolling</span>
                </div>
                <CardTitle className="text-xs font-semibold">{slo.name}</CardTitle>
              </CardHeader>
              <CardContent className="p-3.5 pt-1 space-y-3">
                <div className="flex items-baseline justify-between text-xs">
                  <span className="text-muted-foreground">Achieved:</span>
                  <span className="font-mono font-bold text-sm">
                    {slo.current_percentage.toFixed(2)}%
                    <span className="text-[10px] font-normal text-muted-foreground ml-1">
                      (target: {slo.target_percentage}%)
                    </span>
                  </span>
                </div>

                {/* Error Budget Progress Bar */}
                <div className="space-y-1">
                  <div className="flex justify-between text-[10px]">
                    <span className="text-muted-foreground">Error Budget Remaining:</span>
                    <span className={`font-mono font-semibold ${isAtRisk ? "text-amber-500" : "text-emerald-500"}`}>
                      {slo.error_budget_remaining_percent.toFixed(1)}%
                    </span>
                  </div>
                  <div className="h-2 w-full bg-muted rounded-full overflow-hidden">
                    <div
                      className={`h-full rounded-full transition-all ${
                        slo.error_budget_remaining_percent > 50
                          ? "bg-emerald-500"
                          : slo.error_budget_remaining_percent > 20
                          ? "bg-amber-500"
                          : "bg-rose-500"
                      }`}
                      style={{ width: `${Math.min(slo.error_budget_remaining_percent, 100)}%` }}
                    />
                  </div>
                </div>

                {/* Burn Rate Indicator */}
                <div className="pt-2 border-t border-border/50 flex items-center justify-between text-[11px]">
                  <span className="text-muted-foreground flex items-center gap-1">
                    <TrendingDown className="h-3 w-3" />
                    Burn Rate:
                  </span>
                  <div className="flex items-center gap-1 font-mono">
                    <span>{slo.burn_rate.toFixed(2)}x</span>
                    {slo.burn_rate < 1.0 ? (
                      <CheckCircle2 className="h-3 w-3 text-emerald-500" />
                    ) : (
                      <AlertTriangle className="h-3 w-3 text-amber-500" />
                    )}
                  </div>
                </div>
              </CardContent>
            </Card>
          );
        })}
      </div>
    </div>
  );
}
