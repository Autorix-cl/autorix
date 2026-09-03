"use client";

import * as React from "react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import {
  Zap,
  Play,
  CheckCircle2,
  XCircle,
  Loader2,
  Clock,
  Shield,
  Layers,
  Network,
  Scale,
  Server,
} from "lucide-react";
import { toast } from "sonner";
import type { RequestSimulationTrace } from "@/lib/api/schemas/explorer";

const ENGINE_ICONS: Record<string, React.ElementType> = {
  aegis: Shield,
  ego: Layers,
  janus: Layers,
  vulcan: Layers,
  nexus: Network,
  themis: Scale,
  upstream: Server,
};

export function RequestSimulator() {
  const [method, setMethod] = React.useState("GET");
  const [path, setPath] = React.useState("/api/v1/projects/proj_alpha/deployments");
  const [authHeader, setAuthHeader] = React.useState("Bearer av_live_890abcd");
  const [subject, setSubject] = React.useState("user:usr_operator");
  const [isLoading, setIsLoading] = React.useState(false);
  const [trace, setTrace] = React.useState<RequestSimulationTrace | null>(null);

  const handleSimulate = React.useCallback(async (e?: React.FormEvent) => {
    e?.preventDefault();
    setIsLoading(true);
    try {
      const res = await fetch("/api/explorer/simulate", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          method,
          path,
          headers: authHeader ? { authorization: authHeader } : {},
          subject,
        }),
      });
      if (!res.ok) throw new Error("Simulation failed");
      const data = await res.json();
      setTrace(data);
    } catch {
      toast.error("Failed to run request simulation");
    } finally {
      setIsLoading(false);
    }
  }, [method, path, authHeader, subject]);

  React.useEffect(() => {
    handleSimulate();
  }, [handleSimulate]);


  return (
    <div className="space-y-4">
      {/* Simulation Request Builder Card */}
      <Card className="border-border">
        <CardHeader className="pb-3">
          <CardTitle className="text-xs font-semibold flex items-center gap-1.5">
            <Zap className="w-3.5 h-3.5 text-primary" />
            End-to-End Request Pipeline Simulator
          </CardTitle>
          <CardDescription className="text-[11px]">
            Traces an ingress request entering Aegis through credential resolution, ReBAC check, and ABAC guardrails.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSimulate} className="grid grid-cols-1 md:grid-cols-4 gap-3 text-xs items-end">
            <div className="flex gap-2">
              <div className="space-y-1 w-24">
                <Label className="text-[11px]">Method</Label>
                <Select value={method} onValueChange={setMethod}>
                  <SelectTrigger className="h-8 text-xs font-mono">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="GET">GET</SelectItem>
                    <SelectItem value="POST">POST</SelectItem>
                    <SelectItem value="PUT">PUT</SelectItem>
                    <SelectItem value="DELETE">DELETE</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-1 flex-1">
                <Label className="text-[11px]">Path</Label>
                <Input
                  value={path}
                  onChange={(e) => setPath(e.target.value)}
                  className="h-8 text-xs font-mono"
                />
              </div>
            </div>

            <div className="space-y-1">
              <Label className="text-[11px]">Authorization Header</Label>
              <Input
                value={authHeader}
                onChange={(e) => setAuthHeader(e.target.value)}
                placeholder="Bearer token or cookie"
                className="h-8 text-xs font-mono"
              />
            </div>

            <div className="space-y-1">
              <Label className="text-[11px]">Subject Principal</Label>
              <Input
                value={subject}
                onChange={(e) => setSubject(e.target.value)}
                placeholder="user:usr_operator"
                className="h-8 text-xs font-mono"
              />
            </div>

            <div>
              <Button type="submit" size="sm" disabled={isLoading} className="h-8 text-xs gap-1.5 w-full">
                {isLoading ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : <Play className="w-3.5 h-3.5" />}
                Simulate Request
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>

      {/* Trace Timeline View */}
      {trace && (
        <div className="space-y-4">
          {/* Header Summary */}
          <div className="flex items-center justify-between p-3 rounded-lg border bg-card/60 text-xs">
            <div className="flex items-center gap-3">
              <Badge variant="outline" className="font-mono text-[10px]">
                TRACE: {trace.trace_id}
              </Badge>
              <span className="font-mono font-semibold text-foreground">
                {trace.request.method} {trace.request.path}
              </span>
            </div>

            <div className="flex items-center gap-3">
              <span className="text-muted-foreground text-[11px] flex items-center gap-1">
                <Clock className="w-3.5 h-3.5 text-primary" /> {trace.outcome.latency_total_ms}ms total
              </span>
              <Badge
                variant={trace.outcome.allowed ? "default" : "destructive"}
                className={`text-[10px] px-2 py-0.5 ${
                  trace.outcome.allowed ? "bg-emerald-500" : ""
                }`}
              >
                {trace.outcome.final_decision}
              </Badge>
            </div>
          </div>

          {/* Stepper Pipeline */}
          <div className="space-y-2">
            {trace.steps.map((step, idx) => {
              const Icon = ENGINE_ICONS[step.engine] || Zap;
              return (
                <div
                  key={idx}
                  className="p-3 rounded-lg border bg-background/80 flex items-start justify-between text-xs transition-all hover:border-primary/50"
                >
                  <div className="flex items-start gap-3">
                    <div className="w-7 h-7 rounded-full bg-muted flex items-center justify-center shrink-0 mt-0.5">
                      <Icon className="w-3.5 h-3.5 text-primary" />
                    </div>

                    <div className="space-y-1">
                      <div className="flex items-center gap-2">
                        <span className="font-bold text-foreground">{step.step}</span>
                        <Badge variant="outline" className="text-[9px] uppercase font-mono">
                          {step.engine}
                        </Badge>
                      </div>

                      <div className="p-2 rounded bg-muted/30 font-mono text-[11px] space-y-0.5">
                        {Object.entries(step.details).map(([k, v]) => (
                          <div key={k} className="text-muted-foreground truncate max-w-xl">
                            <span className="text-primary font-semibold">{k}:</span>{" "}
                            <span>{typeof v === "object" ? JSON.stringify(v) : String(v)}</span>
                          </div>
                        ))}
                      </div>
                    </div>
                  </div>

                  <div className="flex items-center gap-2 shrink-0">
                    <span className="text-[10px] text-muted-foreground font-mono">{step.latency_ms}ms</span>
                    {step.status === "pass" ? (
                      <CheckCircle2 className="w-4 h-4 text-emerald-500" />
                    ) : (
                      <XCircle className="w-4 h-4 text-destructive" />
                    )}
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
}
