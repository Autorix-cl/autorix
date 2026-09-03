"use client";

import * as React from "react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import {
  ShieldCheck,
  Network,
  Scale,
  Shield,
  Play,
  CheckCircle2,
  XCircle,
  Loader2,
} from "lucide-react";
import { toast } from "sonner";
import type { EffectiveAccessResult } from "@/lib/api/schemas/explorer";

import { fetchJSON } from "@/lib/api/client";

export function EffectiveAccessExplorer() {
  const [subject, setSubject] = React.useState("user:alice");
  const [resource, setResource] = React.useState("/api/v1/finance/invoices");
  const [action, setAction] = React.useState("GET");
  const [isLoading, setIsLoading] = React.useState(false);
  const [result, setResult] = React.useState<EffectiveAccessResult | null>(null);

  const handleEvaluate = React.useCallback(async (e?: React.FormEvent) => {
    e?.preventDefault();
    setIsLoading(true);
    try {
      const res = await fetchJSON<EffectiveAccessResult>("/api/explorer/effective-access", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ subject, resource, action }),
      });
      if (!res.ok) throw new Error(res.error.message);
      setResult(res.data);
    } catch {
      toast.error("Failed to evaluate effective access");
    } finally {
      setIsLoading(false);
    }
  }, [subject, resource, action]);

  React.useEffect(() => {
    handleEvaluate();
  }, [handleEvaluate]);


  return (
    <div className="space-y-4">
      {/* Parameter Input Form */}
      <Card className="border-border">
        <CardHeader className="pb-3">
          <CardTitle className="text-xs font-semibold flex items-center gap-1.5">
            <ShieldCheck className="w-3.5 h-3.5 text-primary" />
            Zero-Trust Effective Access Query
          </CardTitle>
          <CardDescription className="text-[11px]">
            Synthesizes Aegis proxy routing, Nexus ReBAC relation graph, and Themis ABAC policy constraints.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleEvaluate} className="grid grid-cols-1 md:grid-cols-4 gap-3 text-xs items-end">
            <div className="space-y-1">
              <Label className="text-[11px]">Subject</Label>
              <Input
                value={subject}
                onChange={(e) => setSubject(e.target.value)}
                placeholder="user:alice"
                className="h-8 text-xs font-mono"
              />
            </div>
            <div className="space-y-1 md:col-span-2">
              <Label className="text-[11px]">Target Resource Path</Label>
              <Input
                value={resource}
                onChange={(e) => setResource(e.target.value)}
                placeholder="/api/v1/finance/invoices"
                className="h-8 text-xs font-mono"
              />
            </div>
            <div className="flex gap-2">
              <div className="space-y-1 w-24">
                <Label className="text-[11px]">Action</Label>
                <Select value={action} onValueChange={setAction}>
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
              <Button type="submit" size="sm" disabled={isLoading} className="h-8 text-xs gap-1.5 flex-1 mt-auto">
                {isLoading ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : <Play className="w-3.5 h-3.5" />}
                Evaluate
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>

      {/* Results View */}
      {result && (
        <div className="space-y-4">
          {/* Main Outcome Banner */}
          <div
            className={`p-4 rounded-lg border flex items-center justify-between ${
              result.allowed
                ? "bg-emerald-500/10 border-emerald-500/30 text-emerald-500"
                : "bg-destructive/10 border-destructive/30 text-destructive"
            }`}
          >
            <div className="flex items-center gap-3">
              {result.allowed ? (
                <CheckCircle2 className="w-6 h-6 shrink-0 text-emerald-500" />
              ) : (
                <XCircle className="w-6 h-6 shrink-0 text-destructive" />
              )}
              <div>
                <div className="font-bold text-sm">
                  {result.allowed ? "ACCESS GRANTED · 200 OK" : "ACCESS DENIED · 403 FORBIDDEN"}
                </div>
                <div className="text-xs text-muted-foreground mt-0.5">{result.reason}</div>
              </div>
            </div>

            <Badge
              variant={result.allowed ? "default" : "destructive"}
              className={`text-xs px-2.5 py-1 ${result.allowed ? "bg-emerald-500" : ""}`}
            >
              {result.allowed ? "PERMITTED" : "DENIED"}
            </Badge>
          </div>

          {/* Engine Contribution Attribution Grid */}
          <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
            {/* 1. Aegis Card */}
            <Card className="border-border">
              <CardHeader className="p-3 pb-2">
                <div className="flex items-center justify-between">
                  <span className="text-xs font-semibold flex items-center gap-1.5 text-foreground">
                    <Shield className="w-3.5 h-3.5 text-blue-400" />
                    Aegis Ingress
                  </span>
                  <Badge variant={result.engine_breakdown.aegis.matched ? "default" : "destructive"} className="text-[9px] px-1 py-0">
                    {result.engine_breakdown.aegis.matched ? "Matched" : "No Match"}
                  </Badge>
                </div>
              </CardHeader>
              <CardContent className="p-3 pt-0 text-[11px] font-mono space-y-1">
                <div className="text-muted-foreground truncate">
                  Rule: <span className="text-foreground">{result.engine_breakdown.aegis.rule_name}</span>
                </div>
                <div className="text-muted-foreground truncate">
                  Upstream: <span className="text-foreground">{result.engine_breakdown.aegis.upstream}</span>
                </div>
              </CardContent>
            </Card>

            {/* 2. Nexus Card */}
            <Card className="border-border">
              <CardHeader className="p-3 pb-2">
                <div className="flex items-center justify-between">
                  <span className="text-xs font-semibold flex items-center gap-1.5 text-foreground">
                    <Network className="w-3.5 h-3.5 text-indigo-400" />
                    Nexus ReBAC
                  </span>
                  <Badge
                    variant={result.engine_breakdown.nexus.allowed ? "default" : "destructive"}
                    className={`text-[9px] px-1 py-0 ${
                      result.engine_breakdown.nexus.allowed ? "bg-emerald-500" : ""
                    }`}
                  >
                    {result.engine_breakdown.nexus.allowed ? "Allowed" : "Denied"}
                  </Badge>
                </div>
              </CardHeader>
              <CardContent className="p-3 pt-0 text-[11px] font-mono space-y-1">
                <div className="text-muted-foreground truncate">
                  Target:{" "}
                  <span className="text-foreground">
                    {result.engine_breakdown.nexus.namespace}:{result.engine_breakdown.nexus.object}
                  </span>
                </div>
                <div className="text-muted-foreground truncate">
                  Relation: <span className="text-foreground">{result.engine_breakdown.nexus.relation}</span>
                </div>
              </CardContent>
            </Card>

            {/* 3. Themis Card */}
            <Card className="border-border">
              <CardHeader className="p-3 pb-2">
                <div className="flex items-center justify-between">
                  <span className="text-xs font-semibold flex items-center gap-1.5 text-foreground">
                    <Scale className="w-3.5 h-3.5 text-purple-400" />
                    Themis ABAC
                  </span>
                  <Badge
                    variant={result.engine_breakdown.themis.allowed ? "default" : "destructive"}
                    className={`text-[9px] px-1 py-0 ${
                      result.engine_breakdown.themis.allowed ? "bg-emerald-500" : ""
                    }`}
                  >
                    {result.engine_breakdown.themis.allowed ? "All Passed" : "Policy Failed"}
                  </Badge>
                </div>
              </CardHeader>
              <CardContent className="p-3 pt-0 text-[11px] font-mono space-y-1">
                <div className="text-muted-foreground truncate">
                  Policies: <span className="text-foreground">{result.engine_breakdown.themis.policies_matched} evaluated</span>
                </div>
                {result.engine_breakdown.themis.failed_policy && (
                  <div className="text-destructive truncate">
                    Failed: {result.engine_breakdown.themis.failed_policy}
                  </div>
                )}
              </CardContent>
            </Card>
          </div>
        </div>
      )}
    </div>
  );
}
