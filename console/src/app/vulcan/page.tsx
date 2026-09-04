"use client";

import * as React from "react";
import { KeysTable } from "./keys-table";
import { KeyBuilderSheet } from "./key-builder-sheet";
import { AttenuationStudio } from "./attenuation-studio";
import { MacaroonInspector } from "./macaroon-inspector";
import { ScopeCatalogSheet } from "./scope-catalog-sheet";
import { Shield, ListFilter, Layers, KeyRound, Zap, Activity, SearchCode, Wand2 } from "lucide-react";
import { ServiceHeader } from "@/components/layout/service-header";
import { CloudSection } from "@/components/layout/cloud-section";
import { Card } from "@/components/ui/card";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs";
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

        {/* Studio Navigation Tabs */}
        <Tabs defaultValue="keys" className="space-y-4">
          <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3 border-b border-border pb-3">
            <TabsList className="bg-muted/40 p-1">
              <TabsTrigger value="keys" className="gap-2 text-xs font-medium">
                <KeyRound className="h-3.5 w-3.5 text-amber-400" />
                <span>API Keys Vault</span>
              </TabsTrigger>
              <TabsTrigger value="inspector" className="gap-2 text-xs font-medium">
                <SearchCode className="h-3.5 w-3.5 text-blue-400" />
                <span>Macaroon Inspector</span>
              </TabsTrigger>
              <TabsTrigger value="attenuation" className="gap-2 text-xs font-medium">
                <Wand2 className="h-3.5 w-3.5 text-purple-400" />
                <span>Attenuation Studio</span>
              </TabsTrigger>
            </TabsList>
          </div>

          {/* Tab 1: Keys Table with Telemetry and Rotation */}
          <TabsContent value="keys" className="space-y-4 focus-visible:outline-none">
            <CloudSection
              title="API Key Registry & Zero-Downtime Rotation"
              description="Cryptographically signed API keys, prefix routing, active scopes, and rotation lifecycle"
              icon={KeyRound}
              badge="M2M Credentials"
              badgeVariant="warning"
            >
              <Card className="bg-card/80 p-6 pt-0">
                <KeysTable />
              </Card>
            </CloudSection>
          </TabsContent>

          {/* Tab 2: Macaroon Inspector with Live Verification */}
          <TabsContent value="inspector" className="space-y-4 focus-visible:outline-none">
            <CloudSection
              title="Capability Token Inspector & Live Verification"
              description="Inspect caveats, attenuation provenance, and test live verification against simulated environments"
              icon={SearchCode}
              badge="Live Decoder"
              badgeVariant="warning"
            >
              <MacaroonInspector />
            </CloudSection>
          </TabsContent>

          {/* Tab 3: Attenuation Studio with Provenance Chain */}
          <TabsContent value="attenuation" className="space-y-4 focus-visible:outline-none">
            <CloudSection
              title="Cryptographic Attenuation Studio"
              description="Derive attenuated child macaroons with first-party caveats and delegated permissions"
              icon={Wand2}
              badge="Token Minting"
              badgeVariant="warning"
            >
              <AttenuationStudio />
            </CloudSection>
          </TabsContent>
        </Tabs>

        {/* Scope Catalogue Management Sheet */}
        <ScopeCatalogSheet isOpen={scopeSheetOpen} onOpenChange={setScopeSheetOpen} />
      </div>
    </QueryClientProvider>
  );
}
