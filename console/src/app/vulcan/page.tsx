"use client";

import * as React from "react";
import { KeysTable } from "./keys-table";
import { KeyBuilderSheet } from "./key-builder-sheet";
import { AttenuationStudio } from "./attenuation-studio";
import { MacaroonInspector } from "./macaroon-inspector";
import { ScopeCatalogSheet } from "./scope-catalog-sheet";
import { Shield, ListFilter } from "lucide-react";
import { Button } from "@/components/ui/button";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

export default function VulcanPage() {
  const [queryClient] = React.useState(() => new QueryClient());
  const [scopeSheetOpen, setScopeSheetOpen] = React.useState(false);

  return (
    <QueryClientProvider client={queryClient}>
      <div className="flex-1 space-y-6 p-8">
        <div className="flex items-center justify-between space-y-2">
          <div>
            <h1 className="text-2xl font-bold tracking-tight flex items-center gap-2">
              <Shield className="h-6 w-6 text-primary" />
              Vulcan (API Keys & Macaroons)
            </h1>
            <p className="text-muted-foreground text-sm">
              Machine-to-machine credentials, decentralized capability tokens, and cryptographic attenuation.
            </p>
          </div>
          <div className="flex items-center space-x-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => setScopeSheetOpen(true)}
              className="gap-1.5 text-xs h-8"
            >
              <ListFilter className="w-3.5 h-3.5 text-primary" />
              Scope Catalogue
            </Button>
            <KeyBuilderSheet />
          </div>
        </div>

        {/* Keys Table with Telemetry and Rotation */}
        <div className="space-y-4">
          <KeysTable />
        </div>

        {/* Macaroon Inspector with Live Verification */}
        <MacaroonInspector />

        {/* Attenuation Studio with Provenance Chain */}
        <AttenuationStudio />

        {/* Scope Catalogue Management Sheet */}
        <ScopeCatalogSheet isOpen={scopeSheetOpen} onOpenChange={setScopeSheetOpen} />
      </div>
    </QueryClientProvider>
  );
}
