import * as React from "react";
import { Button } from "@/components/ui/button";
import { LogOut } from "lucide-react";

export interface Session {
  id: string;
  ip: string;
  userAgent: string;
  lastAccess: string;
}

export interface SessionManagerProps {
  sessions: Session[];
  onRevoke: (id: string) => void;
}

export function SessionManager({ sessions, onRevoke }: SessionManagerProps) {
  if (sessions.length === 0) {
    return <div className="text-sm text-muted-foreground p-4">No active sessions found.</div>;
  }

  return (
    <div className="space-y-3">
      {sessions.map((session) => (
        <div key={session.id} className="flex items-center justify-between rounded-md border p-3 bg-card">
          <div className="flex flex-col gap-1">
            <span className="text-sm font-medium">{session.userAgent}</span>
            <div className="flex items-center gap-2 text-xs text-muted-foreground">
              <span>{session.ip}</span>
              <span>&bull;</span>
              <span>Last active: {session.lastAccess}</span>
            </div>
          </div>
          <Button 
            variant="outline" 
            size="sm" 
            onClick={() => onRevoke(session.id)}
            aria-label={`Revoke session for ${session.ip}`}
          >
            <LogOut className="h-4 w-4 mr-2" />
            Revoke
          </Button>
        </div>
      ))}
    </div>
  );
}
