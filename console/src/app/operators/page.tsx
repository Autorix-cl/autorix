"use client";

import { useQuery } from "@tanstack/react-query";
import { ShieldAlert, KeyRound, UserCheck, RefreshCw, AlertCircle, Lock } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from "@/components/ui/card";
import type { OperatorDTO } from "@/lib/api/schemas/operator";

export default function OperatorsPage() {
  const { data: operators, isLoading, error, refetch } = useQuery<OperatorDTO[]>({
    queryKey: ["operators"],
    queryFn: async () => {
      const res = await fetch("/api/operators");
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        throw new Error(data.error || "Failed to load operators");
      }
      return res.json();
    },
  });

  return (
    <div className="space-y-6">
      {/* Header Section */}
      <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
        <div>
          <div className="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full border border-amber-500/20 bg-amber-500/5 text-amber-400 text-xs font-mono mb-2">
            <Lock className="w-3 h-3" />
            CONSOLE IDENTITY & RBAC
          </div>
          <h1 className="text-2xl font-bold tracking-tight text-foreground">Console Operators</h1>
          <p className="text-sm text-muted-foreground">
            Manage authenticated operators, break-glass administrator access, and role assignments.
          </p>
        </div>

        <div className="flex items-center gap-2">
          <Button variant="outline" size="sm" onClick={() => refetch()} className="gap-1.5 text-xs">
            <RefreshCw className="h-3.5 w-3.5" />
            Refresh
          </Button>
        </div>
      </div>

      {/* Break-glass Policy Banner */}
      <div className="p-4 rounded-xl border border-amber-500/30 bg-gradient-to-r from-amber-500/10 via-amber-500/5 to-transparent flex items-start gap-3.5">
        <ShieldAlert className="w-5 h-5 text-amber-400 shrink-0 mt-0.5" />
        <div className="text-xs space-y-1">
          <h3 className="font-semibold text-amber-300">Break-Glass Sovereignty Policy (P3-S1-T3)</h3>
          <p className="text-slate-400">
            Local credentials stored in Argus remain valid when upstream SSO providers (Janus / Ego / Hermes) are unreachable. Break-glass logins trigger high-priority audit events and are locked after 5 consecutive failed attempts.
          </p>
        </div>
      </div>

      {/* Operators List */}
      <Card className="border-border/80 bg-card/60 backdrop-blur-sm shadow-xs">
        <CardHeader className="pb-3">
          <div className="flex items-center justify-between">
            <div>
              <CardTitle className="text-base font-semibold">Registered Operators</CardTitle>
              <CardDescription className="text-xs">
                Active administrative principals with access to the Autorix Control Plane
              </CardDescription>
            </div>
            {operators && (
              <Badge variant="outline" className="font-mono text-xs">
                {operators.length} {operators.length === 1 ? "Operator" : "Operators"}
              </Badge>
            )}
          </div>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="py-12 text-center text-xs text-muted-foreground font-mono">
              Loading operators from control plane registry...
            </div>
          ) : error ? (
            <div className="p-4 rounded-lg border border-red-500/30 bg-red-500/10 text-red-300 text-xs flex items-center gap-2">
              <AlertCircle className="w-4 h-4 text-red-400 shrink-0" />
              <span>{(error as Error).message}</span>
            </div>
          ) : !operators || operators.length === 0 ? (
            <div className="py-12 text-center text-xs text-muted-foreground">
              No operators registered yet.
            </div>
          ) : (
            <div className="divide-y divide-border/60">
              {operators.map((op) => (
                <div key={op.id} className="py-3.5 flex items-center justify-between gap-4">
                  <div className="flex items-center gap-3">
                    <div className="w-9 h-9 rounded-lg bg-slate-800/80 border border-slate-700/60 flex items-center justify-center text-amber-400 font-mono font-bold text-xs">
                      {op.name.slice(0, 2).toUpperCase()}
                    </div>
                    <div>
                      <div className="flex items-center gap-2">
                        <span className="text-sm font-semibold text-foreground">{op.name}</span>
                        {op.is_local ? (
                          <Badge variant="outline" className="text-[10px] bg-amber-500/10 text-amber-400 border-amber-500/30 gap-1 font-mono">
                            <KeyRound className="w-2.5 h-2.5" />
                            Break-Glass / Local
                          </Badge>
                        ) : (
                          <Badge variant="outline" className="text-[10px] bg-blue-500/10 text-blue-400 border-blue-500/30 gap-1 font-mono">
                            <UserCheck className="w-2.5 h-2.5" />
                            SSO Federated
                          </Badge>
                        )}
                      </div>
                      <p className="text-xs text-muted-foreground font-mono">{op.email}</p>
                    </div>
                  </div>

                  <div className="flex items-center gap-3">
                    <span className="text-xs uppercase font-mono px-2 py-0.5 rounded bg-muted text-muted-foreground border border-border/80 font-medium">
                      Role: {op.role}
                    </span>
                    <Badge variant={op.is_active ? "default" : "destructive"} className="text-[10px]">
                      {op.is_active ? "Active" : "Deactivated"}
                    </Badge>
                  </div>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
