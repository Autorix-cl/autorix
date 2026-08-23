"use client";

import { KeysTable } from "./keys-table";
import { KeyBuilderSheet } from "./key-builder-sheet";
import { AttenuationStudio } from "./attenuation-studio";
import { Shield } from "lucide-react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useState } from "react";

export default function VulcanPage() {
  const [queryClient] = useState(() => new QueryClient());

  return (
    <QueryClientProvider client={queryClient}>
      <div className="flex-1 space-y-6 p-8">
        <div className="flex items-center justify-between space-y-2">
          <div>
            <h2 className="text-2xl font-bold tracking-tight flex items-center gap-2">
              <Shield className="h-6 w-6 text-primary" />
              Vulcan (API Keys)
            </h2>
            <p className="text-muted-foreground">
              Manage root keys and machine-to-machine authentication.
            </p>
          </div>
          <div className="flex items-center space-x-2">
            <KeyBuilderSheet />
          </div>
        </div>

        <div className="space-y-4">
          <KeysTable />
        </div>

        <AttenuationStudio />
      </div>
    </QueryClientProvider>
  );
}
