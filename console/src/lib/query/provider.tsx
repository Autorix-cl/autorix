"use client";

/**
 * TanStack Query provider for the app shell (P1-S3-T1). Every page used to
 * own bespoke useState + useEffect fetching with no caching, deduplication
 * or revalidation; this replaces that with one shared client.
 */
import { useState, type ReactNode } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

/** Builds a QueryClient with the console's shared defaults. */
export function createQueryClient(): QueryClient {
  return new QueryClient({
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
