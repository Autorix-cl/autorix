"use client";

import * as React from "react";
import { usePathname } from "next/navigation";

export type EngineType = "ego" | "nexus" | "janus" | "aegis" | "vulcan" | "hermes" | "themis";
export type EngineHealthStatus = "healthy" | "degraded" | "unreachable" | "not_connected" | "unknown";

export interface EngineCapabilityInfo {
  type: EngineType;
  displayName: string;
  status: EngineHealthStatus;
  instanceCount: number;
  version?: string;
  capabilities: string[];
}

export interface CapabilityContextValue {
  engines: Record<EngineType, EngineCapabilityInfo>;
  isLoading: boolean;
  error: string | null;
  isEngineConnected: (type: EngineType) => boolean;
  getEngineStatus: (type: EngineType) => EngineHealthStatus;
  refreshCapabilities: () => Promise<void>;
}

import { useEnvironment } from "@/lib/environment/environment-context";

const PUBLIC_AUTH_ROUTES = ["/login", "/setup", "/403"];

function isPublicAuthPath(pathname: string | null): boolean {
  if (!pathname) return false;
  return PUBLIC_AUTH_ROUTES.some((route) => pathname === route || pathname.startsWith(route + "/"));
}

const DEFAULT_ENGINES: Record<EngineType, EngineCapabilityInfo> = {
  ego: { type: "ego", displayName: "Ego (Identity)", status: "healthy", instanceCount: 1, capabilities: ["identities", "traits", "mfa"] },
  nexus: { type: "nexus", displayName: "Nexus (Zanzibar)", status: "healthy", instanceCount: 1, capabilities: ["tuples", "expand", "explain"] },
  janus: { type: "janus", displayName: "Janus (OAuth2/OIDC)", status: "healthy", instanceCount: 1, capabilities: ["clients", "tokens", "jwks"] },
  aegis: { type: "aegis", displayName: "Aegis (Proxy)", status: "healthy", instanceCount: 1, capabilities: ["rules", "traffic", "handlers"] },
  vulcan: { type: "vulcan", displayName: "Vulcan (API Keys)", status: "healthy", instanceCount: 1, capabilities: ["keys", "macaroons", "quotas"] },
  hermes: { type: "hermes", displayName: "Hermes (Enterprise)", status: "healthy", instanceCount: 1, capabilities: ["saml", "scim", "certificates"] },
  themis: { type: "themis", displayName: "Themis (Policy/CEL)", status: "healthy", instanceCount: 1, capabilities: ["policies", "cel", "dryrun"] },
};

function buildEnginesWithStatus(status: EngineHealthStatus): Record<EngineType, EngineCapabilityInfo> {
  const result = {} as Record<EngineType, EngineCapabilityInfo>;
  for (const [key, val] of Object.entries(DEFAULT_ENGINES)) {
    result[key as EngineType] = {
      ...val,
      status,
      instanceCount: 0,
    };
  }
  return result;
}

function getDisconnectedEngines(): Record<EngineType, EngineCapabilityInfo> {
  return buildEnginesWithStatus("not_connected");
}

function getUnreachableEngines(): Record<EngineType, EngineCapabilityInfo> {
  return buildEnginesWithStatus("unreachable");
}

const CapabilityContext = React.createContext<CapabilityContextValue | null>(null);

export function CapabilityProvider({ children }: { children: React.ReactNode }) {
  const { currentEnv } = useEnvironment();
  const pathname = usePathname();
  const isPublicAuth = isPublicAuthPath(pathname);

  const [engines, setEngines] = React.useState<Record<EngineType, EngineCapabilityInfo>>(() =>
    currentEnv.id === "prod" ? DEFAULT_ENGINES : getDisconnectedEngines(),
  );
  const [isLoading, setIsLoading] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);

  const applyFleetResponse = React.useCallback(
    (data: unknown) => {
      if (Array.isArray(data) && data.length > 0) {
        const updated = buildEnginesWithStatus("not_connected");
        for (const item of data) {
          if (item && typeof item === "object" && "engine_type" in item) {
            const engType = (item as { engine_type: string }).engine_type as EngineType;
            if (updated[engType]) {
              const rec = item as { status?: EngineHealthStatus; instance_count?: number; version?: string };
              updated[engType] = {
                ...updated[engType],
                status: rec.status || "healthy",
                instanceCount: rec.instance_count ?? 1,
                version: rec.version,
              };
            }
          }
        }
        setEngines(updated);
        setError(null);
        return;
      }

      setError(null);
      setEngines(currentEnv.id === "prod" ? DEFAULT_ENGINES : getDisconnectedEngines());
    },
    [currentEnv.id],
  );

  const refreshCapabilities = React.useCallback(async () => {
    if (isPublicAuth) {
      return;
    }

    try {
      setIsLoading(true);
      const res = await fetch(`/api/fleet/engines?environment=${encodeURIComponent(currentEnv.id)}`);
      if (!res.ok) {
        setError(`Failed to fetch fleet capabilities (HTTP ${res.status})`);
        setEngines(getUnreachableEngines());
        return;
      }
      const data = await res.json();
      applyFleetResponse(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to reach fleet capabilities endpoint");
      setEngines(getUnreachableEngines());
    } finally {
      setIsLoading(false);
    }
  }, [applyFleetResponse, currentEnv.id, isPublicAuth]);

  React.useEffect(() => {
    let cancelled = false;

    if (isPublicAuth) {
      setIsLoading(false);
      return;
    }

    if (currentEnv.id !== "prod") {
      setEngines(getDisconnectedEngines());
    } else {
      setEngines(DEFAULT_ENGINES);
    }

    async function fetchEngines() {
      try {
        setIsLoading(true);
        const res = await fetch(`/api/fleet/engines?environment=${encodeURIComponent(currentEnv.id)}`);
        if (cancelled) return;
        if (!res.ok) {
          setError(`Failed to fetch fleet capabilities (HTTP ${res.status})`);
          setEngines(getUnreachableEngines());
          return;
        }
        const data = await res.json();
        if (cancelled) return;
        applyFleetResponse(data);
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : "Failed to reach fleet capabilities endpoint");
          setEngines(getUnreachableEngines());
        }
      } finally {
        if (!cancelled) {
          setIsLoading(false);
        }
      }
    }

    fetchEngines();

    return () => {
      cancelled = true;
    };
  }, [applyFleetResponse, currentEnv.id, isPublicAuth]);

  const isEngineConnected = React.useCallback(
    (type: EngineType) => {
      const eng = engines[type];
      return eng
        ? eng.status !== "not_connected" && eng.status !== "unreachable" && eng.status !== "unknown"
        : false;
    },
    [engines]
  );

  const getEngineStatus = React.useCallback(
    (type: EngineType) => {
      return engines[type]?.status || "not_connected";
    },
    [engines]
  );

  return (
    <CapabilityContext.Provider
      value={{
        engines,
        isLoading,
        error,
        isEngineConnected,
        getEngineStatus,
        refreshCapabilities,
      }}
    >
      {children}
    </CapabilityContext.Provider>
  );
}

export function useCapabilities() {
  const ctx = React.useContext(CapabilityContext);
  if (!ctx) {
    throw new Error("useCapabilities must be used within a CapabilityProvider");
  }
  return ctx;
}
