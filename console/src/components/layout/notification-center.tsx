"use client";

import * as React from "react";
import { Bell, ShieldAlert, CheckCircle2, Info, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { formatRelativeTime } from "@/lib/format/date-time";

export interface FleetNotification {
  id: string;
  type: "security" | "success" | "info";
  title: string;
  message: string;
  timestamp: string;
  read: boolean;
}

const INITIAL_NOTIFICATIONS: FleetNotification[] = [
  {
    id: "notif-1",
    type: "security",
    title: "Break-Glass Operator Authenticated",
    message: "Root owner session established via local credential fallback.",
    timestamp: new Date(Date.now() - 12 * 60 * 1000).toISOString(),
    read: false,
  },
  {
    id: "notif-2",
    type: "success",
    title: "Cluster Engines Synchronized",
    message: "6 IAM engines discovered and registered in Argus control plane.",
    timestamp: new Date(Date.now() - 45 * 60 * 1000).toISOString(),
    read: true,
  },
];

export function NotificationCenter() {
  const [notifications, setNotifications] = React.useState<FleetNotification[]>(INITIAL_NOTIFICATIONS);

  const unreadCount = notifications.filter((n) => !n.read).length;

  const markAllRead = () => {
    setNotifications((prev) => prev.map((n) => ({ ...n, read: true })));
  };

  const clearNotification = (id: string, e: React.MouseEvent) => {
    e.stopPropagation();
    setNotifications((prev) => prev.filter((n) => n.id !== id));
  };

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          variant="ghost"
          size="icon-sm"
          className="h-8 w-8 relative text-muted-foreground hover:text-foreground"
          aria-label="Notifications"
        >
          <Bell className="h-4 w-4" />
          {unreadCount > 0 && (
            <span className="absolute top-1.5 right-1.5 h-2 w-2 rounded-full bg-blue-500 ring-2 ring-background" />
          )}
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-84 p-0 overflow-hidden bg-card border-border/80 shadow-xl">
        <div className="flex items-center justify-between px-4 py-3 border-b border-border/60 bg-muted/30">
          <div className="flex items-center gap-2">
            <span className="text-xs font-bold text-foreground">Fleet Notifications</span>
            {unreadCount > 0 && (
              <Badge variant="default" className="text-[10px] px-1.5 py-0">
                {unreadCount} New
              </Badge>
            )}
          </div>
          {unreadCount > 0 && (
            <button
              type="button"
              onClick={markAllRead}
              className="text-[11px] text-primary hover:underline font-medium"
            >
              Mark all read
            </button>
          )}
        </div>

        <div className="max-h-72 overflow-y-auto divide-y divide-border/50">
          {notifications.length === 0 ? (
            <div className="py-8 text-center text-xs text-muted-foreground">
              No notifications in fleet inbox.
            </div>
          ) : (
            notifications.map((n) => (
              <div
                key={n.id}
                className={`p-3 text-xs transition-colors flex items-start gap-2.5 ${
                  n.read ? "bg-card hover:bg-muted/30" : "bg-primary/5 hover:bg-primary/10"
                }`}
              >
                {n.type === "security" ? (
                  <ShieldAlert className="h-4 w-4 text-amber-400 shrink-0 mt-0.5" />
                ) : n.type === "success" ? (
                  <CheckCircle2 className="h-4 w-4 text-emerald-400 shrink-0 mt-0.5" />
                ) : (
                  <Info className="h-4 w-4 text-blue-400 shrink-0 mt-0.5" />
                )}
                <div className="flex-1 min-w-0">
                  <div className="flex items-center justify-between gap-1">
                    <span className="font-semibold text-foreground truncate">{n.title}</span>
                    <span className="text-[10px] text-muted-foreground font-mono shrink-0">
                      {formatRelativeTime(n.timestamp, "en")}
                    </span>
                  </div>
                  <p className="text-[11px] text-muted-foreground mt-0.5 line-clamp-2">{n.message}</p>
                </div>
                <button
                  type="button"
                  onClick={(e) => clearNotification(n.id, e)}
                  className="text-muted-foreground hover:text-foreground opacity-60 hover:opacity-100"
                >
                  <X className="h-3 w-3" />
                </button>
              </div>
            ))
          )}
        </div>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
