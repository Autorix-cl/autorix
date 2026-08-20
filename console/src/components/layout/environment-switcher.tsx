"use client";

import * as React from "react";
import { Check, ChevronDown } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

import { useEnvironment } from "@/lib/environment/environment-context";

export function EnvironmentSwitcher() {
  const { currentEnv, setEnvironment, environments } = useEnvironment();

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          variant="outline"
          size="sm"
          className="h-8 gap-1.5 px-2.5 text-xs font-semibold border-border/80 bg-card/60"
        >
          <span
            className={`h-2 w-2 rounded-full ${
              currentEnv.isProduction ? "bg-rose-500 ring-2 ring-rose-500/20" : "bg-emerald-500 ring-2 ring-emerald-500/20"
            }`}
          />
          <span>{currentEnv.name}</span>
          <ChevronDown className="h-3 w-3 opacity-50" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start" className="w-44">
        <DropdownMenuLabel className="text-[10px] uppercase text-muted-foreground">
          Active Environment (P4-S5-T1)
        </DropdownMenuLabel>
        <DropdownMenuSeparator />
        {environments.map((env) => (
          <DropdownMenuItem
            key={env.id}
            onClick={() => setEnvironment(env.id)}
            className="flex items-center justify-between text-xs cursor-pointer"
          >
            <div className="flex items-center gap-2">
              <span
                className={`h-2 w-2 rounded-full ${
                  env.isProduction ? "bg-rose-500" : "bg-emerald-500"
                }`}
              />
              <span className="font-medium">{env.name}</span>
            </div>
            {currentEnv.id === env.id && <Check className="h-3.5 w-3.5 text-primary" />}
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
