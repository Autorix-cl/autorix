"use client";

import * as React from "react";
import { FolderOpen, Plus } from "lucide-react";
import { Button } from "@/components/ui/button";

export interface EmptyStateProps {
  icon?: React.ComponentType<{ className?: string }>;
  title: string;
  description: string;
  actionLabel?: string;
  onAction?: () => void;
}

export function EmptyState({
  icon: Icon = FolderOpen,
  title,
  description,
  actionLabel,
  onAction,
}: EmptyStateProps) {
  return (
    <div className="flex flex-col items-center justify-center py-16 px-4 text-center rounded-xl border border-dashed border-border/80 bg-card/40 backdrop-blur-xs">
      <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-muted/80 text-muted-foreground mb-4">
        <Icon className="h-6 w-6" />
      </div>
      <h3 className="text-sm font-semibold text-foreground">{title}</h3>
      <p className="text-xs text-muted-foreground max-w-sm mt-1 mb-6">{description}</p>
      {actionLabel && onAction && (
        <Button size="sm" onClick={onAction} className="gap-1.5 text-xs">
          <Plus className="h-3.5 w-3.5" />
          {actionLabel}
        </Button>
      )}
    </div>
  );
}
