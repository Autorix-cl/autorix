"use client";

import * as React from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Network,
  Clock,
  CheckCircle2,
  XCircle,
  RefreshCw,
  Loader2,
  Layers,
} from "lucide-react";
import { toast } from "sonner";
import type { TraceDetail, TraceSpan } from "@/lib/api/schemas/observability";

interface TraceExplorerProps {
  initialTraceId?: string;
}

export function TraceExplorer({ initialTraceId }: TraceExplorerProps) {
  const [traces, setTraces] = React.useState<TraceDetail[]>([]);
  const [selectedTrace, setSelectedTrace] = React.useState<TraceDetail | null>(null);
  const [selectedSpan, setSelectedSpan] = React.useState<TraceSpan | null>(null);
  const [isLoading, setIsLoading] = React.useState(false);

  const fetchTraces = React.useCallback(async () => {
    setIsLoading(true);
    try {
      const res = await fetch("/api/observability/traces");
      if (!res.ok) throw new Error("Failed to load traces");
      const data = await res.json();
      setTraces(data);
      if (data.length > 0) {
        const found = initialTraceId ? data.find((t: TraceDetail) => t.trace_id === initialTraceId) : data[0];
        setSelectedTrace(found || data[0]);
      }
    } catch {
      toast.error("Error fetching distributed traces");
    } finally {
      setIsLoading(false);
    }
  }, [initialTraceId]);

  React.useEffect(() => {
    fetchTraces();
  }, [fetchTraces]);

  const maxDuration = selectedTrace ? Math.max(selectedTrace.total_duration_ms, 0.1) : 1;

  return (
    <div className="space-y-4">
      {/* Top Header */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-sm font-semibold tracking-tight">Distributed Trace Waterfall (OpenTelemetry)</h2>
          <p className="text-xs text-muted-foreground">
            Correlated cross-engine execution spans across Aegis, Nexus, Themis and backends
          </p>
        </div>
        <Button
          variant="outline"
          size="sm"
          onClick={() => fetchTraces()}
          disabled={isLoading}
          className="h-8 gap-1 text-xs"
        >
          {isLoading ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <RefreshCw className="h-3.5 w-3.5" />}
          Refresh
        </Button>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
        {/* Trace Selection List */}
        <Card className="border-border lg:col-span-1">
          <CardHeader className="p-3 border-b border-border">
            <CardTitle className="text-xs font-semibold flex items-center justify-between">
              <span>Recent Traces ({traces.length})</span>
            </CardTitle>
          </CardHeader>
          <CardContent className="p-0">
            <div className="divide-y divide-border">
              {traces.map((tr) => {
                const isSelected = selectedTrace?.trace_id === tr.trace_id;
                return (
                  <div
                    key={tr.trace_id}
                    onClick={() => {
                      setSelectedTrace(tr);
                      setSelectedSpan(null);
                    }}
                    className={`p-3 cursor-pointer transition-colors ${
                      isSelected ? "bg-muted" : "hover:bg-muted/40"
                    }`}
                  >
                    <div className="flex items-center justify-between text-xs mb-1">
                      <span className="font-mono font-medium">{tr.trace_id}</span>
                      <Badge
                        variant={tr.status === "ok" ? "outline" : "destructive"}
                        className="text-[9px] uppercase px-1 py-0"
                      >
                        {tr.status}
                      </Badge>
                    </div>
                    <div className="text-[11px] text-foreground font-medium truncate">{tr.root_operation}</div>
                    <div className="flex items-center justify-between text-[10px] text-muted-foreground mt-1.5">
                      <span>{tr.spans.length} spans</span>
                      <span className="font-mono">{tr.total_duration_ms.toFixed(1)} ms</span>
                    </div>
                  </div>
                );
              })}
            </div>
          </CardContent>
        </Card>

        {/* Trace Waterfall Panel */}
        <Card className="border-border lg:col-span-2">
          <CardHeader className="p-3 border-b border-border">
            <CardTitle className="text-xs font-semibold flex items-center justify-between">
              <span className="flex items-center gap-1.5 font-mono">
                <Network className="h-4 w-4 text-sky-500" />
                {selectedTrace?.trace_id || "No trace selected"}
              </span>
              <span className="text-xs font-mono text-muted-foreground flex items-center gap-1">
                <Clock className="h-3 w-3" />
                Total: {selectedTrace?.total_duration_ms.toFixed(1)} ms
              </span>
            </CardTitle>
          </CardHeader>
          <CardContent className="p-4 space-y-4">
            {selectedTrace ? (
              <div className="space-y-2">
                {selectedTrace.spans.map((span) => {
                  const isSelected = selectedSpan?.span_id === span.span_id;
                  const leftPercent = (span.start_time_ms / maxDuration) * 100;
                  const widthPercent = Math.max((span.duration_ms / maxDuration) * 100, 2);

                  return (
                    <div
                      key={span.span_id}
                      onClick={() => setSelectedSpan(span)}
                      className={`p-2 rounded border transition-all cursor-pointer ${
                        isSelected
                          ? "border-sky-500 bg-sky-500/10"
                          : "border-border/60 hover:border-border hover:bg-muted/20"
                      }`}
                    >
                      <div className="flex items-center justify-between text-xs mb-1.5">
                        <div className="flex items-center gap-2">
                          <Badge variant="outline" className="text-[9px] uppercase px-1 py-0 font-mono">
                            {span.engine}
                          </Badge>
                          <span className="font-medium text-[11px]">{span.operation}</span>
                        </div>
                        <div className="flex items-center gap-1.5 font-mono text-[11px]">
                          {span.status === "ok" ? (
                            <CheckCircle2 className="h-3 w-3 text-emerald-500" />
                          ) : (
                            <XCircle className="h-3 w-3 text-rose-500" />
                          )}
                          <span>{span.duration_ms.toFixed(1)} ms</span>
                        </div>
                      </div>

                      {/* Waterfall Timeline Bar */}
                      <div className="h-2 w-full bg-muted rounded-full relative overflow-hidden">
                        <div
                          className={`h-full rounded-full ${
                            span.status === "ok" ? "bg-sky-500" : "bg-rose-500"
                          }`}
                          style={{
                            marginLeft: `${leftPercent}%`,
                            width: `${widthPercent}%`,
                          }}
                        />
                      </div>
                    </div>
                  );
                })}

                {/* Selected Span Attributes Detail */}
                {selectedSpan && (
                  <div className="mt-4 p-3 rounded-lg border border-border bg-muted/20 space-y-2 text-xs">
                    <div className="flex items-center justify-between font-semibold">
                      <span className="flex items-center gap-1.5">
                        <Layers className="h-3.5 w-3.5 text-sky-500" />
                        Span Details: {selectedSpan.operation}
                      </span>
                      <span className="font-mono text-[10px] text-muted-foreground">{selectedSpan.span_id}</span>
                    </div>
                    <pre className="p-2 rounded bg-muted/40 font-mono text-[10px] overflow-x-auto">
                      {JSON.stringify(selectedSpan.attributes, null, 2)}
                    </pre>
                  </div>
                )}
              </div>
            ) : (
              <div className="py-8 text-center text-xs text-muted-foreground">Select a trace to view spans</div>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
