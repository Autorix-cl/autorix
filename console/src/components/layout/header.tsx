"use client";

import * as React from "react";
import { Search, Globe, Check, Database, LogOut, ChevronDown, Sun, Moon, Laptop, Menu } from "lucide-react";
import { useTranslation } from "@/lib/i18n";
import { useTheme } from "@/lib/theme-provider";
import { usePermission } from "@/lib/auth/use-permission";
import { useApiQuery } from "@/lib/query/use-api-query";
import { fetchAndParse } from "@/lib/api/schema";
import { healthResponseSchema } from "@/lib/api/schemas/health";
import { cn } from "@/lib/utils";
import { EnvironmentSwitcher } from "./environment-switcher";
import { NotificationCenter } from "./notification-center";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { useEnvironment } from "@/lib/environment/environment-context";

interface HeaderProps {
  onOpenSearch?: () => void;
  onOpenMobileNav?: () => void;
  mobileNavOpen?: boolean;
}

function readCsrfCookie(): string | undefined {
  if (typeof document === "undefined") return undefined;
  const match = document.cookie.match(/(?:^|;\s*)autorix_csrf=([^;]+)/);
  return match ? decodeURIComponent(match[1]) : undefined;
}

export function Header({ onOpenSearch, onOpenMobileNav, mobileNavOpen = false }: HeaderProps) {
  const { t, locale, setLocale } = useTranslation();
  const { theme, setTheme, resolvedTheme } = useTheme();
  const { currentEnv, isProduction } = useEnvironment();

  const { data: healthData } = useApiQuery(["health"], () => fetchAndParse("/api/health", healthResponseSchema));
  const services = healthData?.services ?? [];
  const isHealthy = services.length > 0 && services.every((s) => s.status === "healthy");
  const isEnvHealthy = isProduction && isHealthy;

  return (
    <header className="sticky top-0 z-30 flex h-16 w-full items-center justify-between gap-2 border-b border-border/70 bg-background/80 px-4 sm:px-6 backdrop-blur-md">
      {/* Left: Mobile Menu + Environment Switcher + Quick Search Button */}
      <div className="flex items-center gap-2 sm:gap-3 flex-1 min-w-0 max-w-lg">
        {onOpenMobileNav && (
          <Button
            type="button"
            variant="ghost"
            size="icon-sm"
            onClick={onOpenMobileNav}
            aria-label="Open navigation menu"
            aria-expanded={mobileNavOpen}
            className="h-8 w-8 shrink-0 lg:hidden text-muted-foreground hover:text-foreground"
          >
            <Menu className="h-4 w-4" aria-hidden="true" />
          </Button>
        )}

        <EnvironmentSwitcher />

        <button
          type="button"
          onClick={onOpenSearch}
          className="flex h-9 w-full min-w-0 items-center justify-between rounded-lg border border-border/80 bg-muted/40 px-3 text-xs text-muted-foreground transition-colors hover:border-primary/50 hover:bg-muted/70 hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
          aria-label="Open command palette"
        >
          <div className="flex items-center gap-2 min-w-0">
            <Search className="h-3.5 w-3.5 shrink-0" aria-hidden="true" />
            <span className="truncate">{t("header.searchPlaceholder")}</span>
          </div>
          <kbd className="pointer-events-none hidden sm:inline-flex rounded border border-border bg-background px-1.5 py-0.5 text-[10px] font-mono text-muted-foreground font-semibold shadow-xs">
            ⌘K
          </kbd>
        </button>
      </div>

      {/* Right: Telemetry, Language, Theme, Alerts, User Profile */}
      <div className="flex items-center gap-2 sm:gap-3 shrink-0">
        {/* Theme Switcher */}
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button
              type="button"
              variant="ghost"
              size="icon-sm"
              className="h-8 w-8 text-muted-foreground hover:text-foreground"
              aria-label="Toggle theme"
            >
              {resolvedTheme === "dark" ? (
                <Moon className="h-4 w-4 text-blue-400" aria-hidden="true" />
              ) : (
                <Sun className="h-4 w-4 text-amber-500" aria-hidden="true" />
              )}
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" className="w-36">
            <DropdownMenuLabel className="text-[10px] uppercase text-muted-foreground">Theme</DropdownMenuLabel>
            <DropdownMenuSeparator />
            <DropdownMenuItem
              onClick={() => setTheme("light")}
              className="flex items-center justify-between text-xs cursor-pointer"
            >
              <div className="flex items-center gap-2">
                <Sun className="h-3.5 w-3.5" aria-hidden="true" />
                <span>Light</span>
              </div>
              {theme === "light" && <Check className="h-3.5 w-3.5 text-primary" aria-hidden="true" />}
            </DropdownMenuItem>
            <DropdownMenuItem
              onClick={() => setTheme("dark")}
              className="flex items-center justify-between text-xs cursor-pointer"
            >
              <div className="flex items-center gap-2">
                <Moon className="h-3.5 w-3.5" aria-hidden="true" />
                <span>Dark</span>
              </div>
              {theme === "dark" && <Check className="h-3.5 w-3.5 text-primary" aria-hidden="true" />}
            </DropdownMenuItem>
            <DropdownMenuItem
              onClick={() => setTheme("system")}
              className="flex items-center justify-between text-xs cursor-pointer"
            >
              <div className="flex items-center gap-2">
                <Laptop className="h-3.5 w-3.5" aria-hidden="true" />
                <span>System</span>
              </div>
              {theme === "system" && <Check className="h-3.5 w-3.5 text-primary" aria-hidden="true" />}
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>

        {/* Cluster / PostgreSQL Live Health Indicator */}
        <div className="hidden xl:flex items-center gap-2 rounded-full border border-border/60 bg-muted/30 px-3 py-1 text-xs">
          <Database
            className={cn("h-3.5 w-3.5", isEnvHealthy ? "text-emerald-400" : "text-muted-foreground")}
            aria-hidden="true"
          />
          <span className="text-muted-foreground text-[11px]">PostgreSQL:</span>
          <span
            className={cn(
              "font-medium font-mono text-[11px] flex items-center gap-1.5",
              isEnvHealthy ? "text-emerald-400" : "text-muted-foreground",
            )}
          >
            <span
              className={cn(
                "inline-block h-1.5 w-1.5 rounded-full",
                isEnvHealthy ? "bg-emerald-400 animate-pulse" : "bg-muted-foreground/50",
              )}
              aria-hidden="true"
            />
            {isEnvHealthy
              ? t("header.dbConnected")
              : isProduction
                ? t("header.dbStatusUnavailable")
                : `Isolated (${currentEnv.name})`}
          </span>
        </div>

        {/* Language Switcher Dropdown */}
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button
              type="button"
              variant="outline"
              size="sm"
              aria-label={`Change language (current: ${locale.toUpperCase()})`}
              className="h-8 gap-1.5 px-2.5 text-xs font-semibold border-border/80 bg-card/60"
            >
              <Globe className="h-3.5 w-3.5 text-primary" aria-hidden="true" />
              <span>{locale.toUpperCase()}</span>
              <ChevronDown className="h-3 w-3 opacity-50" aria-hidden="true" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" className="w-40">
            <DropdownMenuLabel className="text-[10px] uppercase text-muted-foreground">
              {t("common.language")}
            </DropdownMenuLabel>
            <DropdownMenuSeparator />
            <DropdownMenuItem
              onClick={() => setLocale("en")}
              className="flex items-center justify-between text-xs cursor-pointer font-medium"
            >
              <div className="flex items-center gap-2">
                <span aria-hidden="true">🇺🇸</span>
                <span>English (US)</span>
              </div>
              {locale === "en" && <Check className="h-3.5 w-3.5 text-primary" aria-hidden="true" />}
            </DropdownMenuItem>
            <DropdownMenuItem
              onClick={() => setLocale("es")}
              className="flex items-center justify-between text-xs cursor-pointer font-medium"
            >
              <div className="flex items-center gap-2">
                <span aria-hidden="true">🇪🇸</span>
                <span>Español</span>
              </div>
              {locale === "es" && <Check className="h-3.5 w-3.5 text-primary" aria-hidden="true" />}
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>

        {/* Notifications / Alerts Inbox */}
        <NotificationCenter />

        <div className="h-4 w-px bg-border/80" aria-hidden="true" />

        {/* Operator Profile with shared AuthProvider session */}
        <OperatorProfileDropdown t={t} />
      </div>
    </header>
  );
}

function OperatorProfileDropdown({ t }: { t: (key: string) => string }) {
  const { operator } = usePermission();

  const handleLogout = async () => {
    try {
      const csrf = readCsrfCookie();
      await fetch("/api/auth/logout", {
        method: "POST",
        headers: csrf ? { "X-CSRF-Token": csrf } : undefined,
      });
    } finally {
      window.location.href = "/login";
    }
  };

  const initials = operator?.name
    ? operator.name
        .split(" ")
        .map((n) => n[0])
        .join("")
        .toUpperCase()
        .slice(0, 2)
    : "OP";

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button
          type="button"
          className="flex items-center gap-2 rounded-lg p-1 transition-colors hover:bg-muted/50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring cursor-pointer"
          aria-label="Operator Profile"
        >
          <Avatar className="h-7 w-7 border-amber-500/40 bg-amber-500/10">
            <AvatarFallback className="text-[11px] font-bold text-amber-400">{initials}</AvatarFallback>
          </Avatar>
          <div className="hidden text-left md:block">
            <div className="text-xs font-semibold leading-none text-foreground">{operator?.name || "Operator"}</div>
            <div className="text-[10px] text-muted-foreground leading-none mt-1 font-mono">
              {operator?.email || "auth:active"}
            </div>
          </div>
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-56">
        <div className="px-2 py-1.5">
          <div className="flex items-center justify-between">
            <p className="text-xs font-semibold text-foreground">{operator?.name || "Operator"}</p>
            <span className="text-[9px] uppercase font-mono px-1.5 py-0.5 rounded bg-amber-500/10 text-amber-400 border border-amber-500/20 font-semibold">
              {operator?.role || "local"}
            </span>
          </div>
          <p className="text-[10px] text-muted-foreground font-mono truncate mt-0.5">{operator?.email}</p>
        </div>
        <DropdownMenuSeparator />
        <DropdownMenuItem
          onClick={handleLogout}
          className="text-xs text-destructive flex items-center gap-2 cursor-pointer"
        >
          <LogOut className="h-3.5 w-3.5" aria-hidden="true" />
          <span>{t("header.logout")}</span>
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
