"use client";

/**
 * TanStack Query provider for the app shell (P1-S3-T1). Every page used to
 * own bespoke useState + useEffect fetching with no caching, deduplication
 * or revalidation; this replaces that with one shared client.
 */
import { useState, type ReactNode } from "react";
import { QueryClient, QueryClientProvider, QueryCache, MutationCache } from "@tanstack/react-query";
import { redirectToLogin, ApiRequestError, type ApiError } from "../api/client";

function isUnauthorizedError(error: unknown): boolean {
  if (!error || typeof error !== "object") return false;
  if (error instanceof ApiRequestError) {
    return error.kind === "unauthorized" || error.status === 401;
  }
  if ("kind" in error) {
    const kind = (error as ApiError).kind;
    if (kind === "forbidden") return false;
    if (kind === "unauthorized") return true;
  }
  if ("status" in error) {
    const status = (error as ApiError).status;
    if (status === 403) return false;
    if (status === 401) return true;
  }
  if ("message" in error && typeof (error as { message: unknown }).message === "string") {
    const msg = (error as { message: string }).message.toLowerCase();
    return msg.includes("unauthorized") || msg.includes("authentication required");
  }
  return false;
}

/** Builds a QueryClient with the console's shared defaults. */
export function createQueryClient(): QueryClient {
  return new QueryClient({
    queryCache: new QueryCache({
      onError: (error) => {
        if (isUnauthorizedError(error)) {
          redirectToLogin();
        }
      },
    }),
    mutationCache: new MutationCache({
      onError: (error) => {
        if (isUnauthorizedError(error)) {
          redirectToLogin();
        }
      },
    }),
    defaultOptions: {
      queries: {
        // Data is considered fresh for 30s: navigating between pages that
        // share a query key doesn't refetch on every mount.
        staleTime: 30_000,
        // Bounded retry — enough to ride out a blip, not enough to hammer
        // a genuinely down engine.
        retry: 2,
        refetchOnWindowFocus: true,
      },
      mutations: {
        retry: 0,
      },
    },
  });
}

export function AppQueryProvider({ children }: { children: ReactNode }) {
  const [client] = useState(createQueryClient);
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
}
