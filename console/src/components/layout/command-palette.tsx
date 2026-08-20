"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import {
  Search,
  LayoutDashboard,
  Users,
  Network,
  KeyRound,
  Shield,
  Layers,
  Building2,
  Scale,
  Sparkles,
  Sun,
  Moon,
  LogOut,
  X,
} from "lucide-react";
import { useTheme } from "@/lib/theme-provider";

interface CommandPaletteProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function CommandPalette({ open, onOpenChange }: CommandPaletteProps) {
  const router = useRouter();
  const { setTheme, resolvedTheme } = useTheme();
  const [search, setSearch] = React.useState("");

  React.useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === "k") {
        e.preventDefault();
        onOpenChange(!open);
      }
      if (e.key === "Escape" && open) {
        onOpenChange(false);
      }
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [open, onOpenChange]);

  if (!open) return null;

  const navigationItems = [
    { label: "Dashboard", href: "/", icon: LayoutDashboard, category: "Navigation" },
    { label: "Identities (Ego)", href: "/identities", icon: Users, category: "Navigation" },
    { label: "Permissions (Nexus)", href: "/permissions", icon: Network, category: "Navigation" },
    { label: "OAuth2 & OIDC (Janus)", href: "/oauth2", icon: KeyRound, category: "Navigation" },
    { label: "Proxy Rules (Aegis)", href: "/proxy-rules", icon: Shield, category: "Navigation" },
    { label: "API Keys (Vulcan)", href: "/api-keys", icon: Layers, category: "Navigation" },
    { label: "Enterprise SSO & SCIM (Hermes)", href: "/enterprise", icon: Building2, category: "Navigation" },
    { label: "Policies (Themis)", href: "/policies", icon: Scale, category: "Navigation" },
    { label: "Operators & RBAC", href: "/operators", icon: Users, category: "Control Plane" },
    { label: "Design System & Tokens", href: "/design-system", icon: Sparkles, category: "Developer" },
  ];

  const actions = [
    {
      label: resolvedTheme === "dark" ? "Switch to Light Theme" : "Switch to Dark Theme",
      action: () => setTheme(resolvedTheme === "dark" ? "light" : "dark"),
      icon: resolvedTheme === "dark" ? Sun : Moon,
      category: "Preferences",
    },
    {
      label: "Log Out of Console",
      action: async () => {
        await fetch("/api/auth/logout", { method: "POST" });
        window.location.href = "/login";
      },
      icon: LogOut,
      category: "Account",
    },
  ];

  const filteredNav = navigationItems.filter((item) =>
    item.label.toLowerCase().includes(search.toLowerCase())
  );

  const filteredActions = actions.filter((act) =>
    act.label.toLowerCase().includes(search.toLowerCase())
  );

  const handleSelectNav = (href: string) => {
    onOpenChange(false);
    router.push(href);
  };

  const handleSelectAction = (actionFn: () => void) => {
    onOpenChange(false);
    actionFn();
  };

  return (
    <div className="fixed inset-0 z-50 bg-black/60 backdrop-blur-xs flex items-start justify-center pt-24 p-4 animate-in fade-in">
      <div
        className="w-full max-w-lg rounded-xl border border-border/80 bg-card shadow-2xl overflow-hidden animate-in zoom-in-95"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Search Input Header */}
        <div className="flex items-center px-4 border-b border-border/70 bg-card">
          <Search className="h-4 w-4 text-muted-foreground shrink-0 mr-3" />
          <input
            type="text"
            placeholder="Type a command or search console pages..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            autoFocus
            className="h-12 w-full bg-transparent text-sm text-foreground placeholder:text-muted-foreground outline-none font-sans"
          />
          <button
            onClick={() => onOpenChange(false)}
            className="p-1 rounded-md text-muted-foreground hover:text-foreground"
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        {/* Results List */}
        <div className="max-h-80 overflow-y-auto p-2 space-y-3">
          {filteredNav.length > 0 && (
            <div>
              <div className="px-2 py-1 text-[10px] font-semibold uppercase text-muted-foreground font-mono">
                Pages & Navigation
              </div>
              <div className="space-y-0.5">
                {filteredNav.map((item) => {
                  const Icon = item.icon;
                  return (
                    <button
                      key={item.href}
                      onClick={() => handleSelectNav(item.href)}
                      className="w-full flex items-center gap-3 px-3 py-2 rounded-lg text-xs font-medium text-foreground hover:bg-primary/10 hover:text-primary transition-colors text-left"
                    >
                      <Icon className="h-4 w-4 shrink-0 text-muted-foreground" />
                      <span>{item.label}</span>
                    </button>
                  );
                })}
              </div>
            </div>
          )}

          {filteredActions.length > 0 && (
            <div>
              <div className="px-2 py-1 text-[10px] font-semibold uppercase text-muted-foreground font-mono">
                Actions & Quick Commands
              </div>
              <div className="space-y-0.5">
                {filteredActions.map((act) => {
                  const Icon = act.icon;
                  return (
                    <button
                      key={act.label}
                      onClick={() => handleSelectAction(act.action)}
                      className="w-full flex items-center gap-3 px-3 py-2 rounded-lg text-xs font-medium text-foreground hover:bg-primary/10 hover:text-primary transition-colors text-left"
                    >
                      <Icon className="h-4 w-4 shrink-0 text-muted-foreground" />
                      <span>{act.label}</span>
                    </button>
                  );
                })}
              </div>
            </div>
          )}

          {filteredNav.length === 0 && filteredActions.length === 0 && (
            <div className="py-8 text-center text-xs text-muted-foreground">
              No matching commands or pages found.
            </div>
          )}
        </div>

        {/* Footer info */}
        <div className="flex items-center justify-between px-4 py-2 border-t border-border/60 bg-muted/40 text-[11px] text-muted-foreground font-mono">
          <span>Navigate with arrows</span>
          <span>ESC to dismiss</span>
        </div>
      </div>
    </div>
  );
}
