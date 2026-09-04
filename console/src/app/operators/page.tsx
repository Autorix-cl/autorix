"use client";

import * as React from "react";
import {
  ShieldAlert,
  KeyRound,
  UserCheck,
  RefreshCw,
  Lock,
  UserPlus,
  Users,
  Shield,
  Activity,
  UserX,
  Trash2,
  AlertTriangle,
  Loader2,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { ServiceHeader } from "@/components/layout/service-header";
import { CloudSection } from "@/components/layout/cloud-section";
import { ErrorState } from "@/components/state/error-state";
import { useApiQuery } from "@/lib/query/use-api-query";
import { useApiMutation } from "@/lib/query/use-api-mutation";
import { useQueryClient } from "@tanstack/react-query";
import { fetchAndParse } from "@/lib/api/schema";
import { operatorsListSchema, operatorSchema, type OperatorDTO } from "@/lib/api/schemas/operator";
import { z } from "zod";
import { OperatorBuilderSheet } from "./operator-builder-sheet";

export default function OperatorsPage() {
  const queryClient = useQueryClient();
  const [isBuilderOpen, setIsBuilderOpen] = React.useState(false);
  const [deactivateTarget, setDeactivateTarget] = React.useState<OperatorDTO | null>(null);
  const [deleteTarget, setDeleteTarget] = React.useState<OperatorDTO | null>(null);

  const {
    data: operators,
    isLoading,
    isError,
    error,
    refetch,
  } = useApiQuery(["operators"], () => fetchAndParse("/api/operators", operatorsListSchema));

  const toggleStatusMutation = useApiMutation(
    ({ id, is_active }: { id: string; is_active: boolean }) =>
      fetchAndParse(`/api/operators/${id}`, operatorSchema, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ is_active }),
      }),
    {
      successMessage: (data) =>
        data.is_active ? "Operator reactivated successfully." : "Operator account deactivated and active sessions revoked.",
      onSuccess: () => {
        queryClient.invalidateQueries({ queryKey: ["operators"] });
        setDeactivateTarget(null);
      },
    }
  );

  const deleteMutation = useApiMutation(
    (id: string) =>
      fetchAndParse(`/api/operators/${id}`, z.any(), {
        method: "DELETE",
      }),
    {
      successMessage: () => "Operator account permanently purged from control plane.",
      onSuccess: () => {
        queryClient.invalidateQueries({ queryKey: ["operators"] });
        setDeleteTarget(null);
      },
    }
  );

  return (
    <div className="space-y-6">
      <OperatorBuilderSheet
        isOpen={isBuilderOpen}
        onOpenChange={setIsBuilderOpen}
        onSuccess={() => refetch()}
      />

      {/* Cloud Service Header & Telemetry HUD */}
      <ServiceHeader
        serviceName="Argus Control Plane"
        title="Console Operators & RBAC"
        description="Manage authenticated operators, break-glass administrator access, and role assignments."
        icon={ShieldAlert}
        iconColor="text-amber-400"
        statusText="SOVEREIGN-LOCAL-VAULT-ACTIVE"
        statusVariant="warning"
        metrics={[
          {
            label: "Active Operators",
            value: operators?.length ?? 0,
            hint: "Administrative Principals",
            icon: Users,
          },
          {
            label: "Sovereign Vault",
            value: "PostgreSQL",
            hint: "Local Argon2id Vault",
            icon: Lock,
          },
          {
            label: "Role Hierarchy",
            value: "4 Roles",
            hint: "Owner › Admin › Op › Auditor",
            icon: Shield,
          },
          {
            label: "Control Gateway",
            value: "Port 4432",
            hint: "Argus Control Plane",
            icon: Activity,
          },
        ]}
        actions={
          <>
            <Button
              size="sm"
              onClick={() => setIsBuilderOpen(true)}
              className="h-8 gap-1.5 text-xs bg-amber-600 hover:bg-amber-700 text-white font-medium shadow-sm"
            >
              <UserPlus className="h-3.5 w-3.5" />
              <span>Provision Operator</span>
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={() => refetch()}
              className="h-8 gap-1.5 text-xs"
            >
              <RefreshCw className="h-3.5 w-3.5" />
              <span>Refresh</span>
            </Button>
          </>
        }
      />

      {/* Cloud Section 1: Sovereignty & Security Policy */}
      <CloudSection
        title="Sovereignty & Security Policy"
        description="Emergency local access procedures when upstream identity federations are unreachable"
        icon={ShieldAlert}
        badge="POLICY ENFORCED"
        badgeVariant="warning"
      >
        <div className="p-4 rounded-xl border border-amber-500/30 bg-gradient-to-r from-amber-500/10 via-amber-500/5 to-transparent flex items-start gap-3.5">
          <ShieldAlert className="w-5 h-5 text-amber-400 shrink-0 mt-0.5" />
          <div className="text-xs space-y-1">
            <h3 className="font-semibold text-amber-300">Break-Glass Sovereignty Policy (P3-S1-T3)</h3>
            <p className="text-slate-400">
              Local credentials stored in Argus remain valid when upstream SSO providers (Janus / Ego / Hermes) are unreachable. Break-glass logins trigger high-priority audit events and are locked after 5 consecutive failed attempts.
            </p>
          </div>
        </div>
      </CloudSection>

      {/* Cloud Section 2: Operators Directory & Role Bindings */}
      <CloudSection
        title="Administrative Directory & Role Bindings"
        description="Active console principals and role-based permissions matrix"
        icon={Users}
        badge={`${operators?.length ?? 0} ACTIVE`}
        badgeVariant="cyan"
        actions={
          <Button
            size="sm"
            variant="outline"
            onClick={() => setIsBuilderOpen(true)}
            className="h-7 gap-1 text-xs"
          >
            <UserPlus className="w-3 h-3 text-amber-400" />
            <span>Add Operator</span>
          </Button>
        }
      >

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
          ) : isError ? (
            <div className="py-6">
              <ErrorState error={error} onRetry={refetch} />
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

                    {/* Operator Lifecycle Actions */}
                    <div className="flex items-center gap-1.5 ml-2">
                      {op.is_active ? (
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => setDeactivateTarget(op)}
                          disabled={toggleStatusMutation.isPending || deleteMutation.isPending}
                          className="h-7 text-xs gap-1 text-rose-400 border-rose-500/30 hover:bg-rose-500/10 hover:text-rose-300"
                          title="Deactivate operator account"
                        >
                          <UserX className="w-3 h-3" />
                          <span>Deactivate</span>
                        </Button>
                      ) : (
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => toggleStatusMutation.mutate({ id: op.id, is_active: true })}
                          disabled={toggleStatusMutation.isPending || deleteMutation.isPending}
                          className="h-7 text-xs gap-1 text-emerald-400 border-emerald-500/30 hover:bg-emerald-500/10 hover:text-emerald-300"
                          title="Reactivate operator account"
                        >
                          <UserCheck className="w-3 h-3" />
                          <span>Reactivate</span>
                        </Button>
                      )}

                      <Button
                        variant="ghost"
                        size="icon-sm"
                        onClick={() => setDeleteTarget(op)}
                        disabled={toggleStatusMutation.isPending || deleteMutation.isPending}
                        className="h-7 w-7 text-muted-foreground hover:text-rose-400 hover:bg-rose-500/10"
                        title="Delete operator permanently"
                      >
                        <Trash2 className="w-3.5 h-3.5" />
                      </Button>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>
      </CloudSection>

      {/* Deactivation Confirmation Modal */}
      <Dialog open={!!deactivateTarget} onOpenChange={(open) => !open && setDeactivateTarget(null)}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <div className="flex items-center gap-2 text-rose-400">
              <AlertTriangle className="w-5 h-5" />
              <DialogTitle>Deactivate Operator Account</DialogTitle>
            </div>
            <DialogDescription className="text-xs pt-2 text-muted-foreground">
              Are you sure you want to deactivate{" "}
              <strong className="text-foreground font-mono">{deactivateTarget?.name}</strong> (
              <span className="font-mono text-muted-foreground">{deactivateTarget?.email}</span>)?
              <br />
              All active sessions will be terminated immediately and future login access will be blocked.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter className="gap-2 sm:gap-0">
            <Button
              variant="outline"
              size="sm"
              onClick={() => setDeactivateTarget(null)}
              disabled={toggleStatusMutation.isPending}
            >
              Cancel
            </Button>
            <Button
              variant="destructive"
              size="sm"
              onClick={() => {
                if (deactivateTarget) {
                  toggleStatusMutation.mutate({ id: deactivateTarget.id, is_active: false });
                }
              }}
              disabled={toggleStatusMutation.isPending}
              className="gap-1.5"
            >
              {toggleStatusMutation.isPending ? (
                <Loader2 className="w-3.5 h-3.5 animate-spin" />
              ) : (
                <UserX className="w-3.5 h-3.5" />
              )}
              <span>Confirm Deactivation</span>
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete Confirmation Modal */}
      <Dialog open={!!deleteTarget} onOpenChange={(open) => !open && setDeleteTarget(null)}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <div className="flex items-center gap-2 text-rose-500">
              <Trash2 className="w-5 h-5" />
              <DialogTitle>Delete Operator Permanently</DialogTitle>
            </div>
            <DialogDescription className="text-xs pt-2 text-muted-foreground">
              Are you sure you want to permanently delete operator{" "}
              <strong className="text-foreground font-mono">{deleteTarget?.name}</strong> (
              <span className="font-mono text-muted-foreground">{deleteTarget?.email}</span>)?
              <br />
              <span className="text-rose-400 font-semibold">This action cannot be undone.</span> Credentials, role bindings, and personal tokens will be purged.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter className="gap-2 sm:gap-0">
            <Button
              variant="outline"
              size="sm"
              onClick={() => setDeleteTarget(null)}
              disabled={deleteMutation.isPending}
            >
              Cancel
            </Button>
            <Button
              variant="destructive"
              size="sm"
              onClick={() => {
                if (deleteTarget) {
                  deleteMutation.mutate(deleteTarget.id);
                }
              }}
              disabled={deleteMutation.isPending}
              className="gap-1.5"
            >
              {deleteMutation.isPending ? (
                <Loader2 className="w-3.5 h-3.5 animate-spin" />
              ) : (
                <Trash2 className="w-3.5 h-3.5" />
              )}
              <span>Delete Permanently</span>
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
