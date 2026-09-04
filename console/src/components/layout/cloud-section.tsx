import * as React from "react";
import { LucideIcon } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";

interface CloudSectionProps extends React.HTMLAttributes<HTMLElement> {
  title: string;
  description?: string;
  icon?: LucideIcon;
  badge?: string;
  badgeVariant?: "cyan" | "purple" | "success" | "warning" | "info" | "rose" | "default";
  actions?: React.ReactNode;
  children: React.ReactNode;
}

/**
 * CloudSection organizes dashboard cards into clean, enterprise-grade cloud sections
 * with consistent visual hierarchy, subtle typography, and optional action slots.
 */
export function CloudSection({
  title,
  description,
  icon: Icon,
  badge,
  badgeVariant = "default",
  actions,
  children,
  className,
  ...props
}: CloudSectionProps) {
  return (
    <section className={cn("space-y-3", className)} {...props}>
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-2 pb-1 border-b border-border/40">
        <div className="flex items-center gap-2 flex-wrap">
          {Icon && (
            <div className="flex items-center justify-center w-6 h-6 rounded-md bg-muted/70 text-muted-foreground">
              <Icon className="w-3.5 h-3.5 text-foreground" />
            </div>
          )}
          <h2 className="text-sm font-semibold tracking-tight text-foreground">{title}</h2>
          {badge && (
            <Badge variant={badgeVariant} className="text-[10px] py-0 px-2 font-mono">
              {badge}
            </Badge>
          )}
          {description && (
            <span className="hidden md:inline text-xs text-muted-foreground border-l border-border/60 pl-2">
              {description}
            </span>
          )}
        </div>
        {actions && <div className="flex items-center gap-2">{actions}</div>}
      </div>
      <div>{children}</div>
    </section>
  );
}
