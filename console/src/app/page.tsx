"use client";

import * as React from "react";
import Link from "next/link";
import {
  Shield,
  Users,
  KeyRound,
  Network,
  Layers,
  Building2,
  Scale,
  ArrowUpRight,
  Server,
  Zap,
  CheckCircle2,
  AlertTriangle,
  XCircle,
  HelpCircle,
  Lock,
  ArrowRight,
  Cpu,
  RefreshCw,
  SlidersHorizontal,
  Key,
} from "lucide-react";
import { SERVICES_CONFIG } from "@/lib/config";
import { useTranslation } from "@/lib/i18n";
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { StatCard } from "@/components/ui/stat-card";
import { useApiQuery } from "@/lib/query/use-api-query";
import { fetchAndParse } from "@/lib/api/schema";
import { healthResponseSchema, type EngineStatus, type ServiceHealth } from "@/lib/api/schemas/health";

// "Never fake a signal" (roadmap principle): every visual here is driven by
// a real, measured EngineStatus — never a hardcoded color or count.
const STATUS_META: Record<
  EngineStatus,
  { dot: string; badge: "success" | "warning" | "destructive" | "outline"; Icon: typeof CheckCircle2; pulse: boolean }
> = {
  healthy: { dot: "bg-emerald-400", badge: "success", Icon: CheckCircle2, pulse: true },
  degraded: { dot: "bg-amber-400", badge: "warning", Icon: AlertTriangle, pulse: false },
  unreachable: { dot: "bg-red-500", badge: "destructive", Icon: XCircle, pulse: false },
  unknown: { dot: "bg-slate-400", badge: "outline", Icon: HelpCircle, pulse: false },
};

function averageLatency(services: ServiceHealth[]): number | null {
  const samples = services.map((s) => s.latencyMs).filter((v): v is number => v !== null);
  if (samples.length === 0) return null;
  return samples.reduce((sum, v) => sum + v, 0) / samples.length;
}

import { useEnvironment } from "@/lib/environment/environment-context";
import { Plus } from "lucide-react";

export default function DashboardPage() {
  const { t } = useTranslation();
  const { currentEnv, isProduction } = useEnvironment();

  const {
    data: healthData,
    isFetching: refreshing,
    refetch: fetchHealth,
  } = useApiQuery(["health", currentEnv.id], () => fetchAndParse("/api/health", healthResponseSchema));

  const rawServices = healthData?.services ?? [];
  const services = isProduction ? rawServices : [];
  const statusByKey = new Map(services.map((s) => [s.key, s]));
  const healthyCount = services.filter((s) => s.status === "healthy").length;
  const totalCount = isProduction ? services.length : 0;
  const overallBadge =
    totalCount === 0
      ? "outline"
      : healthyCount === totalCount
        ? "success"
        : healthyCount === 0
          ? "destructive"
          : "warning";

  const aegisStatus = isProduction ? statusByKey.get("aegis")?.status : undefined;
  const perimeterMeta = aegisStatus ? STATUS_META[aegisStatus] : STATUS_META.unknown;
  const avgLatency = isProduction ? averageLatency(services) : null;

  const engines = [
    {
      key: "nexus",
      config: SERVICES_CONFIG.nexus,
      href: "/permissions",
      icon: Network,
      iconColor: "text-purple-400",
      badgeVariant: "purple" as const,
      borderHover: "hover:border-purple-500/40",
      accentBg: "bg-purple-500/10",
      tech: "Google Zanzibar + CEL",
    },
    {
      key: "ego",
      config: SERVICES_CONFIG.ego,
      href: "/identities",
      icon: Users,
      iconColor: "text-blue-400",
      badgeVariant: "info" as const,
      borderHover: "hover:border-blue-500/40",
      accentBg: "bg-blue-500/10",
      tech: "Argon2id + JSON Schema",
    },
    {
      key: "janus",
      config: SERVICES_CONFIG.janus,
      href: "/oauth2",
      icon: KeyRound,
      iconColor: "text-amber-400",
      badgeVariant: "warning" as const,
      borderHover: "hover:border-amber-500/40",
      accentBg: "bg-amber-500/10",
      tech: "RS256 JWKS + PKCE S256",
    },
    {
      key: "aegis",
      config: SERVICES_CONFIG.aegis,
      href: "/proxy-rules",
      icon: Shield,
      iconColor: "text-emerald-400",
      badgeVariant: "success" as const,
      borderHover: "hover:border-emerald-500/40",
      accentBg: "bg-emerald-500/10",
      tech: "Zero Trust Reverse PEP",
    },
    {
      key: "vulcan",
      config: SERVICES_CONFIG.vulcan,
      href: "/api-keys",
      icon: Layers,
      iconColor: "text-cyan-400",
      badgeVariant: "cyan" as const,
      borderHover: "hover:border-cyan-500/40",
      accentBg: "bg-cyan-500/10",
      tech: "Macaroon HMAC Attenuation",
    },
    {
      key: "hermes",
      config: SERVICES_CONFIG.hermes,
      href: "/enterprise",
      icon: Building2,
      iconColor: "text-rose-400",
      badgeVariant: "rose" as const,
      borderHover: "hover:border-rose-500/40",
      accentBg: "bg-rose-500/10",
      tech: "SAML 2.0 + SCIM 2.0 (RFC 7644)",
    },
    {
      key: "themis",
      config: SERVICES_CONFIG.themis,
      href: "/policies",
      icon: Scale,
      iconColor: "text-purple-400",
      badgeVariant: "purple" as const,
      borderHover: "hover:border-purple-500/40",
      accentBg: "bg-purple-500/10",
      tech: "ABAC Google CEL Engine",
    },
  ];

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
        <div>
          <h1 className="text-xl font-bold tracking-tight text-foreground sm:text-2xl">{t("dashboard.title")}</h1>
          <p className="text-xs text-muted-foreground mt-1">{t("dashboard.subtitle")}</p>
        </div>

        <div className="flex items-center gap-2.5">
          {healthData?.checkedAt && (
            <span className="text-[11px] font-mono text-muted-foreground">
              {t("dashboard.lastChecked", { time: new Date(healthData.checkedAt).toLocaleTimeString() })}
            </span>
          )}
          <Button
            variant="outline"
            size="sm"
            onClick={() => fetchHealth()}
            disabled={refreshing}
            className="gap-1.5 text-xs h-8"
          >
            <RefreshCw className={`h-3.5 w-3.5 ${refreshing ? "animate-spin" : ""}`} />
            <span>{t("common.refresh")}</span>
          </Button>
        </div>
      </div>

      {!isProduction && (
        <div className="p-4 rounded-xl border border-amber-500/30 bg-amber-500/10 flex flex-col sm:flex-row sm:items-center justify-between gap-3">
          <div className="flex items-center gap-3">
            <AlertTriangle className="h-5 w-5 text-amber-400 shrink-0" />
            <div>
              <h4 className="text-xs font-semibold text-amber-300">
                Viewing {currentEnv.name} Environment
              </h4>
              <p className="text-[11px] text-muted-foreground">
                No active engine instances are currently enrolled for {currentEnv.name}. Local cluster engines are active under Production.
              </p>
            </div>
          </div>
          <Link href={`/fleet/enroll?env=${currentEnv.id}`}>
            <Button size="sm" variant="outline" className="text-xs gap-1.5 h-8 border-amber-500/40 text-amber-300 hover:bg-amber-500/20 shrink-0">
              <Plus className="h-3 w-3" />
              Enroll {currentEnv.name} Instance
            </Button>
          </Link>
        </div>
      )}

      {/* Top Telemetry / Stat Cards — every value is a real, measured signal */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <StatCard
          title={t("dashboard.stats.services")}
          value={totalCount === 0 ? "—" : `${healthyCount} / ${totalCount}`}
          subtitle={t("dashboard.stats.servicesSub")}
          icon={Server}
          iconColor="text-blue-400"
          badgeText={
            totalCount === 0
              ? t("dashboard.stats.notMeasured")
              : t(
                  `dashboard.status.${statusByKey.size ? (healthyCount === totalCount ? "healthy" : healthyCount === 0 ? "unreachable" : "degraded") : "unknown"}`,
                )
          }
          badgeVariant={overallBadge}
        />

        <StatCard
          title={t("dashboard.stats.perimeter")}
          value={aegisStatus ? t(`dashboard.status.${aegisStatus}`) : t("dashboard.stats.notMeasured")}
          subtitle={t("dashboard.stats.perimeterSub")}
          icon={Shield}
          iconColor="text-emerald-400"
          badgeText={aegisStatus ? t(`dashboard.status.${aegisStatus}`) : t("dashboard.stats.notMeasured")}
          badgeVariant={perimeterMeta.badge}
        />

        <StatCard
          title={t("dashboard.stats.latency")}
          value={avgLatency !== null ? `${avgLatency.toFixed(1)} ms` : "—"}
          subtitle={t("dashboard.stats.latencySub")}
          icon={Zap}
          iconColor="text-purple-400"
          badgeText={avgLatency !== null ? t("common.online") : t("dashboard.stats.notMeasured")}
          badgeVariant={avgLatency !== null ? "purple" : "outline"}
        />

        <StatCard
          title={t("dashboard.stats.requests")}
          value={t("dashboard.stats.notAvailable")}
          subtitle={t("dashboard.stats.requestsSub")}
          icon={Cpu}
          iconColor="text-cyan-400"
          badgeText={t("dashboard.stats.notAvailable")}
          badgeVariant="outline"
        />
      </div>

      {/* Core Engine Fleet Grid */}
      <div className="space-y-3">
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-base font-semibold tracking-tight text-foreground">{t("dashboard.fleetTitle")}</h2>
            <p className="text-xs text-muted-foreground">{t("dashboard.fleetSubtitle")}</p>
          </div>
        </div>

        <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
          {engines.map(({ key, config, href, icon: Icon, iconColor, badgeVariant, borderHover, accentBg, tech }) => {
            const health = statusByKey.get(key);
            const meta = STATUS_META[health?.status ?? "unknown"];
            return (
              <Card
                key={key}
                className={`group relative overflow-hidden transition-all duration-200 ${borderHover} hover:shadow-md bg-card/80`}
              >
                <CardHeader className="p-5 pb-3">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-3">
                      <div
                        className={`flex h-10 w-10 items-center justify-center rounded-lg border border-border/60 ${accentBg}`}
                      >
                        <Icon className={`h-5 w-5 ${iconColor}`} />
                      </div>
                      <div>
                        <CardTitle className="text-sm font-bold flex items-center gap-2">{config.name}</CardTitle>
                        <span className="text-[10px] font-mono text-muted-foreground">{tech}</span>
                      </div>
                    </div>
                    <Badge variant={badgeVariant} className="text-[10px] font-mono uppercase">
                      {config.protocol}
                    </Badge>
                  </div>
                </CardHeader>

                <CardContent className="p-5 pt-1 space-y-4">
                  <p className="text-xs text-muted-foreground min-h-[36px] leading-relaxed">{config.role}</p>

                  <div className="flex items-center justify-between border-t border-border/50 pt-3 text-xs">
                    <div
                      className="flex items-center gap-2 font-mono text-[11px] text-muted-foreground"
                      title={t(`dashboard.status.${health?.status ?? "unknown"}`)}
                    >
                      <span
                        className={`h-1.5 w-1.5 rounded-full ${meta.dot} ${meta.pulse ? "animate-pulse" : ""}`}
                      ></span>
                      <span>
                        {t("common.port")} :{config.port}
                      </span>
                      {health?.latencyMs !== undefined && health?.latencyMs !== null && (
                        <span className="text-muted-foreground/70">· {health.latencyMs.toFixed(0)}ms</span>
                      )}
                    </div>

                    <Link href={href}>
                      <Button
                        variant="secondary"
                        size="sm"
                        className="h-7 gap-1 px-2.5 text-xs group-hover:bg-primary group-hover:text-primary-foreground transition-colors"
                      >
                        <span>{t("dashboard.openStudio")}</span>
                        <ArrowUpRight className="h-3 w-3" />
                      </Button>
                    </Link>
                  </div>
                </CardContent>
              </Card>
            );
          })}
        </div>
      </div>

      {/* Quick Action Bar & Zero Trust Flow */}
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
        {/* Quick Actions */}
        <Card className="lg:col-span-1 bg-card/80">
          <CardHeader className="p-5 pb-3">
            <CardTitle className="text-sm font-semibold flex items-center gap-2">
              <SlidersHorizontal className="h-4 w-4 text-primary" />
              <span>{t("dashboard.quickActions")}</span>
            </CardTitle>
          </CardHeader>
          <CardContent className="p-5 pt-0 space-y-2">
            <Link href="/identities" className="block">
              <Button variant="outline" className="w-full justify-between h-9 text-xs">
                <div className="flex items-center gap-2">
                  <Users className="h-3.5 w-3.5 text-blue-400" />
                  <span>{t("dashboard.quickNewUser")}</span>
                </div>
                <ArrowRight className="h-3.5 w-3.5 opacity-60" />
              </Button>
            </Link>

            <Link href="/api-keys" className="block">
              <Button variant="outline" className="w-full justify-between h-9 text-xs">
                <div className="flex items-center gap-2">
                  <Key className="h-3.5 w-3.5 text-cyan-400" />
                  <span>{t("dashboard.quickNewKey")}</span>
                </div>
                <ArrowRight className="h-3.5 w-3.5 opacity-60" />
              </Button>
            </Link>

            <Link href="/permissions" className="block">
              <Button variant="outline" className="w-full justify-between h-9 text-xs">
                <div className="flex items-center gap-2">
                  <Network className="h-3.5 w-3.5 text-purple-400" />
                  <span>{t("dashboard.quickSimulate")}</span>
                </div>
                <ArrowRight className="h-3.5 w-3.5 opacity-60" />
              </Button>
            </Link>

            <Link href="/proxy-rules" className="block">
              <Button variant="outline" className="w-full justify-between h-9 text-xs">
                <div className="flex items-center gap-2">
                  <Shield className="h-3.5 w-3.5 text-emerald-400" />
                  <span>{t("dashboard.quickProxyRule")}</span>
                </div>
                <ArrowRight className="h-3.5 w-3.5 opacity-60" />
              </Button>
            </Link>
          </CardContent>
        </Card>

        {/* Architecture Pipeline Summary */}
        <Card className="lg:col-span-2 bg-card/80">
          <CardHeader className="p-5 pb-3">
            <div className="flex items-center justify-between">
              <CardTitle className="text-sm font-semibold flex items-center gap-2">
                <Lock className="h-4 w-4 text-emerald-400" />
                <span>{t("dashboard.architectureSummary")}</span>
              </CardTitle>
              <Badge variant="outline" className="text-[10px] font-mono">
                RFC 7519 / Zanzibar
              </Badge>
            </div>
            <CardDescription className="text-xs mt-1">{t("dashboard.architectureDesc")}</CardDescription>
          </CardHeader>
          <CardContent className="p-5 pt-1">
            <div className="grid grid-cols-1 md:grid-cols-4 gap-2.5 pt-2">
              <div className="rounded-lg border border-border/70 bg-muted/30 p-3 text-center">
                <div className="text-[10px] font-bold uppercase text-emerald-400 mb-1">1. Ingress</div>
                <div className="text-xs font-semibold text-foreground">Aegis PEP</div>
                <div className="text-[10px] text-muted-foreground mt-0.5">:4455 Proxy</div>
              </div>

              <div className="rounded-lg border border-border/70 bg-muted/30 p-3 text-center">
                <div className="text-[10px] font-bold uppercase text-amber-400 mb-1">2. Identity / Auth</div>
                <div className="text-xs font-semibold text-foreground">Janus & Ego</div>
                <div className="text-[10px] text-muted-foreground mt-0.5">RS256 / Argon2id</div>
              </div>

              <div className="rounded-lg border border-border/70 bg-muted/30 p-3 text-center">
                <div className="text-[10px] font-bold uppercase text-purple-400 mb-1">3. Authorization</div>
                <div className="text-xs font-semibold text-foreground">Nexus ReBAC</div>
                <div className="text-[10px] text-muted-foreground mt-0.5">gRPC :50051 + CEL</div>
              </div>

              <div className="rounded-lg border border-border/70 bg-muted/30 p-3 text-center">
                <div className="text-[10px] font-bold uppercase text-cyan-400 mb-1">4. Capability</div>
                <div className="text-xs font-semibold text-foreground">Vulcan & Hermes</div>
                <div className="text-[10px] text-muted-foreground mt-0.5">Macaroons / SCIM</div>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
