import React, { createContext, useContext, useEffect, useState } from "react";
import {
  AutorixClient,
  AutorixConfig,
  CheckPermissionRequest,
  CheckPermissionResponse,
  EvaluatePolicyResponse,
  UserSession,
} from "./client";

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

export function useBatchPermissions(
  requests: Array<{ namespace: string; object: string; relation: string; context?: Record<string, unknown> }>
) {
  const { client, session } = useAutorix();
  const [results, setResults] = useState<CheckPermissionResponse[]>([]);
  const [checking, setChecking] = useState<boolean>(true);

  useEffect(() => {
    let active = true;
    if (!session?.identity?.id || requests.length === 0) {
      setResults(requests.map(() => ({ allowed: false })));
      setChecking(false);
      return;
    }

    setChecking(true);
    const reqs: CheckPermissionRequest[] = requests.map((r) => ({
      ...r,
      subject: session.identity.id,
    }));

    client
      .checkBatch(reqs)
      .then((res) => {
        if (active) {
          setResults(res);
          setChecking(false);
        }
      })
      .catch(() => {
        if (active) {
          setResults(requests.map(() => ({ allowed: false })));
          setChecking(false);
        }
      });

    return () => {
      active = false;
    };
  }, [JSON.stringify(requests), session?.identity?.id]);

  return { results, checking };
}

export function usePolicy(policyContext: Record<string, unknown>, tenantId?: string) {
  const { client } = useAutorix();
  const [result, setResult] = useState<EvaluatePolicyResponse | null>(null);
  const [evaluating, setEvaluating] = useState<boolean>(true);

  useEffect(() => {
    let active = true;
    setEvaluating(true);

    client
      .evaluatePolicy({ context: policyContext, tenantId })
      .then((res) => {
        if (active) {
          setResult(res);
          setEvaluating(false);
        }
      })
      .catch(() => {
        if (active) {
          setResult({ allPassed: false, results: [], totalEvaluated: 0 });
          setEvaluating(false);
        }
      });

  return () => {
      active = false;
    };
  }, [JSON.stringify(policyContext), tenantId]);

  return { passed: result?.allPassed ?? false, result, evaluating };
}
