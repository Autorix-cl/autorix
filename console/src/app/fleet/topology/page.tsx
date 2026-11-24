"use client";

import * as React from "react";
import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
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
  RefreshCw,
} from "lucide-react";
import { PageHeader } from "@/components/layout/page-header";
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { useApiQuery } from "@/lib/query/use-api-query";
import { fetchAndParse } from "@/lib/api/schema";
import { healthResponseSchema, type EngineStatus } from "@/lib/api/schemas/health";

interface TopologyNodeDef {
  id: string;
  label: string;
  engine: string;
  port: string;
  icon: React.ComponentType<{ className?: string }>;
}

interface TopologyEdgeDef {
  from: string;
  to: string;
  protocol: string;
}

const NODE_DEFS: TopologyNodeDef[] = [
  { id: "argus", label: "Argus (Control Plane)", engine: "argus", port: ":4400", icon: Activity },
  { id: "ego", label: "Ego (Identity)", engine: "ego", port: ":4433", icon: Users },
  { id: "nexus", label: "Nexus (Zanzibar)", engine: "nexus", port: ":50051", icon: Network },
  { id: "janus", label: "Janus (OAuth2/OIDC)", engine: "janus", port: ":4444", icon: KeyRound },
  { id: "aegis", label: "Aegis (Proxy)", engine: "aegis", port: ":4455", icon: Shield },
  { id: "vulcan", label: "Vulcan (API Keys)", engine: "vulcan", port: ":4466", icon: Layers },
  { id: "hermes", label: "Hermes (Enterprise)", engine: "hermes", port: ":4477", icon: Building2 },
  { id: "themis", label: "Themis (Policy)", engine: "themis", port: ":4488", icon: Scale },
];

const ARCHITECTURE_EDGES: TopologyEdgeDef[] = [
  { from: "aegis", to: "janus", protocol: "HTTP/2" },
  { from: "aegis", to: "vulcan", protocol: "HTTP/2" },
  { from: "aegis", to: "themis", protocol: "gRPC" },
  { from: "janus", to: "ego", protocol: "HTTP/2" },
  { from: "janus", to: "hermes", protocol: "HTTP/2" },
  { from: "themis", to: "nexus", protocol: "gRPC" },
];

interface LiveTopologyEdge {
  from?: string;
  source?: string;
  to?: string;
  target?: string;
  protocol?: string;
  status?: string;
  reachable?: boolean;
  latency_ms?: number | null;
}

export default function FleetTopologyPage() {
  const {
    data: healthData,
    isFetching: healthFetching,
    refetch: refetchHealth,
  } = useApiQuery(["health"], () => fetchAndParse("/api/health", healthResponseSchema));

  const {
    data: topologyData,
    isFetching: topoFetching,
    refetch: refetchTopology,
  } = useQuery<{
    edges?: LiveTopologyEdge[];
    nodes?: Array<{ id?: string; engine_type?: string; status?: string }>;
  } | null>({
    queryKey: ["fleet", "topology"],
    queryFn: async () => {
      try {
        const res = await fetch("/api/fleet/topology");
        if (!res.ok) return null;
        return await res.json();
      } catch {
        return null;
      }
    },
  });

  const healthByKey = React.useMemo(() => {
    const map = new Map<string, { status: EngineStatus; latencyMs: number | null }>();
    for (const svc of healthData?.services ?? []) {
      map.set(svc.key.toLowerCase(), { status: svc.status, latencyMs: svc.latencyMs });
    }
    return map;
  }, [healthData]);

  const nodes = React.useMemo(
    () =>
      NODE_DEFS.map((def) => {
        const health = healthByKey.get(def.id);
        const topoNode = topologyData?.nodes?.find((n) => (n.engine_type || n.id || "").toLowerCase() === def.id);
        const status: EngineStatus =
          health?.status ??
          (topoNode?.status === "healthy"
            ? "healthy"
            : topoNode?.status === "degraded"
              ? "degraded"
              : topoNode?.status === "unreachable"
                ? "unreachable"
                : "unknown");
        return { ...def, status, latencyMs: health?.latencyMs ?? null };
      }),
    [healthByKey, topologyData],
  );

  const edges = React.useMemo(() => {
    const liveEdges = topologyData?.edges ?? [];
    return ARCHITECTURE_EDGES.map((def) => {
      const matched = liveEdges.find(
        (e) =>
          (e.from || e.source || "").toLowerCase() === def.from && (e.to || e.target || "").toLowerCase() === def.to,
      );
      const fromHealth = healthByKey.get(def.from);
      const toHealth = healthByKey.get(def.to);
      const isHealthy =
        matched?.reachable !== undefined
          ? matched.reachable
          : matched?.status
            ? matched.status === "healthy"
            : fromHealth?.status === "healthy" && toHealth?.status === "healthy";
      const latencyMs = typeof matched?.latency_ms === "number" ? matched.latency_ms : (toHealth?.latencyMs ?? null);
      return {
        ...def,
        healthy: isHealthy,
        latency: latencyMs !== null ? `${latencyMs.toFixed(1)}ms` : "Unmeasured",
      };
    });
  }, [healthByKey, topologyData]);

  const healthyEdgeCount = edges.filter((e) => e.healthy).length;
  const isRefreshing = healthFetching || topoFetching;

  const statusDotClass = (status: EngineStatus) => {
    switch (status) {
      case "healthy":
        return "bg-emerald-500 ring-2 ring-emerald-500/20";
      case "degraded":
        return "bg-amber-500 ring-2 ring-amber-500/20";
      case "unreachable":
        return "bg-rose-500 ring-2 ring-rose-500/20";
      default:
        return "bg-slate-400 ring-2 ring-slate-400/20";
    }
  };

  return (
    <div className="space-y-6">
      <PageHeader
        breadcrumbs={[{ label: "Fleet", href: "/fleet" }, { label: "Topology Graph" }]}
        title="Fleet Dependency Topology"
        description="Live graph visualization of inter-engine dependencies, gRPC channels, and edge health."
        badge="Live Graph"
        actions={
          <Button
            type="button"
            variant="outline"
            size="sm"
            disabled={isRefreshing}
            onClick={() => {
              refetchHealth();
              refetchTopology();
            }}
            className="h-8 gap-1.5 text-xs"
          >
            <RefreshCw className={`h-3.5 w-3.5 ${isRefreshing ? "animate-spin" : ""}`} aria-hidden="true" />
            Refresh
          </Button>
        }
      />

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Main Graph Canvas Visualization */}
        <div className="lg:col-span-2 space-y-4">
          <Card className="border-border/80 bg-card overflow-hidden">
            <CardHeader className="border-b border-border/60 pb-3">
              <div className="flex items-center justify-between gap-2">
                <div>
                  <CardTitle className="text-sm font-semibold">Engine Dependency Nodes</CardTitle>
                  <CardDescription className="text-xs">
                    Live status from Argus topology & engine health probes
                  </CardDescription>
                </div>
                <Badge
                  variant="outline"
                  className={
                    healthyEdgeCount === edges.length
                      ? "text-[10px] font-mono text-emerald-400 border-emerald-500/20"
                      : "text-[10px] font-mono text-amber-400 border-amber-500/30"
                  }
                >
                  {healthyEdgeCount}/{edges.length} Edges Healthy
                </Badge>
              </div>
            </CardHeader>
            <CardContent className="p-6">
              <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-4">
                {nodes.map((node) => {
                  const Icon = node.icon;
                  return (
                    <Link
                      key={node.id}
                      href={node.id === "argus" ? "/fleet" : `/fleet/instances?type=${node.engine}`}
                      className="block p-4 rounded-xl border border-border/80 bg-muted/30 hover:border-primary/50 hover:bg-muted/60 transition-all text-left focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                    >
                      <div className="flex items-center justify-between mb-2">
                        <div className="p-2 rounded-lg bg-card border border-border/60">
                          <Icon className="h-4 w-4 text-primary" />
                        </div>
                        <span className="flex items-center gap-1.5">
                          <span
                            className={`h-2.5 w-2.5 rounded-full ${statusDotClass(node.status)}`}
                            aria-hidden="true"
                          />
                          <span className="sr-only">{node.status}</span>
                        </span>
                      </div>
                      <div className="text-xs font-bold text-foreground">{node.label}</div>
                      <div className="flex items-center justify-between text-[10px] font-mono text-muted-foreground mt-0.5">
                        <span>{node.port}</span>
                        <span className="uppercase">{node.status}</span>
                      </div>
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
              {edges.map((edge) => (
                <div key={`${edge.from}-${edge.to}`} className="p-3 text-xs flex items-center justify-between">
                  <div>
                    <div className="flex items-center gap-1.5 font-semibold text-foreground">
                      <span className="capitalize">{edge.from}</span>
                      <ArrowRight className="h-3 w-3 text-muted-foreground" aria-hidden="true" />
                      <span className="capitalize">{edge.to}</span>
                    </div>
                    <span className="text-[10px] font-mono text-muted-foreground">{edge.protocol}</span>
                  </div>
                  <div className="text-right">
                    <Badge
                      variant="outline"
                      className={
                        edge.healthy
                          ? "text-[10px] font-mono text-emerald-400 border-emerald-500/20"
                          : "text-[10px] font-mono text-amber-400 border-amber-500/30"
                      }
                    >
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
