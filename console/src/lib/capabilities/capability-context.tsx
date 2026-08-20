"use client";

import * as React from "react";

export type EngineType = "ego" | "nexus" | "janus" | "aegis" | "vulcan" | "hermes" | "themis";
export type EngineHealthStatus = "healthy" | "degraded" | "unreachable" | "not_connected";

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
  isEngineConnected: (type: EngineType) => boolean;
  getEngineStatus: (type: EngineType) => EngineHealthStatus;
  refreshCapabilities: () => Promise<void>;
}

import { useEnvironment } from "@/lib/environment/environment-context";

const DEFAULT_ENGINES: Record<EngineType, EngineCapabilityInfo> = {
  ego: { type: "ego", displayName: "Ego (Identity)", status: "healthy", instanceCount: 1, capabilities: ["identities", "traits", "mfa"] },
  nexus: { type: "nexus", displayName: "Nexus (Zanzibar)", status: "healthy", instanceCount: 1, capabilities: ["tuples", "expand", "explain"] },
  janus: { type: "janus", displayName: "Janus (OAuth2/OIDC)", status: "healthy", instanceCount: 1, capabilities: ["clients", "tokens", "jwks"] },
  aegis: { type: "aegis", displayName: "Aegis (Proxy)", status: "healthy", instanceCount: 1, capabilities: ["rules", "traffic", "handlers"] },
  vulcan: { type: "vulcan", displayName: "Vulcan (API Keys)", status: "healthy", instanceCount: 1, capabilities: ["keys", "macaroons", "quotas"] },
  hermes: { type: "hermes", displayName: "Hermes (Enterprise)", status: "healthy", instanceCount: 1, capabilities: ["saml", "scim", "certificates"] },
  themis: { type: "themis", displayName: "Themis (Policy/CEL)", status: "healthy", instanceCount: 1, capabilities: ["policies", "cel", "dryrun"] },
};

function getDisconnectedEngines(): Record<EngineType, EngineCapabilityInfo> {
  const disconnected = {} as Record<EngineType, EngineCapabilityInfo>;
  for (const [key, val] of Object.entries(DEFAULT_ENGINES)) {
    disconnected[key as EngineType] = {
      ...val,
      status: "not_connected",
      instanceCount: 0,
    };
  }
  return disconnected;
}

const CapabilityContext = React.createContext<CapabilityContextValue | null>(null);

export function CapabilityProvider({ children }: { children: React.ReactNode }) {
  const { currentEnv } = useEnvironment();
  const [engines, setEngines] = React.useState<Record<EngineType, EngineCapabilityInfo>>(() =>
    currentEnv.id === "prod" ? DEFAULT_ENGINES : getDisconnectedEngines(),
  );
  const [isLoading, setIsLoading] = React.useState(false);

  const refreshCapabilities = React.useCallback(async () => {
    try {
      setIsLoading(true);
      const res = await fetch(`/api/fleet/engines?environment=${currentEnv.id}`);
      if (res.ok) {
        const data = await res.json();
        if (Array.isArray(data) && data.length > 0) {
          const updated = { ...DEFAULT_ENGINES };
          for (const item of data) {
            if (item.engine_type && updated[item.engine_type as EngineType]) {
              updated[item.engine_type as EngineType] = {
                ...updated[item.engine_type as EngineType],
                status: item.status || "healthy",
                instanceCount: item.instance_count || 1,
                version: item.version,
              };
            }
          }
          setEngines(updated);
          return;
        }
      }
      setEngines(currentEnv.id === "prod" ? DEFAULT_ENGINES : getDisconnectedEngines());
    } catch {
      setEngines(currentEnv.id === "prod" ? DEFAULT_ENGINES : getDisconnectedEngines());
    } finally {
      setIsLoading(false);
    }
  }, [currentEnv.id]);

  React.useEffect(() => {
    let cancelled = false;

    if (currentEnv.id !== "prod") {
      setEngines(getDisconnectedEngines());
    } else {
      setEngines(DEFAULT_ENGINES);
    }

    async function fetchEngines() {
      try {
        setIsLoading(true);
        const res = await fetch(`/api/fleet/engines?environment=${currentEnv.id}`);
        if (cancelled) return;
        if (res.ok) {
          const data = await res.json();
          if (cancelled) return;
          if (Array.isArray(data) && data.length > 0) {
            const updated = { ...DEFAULT_ENGINES };
            for (const item of data) {
              if (item.engine_type && updated[item.engine_type as EngineType]) {
                updated[item.engine_type as EngineType] = {
                  ...updated[item.engine_type as EngineType],
                  status: item.status || "healthy",
                  instanceCount: item.instance_count || 1,
                  version: item.version,
                };
              }
            }
            setEngines(updated);
            return;
          }
        }
        if (!cancelled) {
          setEngines(currentEnv.id === "prod" ? DEFAULT_ENGINES : getDisconnectedEngines());
        }
      } catch {
        if (!cancelled) {
          setEngines(currentEnv.id === "prod" ? DEFAULT_ENGINES : getDisconnectedEngines());
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
  }, [currentEnv.id]);

  const isEngineConnected = React.useCallback(
    (type: EngineType) => {
      const eng = engines[type];
      return eng ? eng.status !== "not_connected" && eng.status !== "unreachable" : false;
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
