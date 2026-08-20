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

export function NotificationCenter() {
  const [notifications, setNotifications] = React.useState<FleetNotification[]>([]);
  const [readIds, setReadIds] = React.useState<Set<string>>(new Set());
  const [dismissedIds, setDismissedIds] = React.useState<Set<string>>(new Set());
  const [loading, setLoading] = React.useState(true);

  // Load read and dismissed IDs from localStorage
  React.useEffect(() => {
    try {
      const storedRead = localStorage.getItem("autorix_read_notifications");
      if (storedRead) {
        setReadIds(new Set(JSON.parse(storedRead)));
      }
      const storedDismissed = localStorage.getItem("autorix_dismissed_notifications");
      if (storedDismissed) {
        setDismissedIds(new Set(JSON.parse(storedDismissed)));
      }
    } catch {
      // Ignore storage errors
    }
  }, []);

  const fetchNotifications = React.useCallback(async () => {
    try {
      const res = await fetch("/api/notifications");
      if (res.ok) {
        const data = await res.json();
        if (Array.isArray(data?.notifications)) {
          setNotifications(data.notifications);
        }
      }
    } catch {
      // Keep existing
    } finally {
      setLoading(false);
    }
  }, []);

  React.useEffect(() => {
    fetchNotifications();
    const interval = setInterval(fetchNotifications, 15000);
    return () => clearInterval(interval);
  }, [fetchNotifications]);

  const activeNotifications = notifications
    .filter((n) => !dismissedIds.has(n.id))
    .map((n) => ({
      ...n,
      read: n.read || readIds.has(n.id),
    }));

  const unreadCount = activeNotifications.filter((n) => !n.read).length;

  const markAllRead = () => {
    const nextRead = new Set(readIds);
    activeNotifications.forEach((n) => nextRead.add(n.id));
    setReadIds(nextRead);
    try {
      localStorage.setItem("autorix_read_notifications", JSON.stringify(Array.from(nextRead)));
    } catch {
      // Ignore
    }
  };

  const clearNotification = (id: string, e: React.MouseEvent) => {
    e.stopPropagation();
    const nextDismissed = new Set(dismissedIds);
    nextDismissed.add(id);
    setDismissedIds(nextDismissed);
    try {
      localStorage.setItem("autorix_dismissed_notifications", JSON.stringify(Array.from(nextDismissed)));
    } catch {
      // Ignore
    }
  };

  return (
    <DropdownMenu onOpenChange={(open) => open && fetchNotifications()}>
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
              className="text-[11px] text-primary hover:underline font-medium cursor-pointer"
            >
              Mark all read
            </button>
          )}
        </div>

        <div className="max-h-72 overflow-y-auto divide-y divide-border/50">
          {loading && activeNotifications.length === 0 ? (
            <div className="py-8 text-center text-xs text-muted-foreground">
              Loading notifications...
            </div>
          ) : activeNotifications.length === 0 ? (
            <div className="py-8 text-center text-xs text-muted-foreground">
              No notifications in fleet inbox.
            </div>
          ) : (
            activeNotifications.map((n) => (
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
                  className="text-muted-foreground hover:text-foreground opacity-60 hover:opacity-100 cursor-pointer"
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
