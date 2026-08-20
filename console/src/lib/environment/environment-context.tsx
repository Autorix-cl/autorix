"use client";

import * as React from "react";

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

const EnvironmentContext = React.createContext<EnvironmentContextValue | null>(null);

export function EnvironmentProvider({ children }: { children: React.ReactNode }) {
  const [currentEnv, setCurrentEnv] = React.useState<EnvironmentOption>(DEFAULT_ENVIRONMENTS[0]);

  React.useEffect(() => {
    const saved = localStorage.getItem("autorix_active_env");
    if (saved) {
      const match = DEFAULT_ENVIRONMENTS.find((e) => e.id === saved || e.slug === saved);
      if (match) setCurrentEnv(match);
    }
  }, []);

  const setEnvironment = React.useCallback((envId: string) => {
    const match = DEFAULT_ENVIRONMENTS.find((e) => e.id === envId || e.slug === envId);
    if (match) {
      localStorage.setItem("autorix_active_env", match.id);
      document.cookie = `autorix_active_env=${match.id}; path=/; max-age=31536000; SameSite=Lax`;
      setCurrentEnv(match);
    }
  }, []);

  return (
    <EnvironmentContext.Provider
      value={{
        currentEnv,
        setEnvironment,
        environments: DEFAULT_ENVIRONMENTS,
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
