import * as React from "react";
import { Activity } from "lucide-react";

export interface AuditEvent {
  id: string;
  action: string;
  actor: string;
  timestamp: string;
}

export interface AuditTimelineProps {
  events: AuditEvent[];
}

export function AuditTimeline({ events }: AuditTimelineProps) {
  if (events.length === 0) {
    return <div className="text-sm text-muted-foreground p-4 text-center border rounded-md border-dashed">No recent activity.</div>;
  }

  return (
    <div className="space-y-4">
      {events.map((evt) => (
        <div key={evt.id} className="flex gap-4">
          <div className="mt-1 bg-muted rounded-full p-1">
            <Activity className="h-4 w-4 text-muted-foreground" />
          </div>
          <div className="flex flex-col space-y-1">
            <span className="text-sm font-medium">{evt.action}</span>
            <div className="flex items-center gap-2 text-xs text-muted-foreground">
              <span>by {evt.actor}</span>
              <span>&bull;</span>
              <span>{evt.timestamp}</span>
            </div>
          </div>
        </div>
      ))}
    </div>
  );
}
