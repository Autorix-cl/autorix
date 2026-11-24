"use client";

import * as React from "react";
import Link from "next/link";
import {
  Server,
  Network,
  Shield,
  Layers,
  KeyRound,
  Building2,
  Scale,
  Users,
  Activity,
  Plus,
  ArrowRight,
  RefreshCw,
  Cpu,
  CheckCircle2,
  AlertTriangle,
  XCircle,
} from "lucide-react";
import { PageHeader } from "@/components/layout/page-header";
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { useCapabilities, type EngineType } from "@/lib/capabilities/capability-context";
import { Can } from "@/lib/auth/use-permission";

interface EngineCardDef {
  type: EngineType;
  title: string;
  desc: string;
  icon: React.ComponentType<{ className?: string }>;
  port: string;
}

const ENGINE_CARDS: EngineCardDef[] = [
  { type: "ego", title: "Ego (Identity)", desc: "Identity & trait management with MFA", icon: Users, port: ":4433" },
  { type: "nexus", title: "Nexus (Zanzibar)", desc: "Fine-grained relationship authorization & graph queries", icon: Network, port: ":50051" },
  { type: "janus", title: "Janus (OAuth2/OIDC)", desc: "Client lifecycle, JWKS & token introspection", icon: KeyRound, port: ":4444" },
  { type: "aegis", title: "Aegis (Proxy)", desc: "Reverse proxy policy enforcement & path rewrites", icon: Shield, port: ":4455" },
  { type: "vulcan", title: "Vulcan (API Keys)", desc: "API key issuance, macaroons & quotas", icon: Layers, port: ":4466" },
  { type: "hermes", title: "Hermes (Enterprise)", desc: "SAML 2.0 federation & SCIM 2.0 sync", icon: Building2, port: ":4477" },
  { type: "themis", title: "Themis (Policy)", desc: "CEL expression compilation & dry-run evaluation", icon: Scale, port: ":4488" },
];

export default function FleetOverviewPage() {
  const { engines, isLoading, refreshCapabilities } = useCapabilities();

  const totalInstances = Object.values(engines).reduce((acc, curr) => acc + (curr.instanceCount ?? 0), 0);
  const healthyCount = Object.values(engines).filter((e) => e.status === "healthy").length;
  const connectedTypes = Object.values(engines).filter(
    (e) => e.status === "healthy" || e.status === "degraded" || (e.instanceCount ?? 0) > 0,
  ).length;

  return (
    <div className="space-y-6">
      <PageHeader
        breadcrumbs={[{ label: "Fleet" }]}
        title="Fleet & Engine Topology"
        description="Argus control plane: real-time registration, heartbeats and engine telemetry."
        badge="Argus Control Plane"
        arn="arn:autorix:argus:prod:default:fleet/control-plane"
        actions={
          <div className="flex items-center gap-2">
            <Button type="button" variant="outline" size="sm" onClick={refreshCapabilities} disabled={isLoading} className="h-8 gap-1.5 text-xs">
              <RefreshCw className={`h-3.5 w-3.5 ${isLoading ? "animate-spin" : ""}`} aria-hidden="true" />
              Refresh
            </Button>
            <Can perform="fleet:manage">
              <Button asChild size="sm" className="h-8 gap-1.5 text-xs">
                <Link href="/fleet/enroll">
                  <Plus className="h-3.5 w-3.5" aria-hidden="true" />
                  Enroll Engine
                </Link>
              </Button>
            </Can>
          </div>
        }
      />

      {/* Fleet Stats Overview */}
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
        <Card className="border-border/80 bg-card/60 backdrop-blur-xs">
          <CardHeader className="p-4 pb-2">
            <CardDescription className="text-xs">Connected Engine Types</CardDescription>
            <CardTitle className="text-2xl font-bold font-mono text-foreground flex items-center justify-between">
              <span>{connectedTypes} / 7</span>
              <Cpu className="h-5 w-5 text-primary opacity-60" aria-hidden="true" />
            </CardTitle>
          </CardHeader>
          <CardContent className="p-4 pt-0 text-[11px] text-muted-foreground">
            Registered IAM engine categories
          </CardContent>
        </Card>

        <Card className="border-border/80 bg-card/60 backdrop-blur-xs">
          <CardHeader className="p-4 pb-2">
            <CardDescription className="text-xs">Registered Instances</CardDescription>
            <CardTitle className="text-2xl font-bold font-mono text-foreground flex items-center justify-between">
              <span>{totalInstances}</span>
              <Server className="h-5 w-5 text-emerald-400 opacity-60" aria-hidden="true" />
            </CardTitle>
          </CardHeader>
          <CardContent className="p-4 pt-0 text-[11px] text-muted-foreground">
            Active heartbeats monitored by Argus
          </CardContent>
        </Card>

        <Card className="border-border/80 bg-card/60 backdrop-blur-xs">
          <CardHeader className="p-4 pb-2">
            <CardDescription className="text-xs">Cluster Health</CardDescription>
            <CardTitle className={`text-2xl font-bold font-mono flex items-center justify-between ${healthyCount === 7 ? "text-emerald-400" : healthyCount > 0 ? "text-amber-400" : "text-muted-foreground"}`}>
              <span>{Math.round((healthyCount / 7) * 100)}%</span>
              <Activity className="h-5 w-5 opacity-60" aria-hidden="true" />
            </CardTitle>
          </CardHeader>
          <CardContent className="p-4 pt-0 text-[11px] text-muted-foreground">
            {healthyCount} of 7 engines reporting healthy
          </CardContent>
        </Card>
      </div>

      {/* Engine Rollup Cards */}
      <div>
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-sm font-bold uppercase tracking-wider text-muted-foreground font-mono">
            Engine Fleet Status
          </h2>
          <Link href="/fleet/instances" className="text-xs text-primary hover:underline flex items-center gap-1">
            View all instances <ArrowRight className="h-3 w-3" aria-hidden="true" />
          </Link>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {ENGINE_CARDS.map((card) => {
            const Icon = card.icon;
            const engineState = engines[card.type];
            const status = engineState?.status ?? "unconnected";
            const isHealthy = status === "healthy";
            const isDegraded = status === "degraded";
            const count = engineState?.instanceCount ?? 0;

            return (
              <Card key={card.type} className="border-border/80 bg-card hover:border-primary/40 transition-colors">
                <CardHeader className="p-4 pb-3">
                  <div className="flex items-start justify-between">
                    <div className="flex items-center gap-2.5">
                      <div className="p-2 rounded-lg bg-muted border border-border/60">
                        <Icon className="h-4 w-4 text-primary" />
                      </div>
                      <div>
                        <CardTitle className="text-sm font-semibold text-foreground">{card.title}</CardTitle>
                        <span className="text-[10px] font-mono text-muted-foreground">{card.port}</span>
                      </div>
                    </div>

                    <Badge
                      variant={isHealthy ? "default" : isDegraded ? "outline" : "destructive"}
                      className={`text-[10px] uppercase font-mono px-2 py-0.5 ${
                        isHealthy ? "bg-emerald-500/10 text-emerald-400 border-emerald-500/20" : ""
                      }`}
                    >
                      {isHealthy ? (
                        <CheckCircle2 className="h-3 w-3 mr-1" aria-hidden="true" />
                      ) : isDegraded ? (
                        <AlertTriangle className="h-3 w-3 mr-1 text-amber-400" aria-hidden="true" />
                      ) : (
                        <XCircle className="h-3 w-3 mr-1" aria-hidden="true" />
                      )}
                      {status}
                    </Badge>
                  </div>
                  <CardDescription className="text-xs mt-2 line-clamp-2">{card.desc}</CardDescription>
                </CardHeader>
                <CardContent className="p-4 pt-3 border-t border-border/50 mt-3 flex items-center justify-between text-xs text-muted-foreground">
                  <div className="flex items-center gap-2">
                    <span className="font-mono text-foreground font-semibold">{count}</span>
                    <span>instance(s)</span>
                  </div>
                  <Link
                    href={`/fleet/instances?type=${card.type}`}
                    className="text-primary hover:underline text-[11px] font-medium"
                  >
                    Details →
                  </Link>
                </CardContent>
              </Card>
            );
          })}
        </div>
      </div>
    </div>
  );
}
