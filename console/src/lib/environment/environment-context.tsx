"use client";

import * as React from "react";
import { usePathname } from "next/navigation";

export interface EnvironmentOption {
  id: string; // "prod" | "staging" | "dev"
  slug: string;
  name: string;
  isProduction: boolean;
}

export const DEFAULT_ENVIRONMENTS: EnvironmentOption[] = [
  { id: "prod", slug: "production", name: "Production", isProduction: true },
  { id: "staging", slug: "staging", name: "Staging", isProduction: false },
  { id: "dev", slug: "development", name: "Development", isProduction: false },
];

export interface EnvironmentContextValue {
  currentEnv: EnvironmentOption;
  setEnvironment: (envId: string) => void;
  environments: EnvironmentOption[];
  isProduction: boolean;
}

const PUBLIC_AUTH_ROUTES = ["/login", "/setup", "/403"];

function isPublicAuthPath(pathname: string | null): boolean {
  if (!pathname) return false;
  return PUBLIC_AUTH_ROUTES.some((route) => pathname === route || pathname.startsWith(route + "/"));
}

function safeGetStorage(key: string): string | null {
  try {
    return localStorage.getItem(key);
  } catch {
    return null;
  }
}

function safeSetStorage(key: string, value: string): void {
  try {
    localStorage.setItem(key, value);
  } catch {
    // Storage may be unavailable in private browsing or restricted sandboxes
  }
}

function normalizeRemoteEnvironment(item: unknown): EnvironmentOption | null {
  if (!item || typeof item !== "object") return null;
  const rec = item as Record<string, unknown>;
  const id = typeof rec.id === "string" && rec.id ? rec.id : typeof rec.slug === "string" ? rec.slug : "";
  const slug = typeof rec.slug === "string" && rec.slug ? rec.slug : id;
  const name = typeof rec.name === "string" && rec.name ? rec.name : slug;
  if (!id || !slug) return null;
  const isProduction =
    typeof rec.is_production === "boolean"
      ? rec.is_production
      : typeof rec.isProduction === "boolean"
        ? rec.isProduction
        : slug === "production" || id === "prod";
  return { id, slug, name, isProduction };
}

const EnvironmentContext = React.createContext<EnvironmentContextValue | null>(null);

export function EnvironmentProvider({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const isPublicAuth = isPublicAuthPath(pathname);
  const [environments, setEnvironments] = React.useState<EnvironmentOption[]>(DEFAULT_ENVIRONMENTS);
  const [currentEnv, setCurrentEnv] = React.useState<EnvironmentOption>(DEFAULT_ENVIRONMENTS[0]);

  React.useEffect(() => {
    const saved = safeGetStorage("autorix_active_env");
    if (saved) {
      const match = DEFAULT_ENVIRONMENTS.find((e) => e.id === saved || e.slug === saved);
      if (match) setCurrentEnv(match);
    }
  }, []);

  React.useEffect(() => {
    if (isPublicAuth || typeof fetch !== "function") return;
    let cancelled = false;

    async function loadEnvironments() {
      try {
        const res = await fetch("/api/governance/environments");
        if (!res.ok || cancelled) return;
        const json = await res.json();
        const rawList = Array.isArray(json) ? json : Array.isArray(json?.data) ? json.data : [];
        const parsed = rawList
          .map(normalizeRemoteEnvironment)
          .filter((item: EnvironmentOption | null): item is EnvironmentOption => item !== null);
        if (parsed.length === 0 || cancelled) return;

        setEnvironments(parsed);
        const saved = safeGetStorage("autorix_active_env");
        if (saved) {
          const match = parsed.find((e: EnvironmentOption) => e.id === saved || e.slug === saved);
          if (match) setCurrentEnv(match);
        }
      } catch {
        // Gracefully keep DEFAULT_ENVIRONMENTS fallback
      }
    }

    loadEnvironments();
    return () => {
      cancelled = true;
    };
  }, [isPublicAuth]);

  const setEnvironment = React.useCallback(
    (envId: string) => {
      const match = environments.find((e) => e.id === envId || e.slug === envId);
      if (match) {
        safeSetStorage("autorix_active_env", match.id);
        const secureAttr =
          typeof window !== "undefined" && window.location.protocol === "https:" ? "; Secure" : "";
        document.cookie = `autorix_active_env=${encodeURIComponent(match.id)}; path=/; max-age=31536000; SameSite=Lax${secureAttr}`;
        setCurrentEnv(match);
      }
    },
    [environments],
  );

  return (
    <EnvironmentContext.Provider
      value={{
        currentEnv,
        setEnvironment,
        environments,
        isProduction: currentEnv.isProduction,
      }}
    >
      {children}
    </EnvironmentContext.Provider>
  );
}

export function useEnvironment() {
  const ctx = React.useContext(EnvironmentContext);
  if (!ctx) {
    throw new Error("useEnvironment must be used within an EnvironmentProvider");
  }
  return ctx;
}

export function useOptionalEnvironment() {
  return React.useContext(EnvironmentContext);
}
