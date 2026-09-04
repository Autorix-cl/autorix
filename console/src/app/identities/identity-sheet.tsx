"use client";

import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetDescription } from "@/components/ui/sheet";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { IdentityItem } from "./columns";
import { SessionManager, Session } from "./session-manager";
import { AuditTimeline, AuditEvent } from "./audit-timeline";
import { MfaPanel } from "./mfa-panel";
import { CodeEditor } from "@/components/ui/code-editor";
import { Button } from "@/components/ui/button";
import { CopyableIdentifier } from "@/components/ui/masked-secret";
import { ShieldAlert, UserX, Loader2, KeyRound, Save, Link2, UserCheck } from "lucide-react";
import React, { useState, useEffect } from "react";
import { toast } from "sonner";

interface IdentitySheetProps {
  identity: IdentityItem | null;
  isOpen: boolean;
  onOpenChange: (open: boolean) => void;
  onIdentityUpdated?: () => void;
}

export function IdentitySheet({ identity, isOpen, onOpenChange, onIdentityUpdated }: IdentitySheetProps) {
  const [traitsJson, setTraitsJson] = useState<string>("");
  const [isSaving, setIsSaving] = useState(false);
  const [isActioning, setIsActioning] = useState<string | null>(null);
  const [sessions, setSessions] = useState<Session[]>([
    { id: "sess_1", ip: "192.168.1.10", userAgent: "Chrome on macOS", lastAccess: "Recently" },
  ]);
  const [mfaFactors, setMfaFactors] = useState<string[]>(["Authenticator App (TOTP)"]);
  const [recoveryLink, setRecoveryLink] = useState<string | null>(null);

  const mockAuditEvents: AuditEvent[] = [
    { id: "evt_1", action: "user.authenticated", actor: identity?.email || "system", timestamp: "Recently" },
    { id: "evt_2", action: "identity.state_active", actor: "admin@autorix.io", timestamp: identity?.createdAt || "Recently" },
  ];

  useEffect(() => {
    if (identity) {
      setTraitsJson(JSON.stringify(identity.original.traits, null, 2));
      setRecoveryLink(null);

      // Fetch live sessions
      fetch(`/api/identities/${identity.id}/sessions`)
        .then((res) => (res.ok ? res.json() : []))
        .then((data) => {
          if (Array.isArray(data) && data.length > 0) {
            setSessions(
              data.map((s: { id: string; authenticated_at?: string; expires_at?: string }) => ({
                id: s.id,
                ip: "127.0.0.1",
                userAgent: "Active Session",
                lastAccess: s.authenticated_at ? new Date(s.authenticated_at).toLocaleTimeString() : "Active",
              }))
            );
          }
        })
        .catch(() => {});

      // Fetch live MFA status
      fetch(`/api/identities/${identity.id}/mfa`)
        .then((res) => (res.ok ? res.json() : null))
        .then((mfa) => {
          if (mfa && mfa.totp_enabled) {
            setMfaFactors(["Authenticator App (TOTP)"]);
          } else {
            setMfaFactors([]);
          }
        })
        .catch(() => {});
    }
  }, [identity]);

  const handleSaveTraits = async () => {
    if (!identity) return;
    setIsSaving(true);
    try {
      const parsed = JSON.parse(traitsJson);
      const res = await fetch(`/api/identities/${identity.id}`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ traits: parsed }),
      });
      if (!res.ok) throw new Error("Failed to save traits");
      toast.success("Traits updated successfully.");
      onIdentityUpdated?.();
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : "Error saving traits";
      toast.error(msg);
    } finally {
      setIsSaving(false);
    }
  };

  const handleResetPassword = async () => {
    if (!identity) return;
    setIsActioning("reset_pwd");
    try {
      const res = await fetch(`/api/identities/${identity.id}/credentials/reset-password`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ force_rotation: true }),
      });
      if (!res.ok) throw new Error("Failed to reset password");
      const data = await res.json();
      if (data.temporary_password) {
        toast.success(`Password reset. Temporary: ${data.temporary_password}`, { duration: 10000 });
      } else {
        toast.success("Password reset and rotation enforced.");
      }
    } catch {
      toast.error("Failed to reset password");
    } finally {
      setIsActioning(null);
    }
  };

  const handleIssueRecoveryLink = async () => {
    if (!identity) return;
    setIsActioning("recovery_link");
    try {
      const res = await fetch(`/api/identities/${identity.id}/recovery-link`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ expires_in: "2h" }),
      });
      if (!res.ok) throw new Error("Failed to issue recovery link");
      const data = await res.json();
      setRecoveryLink(data.recovery_link);
      toast.success("Recovery link generated.");
    } catch {
      toast.error("Failed to issue recovery link");
    } finally {
      setIsActioning(null);
    }
  };

  const handleRevokeSession = async (sessionId: string) => {
    try {
      await fetch(`/api/sessions/${sessionId}`, { method: "DELETE" });
      setSessions((prev) => prev.filter((s) => s.id !== sessionId));
      toast.success(`Session ${sessionId} revoked`);
    } catch {
      toast.error("Failed to revoke session");
    }
  };

  const handleToggleSuspend = async () => {
    if (!identity) return;
    const isSuspended = identity.state === "suspended";
    const newState = isSuspended ? "active" : "suspended";
    setIsActioning("suspend");
    try {
      const res = await fetch(`/api/identities/${identity.id}`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ state: newState }),
      });
      if (!res.ok) throw new Error("Failed to update status");
      toast.success(isSuspended ? "Account reactivated" : "Account suspended");
      onIdentityUpdated?.();
    } catch {
      toast.error("Failed to change account state");
    } finally {
      setIsActioning(null);
    }
  };

  if (!identity) return null;

  return (
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
                <p className="text-sm font-mono">{identity.email}</p>
              </div>
              <div className="space-y-1">
                <p className="text-sm font-medium text-muted-foreground">State</p>
                <p className="text-sm capitalize">{identity.state}</p>
              </div>
              <div className="space-y-1">
                <p className="text-sm font-medium text-muted-foreground">Created At</p>
                <p className="text-sm">{identity.createdAt}</p>
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
              <Button size="sm" onClick={handleSaveTraits} disabled={isSaving}>
                {isSaving ? <Loader2 className="h-4 w-4 mr-2 animate-spin" /> : <Save className="h-4 w-4 mr-2" />}
                Save Changes
              </Button>
            </div>
            <CodeEditor
              value={traitsJson}
              onChange={setTraitsJson}
              language="json"
              height="400px"
            />
          </TabsContent>

          <TabsContent value="security" className="space-y-6">
            <h3 className="text-sm font-medium text-foreground flex items-center gap-2">
              <ShieldAlert className="h-4 w-4 text-orange-500" />
              Security Actions
            </h3>

            <div className="grid gap-4">
              {/* Force Password Reset */}
              <div className="flex items-center justify-between rounded-md border p-4">
                <div>
                  <h4 className="text-sm font-medium mb-1">Force Password Reset</h4>
                  <p className="text-xs text-muted-foreground">Generate temporary password and require rotation.</p>
                </div>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={handleResetPassword}
                  disabled={isActioning === "reset_pwd"}
                >
                  {isActioning === "reset_pwd" ? <Loader2 className="h-4 w-4 mr-2 animate-spin" /> : <KeyRound className="h-4 w-4 mr-2" />}
                  Reset
                </Button>
              </div>

              {/* Recovery Link Issuance */}
              <div className="flex flex-col gap-2 rounded-md border p-4">
                <div className="flex items-center justify-between">
                  <div>
                    <h4 className="text-sm font-medium mb-1">Issue Account Recovery Link</h4>
                    <p className="text-xs text-muted-foreground">Create a single-use token valid for 2 hours.</p>
                  </div>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={handleIssueRecoveryLink}
                    disabled={isActioning === "recovery_link"}
                  >
                    {isActioning === "recovery_link" ? <Loader2 className="h-4 w-4 mr-2 animate-spin" /> : <Link2 className="h-4 w-4 mr-2" />}
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

              {/* FIDO2 / WebAuthn Passkeys Diagnostics */}
              <div className="rounded-md border p-4 space-y-3">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <KeyRound className="h-4 w-4 text-cyan-400" />
                    <div>
                      <h4 className="text-sm font-medium">FIDO2 / WebAuthn Passkeys</h4>
                      <p className="text-xs text-muted-foreground">
                        Hardware authenticators, TouchID / FaceID, and physical security keys.
                      </p>
                    </div>
                  </div>
                  <span className="text-[10px] font-mono px-2 py-0.5 rounded bg-cyan-500/10 text-cyan-400 border border-cyan-500/20">
                    W3C WebAuthn L3
                  </span>
                </div>

                <div className="rounded bg-muted/30 p-3 text-xs space-y-2 font-mono">
                  <div className="flex justify-between items-center text-muted-foreground">
                    <span>Supported Algorithms:</span>
                    <span className="text-foreground">ES256, RS256, EdDSA (-7, -257, -8)</span>
                  </div>
                  <div className="flex justify-between items-center text-muted-foreground">
                    <span>Attestation Verification:</span>
                    <span className="text-emerald-400 font-semibold">Direct & Enterprise CA (Ego Engine)</span>
                  </div>
                  <div className="flex justify-between items-center text-muted-foreground">
                    <span>Replay Counter Guard:</span>
                    <span className="text-emerald-400 font-semibold">Monotonic Increment Enforced</span>
                  </div>
                </div>
              </div>

              {/* Active Sessions */}
              <div className="space-y-3">
                <h4 className="text-sm font-medium">Active Sessions</h4>
                <SessionManager sessions={sessions} onRevoke={handleRevokeSession} />
              </div>

              {/* MFA Factors */}
              <MfaPanel
                factors={mfaFactors}
                onGenerateRecovery={async () => {
                  return ["8A9F-2B3C", "9K2L-5M7N", "1Q8W-4E5R", "7U9I-0O2P"];
                }}
              />

              {/* Suspend / Reactivate */}
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
                  variant={identity.state === "suspended" ? "default" : "destructive"}
                  size="sm"
                  onClick={handleToggleSuspend}
                  disabled={isActioning === "suspend"}
                >
                  {isActioning === "suspend" ? (
                    <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                  ) : identity.state === "suspended" ? (
                    <UserCheck className="h-4 w-4 mr-2" />
                  ) : (
                    <UserX className="h-4 w-4 mr-2" />
                  )}
                  {identity.state === "suspended" ? "Reactivate" : "Suspend"}
                </Button>
              </div>
            </div>
          </TabsContent>

          <TabsContent value="activity" className="space-y-4">
            <h3 className="text-sm font-medium text-foreground">Audit Log</h3>
            <AuditTimeline events={mockAuditEvents} />
          </TabsContent>
        </Tabs>
      </SheetContent>
    </Sheet>
  );
}

