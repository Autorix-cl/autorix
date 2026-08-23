"use client";

import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetDescription } from "@/components/ui/sheet";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { IdentityItem } from "./columns";
import { SessionManager, Session } from "./session-manager";
import { AuditTimeline, AuditEvent } from "./audit-timeline";
import { MfaPanel } from "./mfa-panel";
import { CodeEditor } from "@/components/ui/code-editor";
import { Button } from "@/components/ui/button";
import { ShieldAlert, UserX, Loader2, KeyRound, Save } from "lucide-react";
import React, { useState } from "react";
import { toast } from "sonner";

interface IdentitySheetProps {
  identity: IdentityItem | null;
  isOpen: boolean;
  onOpenChange: (open: boolean) => void;
}

export function IdentitySheet({ identity, isOpen, onOpenChange }: IdentitySheetProps) {
  const [traitsJson, setTraitsJson] = useState<string>("");
  const [isSaving, setIsSaving] = useState(false);
  const [isActioning, setIsActioning] = useState<string | null>(null);

  
  const mockAuditEvents: AuditEvent[] = [
    { id: "evt_1", action: "user.suspended", actor: "admin@autorix.com", timestamp: "2023-10-12 09:00:00" },
    { id: "evt_2", action: "mfa.removed", actor: "system", timestamp: "2023-10-11 14:20:00" },
    { id: "evt_3", action: "user.created", actor: "admin@autorix.com", timestamp: "2023-10-01 10:00:00" }
  ];

  const mockSessions: Session[] = [
    { id: "sess_1", ip: "192.168.1.10", userAgent: "Chrome on macOS", lastAccess: "2023-10-10 10:00:00" },
    { id: "sess_2", ip: "10.0.0.5", userAgent: "Safari on iOS", lastAccess: "2023-10-11 15:30:00" }
  ];


  React.useEffect(() => {
    if (identity) {
      setTraitsJson(JSON.stringify(identity.original.traits, null, 2));
    }
  }, [identity]);

  const handleAction = (actionId: string, successMessage: string) => {
    setIsActioning(actionId);
    // TODO: Wire up to real useApiMutation once endpoint is ready
    setTimeout(() => {
      setIsActioning(null);
      toast.success(successMessage);
    }, 1000);
  };

  const handleSaveTraits = () => {
    setIsSaving(true);
    setTimeout(() => {
      setIsSaving(false);
      toast.success("Traits updated successfully.");
    }, 1000);
  };

  if (!identity) return null;

  return (
    <Sheet open={isOpen} onOpenChange={onOpenChange}>
      <SheetContent className="sm:max-w-[600px] overflow-y-auto">
        <SheetHeader className="mb-6">
          <SheetTitle className="text-xl font-semibold flex items-center gap-2">
            User Profile: {identity.name}
          </SheetTitle>
          <SheetDescription>
            ID: {identity.id}
          </SheetDescription>
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
                <p className="text-sm">{identity.email}</p>
              </div>
              <div className="space-y-1">
                <p className="text-sm font-medium text-muted-foreground">State</p>
                <p className="text-sm capitalize">{identity.state}</p>
              </div>
              <div className="space-y-1">
                <p className="text-sm font-medium text-muted-foreground">Created At</p>
                <p className="text-sm">{identity.createdAt}</p>
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
              <div className="flex items-center justify-between rounded-md border p-4">
                <div>
                  <h4 className="text-sm font-medium mb-1">Force Password Reset</h4>
                  <p className="text-xs text-muted-foreground">Invalidate current password and email a reset link.</p>
                </div>
                <Button variant="outline" size="sm" onClick={() => handleAction("reset_pwd", "Password reset email sent")} disabled={isActioning === "reset_pwd"}>
                  {isActioning === "reset_pwd" ? <Loader2 className="h-4 w-4 mr-2 animate-spin" /> : <KeyRound className="h-4 w-4 mr-2" />}
                  Reset
                </Button>
              </div>

              <div className="space-y-3">
                <h4 className="text-sm font-medium">Active Sessions</h4>
                <SessionManager sessions={mockSessions} onRevoke={(id) => handleAction("revoke_sess_" + id, "Session " + id + " revoked")} />
              </div>

              <MfaPanel factors={["Authenticator App", "WebAuthn (TouchID)"]} onGenerateRecovery={async () => {
                await new Promise(r => setTimeout(r, 800));
                return ["8A9F-2B3C", "9K2L-5M7N", "1Q8W-4E5R", "7U9I-0O2P"];
              }} />

              <div className="flex items-center justify-between rounded-md border border-destructive/20 bg-destructive/10 p-4 mt-4">
                <div>
                  <h4 className="text-sm font-medium text-destructive mb-1">Suspend Account</h4>
                  <p className="text-xs text-muted-foreground">Block new logins and revoke active sessions.</p>
                </div>
                <Button variant="destructive" size="sm" onClick={() => handleAction("suspend", "Account suspended")} disabled={isActioning === "suspend"}>
                  {isActioning === "suspend" ? <Loader2 className="h-4 w-4 mr-2 animate-spin" /> : <UserX className="h-4 w-4 mr-2" />}
                  Suspend
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
