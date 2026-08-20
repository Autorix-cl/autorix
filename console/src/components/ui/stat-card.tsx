"use client";

import * as React from "react";
import { LucideIcon } from "lucide-react";
import { cn } from "@/lib/utils";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";

interface StatCardProps {
  title: string;
  value: string | number;
  subtitle?: string;
  icon: LucideIcon;
  iconColor?: string;
  badgeText?: string;
  badgeVariant?:
    "default" | "secondary" | "destructive" | "outline" | "success" | "warning" | "info" | "purple" | "cyan" | "rose";
  trend?: {
    value: string;
    positive?: boolean;
  };
  className?: string;
}

export function StatCard({
  title,
  value,
  subtitle,
  icon: Icon,
  iconColor = "text-primary",
  badgeText,
  badgeVariant = "success",
  trend,
  className,
}: StatCardProps) {
  return (
    <Card
      className={cn(
        "relative overflow-hidden border-border/70 bg-card/70 backdrop-blur-sm hover:border-primary/40 hover:shadow-lg transition-all duration-300",
        className,
      )}
    >
      <CardContent className="p-5">
        <div className="flex items-center justify-between">
          <span className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">{title}</span>
          <div className={cn("rounded-lg p-2 bg-muted/60 border border-border/50", iconColor)}>
            <Icon className="h-4 w-4" />
          </div>
        </div>

        <div className="mt-3 flex items-baseline justify-between">
          <div className="text-2xl font-bold tracking-tight text-foreground font-mono">{value}</div>
          {badgeText && (
            <Badge variant={badgeVariant} className="text-[10px] px-2 py-0.5 font-semibold">
              {badgeText}
            </Badge>
          )}
          {trend && (
            <span className={cn("text-xs font-medium", trend.positive ? "text-emerald-400" : "text-rose-400")}>
              {trend.positive ? "+" : ""}
              {trend.value}
            </span>
          )}
        </div>

        {subtitle && <p className="mt-1.5 text-xs text-muted-foreground/90 truncate">{subtitle}</p>}
      </CardContent>
    </Card>
  );
}
