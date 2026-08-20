"use client";

import * as React from "react";
import Link from "next/link";
import { ChevronRight, Home, Copy, Check } from "lucide-react";
import { Badge } from "@/components/ui/badge";

export interface BreadcrumbItem {
  label: string;
  href?: string;
}

export interface PageHeaderProps {
  breadcrumbs: BreadcrumbItem[];
  title: string;
  description?: string;
  badge?: string;
  arn?: string; // Canonical Resource Identifier (P4-S5-T2)
  actions?: React.ReactNode;
}

export function PageHeader({
  breadcrumbs,
  title,
  description,
  badge,
  arn,
  actions,
}: PageHeaderProps) {
  const [copiedArn, setCopiedArn] = React.useState(false);

  const handleCopyArn = () => {
    if (!arn) return;
    navigator.clipboard.writeText(arn);
    setCopiedArn(true);
    setTimeout(() => setCopiedArn(false), 2000);
  };

  return (
    <div className="space-y-3 border-b border-border/60 pb-5">
      {/* Breadcrumb Trail */}
      <nav aria-label="Breadcrumbs" className="flex items-center gap-1.5 text-xs text-muted-foreground font-medium">
        <Link href="/" className="hover:text-foreground transition-colors flex items-center gap-1">
          <Home className="h-3.5 w-3.5" />
          <span>Home</span>
        </Link>
        {breadcrumbs.map((crumb) => (
          <React.Fragment key={crumb.label}>
            <ChevronRight className="h-3 w-3 opacity-40 shrink-0" />
            {crumb.href ? (
              <Link href={crumb.href} className="hover:text-foreground transition-colors">
                {crumb.label}
              </Link>
            ) : (
              <span className="text-foreground font-semibold">{crumb.label}</span>
            )}
          </React.Fragment>
        ))}
      </nav>

      {/* Main Title & Action Bar */}
      <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
        <div>
          <div className="flex items-center gap-2.5">
            <h1 className="text-2xl font-bold tracking-tight text-foreground">{title}</h1>
            {badge && (
              <Badge variant="outline" className="font-mono text-xs uppercase">
                {badge}
              </Badge>
            )}
          </div>
          {description && <p className="text-sm text-muted-foreground mt-0.5">{description}</p>}
          {arn && (
            <div className="inline-flex items-center gap-2 mt-2 px-2.5 py-1 rounded-md bg-muted/60 border border-border text-[11px] font-mono text-muted-foreground">
              <span className="text-primary font-semibold">ARN:</span>
              <span className="truncate max-w-md">{arn}</span>
              <button
                type="button"
                onClick={handleCopyArn}
                className="hover:text-foreground transition-colors ml-1"
                aria-label="Copy ARN"
              >
                {copiedArn ? <Check className="h-3 w-3 text-emerald-400" /> : <Copy className="h-3 w-3" />}
              </button>
            </div>
          )}
        </div>

        {actions && <div className="flex items-center gap-2 shrink-0">{actions}</div>}
      </div>
    </div>
  );
}
