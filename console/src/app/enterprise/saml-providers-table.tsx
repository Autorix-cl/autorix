"use client";

import * as React from "react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Switch } from "@/components/ui/switch";
import {
  Building2,
  Plus,
  Settings2,
  AlertTriangle,
  CheckCircle2,
  XCircle,
} from "lucide-react";

import { toast } from "sonner";
import type { SAMLProvider } from "@/lib/api/schemas/hermes";
import { SAMLProviderSheet } from "./saml-provider-sheet";
import { SAMLConnectionWizard } from "./saml-connection-wizard";

interface SAMLProvidersTableProps {
  providers: SAMLProvider[];
  isLoading?: boolean;
  onRefresh?: () => void;
}

export function SAMLProvidersTable({
  providers,
  isLoading,
  onRefresh,
}: SAMLProvidersTableProps) {
  const [selectedProvider, setSelectedProvider] = React.useState<SAMLProvider | null>(null);
  const [isWizardOpen, setIsWizardOpen] = React.useState(false);

  const handleToggle = async (provider: SAMLProvider, nextState: boolean) => {
    try {
      const endpoint = nextState ? "enable" : "disable";
      const res = await fetch(`/api/enterprise/saml/${encodeURIComponent(provider.id)}/${endpoint}`, {
        method: "POST",
      });
      if (!res.ok) throw new Error("Failed to toggle provider");
      toast.success(`Provider ${nextState ? "enabled" : "disabled"}`);
      onRefresh?.();
    } catch {
      toast.error("Failed to toggle provider");
    }
  };

  return (
    <>
      <Card className="border-border">
        <CardHeader className="pb-3">
          <div className="flex items-center justify-between">
            <div>
              <CardTitle className="text-sm font-semibold flex items-center gap-2">
                <Building2 className="w-4 h-4 text-primary" />
                SAML 2.0 Identity Providers
              </CardTitle>
              <CardDescription className="text-xs">
                Federated enterprise connections with automated claim transformation.
              </CardDescription>
            </div>
            <Button
              size="sm"
              onClick={() => setIsWizardOpen(true)}
              className="h-8 gap-1.5 text-xs"
            >
              <Plus className="w-3.5 h-3.5" />
              New Connection Wizard
            </Button>
          </div>
        </CardHeader>

        <CardContent className="text-xs">
          {isLoading ? (
            <div className="py-6 text-center text-muted-foreground">Loading identity providers...</div>
          ) : providers.length === 0 ? (
            <div className="py-8 text-center text-muted-foreground border rounded-lg border-dashed">
              No SAML 2.0 connections configured yet.
            </div>
          ) : (
            <div className="border rounded-lg overflow-hidden">
              <div className="grid grid-cols-12 p-2.5 bg-muted/40 font-semibold text-[11px] text-muted-foreground">
                <span className="col-span-4">Provider / Entity</span>
                <span className="col-span-4">Certificate Diagnostics</span>
                <span className="col-span-2 text-center">Status</span>
                <span className="col-span-2 text-right">Actions</span>
              </div>
              <div className="divide-y max-h-64 overflow-y-auto">
                {providers.map((p) => {
                  const cert = p.certificates?.[0];
                  return (
                    <div key={p.id} className="grid grid-cols-12 p-2.5 items-center text-[11px]">
                      <div className="col-span-4 space-y-0.5 truncate pr-2">
                        <div className="font-semibold text-foreground truncate">
                          {p.display_name || p.id}
                        </div>
                        <div className="text-[10px] text-muted-foreground font-mono truncate">
                          {p.idp_entity_id}
                        </div>
                      </div>

                      <div className="col-span-4 space-y-0.5 pr-2">
                        {cert ? (
                          <div className="flex items-center gap-1.5">
                            {cert.expired ? (
                              <Badge variant="destructive" className="text-[9px] gap-1 px-1 py-0">
                                <XCircle className="w-2.5 h-2.5" /> Expired
                              </Badge>
                            ) : cert.expiring_soon ? (
                              <Badge variant="outline" className="text-[9px] text-amber-500 border-amber-500/30 gap-1 px-1 py-0">
                                <AlertTriangle className="w-2.5 h-2.5" /> {cert.days_until_expiry}d left
                              </Badge>
                            ) : (
                              <Badge variant="default" className="bg-emerald-500 text-[9px] gap-1 px-1 py-0">
                                <CheckCircle2 className="w-2.5 h-2.5" /> Valid
                              </Badge>
                            )}
                            <span className="text-[10px] text-muted-foreground font-mono truncate">
                              Expires {new Date(cert.not_after).toLocaleDateString()}
                            </span>
                          </div>
                        ) : p.idp_cert_expires_at ? (
                          <span className="text-[10px] text-muted-foreground font-mono">
                            Expires {new Date(p.idp_cert_expires_at).toLocaleDateString()}
                          </span>
                        ) : (
                          <span className="text-[10px] text-muted-foreground italic">
                            Certificate valid
                          </span>
                        )}
                      </div>

                      <div className="col-span-2 flex items-center justify-center gap-1.5">
                        <Switch
                          checked={p.enabled}
                          onCheckedChange={(c) => handleToggle(p, c)}
                        />
                        <span className="text-[10px] text-muted-foreground">
                          {p.enabled ? "Active" : "Off"}
                        </span>
                      </div>

                      <div className="col-span-2 text-right">
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => setSelectedProvider(p)}
                          className="h-7 text-[11px] gap-1 px-2"
                        >
                          <Settings2 className="w-3 h-3 text-primary" />
                          Configure
                        </Button>
                      </div>
                    </div>
                  );
                })}
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Provider Details & Attribute Mapper Sheet */}
      <SAMLProviderSheet
        provider={selectedProvider}
        isOpen={Boolean(selectedProvider)}
        onOpenChange={(open) => !open && setSelectedProvider(null)}
        onSuccess={onRefresh}
      />

      {/* 3-Step SAML Connection Wizard */}
      <SAMLConnectionWizard
        isOpen={isWizardOpen}
        onOpenChange={setIsWizardOpen}
        onSuccess={onRefresh}
      />
    </>
  );
}
