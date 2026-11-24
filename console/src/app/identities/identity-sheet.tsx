"use client";

import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetDescription } from "@/components/ui/sheet";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { IdentityItem } from "./columns";
import { SessionManager, Session } from "./session-manager";
import { AuditTimeline, AuditEvent } from "./audit-timeline";
import { MfaPanel } from "./mfa-panel";
import { CodeEditor } from "@/components/ui/code-editor";
import { Button } from "@/components/ui/button";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { CopyableIdentifier, MaskedSecret } from "@/components/ui/masked-secret";
import { Can } from "@/lib/auth/use-permission";
import { formatDateTime } from "@/lib/format/date-time";
import { ShieldAlert, UserX, Loader2, KeyRound, Save, Link2, UserCheck } from "lucide-react";
import React, { useState, useEffect } from "react";
import { toast } from "sonner";

interface IdentitySheetProps {
  identity: IdentityItem | null;
  isOpen: boolean;
  onOpenChange: (open: boolean) => void;
  onIdentityUpdated?: () => void;
}

function readCsrfCookie(): string | undefined {
  if (typeof document === "undefined") return undefined;
  const match = document.cookie.match(/(?:^|;\s*)autorix_csrf=([^;]+)/);
  return match ? decodeURIComponent(match[1]) : undefined;
}

export function IdentitySheet({ identity, isOpen, onOpenChange, onIdentityUpdated }: IdentitySheetProps) {
  const [traitsJson, setTraitsJson] = useState<string>("");
  const [isSaving, setIsSaving] = useState(false);
  const [isActioning, setIsActioning] = useState<string | null>(null);
  const [sessions, setSessions] = useState<Session[]>([]);
  const [mfaFactors, setMfaFactors] = useState<string[]>([]);
  const [auditEvents, setAuditEvents] = useState<AuditEvent[]>([]);
  const [recoveryLink, setRecoveryLink] = useState<string | null>(null);
  const [temporaryPassword, setTemporaryPassword] = useState<string | null>(null);
  const [confirmResetOpen, setConfirmResetOpen] = useState(false);
  const [confirmSuspendOpen, setConfirmSuspendOpen] = useState(false);

  useEffect(() => {
    if (!identity) return;
    const controller = new AbortController();

    setTraitsJson(JSON.stringify(identity.original.traits, null, 2));
    setRecoveryLink(null);
    setTemporaryPassword(null);
    setSessions([]);
    setMfaFactors([]);
    setAuditEvents([]);

    const encodedId = encodeURIComponent(identity.id);

    // Fetch live sessions
    fetch(`/api/identities/${encodedId}/sessions`, { signal: controller.signal })
      .then((res) => (res.ok ? res.json() : []))
      .then((data) => {
        const list = Array.isArray(data) ? data : data?.data || [];
        setSessions(
          list.map((s: { id: string; ip_address?: string; user_agent?: string; authenticated_at?: string }) => ({
            id: s.id,
            ip: s.ip_address || "Unknown IP",
            userAgent: s.user_agent || "Active Session",
            lastAccess: s.authenticated_at ? formatDateTime(s.authenticated_at) : "Active",
          })),
        );
      })
      .catch(() => {});

    // Fetch live MFA status
    fetch(`/api/identities/${encodedId}/mfa`, { signal: controller.signal })
      .then((res) => (res.ok ? res.json() : null))
      .then((mfa) => {
        if (mfa && mfa.totp_enabled) {
          setMfaFactors(["Authenticator App (TOTP)"]);
        } else {
          setMfaFactors([]);
        }
      })
      .catch(() => {});

    // Fetch live audit events for this identity from Argus
    fetch(`/api/audit?resource_type=identity`, { signal: controller.signal })
      .then((res) => (res.ok ? res.json() : []))
      .then((data) => {
        const list = Array.isArray(data) ? data : data?.data || [];
        const matched = list
          .filter(
            (e: { resource_id?: string; actor?: string }) =>
              e.resource_id === identity.id || e.actor === identity.email,
          )
          .slice(0, 20)
          .map((e: { id: string; action: string; actor?: string; timestamp?: string }) => ({
            id: e.id,
            action: e.action,
            actor: e.actor || "system",
            timestamp: e.timestamp ? formatDateTime(e.timestamp) : "—",
          }));
        setAuditEvents(matched);
      })
      .catch(() => {});

    return () => controller.abort();
  }, [identity]);

  const jsonHeaders = () => {
    const csrf = readCsrfCookie();
    return {
      "Content-Type": "application/json",
      ...(csrf ? { "X-CSRF-Token": csrf } : {}),
    };
  };

  const handleSaveTraits = async () => {
    if (!identity) return;
    setIsSaving(true);
    try {
      const parsed = JSON.parse(traitsJson);
      const res = await fetch(`/api/identities/${encodeURIComponent(identity.id)}`, {
        method: "PATCH",
        headers: jsonHeaders(),
        body: JSON.stringify({ traits: parsed }),
      });
      if (!res.ok) {
        const errBody = await res.json().catch(() => ({}));
        throw new Error(errBody.error || "Failed to save traits");
      }
      toast.success("Traits updated successfully.");
      onIdentityUpdated?.();
    } catch (err: unknown) {
      toast.error(err instanceof Error ? err.message : "Error saving traits");
    } finally {
      setIsSaving(false);
    }
  };

  const handleResetPassword = async () => {
    if (!identity) return;
    setIsActioning("reset_pwd");
    setTemporaryPassword(null);
    try {
      const res = await fetch(`/api/identities/${encodeURIComponent(identity.id)}/credentials/reset-password`, {
        method: "POST",
        headers: jsonHeaders(),
        body: JSON.stringify({ force_rotation: true }),
      });
      if (!res.ok) {
        const errBody = await res.json().catch(() => ({}));
        throw new Error(errBody.error || "Failed to reset password");
      }
      const data = await res.json();
      if (data.temporary_password) {
        setTemporaryPassword(data.temporary_password);
      }
      toast.success("Password reset and rotation enforced.");
      setConfirmResetOpen(false);
    } catch (err: unknown) {
      toast.error(err instanceof Error ? err.message : "Failed to reset password");
    } finally {
      setIsActioning(null);
    }
  };

  const handleIssueRecoveryLink = async () => {
    if (!identity) return;
    setIsActioning("recovery_link");
    try {
      const res = await fetch(`/api/identities/${encodeURIComponent(identity.id)}/recovery-link`, {
        method: "POST",
        headers: jsonHeaders(),
        body: JSON.stringify({ expires_in: "2h" }),
      });
      if (!res.ok) {
        const errBody = await res.json().catch(() => ({}));
        throw new Error(errBody.error || "Failed to issue recovery link");
      }
      const data = await res.json();
      setRecoveryLink(data.recovery_link);
      toast.success("Recovery link generated.");
    } catch (err: unknown) {
      toast.error(err instanceof Error ? err.message : "Failed to issue recovery link");
    } finally {
      setIsActioning(null);
    }
  };

  const handleRevokeSession = async (sessionId: string) => {
    try {
      const csrf = readCsrfCookie();
      const res = await fetch(`/api/sessions/${encodeURIComponent(sessionId)}`, {
        method: "DELETE",
        headers: csrf ? { "X-CSRF-Token": csrf } : undefined,
      });
      if (!res.ok) {
        const errBody = await res.json().catch(() => ({}));
        throw new Error(errBody.error || "Failed to revoke session");
      }
      setSessions((prev) => prev.filter((s) => s.id !== sessionId));
      toast.success(`Session ${sessionId.slice(0, 8)}... revoked`);
    } catch (err: unknown) {
      toast.error(err instanceof Error ? err.message : "Failed to revoke session");
    }
  };

  const handleToggleSuspend = async () => {
    if (!identity) return;
    const isSuspended = identity.state === "suspended";
    const newState = isSuspended ? "active" : "suspended";
    setIsActioning("suspend");
    try {
      const res = await fetch(`/api/identities/${encodeURIComponent(identity.id)}`, {
        method: "PATCH",
        headers: jsonHeaders(),
        body: JSON.stringify({ state: newState }),
      });
      if (!res.ok) {
        const errBody = await res.json().catch(() => ({}));
        throw new Error(errBody.error || "Failed to update status");
      }
      toast.success(isSuspended ? "Account reactivated" : "Account suspended and sessions revoked");
      setConfirmSuspendOpen(false);
      onIdentityUpdated?.();
    } catch (err: unknown) {
      toast.error(err instanceof Error ? err.message : "Failed to change account state");
    } finally {
      setIsActioning(null);
    }
  };

  if (!identity) return null;

  return (
    <>
      <Sheet open={isOpen} onOpenChange={onOpenChange}>
        <SheetContent className="sm:max-w-[600px] overflow-y-auto">
          <SheetHeader className="mb-6">
            <SheetTitle className="text-xl font-semibold flex items-center gap-2">
              User Profile: {identity.name}
            </SheetTitle>
            <SheetDescription>ID: {identity.id}</SheetDescription>
          </SheetHeader>

          <Tabs defaultValue="overview" className="w-full">
            <TabsList className="w-full grid grid-cols-4 mb-6">
              <TabsTrigger value="overview">Overview</TabsTrigger>
              <TabsTrigger value="traits">Traits</TabsTrigger>
              <TabsTrigger value="security">Security</TabsTrigger>
              <TabsTrigger value="activity">Activity</TabsTrigger>
            </TabsList>

            <TabsContent value="overview" className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-1">
                  <p className="text-sm font-medium text-muted-foreground">Email</p>
                  <p className="text-sm font-mono break-all">{identity.email}</p>
                </div>
                <div className="space-y-1">
                  <p className="text-sm font-medium text-muted-foreground">State</p>
                  <p className="text-sm capitalize">{identity.state}</p>
                </div>
                <div className="space-y-1">
                  <p className="text-sm font-medium text-muted-foreground">Created At</p>
                  <p className="text-sm font-mono">{identity.createdAt}</p>
                </div>
                <div className="space-y-1">
                  <p className="text-sm font-medium text-muted-foreground">Schema ID</p>
                  <p className="text-sm font-mono">{identity.original.schema_id || "default"}</p>
                </div>
              </div>
            </TabsContent>

            <TabsContent value="traits" className="space-y-4">
              <div className="flex items-center justify-between">
                <h3 className="text-sm font-medium text-foreground">Traits (JSON)</h3>
                <Can perform="identities:write">
                  <Button type="button" size="sm" onClick={handleSaveTraits} disabled={isSaving}>
                    {isSaving ? <Loader2 className="h-4 w-4 mr-2 animate-spin" aria-hidden="true" /> : <Save className="h-4 w-4 mr-2" aria-hidden="true" />}
                    Save Changes
                  </Button>
                </Can>
              </div>
              <CodeEditor value={traitsJson} onChange={setTraitsJson} language="json" height="400px" />
            </TabsContent>

            <TabsContent value="security" className="space-y-6">
              <h3 className="text-sm font-medium text-foreground flex items-center gap-2">
                <ShieldAlert className="h-4 w-4 text-orange-500" aria-hidden="true" />
                Security Actions
              </h3>

              <div className="grid gap-4">
                {/* Force Password Reset */}
                <Can perform="identities:write">
                  <div className="flex flex-col gap-3 rounded-md border p-4">
                    <div className="flex items-center justify-between gap-3">
                      <div>
                        <h4 className="text-sm font-medium mb-1">Force Password Reset</h4>
                        <p className="text-xs text-muted-foreground">
                          Generate a one-time temporary password, revoke active sessions, and require rotation.
                        </p>
                      </div>
                      <Button
                        type="button"
                        variant="outline"
                        size="sm"
                        onClick={() => setConfirmResetOpen(true)}
                        disabled={isActioning === "reset_pwd"}
                      >
                        {isActioning === "reset_pwd" ? (
                          <Loader2 className="h-4 w-4 mr-2 animate-spin" aria-hidden="true" />
                        ) : (
                          <KeyRound className="h-4 w-4 mr-2" aria-hidden="true" />
                        )}
                        Reset
                      </Button>
                    </div>
                    {temporaryPassword && (
                      <div className="rounded-md border border-amber-500/30 bg-amber-500/5 p-3 space-y-1.5">
                        <p className="text-[11px] font-medium text-amber-400">
                          One-time temporary password (not shown again after closing):
                        </p>
                        <MaskedSecret secret={temporaryPassword} label="Temporary password" />
                      </div>
                    )}
                  </div>
                </Can>

                {/* Recovery Link Issuance */}
                <Can perform="identities:write">
                  <div className="flex flex-col gap-2 rounded-md border p-4">
                    <div className="flex items-center justify-between gap-3">
                      <div>
                        <h4 className="text-sm font-medium mb-1">Issue Account Recovery Link</h4>
                        <p className="text-xs text-muted-foreground">Create a single-use token valid for 2 hours.</p>
                      </div>
                      <Button
                        type="button"
                        variant="outline"
                        size="sm"
                        onClick={handleIssueRecoveryLink}
                        disabled={isActioning === "recovery_link"}
                      >
                        {isActioning === "recovery_link" ? (
                          <Loader2 className="h-4 w-4 mr-2 animate-spin" aria-hidden="true" />
                        ) : (
                          <Link2 className="h-4 w-4 mr-2" aria-hidden="true" />
                        )}
                        Generate Link
                      </Button>
                    </div>
                    {recoveryLink && (
                      <div className="mt-2 p-2 bg-muted/40 rounded border font-mono text-xs text-emerald-400 select-all">
                        <CopyableIdentifier
                          value={recoveryLink}
                          label="Recovery Link"
                          truncateLength={64}
                          className="w-full justify-between"
                        />
                      </div>
                    )}
                  </div>
                </Can>

                {/* Active Sessions */}
                <div className="space-y-3">
                  <h4 className="text-sm font-medium">Active Sessions</h4>
                  <SessionManager sessions={sessions} onRevoke={handleRevokeSession} />
                </div>

                {/* MFA Factors */}
                <MfaPanel
                  factors={mfaFactors}
                  onGenerateRecovery={async () => {
                    const res = await fetch(`/api/identities/${encodeURIComponent(identity.id)}/recovery-link`, {
                      method: "POST",
                      headers: jsonHeaders(),
                      body: JSON.stringify({ expires_in: "2h" }),
                    });
                    if (!res.ok) {
                      toast.error("Failed to generate recovery token from Ego");
                      return [];
                    }
                    const data = await res.json();
                    const token = data.recovery_token || data.recovery_link;
                    return token ? [String(token)] : [];
                  }}
                />

                {/* Suspend / Reactivate */}
                <Can perform="identities:write">
                  <div className="flex items-center justify-between rounded-md border border-destructive/20 bg-destructive/10 p-4 mt-4">
                    <div>
                      <h4 className="text-sm font-medium text-destructive mb-1">
                        {identity.state === "suspended" ? "Reactivate Account" : "Suspend Account"}
                      </h4>
                      <p className="text-xs text-muted-foreground">
                        {identity.state === "suspended"
                          ? "Restore access and allow user authentication."
                          : "Block new logins and revoke active sessions."}
                      </p>
                    </div>
                    <Button
                      type="button"
                      variant={identity.state === "suspended" ? "default" : "destructive"}
                      size="sm"
                      onClick={() =>
                        identity.state === "suspended" ? handleToggleSuspend() : setConfirmSuspendOpen(true)
                      }
                      disabled={isActioning === "suspend"}
                    >
                      {isActioning === "suspend" ? (
                        <Loader2 className="h-4 w-4 mr-2 animate-spin" aria-hidden="true" />
                      ) : identity.state === "suspended" ? (
                        <UserCheck className="h-4 w-4 mr-2" aria-hidden="true" />
                      ) : (
                        <UserX className="h-4 w-4 mr-2" aria-hidden="true" />
                      )}
                      {identity.state === "suspended" ? "Reactivate" : "Suspend"}
                    </Button>
                  </div>
                </Can>
              </div>
            </TabsContent>

            <TabsContent value="activity" className="space-y-4">
              <h3 className="text-sm font-medium text-foreground">Audit Log</h3>
              <AuditTimeline events={auditEvents} />
            </TabsContent>
          </Tabs>
        </SheetContent>
      </Sheet>

      <ConfirmDialog
        open={confirmResetOpen}
        onOpenChange={setConfirmResetOpen}
        title="Force Password Reset"
        description={`Reset the password for ${identity.email}? Existing sessions will be revoked and the user must rotate credentials on next sign-in.`}
        confirmLabel="Reset Password"
        isLoading={isActioning === "reset_pwd"}
        onConfirm={handleResetPassword}
      />

      <ConfirmDialog
        open={confirmSuspendOpen}
        onOpenChange={setConfirmSuspendOpen}
        title="Suspend Identity"
        description={`Suspend ${identity.email}? All active sessions will be invalidated immediately and logins will be blocked.`}
        confirmLabel="Suspend Account"
        isLoading={isActioning === "suspend"}
        onConfirm={handleToggleSuspend}
      />
    </>
  );
}
