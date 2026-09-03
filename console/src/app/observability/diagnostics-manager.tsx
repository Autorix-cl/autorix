"use client";

import * as React from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import {
  Wrench,
  Radio,
  GitFork,
  Database,
  Calendar,
  Download,
  CheckCircle2,
  AlertTriangle,
  XCircle,
  Loader2,
  Play,
  Clock,
  ArrowRight,
} from "lucide-react";
import { toast } from "sonner";
import type {
  ConnectivityProbeResult,
  ConfigDriftFinding,
  MigrationStatus,
  ChangeTimelineEvent,
  DiagnosticBundle,
} from "@/lib/api/schemas/diagnostics";
import { fetchJSON } from "@/lib/api/client";

export function DiagnosticsManager() {
  const [subTab, setSubTab] = React.useState<"probe" | "drift" | "migrations" | "timeline">("probe");

  // Connectivity probe state
  const [sourceEngine, setSourceEngine] = React.useState("aegis");
  const [targetEngine, setTargetEngine] = React.useState("nexus");
  const [isProbing, setIsProbing] = React.useState(false);
  const [probeResult, setProbeResult] = React.useState<ConnectivityProbeResult | null>(null);

  // Drift & Migrations & Timeline state
  const [driftFindings, setDriftFindings] = React.useState<ConfigDriftFinding[]>([]);
  const [migrations, setMigrations] = React.useState<MigrationStatus[]>([]);
  const [timeline, setTimeline] = React.useState<ChangeTimelineEvent[]>([]);
  const [isExporting, setIsExporting] = React.useState(false);

  const runProbe = React.useCallback(async () => {
    setIsProbing(true);
    try {
      const res = await fetchJSON<ConnectivityProbeResult>("/api/diagnostics/probe", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ source_engine: sourceEngine, target_engine: targetEngine }),
      });
      if (!res.ok) throw new Error(res.error.message);
      setProbeResult(res.data);
    } catch {
      toast.error("Connectivity probe failed");
    } finally {
      setIsProbing(false);
    }
  }, [sourceEngine, targetEngine]);

  const fetchDiagnosticsData = React.useCallback(async () => {
    try {
      const [driftRes, migRes, timeRes] = await Promise.all([
        fetchJSON<ConfigDriftFinding[]>("/api/diagnostics/drift"),
        fetchJSON<MigrationStatus[]>("/api/diagnostics/migrations"),
        fetchJSON<ChangeTimelineEvent[]>("/api/diagnostics/timeline"),
      ]);
      if (driftRes.ok) setDriftFindings(driftRes.data);
      if (migRes.ok) setMigrations(migRes.data);
      if (timeRes.ok) setTimeline(timeRes.data);
    } catch {
      // Ignored non-critical errors
    }
  }, []);

  React.useEffect(() => {
    fetchDiagnosticsData();
  }, [fetchDiagnosticsData]);

  const handleExportBundle = async () => {
    setIsExporting(true);
    try {
      const res = await fetchJSON<DiagnosticBundle>("/api/diagnostics/bundle", { method: "POST" });
      if (!res.ok) throw new Error(res.error.message);
      toast.success(`Diagnostic bundle ${res.data.bundle_id} generated successfully!`);
    } catch {
      toast.error("Failed to generate diagnostic bundle");
    } finally {
      setIsExporting(false);
    }
  };

  const renderStageIcon = (status: string) => {
    switch (status) {
      case "healthy":
        return <CheckCircle2 className="h-4 w-4 text-emerald-500" />;
      case "degraded":
        return <AlertTriangle className="h-4 w-4 text-amber-500" />;
      case "failed":
        return <XCircle className="h-4 w-4 text-rose-500" />;
      default:
        return <span className="text-[10px] text-muted-foreground font-mono">SKIP</span>;
    }
  };

  return (
    <div className="space-y-4">
      {/* Top Header & Export Bundle */}
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-border pb-3">
        <div>
          <h2 className="text-sm font-semibold tracking-tight flex items-center gap-2">
            <Wrench className="h-4 w-4 text-sky-500" />
            Diagnostics &amp; Incident Support
          </h2>
          <p className="text-xs text-muted-foreground">
            Active connectivity probing, configuration drift detection, schema versioning, and change correlation
          </p>
        </div>

        <Button
          variant="outline"
          size="sm"
          onClick={handleExportBundle}
          disabled={isExporting}
          className="h-8 gap-1.5 text-xs border-sky-500/30 text-sky-400 hover:text-sky-300"
        >
          {isExporting ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Download className="h-3.5 w-3.5" />}
          Export Diagnostic Bundle
        </Button>
      </div>

      {/* Sub-Tabs Selector */}
      <div className="flex flex-wrap items-center gap-1.5">
        <Button
          variant={subTab === "probe" ? "default" : "outline"}
          size="sm"
          onClick={() => setSubTab("probe")}
          className="h-8 text-xs gap-1.5"
        >
          <Radio className="h-3.5 w-3.5" />
          Connectivity Troubleshooter
        </Button>
        <Button
          variant={subTab === "drift" ? "default" : "outline"}
          size="sm"
          onClick={() => setSubTab("drift")}
          className="h-8 text-xs gap-1.5"
        >
          <GitFork className="h-3.5 w-3.5" />
          Config Drift ({driftFindings.length})
        </Button>
        <Button
          variant={subTab === "migrations" ? "default" : "outline"}
          size="sm"
          onClick={() => setSubTab("migrations")}
          className="h-8 text-xs gap-1.5"
        >
          <Database className="h-3.5 w-3.5" />
          Migration Status
        </Button>
        <Button
          variant={subTab === "timeline" ? "default" : "outline"}
          size="sm"
          onClick={() => setSubTab("timeline")}
          className="h-8 text-xs gap-1.5"
        >
          <Calendar className="h-3.5 w-3.5" />
          Change Correlation Timeline
        </Button>
      </div>

      {/* 1. Connectivity Troubleshooter Sub-Tab */}
      {subTab === "probe" && (
        <div className="space-y-4">
          <Card className="border-border">
            <CardHeader className="p-4 pb-2">
              <CardTitle className="text-xs font-semibold">Instance-to-Instance Live Probe</CardTitle>
            </CardHeader>
            <CardContent className="p-4 pt-1 space-y-3">
              <div className="flex flex-wrap items-center gap-3">
                <div className="space-y-1">
                  <span className="text-[11px] text-muted-foreground">Source Engine</span>
                  <Select value={sourceEngine} onValueChange={setSourceEngine}>
                    <SelectTrigger className="w-[140px] h-8 text-xs uppercase font-mono">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="aegis">Aegis</SelectItem>
                      <SelectItem value="vulcan">Vulcan</SelectItem>
                      <SelectItem value="themis">Themis</SelectItem>
                      <SelectItem value="argus">Argus</SelectItem>
                    </SelectContent>
                  </Select>
                </div>

                <ArrowRight className="h-4 w-4 text-muted-foreground mt-4" />

                <div className="space-y-1">
                  <span className="text-[11px] text-muted-foreground">Target Engine</span>
                  <Select value={targetEngine} onValueChange={setTargetEngine}>
                    <SelectTrigger className="w-[140px] h-8 text-xs uppercase font-mono">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="nexus">Nexus</SelectItem>
                      <SelectItem value="themis">Themis</SelectItem>
                      <SelectItem value="ego">Ego</SelectItem>
                      <SelectItem value="janus">Janus</SelectItem>
                      <SelectItem value="vulcan">Vulcan</SelectItem>
                      <SelectItem value="hermes">Hermes</SelectItem>
                    </SelectContent>
                  </Select>
                </div>

                <Button
                  size="sm"
                  onClick={runProbe}
                  disabled={isProbing}
                  className="h-8 gap-1.5 text-xs mt-4"
                >
                  {isProbing ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Play className="h-3.5 w-3.5" />}
                  Execute Probe
                </Button>
              </div>

              {probeResult && (
                <div className="mt-4 pt-3 border-t border-border space-y-3">
                  <div className="flex items-center justify-between">
                    <span className="text-xs font-semibold">Probe Breakdown &amp; Network Health:</span>
                    <Badge
                      variant={probeResult.overall_status === "healthy" ? "default" : "destructive"}
                      className="text-[10px] uppercase font-mono"
                    >
                      {probeResult.overall_status}
                    </Badge>
                  </div>

                  <div className="grid grid-cols-2 md:grid-cols-4 gap-2">
                    <div className="p-2.5 rounded border border-border bg-muted/20 space-y-1">
                      <div className="flex items-center justify-between text-xs font-medium">
                        <span>1. DNS Lookup</span>
                        {renderStageIcon(probeResult.dns_status)}
                      </div>
                      <p className="text-[11px] font-mono text-muted-foreground">
                        {probeResult.dns_latency_ms.toFixed(1)} ms
                      </p>
                    </div>

                    <div className="p-2.5 rounded border border-border bg-muted/20 space-y-1">
                      <div className="flex items-center justify-between text-xs font-medium">
                        <span>2. TCP Connect</span>
                        {renderStageIcon(probeResult.tcp_status)}
                      </div>
                      <p className="text-[11px] font-mono text-muted-foreground">
                        {probeResult.tcp_latency_ms.toFixed(1)} ms
                      </p>
                    </div>

                    <div className="p-2.5 rounded border border-border bg-muted/20 space-y-1">
                      <div className="flex items-center justify-between text-xs font-medium">
                        <span>3. TLS Handshake</span>
                        {renderStageIcon(probeResult.tls_status)}
                      </div>
                      <p className="text-[11px] font-mono text-muted-foreground">
                        {probeResult.tls_status === "skipped" ? "Mesh mTLS / Clear" : `${probeResult.tls_latency_ms} ms`}
                      </p>
                    </div>

                    <div className="p-2.5 rounded border border-border bg-muted/20 space-y-1">
                      <div className="flex items-center justify-between text-xs font-medium">
                        <span>4. Application HTTP</span>
                        {renderStageIcon(probeResult.http_status)}
                      </div>
                      <p className="text-[11px] font-mono text-muted-foreground">
                        {probeResult.http_code} OK ({probeResult.http_latency_ms.toFixed(1)} ms)
                      </p>
                    </div>
                  </div>
                </div>
              )}
            </CardContent>
          </Card>
        </div>
      )}

      {/* 2. Configuration Drift Sub-Tab */}
      {subTab === "drift" && (
        <Card className="border-border">
          <CardHeader className="p-4 pb-2">
            <CardTitle className="text-xs font-semibold">Divergent Configuration Across Replicas</CardTitle>
          </CardHeader>
          <CardContent className="p-0">
            <div className="divide-y divide-border">
              {driftFindings.length === 0 ? (
                <div className="p-6 text-center text-xs text-muted-foreground">
                  No configuration drift detected across fleet instances.
                </div>
              ) : (
                driftFindings.map((finding) => (
                  <div key={finding.id} className="p-3.5 hover:bg-muted/20 space-y-2">
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-2">
                        <Badge variant="outline" className="text-[9px] uppercase px-1 py-0 font-mono">
                          {finding.engine_type}
                        </Badge>
                        <span className="font-mono text-xs font-semibold">{finding.parameter}</span>
                        <Badge
                          variant={finding.severity === "critical" ? "destructive" : "secondary"}
                          className="text-[9px] uppercase px-1 py-0"
                        >
                          {finding.severity}
                        </Badge>
                      </div>
                      <span className="text-[11px] text-muted-foreground">Env: {finding.environment}</span>
                    </div>
                    <div className="text-[11px] text-muted-foreground flex items-center gap-4">
                      <span>
                        Reference Value: <b className="font-mono text-foreground">{finding.reference_value}</b>
                      </span>
                      <span>
                        Divergent Replicas:{" "}
                        {finding.divergent_instances.map((d) => (
                          <span key={d.instance_id} className="font-mono text-amber-500 ml-1">
                            {d.instance_id} ({d.value})
                          </span>
                        ))}
                      </span>
                    </div>
                  </div>
                ))
              )}
            </div>
          </CardContent>
        </Card>
      )}

      {/* 3. Migration Status Sub-Tab */}
      {subTab === "migrations" && (
        <Card className="border-border">
          <CardHeader className="p-4 pb-2">
            <CardTitle className="text-xs font-semibold">Engine Schema Migration State (Postgres)</CardTitle>
          </CardHeader>
          <CardContent className="p-0">
            <table className="w-full text-xs text-left">
              <thead className="bg-muted/50 border-y border-border text-muted-foreground">
                <tr>
                  <th className="px-4 py-2 font-medium">Engine</th>
                  <th className="px-3 py-2 font-medium">Instance</th>
                  <th className="px-3 py-2 font-medium">Current Version</th>
                  <th className="px-3 py-2 font-medium">Expected Version</th>
                  <th className="px-3 py-2 font-medium">Status</th>
                  <th className="px-3 py-2 font-medium text-right">Last Migrated</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border">
                {migrations.map((mig) => (
                  <tr key={mig.engine_type} className="hover:bg-muted/20">
                    <td className="px-4 py-2.5 font-semibold uppercase tracking-wide">{mig.engine_type}</td>
                    <td className="px-3 py-2.5 font-mono text-muted-foreground">{mig.instance_id}</td>
                    <td className="px-3 py-2.5 font-mono">v{mig.current_schema_version}</td>
                    <td className="px-3 py-2.5 font-mono">v{mig.expected_schema_version}</td>
                    <td className="px-3 py-2.5">
                      <Badge variant="default" className="text-[10px] px-1.5 py-0 bg-emerald-600">
                        Up to date
                      </Badge>
                    </td>
                    <td className="px-3 py-2.5 text-right text-muted-foreground font-mono">
                      {new Date(mig.last_migrated_at).toLocaleDateString()}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </CardContent>
        </Card>
      )}

      {/* 4. Change Correlation Timeline Sub-Tab */}
      {subTab === "timeline" && (
        <Card className="border-border">
          <CardHeader className="p-4 pb-2">
            <CardTitle className="text-xs font-semibold">Change Correlation &amp; Incident Overlay</CardTitle>
          </CardHeader>
          <CardContent className="p-0">
            <div className="divide-y divide-border">
              {timeline.map((event) => (
                <div key={event.id} className="p-3.5 hover:bg-muted/20 space-y-1.5">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-2">
                      <Badge variant="outline" className="text-[9px] uppercase px-1 py-0 font-mono">
                        {event.engine_type}
                      </Badge>
                      <Badge variant="secondary" className="text-[9px] uppercase px-1 py-0">
                        {event.event_type.replace("_", " ")}
                      </Badge>
                      <span className="font-semibold text-xs">{event.title}</span>
                      {event.correlated_error_spike && (
                        <Badge variant="destructive" className="text-[9px] uppercase px-1.5 py-0">
                          Error Spike (+{(event.error_spike_rate! * 100).toFixed(1)}%)
                        </Badge>
                      )}
                    </div>
                    <span className="text-[11px] text-muted-foreground flex items-center gap-1 font-mono">
                      <Clock className="h-3 w-3" />
                      {new Date(event.timestamp).toLocaleTimeString()}
                    </span>
                  </div>
                  <p className="text-[11px] text-muted-foreground">{event.description}</p>
                  <div className="text-[10px] text-muted-foreground">Actor: {event.actor}</div>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
