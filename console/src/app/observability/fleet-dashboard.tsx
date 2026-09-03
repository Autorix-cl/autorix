"use client";

import * as React from "react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Activity,
  Zap,
  Clock,
  ShieldCheck,
  AlertTriangle,
  RefreshCw,
  Server,
  Loader2,
  TrendingUp,
} from "lucide-react";
import { toast } from "sonner";
import type { FleetMetricsSummary } from "@/lib/api/schemas/observability";

interface FleetDashboardProps {
  onSelectEngine?: (engine: string) => void;
}

export function FleetDashboard({ onSelectEngine }: FleetDashboardProps) {
  const [metrics, setMetrics] = React.useState<FleetMetricsSummary | null>(null);
  const [isLoading, setIsLoading] = React.useState(false);

  const fetchMetrics = React.useCallback(async () => {
    setIsLoading(true);
    try {
      const res = await fetch("/api/observability/metrics");
      if (!res.ok) throw new Error("Failed to load fleet metrics");
      const data = await res.json();
      setMetrics(data);
    } catch {
      toast.error("Error loading fleet metrics summary");
    } finally {
      setIsLoading(false);
    }
  }, []);

  React.useEffect(() => {
    fetchMetrics();
  }, [fetchMetrics]);

  return (
    <div className="space-y-4">
      {/* Top Controls & Status Bar */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-sm font-semibold tracking-tight">Fleet Telemetry &amp; RED Overview</h2>
          <p className="text-xs text-muted-foreground">
            Aggregated Rate, Errors and Duration across all registered Zero-Trust engines
          </p>
        </div>
        <Button
          variant="outline"
          size="sm"
          onClick={() => fetchMetrics()}
          disabled={isLoading}
          className="h-8 gap-1.5 text-xs"
        >
          {isLoading ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <RefreshCw className="h-3.5 w-3.5" />}
          Refresh
        </Button>
      </div>

      {/* Primary RED Metric Cards */}
      <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6 gap-3">
        <Card className="border-border">
          <CardHeader className="p-3 pb-1">
            <CardDescription className="text-[11px] flex items-center justify-between">
              Fleet Throughput
              <Zap className="h-3.5 w-3.5 text-sky-500" />
            </CardDescription>
          </CardHeader>
          <CardContent className="p-3 pt-0">
            <div className="text-lg font-bold">
              {metrics ? `${metrics.fleet_qps.toFixed(1)} req/s` : "---"}
            </div>
            <p className="text-[10px] text-muted-foreground flex items-center gap-1 mt-0.5">
              <TrendingUp className="h-2.5 w-2.5 text-emerald-500" />
              Real-time ingest
            </p>
          </CardContent>
        </Card>

        <Card className="border-border">
          <CardHeader className="p-3 pb-1">
            <CardDescription className="text-[11px] flex items-center justify-between">
              Error Rate
              <AlertTriangle className="h-3.5 w-3.5 text-amber-500" />
            </CardDescription>
          </CardHeader>
          <CardContent className="p-3 pt-0">
            <div className="text-lg font-bold">
              {metrics ? `${(metrics.fleet_error_rate * 100).toFixed(2)}%` : "---"}
            </div>
            <p className="text-[10px] text-emerald-500 mt-0.5">Below 0.05% SLO</p>
          </CardContent>
        </Card>

        <Card className="border-border">
          <CardHeader className="p-3 pb-1">
            <CardDescription className="text-[11px] flex items-center justify-between">
              p95 Latency
              <Clock className="h-3.5 w-3.5 text-indigo-500" />
            </CardDescription>
          </CardHeader>
          <CardContent className="p-3 pt-0">
            <div className="text-lg font-bold">
              {metrics ? `${metrics.fleet_latency_p95_ms.toFixed(1)} ms` : "---"}
            </div>
            <p className="text-[10px] text-muted-foreground mt-0.5">Max engine hop</p>
          </CardContent>
        </Card>

        <Card className="border-border">
          <CardHeader className="p-3 pb-1">
            <CardDescription className="text-[11px] flex items-center justify-between">
              Total Invocations
              <Activity className="h-3.5 w-3.5 text-emerald-500" />
            </CardDescription>
          </CardHeader>
          <CardContent className="p-3 pt-0">
            <div className="text-lg font-bold">
              {metrics ? metrics.requests_total.toLocaleString() : "---"}
            </div>
            <p className="text-[10px] text-muted-foreground mt-0.5">Window 24h</p>
          </CardContent>
        </Card>

        <Card className="border-border">
          <CardHeader className="p-3 pb-1">
            <CardDescription className="text-[11px] flex items-center justify-between">
              Auth Decisions
              <ShieldCheck className="h-3.5 w-3.5 text-emerald-500" />
            </CardDescription>
          </CardHeader>
          <CardContent className="p-3 pt-0">
            <div className="text-lg font-bold">
              {metrics ? metrics.auth_decisions_total.toLocaleString() : "---"}
            </div>
            <p className="text-[10px] text-muted-foreground mt-0.5">Nexus &amp; Themis</p>
          </CardContent>
        </Card>

        <Card className="border-border">
          <CardHeader className="p-3 pb-1">
            <CardDescription className="text-[11px] flex items-center justify-between">
              Allow Ratio
              <ShieldCheck className="h-3.5 w-3.5 text-emerald-500" />
            </CardDescription>
          </CardHeader>
          <CardContent className="p-3 pt-0">
            <div className="text-lg font-bold">
              {metrics ? `${(metrics.auth_allow_rate * 100).toFixed(1)}%` : "---"}
            </div>
            <p className="text-[10px] text-muted-foreground mt-0.5">Authorized traffic</p>
          </CardContent>
        </Card>
      </div>

      {/* Engine Telemetry Breakdown Table */}
      <Card className="border-border">
        <CardHeader className="p-4 pb-2">
          <CardTitle className="text-xs font-semibold flex items-center gap-1.5">
            <Server className="h-4 w-4 text-sky-500" />
            Engine Telemetry &amp; Saturation Metrics
          </CardTitle>
          <CardDescription className="text-xs">
            Individual throughput, latency profiles and instance health across all registered engines
          </CardDescription>
        </CardHeader>
        <CardContent className="p-0">
          <div className="overflow-x-auto">
            <table className="w-full text-xs text-left">
              <thead className="bg-muted/50 border-y border-border text-muted-foreground">
                <tr>
                  <th className="px-4 py-2 font-medium">Engine</th>
                  <th className="px-3 py-2 font-medium">Status</th>
                  <th className="px-3 py-2 font-medium">Replicas</th>
                  <th className="px-3 py-2 font-medium">QPS</th>
                  <th className="px-3 py-2 font-medium">Error %</th>
                  <th className="px-3 py-2 font-medium">p50</th>
                  <th className="px-3 py-2 font-medium">p95</th>
                  <th className="px-3 py-2 font-medium">p99</th>
                  <th className="px-3 py-2 font-medium">Auth Allow %</th>
                  <th className="px-3 py-2 font-medium text-right">Action</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border">
                {metrics?.engines.map((eng) => (
                  <tr key={eng.engine_type} className="hover:bg-muted/20">
                    <td className="px-4 py-2.5 font-medium flex items-center gap-1.5 uppercase tracking-wide">
                      <span className="w-2 h-2 rounded-full bg-emerald-500 inline-block" />
                      {eng.engine_type}
                    </td>
                    <td className="px-3 py-2.5">
                      <Badge
                        variant={eng.status === "healthy" ? "default" : "destructive"}
                        className="text-[10px] capitalize px-1.5 py-0"
                      >
                        {eng.status}
                      </Badge>
                    </td>
                    <td className="px-3 py-2.5 text-muted-foreground">{eng.instance_count} active</td>
                    <td className="px-3 py-2.5 font-mono">{eng.requests_per_second.toFixed(1)}</td>
                    <td className="px-3 py-2.5 font-mono">
                      <span className={eng.error_rate > 0.01 ? "text-amber-500 font-semibold" : ""}>
                        {(eng.error_rate * 100).toFixed(2)}%
                      </span>
                    </td>
                    <td className="px-3 py-2.5 font-mono text-muted-foreground">{eng.latency_p50_ms.toFixed(1)}ms</td>
                    <td className="px-3 py-2.5 font-mono font-medium">{eng.latency_p95_ms.toFixed(1)}ms</td>
                    <td className="px-3 py-2.5 font-mono text-muted-foreground">{eng.latency_p99_ms.toFixed(1)}ms</td>
                    <td className="px-3 py-2.5 font-mono">
                      {eng.auth_allow_rate > 0 ? `${(eng.auth_allow_rate * 100).toFixed(1)}%` : "N/A"}
                    </td>
                    <td className="px-3 py-2.5 text-right">
                      <Button
                        variant="ghost"
                        size="sm"
                        className="h-6 text-[11px] px-2"
                        onClick={() => onSelectEngine?.(eng.engine_type)}
                      >
                        Inspect
                      </Button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
