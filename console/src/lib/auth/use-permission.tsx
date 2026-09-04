"use client";

import React, { createContext, useContext, useEffect, useState, useCallback } from "react";
import { usePathname } from "next/navigation";
import type { Operator, OperatorRole } from "./types";
import { hasPermission } from "./types";
import { redirectToLogin } from "../api/client";

const PUBLIC_ROUTES = ["/login", "/setup", "/session-expired", "/403"];

interface AuthContextType {
  operator: Operator | null;
  isLoading: boolean;
  can: (permission: string) => boolean;
  hasRole: (roles: OperatorRole | OperatorRole[]) => boolean;
  checkSession: () => Promise<boolean>;
}

const AuthContext = createContext<AuthContextType>({
  operator: null,
  isLoading: true,
  can: () => false,
  hasRole: () => false,
  checkSession: async () => false,
});

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [operator, setOperator] = useState<Operator | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  const pathname = usePathname();
  const currentPath = pathname || "";

  const checkSession = useCallback(async (): Promise<boolean> => {
    try {
      const res = await fetch("/api/auth/me");
      if (res.ok) {
        const data = await res.json();
        if (data?.authenticated && data.operator) {
          setOperator(data.operator);
          return true;
        }
      }
      setOperator(null);
      const isPublic = PUBLIC_ROUTES.some(
        (p) => currentPath === p || currentPath?.startsWith(p + "/")
      );
      if (!isPublic) {
        redirectToLogin(currentPath);
      }
      return false;
    } catch {
      setOperator(null);
      return false;
    } finally {
      setIsLoading(false);
    }
  }, [currentPath]);

  useEffect(() => {
    checkSession();
  }, [checkSession]);

  useEffect(() => {
    if (typeof window === "undefined") return;
    const isPublic = PUBLIC_ROUTES.some(
      (p) => currentPath === p || currentPath?.startsWith(p + "/")
    );
    if (isPublic) return;

    // Check session on window focus / visibility change
    const onFocus = () => {
      checkSession();
    };
    const onVisibilityChange = () => {
      if (document.visibilityState === "visible") {
        checkSession();
      }
    };

    window.addEventListener("focus", onFocus);
    document.addEventListener("visibilitychange", onVisibilityChange);

    // Heartbeat every 45 seconds
    const interval = setInterval(() => {
      checkSession();
    }, 45_000);

    return () => {
      window.removeEventListener("focus", onFocus);
      document.removeEventListener("visibilitychange", onVisibilityChange);
      clearInterval(interval);
    };
  }, [currentPath, checkSession]);

  const can = (permission: string): boolean => {
    if (!operator) return false;
    return hasPermission(operator.role, permission);
  };

  const hasRole = (roles: OperatorRole | OperatorRole[]): boolean => {
    if (!operator) return false;
    const list = Array.isArray(roles) ? roles : [roles];
    return list.includes(operator.role);
  };

  return (
    <AuthContext.Provider value={{ operator, isLoading, can, hasRole, checkSession }}>
      {children}
    </AuthContext.Provider>
  );
}

export function usePermission() {
  return useContext(AuthContext);
}

export interface CanProps {
  perform: string;
  children: React.ReactNode;
  fallback?: React.ReactNode;
}

export function Can({ perform, children, fallback = null }: CanProps) {
  const { can, isLoading } = usePermission();
  if (isLoading) return null;
  if (!can(perform)) return <>{fallback}</>;
  return <>{children}</>;
}
