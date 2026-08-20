import * as React from "react";
import { cva, type VariantProps } from "class-variance-authority";
import { cn } from "@/lib/utils";

const badgeVariants = cva(
  "inline-flex items-center gap-1.5 rounded-full border px-2.5 py-0.5 text-xs font-medium transition-colors focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 tracking-wide",
  {
    variants: {
      variant: {
        default: "border-transparent bg-primary text-primary-foreground hover:bg-primary/80",
        secondary: "border-transparent bg-secondary text-secondary-foreground hover:bg-secondary/80",
        destructive: "border-transparent bg-destructive text-destructive-foreground hover:bg-destructive/80",
        outline: "text-foreground border-border",
        success: "border-emerald-500/30 bg-emerald-500/10 text-emerald-400 dark:text-emerald-300",
        warning: "border-amber-500/30 bg-amber-500/10 text-amber-400 dark:text-amber-300",
        info: "border-blue-500/30 bg-blue-500/10 text-blue-400 dark:text-blue-300",
        purple: "border-purple-500/30 bg-purple-500/10 text-purple-400 dark:text-purple-300",
        cyan: "border-cyan-500/30 bg-cyan-500/10 text-cyan-400 dark:text-cyan-300",
        rose: "border-rose-500/30 bg-rose-500/10 text-rose-400 dark:text-rose-300",
      },
    },
    defaultVariants: {
      variant: "default",
    },
  },
);

export interface BadgeProps extends React.HTMLAttributes<HTMLDivElement>, VariantProps<typeof badgeVariants> {}

function Badge({ className, variant, ...props }: BadgeProps) {
  return <div className={cn(badgeVariants({ variant }), className)} {...props} />;
}

export { Badge, badgeVariants };
