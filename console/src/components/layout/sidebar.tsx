"use client";

import * as React from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  Shield,
  Users,
  KeyRound,
  Network,
  Layers,
  Building2,
  Scale,
  LayoutDashboard,
  FileCode2,
  ExternalLink,
  Server,
  Activity,
  PlusCircle,
  ScrollText,
  ShieldCheck,
  Compass,
} from "lucide-react";
import { cn } from "@/lib/utils";
import { useTranslation } from "@/lib/i18n";
import { useApiQuery } from "@/lib/query/use-api-query";
import { fetchAndParse } from "@/lib/api/schema";
import { healthResponseSchema } from "@/lib/api/schemas/health";
import { Sheet, SheetContent, SheetTitle } from "@/components/ui/sheet";

export interface SidebarProps {
  mobileOpen?: boolean;
  onMobileOpenChange?: (open: boolean) => void;
}

function isNavItemActive(pathname: string | null, href: string, allHrefs: string[]): boolean {
  if (!pathname) return false;
  if (pathname === href) return true;
  if (href === "/") return false;
  if (!pathname.startsWith(`${href}/`)) return false;
  // Avoid highlighting a parent route (e.g. /fleet) when a more specific sibling nav item matches (/fleet/enroll)
  return !allHrefs.some(
    (other) =>
      other !== href && other.startsWith(`${href}/`) && (pathname === other || pathname.startsWith(`${other}/`)),
  );
}

export function Sidebar({ mobileOpen = false, onMobileOpenChange }: SidebarProps) {
  const pathname = usePathname();
  const { t } = useTranslation();

  // Shares the ["health"] query with the Dashboard's real /health/ready probe
  // (React Query dedupes by key) instead of the "7/7 UP" claim this footer
  // used to hardcode regardless of whether any engine was actually running.
  const { data: healthData, isLoading: healthLoading } = useApiQuery(["health"], () =>
    fetchAndParse("/api/health", healthResponseSchema),
  );
  const services = healthData?.services ?? [];
  const healthyCount = services.filter((s) => s.status === "healthy").length;
  const totalCount = services.length;
  const allHealthy = totalCount > 0 && healthyCount === totalCount;

  const navItems = [
    {
      group: t("nav.overview"),
      items: [
        {
          name: t("nav.dashboard"),
          href: "/",
          icon: LayoutDashboard,
        },
        {
          name: "Observability",
          href: "/observability",
          icon: Activity,
        },
        {
          name: "Cross-Engine Explorer",
          href: "/explorer",
          icon: Compass,
        },
        {
          name: "Fleet & Engines",
          href: "/fleet",
          icon: Server,
        },
        {
          name: "Fleet Topology",
          href: "/fleet/topology",
          icon: Activity,
        },
      ],
    },
    {
      group: t("nav.coreEngines"),
      items: [
        {
          name: t("nav.ego"),
          href: "/identities",
          icon: Users,
        },
        {
          name: t("nav.nexus"),
          href: "/permissions",
          icon: Network,
        },
        {
          name: t("nav.janus"),
          href: "/oauth2",
          icon: KeyRound,
        },
        {
          name: t("nav.aegis"),
          href: "/proxy-rules",
          icon: Shield,
        },
        {
          name: t("nav.vulcan"),
          href: "/vulcan",
          icon: Layers,
        },
        {
          name: t("nav.hermes"),
          href: "/enterprise",
          icon: Building2,
        },
        {
          name: t("nav.themis"),
          href: "/policies",
          icon: Scale,
        },
      ],
    },
    {
      group: "Control Plane",
      items: [
        {
          name: "Operators & RBAC",
          href: "/operators",
          icon: Users,
        },
        {
          name: "Enroll Engine",
          href: "/fleet/enroll",
          icon: PlusCircle,
        },
      ],
    },
    {
      group: "Governance & Audit",
      items: [
        {
          name: "Audit Log",
          href: "/audit",
          icon: ScrollText,
        },
        {
          name: "Compliance",
          href: "/compliance",
          icon: ShieldCheck,
        },
      ],
    },
  ];

  const allHrefs = navItems.flatMap((section) => section.items.map((item) => item.href));

  const renderSidebarContent = (onNavigate?: () => void) => (
    <div className="flex h-full flex-col">
      {/* Brand Header */}
      <div className="flex h-16 shrink-0 items-center gap-3 border-b border-border/60 px-5 bg-card/40">
        <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-gradient-to-br from-blue-600 via-indigo-600 to-violet-600 shadow-md shadow-blue-500/20 ring-1 ring-white/20">
          <Shield className="h-5 w-5 text-white" aria-hidden="true" />
        </div>
        <div>
          <div className="flex items-center gap-1.5">
            <span className="text-sm font-bold tracking-tight text-foreground">{t("nav.brand")}</span>
            <span className="rounded bg-primary/15 px-1.5 py-0.2 text-[10px] font-bold text-primary font-mono">
              v1.0
            </span>
          </div>
          <p className="text-[11px] font-medium text-muted-foreground tracking-wide">{t("nav.suite")}</p>
        </div>
      </div>

      {/* Navigation List */}
      <nav aria-label="Primary" className="flex-1 overflow-y-auto px-3 py-4 space-y-6">
        {navItems.map((section) => (
          <div key={section.group} className="space-y-1">
            <div className="px-3 pb-2 text-[10px] font-bold uppercase tracking-wider text-muted-foreground/70">
              {section.group}
            </div>
            <ul className="space-y-1">
              {section.items.map((item) => {
                const isActive = isNavItemActive(pathname, item.href, allHrefs);
                const Icon = item.icon;

                return (
                  <li key={item.href}>
                    <Link
                      href={item.href}
                      aria-current={isActive ? "page" : undefined}
                      onClick={onNavigate}
                      className={cn(
                        "group flex items-center justify-between rounded-lg px-3 py-2 text-xs font-medium transition-all duration-150 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
                        isActive
                          ? "bg-primary/10 text-primary border border-primary/20 shadow-sm"
                          : "text-muted-foreground hover:bg-accent hover:text-foreground border border-transparent",
                      )}
                    >
                      <div className="flex items-center gap-2.5 min-w-0">
                        <Icon
                          aria-hidden="true"
                          className={cn(
                            "h-4 w-4 shrink-0 transition-colors",
                            isActive ? "text-primary" : "text-muted-foreground group-hover:text-foreground",
                          )}
                        />
                        <span className="truncate">{item.name}</span>
                      </div>
                    </Link>
                  </li>
                );
              })}
            </ul>
          </div>
        ))}

        {/* Documentation Links */}
        <div className="space-y-1 pt-2 border-t border-border/40">
          <div className="px-3 pb-2 text-[10px] font-bold uppercase tracking-wider text-muted-foreground/70">
            {t("common.documentation")}
          </div>
          <a
            href="https://github.com/Autorix-cl/autorix"
            target="_blank"
            rel="noopener noreferrer"
            className="flex items-center justify-between rounded-lg px-3 py-2 text-xs font-medium text-muted-foreground hover:bg-accent hover:text-foreground transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
          >
            <div className="flex items-center gap-2.5">
              <FileCode2 className="h-4 w-4" aria-hidden="true" />
              <span>GitHub Architecture</span>
            </div>
            <ExternalLink className="h-3 w-3 opacity-60" aria-hidden="true" />
          </a>
        </div>
      </nav>

      {/* Footer / Cluster Status — driven by the real /api/health probe */}
      <div className="shrink-0 border-t border-border/60 bg-muted/20 p-4">
        <div
          className={cn(
            "rounded-lg border p-3",
            allHealthy ? "border-emerald-500/20 bg-emerald-500/5" : "border-border/60 bg-muted/30",
          )}
        >
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <span className="relative flex h-2 w-2" aria-hidden="true">
                {allHealthy && (
                  <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-emerald-400 opacity-75" />
                )}
                <span
                  className={cn(
                    "relative inline-flex h-2 w-2 rounded-full",
                    allHealthy ? "bg-emerald-500" : totalCount === 0 ? "bg-slate-400" : "bg-amber-500",
                  )}
                />
              </span>
              <span className={cn("text-xs font-semibold", allHealthy ? "text-emerald-400" : "text-muted-foreground")}>
                {t("common.clusterStatus")}
              </span>
            </div>
            <span
              className={cn(
                "rounded px-1.5 py-0.5 text-[10px] font-bold font-mono",
                allHealthy ? "bg-emerald-500/10 text-emerald-400" : "bg-muted text-muted-foreground",
              )}
            >
              {healthLoading ? t("common.checking") : `${healthyCount}/${totalCount || 7} ONLINE`}
            </span>
          </div>
          <p className="mt-1 text-[11px] text-muted-foreground">
            {allHealthy ? t("common.perimeterActive") : t("common.perimeterUnverified")}
          </p>
        </div>
      </div>
    </div>
  );

  return (
    <>
      <aside className="hidden lg:flex w-64 flex-shrink-0 flex-col border-r border-border/70 bg-card/60 backdrop-blur-md">
        {renderSidebarContent()}
      </aside>

      <Sheet open={mobileOpen} onOpenChange={onMobileOpenChange}>
        <SheetContent side="left" className="w-72 p-0 bg-card border-r border-border/70 lg:hidden">
          <SheetTitle className="sr-only">Navigation menu</SheetTitle>
          {renderSidebarContent(() => onMobileOpenChange?.(false))}
        </SheetContent>
      </Sheet>
    </>
  );
}
