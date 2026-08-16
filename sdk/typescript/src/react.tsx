import React, { createContext, useContext, useEffect, useState } from "react";
import { AutorixClient, AutorixConfig, UserSession } from "./client";

interface AutorixContextValue {
  client: AutorixClient;
  session: UserSession | null;
  loading: boolean;
  isAuthenticated: boolean;
  refreshSession: () => Promise<void>;
}

const AutorixContext = createContext<AutorixContextValue | null>(null);

export interface AutorixProviderProps {
  children: React.ReactNode;
  config?: AutorixConfig;
}

export function AutorixProvider({ children, config }: AutorixProviderProps) {
  const [client] = useState(() => new AutorixClient(config));
  const [session, setSession] = useState<UserSession | null>(null);
  const [loading, setLoading] = useState(true);

  const refreshSession = async () => {
    setLoading(true);
    const s = await client.whoami();
    setSession(s);
    setLoading(false);
  };

  useEffect(() => {
    refreshSession();
  }, []);

  return (
    <AutorixContext.Provider
      value={{
        client,
        session,
        loading,
        isAuthenticated: !!session,
        refreshSession,
      }}
    >
      {children}
    </AutorixContext.Provider>
  );
}

export function useAutorix(): AutorixContextValue {
  const ctx = useContext(AutorixContext);
  if (!ctx) {
    throw new Error("useAutorix must be used within an <AutorixProvider>");
  }
  return ctx;
}

export function useSession() {
  const { session, loading, isAuthenticated, refreshSession } = useAutorix();
  return {
    session,
    user: session?.identity || null,
    loading,
    isAuthenticated,
    refreshSession,
  };
}

export function usePermission(
  namespace: string,
  object: string,
  relation: string,
  context?: Record<string, unknown>
) {
  const { client, session } = useAutorix();
  const [allowed, setAllowed] = useState<boolean>(false);
  const [checking, setChecking] = useState<boolean>(true);

  useEffect(() => {
    let active = true;
    if (!session?.identity?.id) {
      setAllowed(false);
      setChecking(false);
      return;
    }

    setChecking(true);
    client
      .check({
        namespace,
        object,
        relation,
        subject: session.identity.id,
        context,
      })
      .then((res) => {
        if (active) {
          setAllowed(res.allowed);
          setChecking(false);
        }
      })
      .catch(() => {
        if (active) {
          setAllowed(false);
          setChecking(false);
        }
      });

    return () => {
      active = false;
    };
  }, [namespace, object, relation, session?.identity?.id]);

  return { allowed, checking };
}
