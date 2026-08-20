"use client";

import React, { createContext, useContext, useEffect, useState } from "react";
import type { Operator, OperatorRole } from "./types";
import { hasPermission } from "./types";

interface AuthContextType {
  operator: Operator | null;
  isLoading: boolean;
  can: (permission: string) => boolean;
  hasRole: (roles: OperatorRole | OperatorRole[]) => boolean;
}

const AuthContext = createContext<AuthContextType>({
  operator: null,
  isLoading: true,
  can: () => false,
  hasRole: () => false,
});

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [operator, setOperator] = useState<Operator | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    fetch("/api/auth/me")
      .then((res) => (res.ok ? res.json() : null))
      .then((data) => {
        if (data?.authenticated && data.operator) {
          setOperator(data.operator);
        } else {
          setOperator(null);
        }
      })
      .catch(() => setOperator(null))
      .finally(() => setIsLoading(false));
  }, []);

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
    <AuthContext.Provider value={{ operator, isLoading, can, hasRole }}>
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
