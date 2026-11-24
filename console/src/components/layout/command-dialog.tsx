"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import {
  Users,
  Network,
  KeyRound,
  Shield,
  Layers,
  Building2,
  LayoutDashboard,
  Search,
  ArrowRight,
  Activity,
  Compass,
  Server,
  ScrollText,
  ShieldCheck,
  Scale,
  PlusCircle,
  Loader2,
} from "lucide-react";
import { useTranslation } from "@/lib/i18n";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";

interface CommandDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

interface CommandItem {
  id: string;
  title: string;
  desc: string;
  href: string;
  badge: string;
  group: "modules" | "results";
  icon: React.ComponentType<{ className?: string }>;
}

interface RemoteSearchResult {
  id: string;
  type: string;
  title: string;
  subtitle?: string;
  href: string;
}

export function CommandDialog({ open, onOpenChange }: CommandDialogProps) {
  const router = useRouter();
  const { t } = useTranslation();
  const [query, setQuery] = React.useState("");
  const [activeIndex, setActiveIndex] = React.useState(0);
  const [remoteResults, setRemoteResults] = React.useState<RemoteSearchResult[]>([]);
  const [searching, setSearching] = React.useState(false);
  const listboxId = React.useId();

  const quickLinks: CommandItem[] = React.useMemo(
    () => [
      {
        id: "nav-dashboard",
        title: t("nav.dashboard"),
        desc: t("dashboard.subtitle"),
        href: "/",
        icon: LayoutDashboard,
        badge: "Overview",
        group: "modules",
      },
      {
        id: "nav-observability",
        title: "Observability & Diagnostics",
        desc: "Cluster telemetry, alerts, and live diagnostics",
        href: "/observability",
        icon: Activity,
        badge: "Metrics",
        group: "modules",
      },
      {
        id: "nav-explorer",
        title: "Cross-Engine Explorer",
        desc: "Unified subject view, effective access, and request simulator",
        href: "/explorer",
        icon: Compass,
        badge: "Explorer",
        group: "modules",
      },
      {
        id: "nav-fleet",
        title: "Fleet & Engines",
        desc: "Monitor registered engine instances and enrollment",
        href: "/fleet",
        icon: Server,
        badge: ":4400",
        group: "modules",
      },
      {
        id: "nav-topology",
        title: "Fleet Topology",
        desc: "Live service graph and inter-engine dependencies",
        href: "/fleet/topology",
        icon: Activity,
        badge: "Argus",
        group: "modules",
      },
      {
        id: "nav-enroll",
        title: "Enroll Engine",
        desc: "Mint single-use enrollment tokens and deployment snippets",
        href: "/fleet/enroll",
        icon: PlusCircle,
        badge: "Argus",
        group: "modules",
      },
      {
        id: "nav-identities",
        title: t("nav.ego"),
        desc: t("identities.subtitle"),
        href: "/identities",
        icon: Users,
        badge: ":4433",
        group: "modules",
      },
      {
        id: "nav-permissions",
        title: t("nav.nexus"),
        desc: t("permissions.subtitle"),
        href: "/permissions",
        icon: Network,
        badge: ":8080",
        group: "modules",
      },
      {
        id: "nav-oauth2",
        title: t("nav.janus"),
        desc: t("oauth2.subtitle"),
        href: "/oauth2",
        icon: KeyRound,
        badge: ":4444",
        group: "modules",
      },
      {
        id: "nav-proxy",
        title: t("nav.aegis"),
        desc: t("proxyRules.subtitle"),
        href: "/proxy-rules",
        icon: Shield,
        badge: ":4455",
        group: "modules",
      },
      {
        id: "nav-vulcan",
        title: t("nav.vulcan"),
        desc: t("apiKeys.subtitle"),
        href: "/vulcan",
        icon: Layers,
        badge: ":4466",
        group: "modules",
      },
      {
        id: "nav-enterprise",
        title: t("nav.hermes"),
        desc: t("enterprise.subtitle"),
        href: "/enterprise",
        icon: Building2,
        badge: ":4477",
        group: "modules",
      },
      {
        id: "nav-policies",
        title: "Policies & Themis",
        desc: "ABAC CEL policy compilation, evaluation, and testing",
        href: "/policies",
        icon: Scale,
        badge: ":4488",
        group: "modules",
      },
      {
        id: "nav-operators",
        title: "Console Operators & RBAC",
        desc: "Manage operator principals and role bindings",
        href: "/operators",
        icon: Users,
        badge: "Argus",
        group: "modules",
      },
      {
        id: "nav-audit",
        title: "Audit Log",
        desc: "Cryptographic SHA-256 hash-chained audit trail",
        href: "/audit",
        icon: ScrollText,
        badge: "Audit",
        group: "modules",
      },
      {
        id: "nav-compliance",
        title: "Compliance & Evidence",
        desc: "SOC 2 / ISO 27001 control checks and evidence exports",
        href: "/compliance",
        icon: ShieldCheck,
        badge: "GRC",
        group: "modules",
      },
    ],
    [t],
  );

  // Debounced live entity search against /api/search
  React.useEffect(() => {
    const trimmed = query.trim();
    if (!open || trimmed.length < 2) {
      setRemoteResults([]);
      setSearching(false);
      return;
    }

    const controller = new AbortController();
    const timer = setTimeout(async () => {
      setSearching(true);
      try {
        const res = await fetch(`/api/search?q=${encodeURIComponent(trimmed)}`, {
          signal: controller.signal,
        });
        if (!res.ok) {
          setRemoteResults([]);
          return;
        }
        const data = await res.json();
        const items = Array.isArray(data?.results) ? data.results : [];
        setRemoteResults(items);
      } catch (err) {
        if ((err as Error)?.name !== "AbortError") {
          setRemoteResults([]);
        }
      } finally {
        if (!controller.signal.aborted) {
          setSearching(false);
        }
      }
    }, 250);

    return () => {
      clearTimeout(timer);
      controller.abort();
    };
  }, [open, query]);

  const filteredModules = React.useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return quickLinks;
    return quickLinks.filter(
      (item) =>
        item.title.toLowerCase().includes(q) ||
        item.desc.toLowerCase().includes(q) ||
        item.badge.toLowerCase().includes(q) ||
        item.href.toLowerCase().includes(q),
    );
  }, [quickLinks, query]);

  const mappedRemote: CommandItem[] = React.useMemo(
    () =>
      remoteResults.map((r) => ({
        id: `remote-${r.type}-${r.id}`,
        title: r.title,
        desc: r.subtitle || r.href,
        href: r.href,
        badge: r.type,
        group: "results",
        icon: r.type === "identity" ? Users : r.type === "api_key" ? Layers : Search,
      })),
    [remoteResults],
  );

  const allItems = React.useMemo(() => [...mappedRemote, ...filteredModules], [mappedRemote, filteredModules]);

  React.useEffect(() => {
    setActiveIndex(0);
  }, [query, allItems.length]);

  const handleOpenChange = React.useCallback(
    (isOpen: boolean) => {
      if (!isOpen) {
        setQuery("");
        setRemoteResults([]);
        setActiveIndex(0);
      }
      onOpenChange(isOpen);
    },
    [onOpenChange],
  );

  const handleSelect = React.useCallback(
    (href: string) => {
      router.push(href);
      handleOpenChange(false);
    },
    [router, handleOpenChange],
  );

  React.useEffect(() => {
    const handleGlobalKeyDown = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === "k") {
        e.preventDefault();
        handleOpenChange(!open);
      }
    };
    window.addEventListener("keydown", handleGlobalKeyDown);
    return () => window.removeEventListener("keydown", handleGlobalKeyDown);
  }, [open, handleOpenChange]);

  const handleInputKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.nativeEvent.isComposing || allItems.length === 0) return;

    if (e.key === "ArrowDown") {
      e.preventDefault();
      setActiveIndex((prev) => (prev + 1) % allItems.length);
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      setActiveIndex((prev) => (prev - 1 + allItems.length) % allItems.length);
    } else if (e.key === "Home") {
      e.preventDefault();
      setActiveIndex(0);
    } else if (e.key === "End") {
      e.preventDefault();
      setActiveIndex(allItems.length - 1);
    } else if (e.key === "Enter") {
      e.preventDefault();
      const target = allItems[activeIndex];
      if (target) handleSelect(target.href);
    }
  };

  const activeItem = allItems[activeIndex];

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="max-w-xl p-0 gap-0 overflow-hidden border-border/80 bg-card">
        <DialogHeader className="p-4 border-b border-border/60">
          <DialogTitle className="sr-only">Quick Command Navigation</DialogTitle>
          <div className="flex items-center gap-3 pr-6">
            {searching ? (
              <Loader2 className="h-4 w-4 animate-spin text-primary shrink-0" aria-hidden="true" />
            ) : (
              <Search className="h-4 w-4 text-muted-foreground shrink-0" aria-hidden="true" />
            )}
            <Input
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              onKeyDown={handleInputKeyDown}
              placeholder={t("header.searchPlaceholder")}
              aria-label="Search modules and resources"
              role="combobox"
              aria-expanded={true}
              aria-controls={listboxId}
              aria-activedescendant={activeItem ? activeItem.id : undefined}
              autoComplete="off"
              autoCorrect="off"
              autoCapitalize="none"
              spellCheck={false}
              className="border-0 bg-transparent p-0 text-sm focus-visible:ring-0 focus-visible:border-transparent placeholder:text-muted-foreground/60 shadow-none h-7"
              autoFocus
            />
          </div>
        </DialogHeader>

        <div
          id={listboxId}
          role="listbox"
          aria-label="Search results"
          className="max-h-80 overflow-y-auto p-2 space-y-1"
        >
          {mappedRemote.length > 0 && (
            <>
              <div className="px-2 py-1 text-[10px] font-bold uppercase tracking-wider text-muted-foreground/70">
                Matching Resources
              </div>
              {mappedRemote.map((item, idx) => {
                const Icon = item.icon;
                const isSelected = idx === activeIndex;
                return (
                  <button
                    key={item.id}
                    id={item.id}
                    type="button"
                    role="option"
                    aria-selected={isSelected}
                    onMouseEnter={() => setActiveIndex(idx)}
                    onClick={() => handleSelect(item.href)}
                    className={cn(
                      "flex w-full items-center justify-between rounded-lg p-2.5 text-left text-xs transition-colors group cursor-pointer",
                      isSelected ? "bg-accent text-accent-foreground" : "hover:bg-accent/60",
                    )}
                  >
                    <div className="flex items-center gap-3 min-w-0">
                      <div className="rounded-md border border-primary/30 bg-primary/10 p-1.5 text-primary shrink-0">
                        <Icon className="h-4 w-4" />
                      </div>
                      <div className="min-w-0">
                        <div className="font-semibold text-foreground truncate">{item.title}</div>
                        <div className="text-[11px] text-muted-foreground truncate">{item.desc}</div>
                      </div>
                    </div>
                    <div className="flex items-center gap-2 shrink-0">
                      <span className="rounded bg-primary/10 px-1.5 py-0.5 text-[10px] font-mono text-primary uppercase">
                        {item.badge}
                      </span>
                      <ArrowRight
                        className={cn(
                          "h-3.5 w-3.5 text-primary transition-opacity",
                          isSelected ? "opacity-100" : "opacity-0",
                        )}
                        aria-hidden="true"
                      />
                    </div>
                  </button>
                );
              })}
            </>
          )}

          <div className="px-2 py-1 text-[10px] font-bold uppercase tracking-wider text-muted-foreground/70">
            {t("nav.coreEngines")}
          </div>

          {allItems.length === 0 ? (
            <div className="py-8 text-center text-xs text-muted-foreground">
              No matching modules or resources found.
            </div>
          ) : (
            filteredModules.map((item, localIdx) => {
              const globalIdx = mappedRemote.length + localIdx;
              const Icon = item.icon;
              const isSelected = globalIdx === activeIndex;
              return (
                <button
                  key={item.id}
                  id={item.id}
                  type="button"
                  role="option"
                  aria-selected={isSelected}
                  onMouseEnter={() => setActiveIndex(globalIdx)}
                  onClick={() => handleSelect(item.href)}
                  className={cn(
                    "flex w-full items-center justify-between rounded-lg p-2.5 text-left text-xs transition-colors group cursor-pointer",
                    isSelected ? "bg-accent text-accent-foreground" : "hover:bg-accent/60",
                  )}
                >
                  <div className="flex items-center gap-3 min-w-0">
                    <div
                      className={cn(
                        "rounded-md border p-1.5 shrink-0",
                        isSelected
                          ? "border-primary/40 bg-primary/10 text-primary"
                          : "border-border/60 bg-muted/60 text-muted-foreground",
                      )}
                    >
                      <Icon className="h-4 w-4" />
                    </div>
                    <div className="min-w-0">
                      <div className="font-semibold text-foreground truncate">{item.title}</div>
                      <div className="text-[11px] text-muted-foreground truncate max-w-sm">{item.desc}</div>
                    </div>
                  </div>
                  <div className="flex items-center gap-2 shrink-0">
                    <span className="rounded bg-muted px-1.5 py-0.5 text-[10px] font-mono text-muted-foreground">
                      {item.badge}
                    </span>
                    <ArrowRight
                      className={cn(
                        "h-3.5 w-3.5 text-primary transition-opacity",
                        isSelected ? "opacity-100" : "opacity-0",
                      )}
                      aria-hidden="true"
                    />
                  </div>
                </button>
              );
            })
          )}
        </div>

        <div className="flex items-center justify-between border-t border-border/60 bg-muted/20 px-4 py-2 text-[11px] text-muted-foreground">
          <span>
            Navigate with <kbd className="rounded border bg-background px-1 py-0.5 font-mono">↑</kbd>{" "}
            <kbd className="rounded border bg-background px-1 py-0.5 font-mono">↓</kbd> · Select with{" "}
            <kbd className="rounded border bg-background px-1 py-0.5 font-mono">↵</kbd>
          </span>
          <span>
            Press <kbd className="rounded border bg-background px-1 py-0.5 font-mono">ESC</kbd> to close
          </span>
        </div>
      </DialogContent>
    </Dialog>
  );
}
