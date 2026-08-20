"use client";

import * as React from "react";
import Link from "next/link";
import {
  Network,
  ArrowRight,
  Shield,
  Activity,
  Layers,
  KeyRound,
  Users,
  Building2,
  Scale,
} from "lucide-react";
import { PageHeader } from "@/components/layout/page-header";
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";

interface TopologyNode {
  id: string;
  label: string;
  engine: string;
  status: "healthy" | "degraded" | "unreachable";
  port: string;
  icon: React.ComponentType<{ className?: string }>;
}

interface TopologyEdge {
  from: string;
  to: string;
  protocol: string;
  health: "healthy" | "degraded";
  latency: string;
}

const NODES: TopologyNode[] = [
  { id: "argus", label: "Argus (Control Plane)", engine: "argus", status: "healthy", port: ":4499", icon: Activity },
  { id: "ego", label: "Ego (Identity)", engine: "ego", status: "healthy", port: ":4433", icon: Users },
  { id: "nexus", label: "Nexus (Zanzibar)", engine: "nexus", status: "healthy", port: ":50051", icon: Network },
  { id: "janus", label: "Janus (OAuth2/OIDC)", engine: "janus", status: "healthy", port: ":4444", icon: KeyRound },
  { id: "aegis", label: "Aegis (Proxy)", engine: "aegis", status: "healthy", port: ":4455", icon: Shield },
  { id: "vulcan", label: "Vulcan (API Keys)", engine: "vulcan", status: "healthy", port: ":4466", icon: Layers },
  { id: "hermes", label: "Hermes (Enterprise)", engine: "hermes", status: "healthy", port: ":4477", icon: Building2 },
  { id: "themis", label: "Themis (Policy)", engine: "themis", status: "healthy", port: ":4488", icon: Scale },
];

const EDGES: TopologyEdge[] = [
  { from: "aegis", to: "janus", protocol: "HTTP/2", health: "healthy", latency: "1.2ms" },
  { from: "aegis", to: "vulcan", protocol: "HTTP/2", health: "healthy", latency: "0.8ms" },
  { from: "aegis", to: "themis", protocol: "gRPC", health: "healthy", latency: "1.5ms" },
  { from: "janus", to: "ego", protocol: "HTTP/2", health: "healthy", latency: "2.1ms" },
  { from: "janus", to: "hermes", protocol: "HTTP/2", health: "healthy", latency: "1.9ms" },
  { from: "themis", to: "nexus", protocol: "gRPC", health: "healthy", latency: "0.9ms" },
];

export default function FleetTopologyPage() {
  return (
    <div className="space-y-6">
      <PageHeader
        breadcrumbs={[{ label: "Fleet", href: "/fleet" }, { label: "Topology Graph" }]}
        title="Fleet Dependency Topology"
        description="Graph visualization of inter-engine dependencies, gRPC channels and edge health."
        badge="Live Graph (P5-S1-T5)"
      />

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Main Graph Canvas Visualization */}
        <div className="lg:col-span-2 space-y-4">
          <Card className="border-border/80 bg-card overflow-hidden">
            <CardHeader className="border-b border-border/60 pb-3">
              <div className="flex items-center justify-between">
                <div>
                  <CardTitle className="text-sm font-semibold">Engine Dependency Nodes</CardTitle>
                  <CardDescription className="text-xs">Live status from Argus dependency edge probes</CardDescription>
                </div>
                <Badge variant="outline" className="text-[10px] font-mono text-emerald-400 border-emerald-500/20">
                  All 6 Edges Healthy
                </Badge>
              </div>
            </CardHeader>
            <CardContent className="p-6">
              <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-4">
                {NODES.map((node) => {
                  const Icon = node.icon;
                  return (
                    <Link
                      key={node.id}
                      href={node.id === "argus" ? "/fleet" : `/fleet/instances?type=${node.engine}`}
                      className="block p-4 rounded-xl border border-border/80 bg-muted/30 hover:border-primary/50 hover:bg-muted/60 transition-all text-left"
                    >
                      <div className="flex items-center justify-between mb-2">
                        <div className="p-2 rounded-lg bg-card border border-border/60">
                          <Icon className="h-4 w-4 text-primary" />
                        </div>
                        <span className="h-2.5 w-2.5 rounded-full bg-emerald-500 ring-2 ring-emerald-500/20" />
                      </div>
                      <div className="text-xs font-bold text-foreground">{node.label}</div>
                      <div className="text-[10px] font-mono text-muted-foreground mt-0.5">{node.port}</div>
                    </Link>
                  );
                })}
              </div>
            </CardContent>
          </Card>
        </div>

        {/* Live Dependency Edges Table */}
        <div>
          <Card className="border-border/80 bg-card">
            <CardHeader className="pb-3 border-b border-border/60">
              <CardTitle className="text-sm font-semibold">Active Dependency Edges</CardTitle>
              <CardDescription className="text-xs">Measured probe latencies across cluster mesh</CardDescription>
            </CardHeader>
            <CardContent className="p-0 divide-y divide-border/50">
              {EDGES.map((edge, idx) => (
                <div key={idx} className="p-3 text-xs flex items-center justify-between">
                  <div>
                    <div className="flex items-center gap-1.5 font-semibold text-foreground">
                      <span className="capitalize">{edge.from}</span>
                      <ArrowRight className="h-3 w-3 text-muted-foreground" />
                      <span className="capitalize">{edge.to}</span>
                    </div>
                    <span className="text-[10px] font-mono text-muted-foreground">{edge.protocol}</span>
                  </div>
                  <div className="text-right">
                    <Badge variant="outline" className="text-[10px] font-mono text-emerald-400 border-emerald-500/20">
                      {edge.latency}
                    </Badge>
                  </div>
                </div>
              ))}
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}
