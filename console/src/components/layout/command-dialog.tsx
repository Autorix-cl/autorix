"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import { Users, Network, KeyRound, Shield, Layers, Building2, LayoutDashboard, Search, ArrowRight } from "lucide-react";
import { useTranslation } from "@/lib/i18n";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";

interface CommandDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function CommandDialog({ open, onOpenChange }: CommandDialogProps) {
  const router = useRouter();
  const { t } = useTranslation();
  const [query, setQuery] = React.useState("");

  const quickLinks = [
    {
      title: t("nav.dashboard"),
      desc: t("dashboard.subtitle"),
      href: "/",
      icon: LayoutDashboard,
      badge: "Overview",
    },
    {
      title: t("nav.ego"),
      desc: t("identities.subtitle"),
      href: "/identities",
      icon: Users,
      badge: ":4433",
    },
    {
      title: t("nav.nexus"),
      desc: t("permissions.subtitle"),
      href: "/permissions",
      icon: Network,
      badge: ":50051",
    },
    {
      title: t("nav.janus"),
      desc: t("oauth2.subtitle"),
      href: "/oauth2",
      icon: KeyRound,
      badge: ":4444",
    },
    {
      title: t("nav.aegis"),
      desc: t("proxyRules.subtitle"),
      href: "/proxy-rules",
      icon: Shield,
      badge: ":4455",
    },
    {
      title: t("nav.vulcan"),
      desc: t("apiKeys.subtitle"),
      href: "/api-keys",
      icon: Layers,
      badge: ":4466",
    },
    {
      title: t("nav.hermes"),
      desc: t("enterprise.subtitle"),
      href: "/enterprise",
      icon: Building2,
      badge: ":4477",
    },
    {
      title: "Policies & Themis",
      desc: "Policy compilation, evaluation and dry-run testing",
      href: "/policies",
      icon: Shield,
      badge: ":4488",
    },
    {
      title: "Console Operators & RBAC",
      desc: "Manage operator principals, break-glass access and role bindings",
      href: "/operators",
      icon: Users,
      badge: "Argus",
    },
    {
      title: "Design System & Tokens",
      desc: "Visual token palette, typography and component primitives",
      href: "/design-system",
      icon: Layers,
      badge: "UI",
    },
  ];

  const filteredLinks = quickLinks.filter(
    (item) =>
      item.title.toLowerCase().includes(query.toLowerCase()) || item.desc.toLowerCase().includes(query.toLowerCase()),
  );

  const handleSelect = (href: string) => {
    router.push(href);
    onOpenChange(false);
  };

  React.useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === "k") {
        e.preventDefault();
        onOpenChange(!open);
      }
    };
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [open, onOpenChange]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-xl p-0 gap-0 overflow-hidden border-border/80 bg-card">
        <DialogHeader className="p-4 border-b border-border/60">
          <DialogTitle className="sr-only">Quick Command Navigation</DialogTitle>
          <div className="flex items-center gap-3">
            <Search className="h-4 w-4 text-muted-foreground" />
            <Input
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder={t("header.searchPlaceholder")}
              className="border-0 bg-transparent p-0 text-sm focus-visible:ring-0 focus-visible:border-transparent placeholder:text-muted-foreground/60 shadow-none h-7"
              autoFocus
            />
          </div>
        </DialogHeader>

        <div className="max-h-80 overflow-y-auto p-2 space-y-1">
          <div className="px-2 py-1 text-[10px] font-bold uppercase tracking-wider text-muted-foreground/70">
            {t("nav.coreEngines")}
          </div>

          {filteredLinks.length === 0 ? (
            <div className="py-8 text-center text-xs text-muted-foreground">No matching modules found.</div>
          ) : (
            filteredLinks.map((item) => {
              const Icon = item.icon;
              return (
                <button
                  key={item.href}
                  onClick={() => handleSelect(item.href)}
                  className="flex w-full items-center justify-between rounded-lg p-2.5 text-left text-xs transition-colors hover:bg-accent hover:text-accent-foreground group cursor-pointer"
                >
                  <div className="flex items-center gap-3">
                    <div className="rounded-md border border-border/60 bg-muted/60 p-1.5 text-muted-foreground group-hover:text-primary group-hover:border-primary/40">
                      <Icon className="h-4 w-4" />
                    </div>
                    <div>
                      <div className="font-semibold text-foreground">{item.title}</div>
                      <div className="text-[11px] text-muted-foreground truncate max-w-sm">{item.desc}</div>
                    </div>
                  </div>
                  <div className="flex items-center gap-2">
                    <span className="rounded bg-muted px-1.5 py-0.5 text-[10px] font-mono text-muted-foreground">
                      {item.badge}
                    </span>
                    <ArrowRight className="h-3.5 w-3.5 opacity-0 group-hover:opacity-100 transition-opacity text-primary" />
                  </div>
                </button>
              );
            })
          )}
        </div>

        <div className="flex items-center justify-between border-t border-border/60 bg-muted/20 px-4 py-2 text-[11px] text-muted-foreground">
          <span>
            Navigate with <kbd className="rounded border bg-background px-1 py-0.5 font-mono">↑</kbd>{" "}
            <kbd className="rounded border bg-background px-1 py-0.5 font-mono">↓</kbd>
          </span>
          <span>
            Press <kbd className="rounded border bg-background px-1 py-0.5 font-mono">ESC</kbd> to close
          </span>
        </div>
      </DialogContent>
    </Dialog>
  );
}
