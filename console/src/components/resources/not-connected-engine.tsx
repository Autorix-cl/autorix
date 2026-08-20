"use client";

import * as React from "react";
import Link from "next/link";
import { ServerOff, Plus, ArrowRight, Radio } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from "@/components/ui/card";
import type { EngineType } from "@/lib/capabilities/capability-context";

export interface NotConnectedEngineProps {
  engineType: EngineType;
  engineName: string;
  description: string;
}

export function NotConnectedEngine({
  engineType,
  engineName,
  description,
}: NotConnectedEngineProps) {
  return (
    <Card className="max-w-xl mx-auto my-12 border-dashed border-border/80 bg-card/60 text-center p-6">
      <CardHeader className="pb-3">
        <div className="mx-auto flex h-12 w-12 items-center justify-center rounded-2xl bg-amber-500/10 text-amber-400 mb-2">
          <ServerOff className="h-6 w-6" />
        </div>
        <CardTitle className="text-lg font-bold">{engineName} Is Not Connected</CardTitle>
        <CardDescription className="text-xs max-w-sm mx-auto mt-1">
          {description} No active {engineName} instance has registered with the Argus control plane in the current environment.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4 pt-2">
        <div className="p-3 rounded-lg bg-muted/50 border border-border text-[11px] font-mono text-muted-foreground flex items-center justify-center gap-2">
          <Radio className="h-3.5 w-3.5 text-amber-400 animate-pulse" />
          <span>Awaiting registration token or heartbeat on :4499</span>
        </div>

        <div className="flex items-center justify-center gap-3">
          <Link href={`/fleet/enroll?type=${engineType}`}>
            <Button size="sm" className="text-xs gap-1.5">
              <Plus className="h-3.5 w-3.5" />
              Enroll {engineName}
            </Button>
          </Link>
          <Link href="/fleet">
            <Button variant="outline" size="sm" className="text-xs gap-1.5">
              Fleet Overview <ArrowRight className="h-3.5 w-3.5" />
            </Button>
          </Link>
        </div>
      </CardContent>
    </Card>
  );
}
