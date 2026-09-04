"use client";

import * as React from "react";
import Link from "next/link";
import { LucideIcon, ChevronRight, Activity } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { useOptionalEnvironment, DEFAULT_ENVIRONMENTS } from "@/lib/environment/environment-context";
import { cn } from "@/lib/utils";

export interface ServiceMetric {
  label: string;
  value: React.ReactNode;
  hint?: string;
  icon?: LucideIcon;
  trend?: "neutral" | "positive" | "warning";
}

interface ServiceHeaderProps {
  serviceName: string;
  title: string;
  description: string;
  icon: LucideIcon;
  iconColor?: string;
  statusText?: string;
  statusVariant?: "cyan" | "purple" | "success" | "warning" | "info" | "rose" | "default";
  actions?: React.ReactNode;
  metrics?: ServiceMetric[];
  className?: string;
}

/**
 * ServiceHeader provides a top-tier Cloud Console header experience
 * inspired by Cloudflare Zero Trust, Supabase, and Vercel Dashboard.
 */
export function ServiceHeader({
  serviceName,
  title,
  description,
  icon: Icon,
  iconColor = "text-cyan-400",
  statusText,
  statusVariant = "cyan",
  actions,
  metrics,
  className,
}: ServiceHeaderProps) {
  const envCtx = useOptionalEnvironment();
  const currentEnv = envCtx?.currentEnv ?? DEFAULT_ENVIRONMENTS[0];

  return (
    <div className={cn("space-y-4 pb-2", className)}>
      {/* 1. Contextual Cloud Breadcrumb */}
      <div className="flex items-center gap-1.5 text-xs text-muted-foreground font-mono">
        <Link href="/" className="hover:text-foreground transition-colors">
          Autorix Cloud
        </Link>
        <ChevronRight className="h-3 w-3 text-muted-foreground/60" />
        <span className="capitalize text-foreground/80">{currentEnv.name}</span>
        <ChevronRight className="h-3 w-3 text-muted-foreground/60" />
        <span className="text-foreground font-semibold">{serviceName}</span>
      </div>

      {/* 2. Title, Live Status & Actions Row */}
      <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
        <div className="flex items-start gap-3.5">
          <div
            className={cn(
              "flex h-11 w-11 shrink-0 items-center justify-center rounded-xl border border-border/80 bg-muted/40 shadow-xs",
              "transition-transform duration-200 hover:scale-105"
            )}
          >
            <Icon className={cn("h-5 w-5", iconColor)} />
          </div>

          <div className="space-y-1">
            <div className="flex items-center gap-2.5 flex-wrap">
              <h1 className="text-2xl font-bold tracking-tight text-foreground font-sans">
                {title}
              </h1>
              {statusText && (
                <Badge
                  variant={statusVariant}
                  className="gap-1.5 py-0.5 px-2.5 text-[11px] font-mono whitespace-nowrap shrink-0"
                >
                  <span className="relative flex h-2 w-2">
                    <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-current opacity-75" />
                    <span className="relative inline-flex rounded-full h-2 w-2 bg-current" />
                  </span>
                  <span>{statusText}</span>
                </Badge>
              )}
            </div>
            <p className="text-xs text-muted-foreground max-w-2xl leading-relaxed">
              {description}
            </p>
          </div>
        </div>

        {actions && (
          <div className="flex flex-wrap items-center gap-2 pt-1 lg:pt-0">
            {actions}
          </div>
        )}
      </div>

      {/* 3. Cloud Telemetry HUD Strip */}
      {metrics && metrics.length > 0 && (
        <div className="grid grid-cols-2 gap-2.5 sm:grid-cols-4 pt-2">
          {metrics.map((m, idx) => {
            const MetricIcon = m.icon || Activity;
            return (
              <div
                key={idx}
                className="cloud-hud-item rounded-lg p-3 flex flex-col justify-between"
              >
                <div className="flex items-center justify-between text-muted-foreground">
                  <span className="text-[11px] font-medium tracking-wide uppercase">
                    {m.label}
                  </span>
                  <MetricIcon className="h-3.5 w-3.5 text-muted-foreground/60" />
                </div>
                <div className="mt-1.5 flex items-baseline gap-2">
                  <span className="text-lg font-bold font-mono tracking-tight text-foreground">
                    {m.value}
                  </span>
                  {m.hint && (
                    <span className="text-[10px] text-muted-foreground truncate">
                      {m.hint}
                    </span>
                  )}
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
