"use client";

import * as React from "react";
import { KeysTable } from "./keys-table";
import { KeyBuilderSheet } from "./key-builder-sheet";
import { AttenuationStudio } from "./attenuation-studio";
import { MacaroonInspector } from "./macaroon-inspector";
import { ScopeCatalogSheet } from "./scope-catalog-sheet";
import { Shield, ListFilter, Layers, KeyRound, Zap, Activity } from "lucide-react";
import { ServiceHeader } from "@/components/layout/service-header";
import { Button } from "@/components/ui/button";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

export default function VulcanPage() {
  const [queryClient] = React.useState(() => new QueryClient());
  const [scopeSheetOpen, setScopeSheetOpen] = React.useState(false);

  return (
    <QueryClientProvider client={queryClient}>
      <div className="space-y-6">
        {/* Cloud Service Header & Telemetry HUD */}
        <ServiceHeader
          serviceName="Vulcan Capability Engine"
          title="Vulcan (API Keys & Macaroons)"
          description="Machine-to-machine credentials, decentralized capability tokens, and cryptographic attenuation."
          icon={Layers}
          iconColor="text-amber-400"
          statusText="CHAVEZ-ATTENUATED"
          statusVariant="warning"
          metrics={[
            {
              label: "Token Model",
              value: "Macaroons v2",
              hint: "Chained Caveat Hashes",
              icon: Shield,
            },
            {
              label: "Caveat Discharge",
              value: "Third-Party",
              hint: "Encrypted Location Tickets",
              icon: KeyRound,
            },
            {
              label: "Verification Latency",
              value: "< 0.8ms",
              hint: "HMAC-SHA256 Cascade",
              icon: Zap,
            },
            {
              label: "Engine Gateway",
              value: "Port 4439",
              hint: "API Key Authority & Signer",
              icon: Activity,
            },
          ]}
          actions={
            <>
              <Button
                variant="outline"
                size="sm"
                onClick={() => setScopeSheetOpen(true)}
                className="h-8 gap-1.5 text-xs"
              >
                <ListFilter className="h-3.5 w-3.5" />
                <span>Scope Catalogue</span>
              </Button>
              <KeyBuilderSheet />
            </>
          }
        />

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
