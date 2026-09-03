"use client";

import * as React from "react";
import { useParams } from "next/navigation";
import Link from "next/link";
import { ArrowLeft } from "lucide-react";
import { Button } from "@/components/ui/button";
import { PerEngineDashboard } from "../../per-engine-dashboard";

export default function EngineObservabilityPage() {
  const params = useParams();
  const engineType = typeof params.type === "string" ? params.type : "nexus";

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-2">
        <Link href="/observability">
          <Button variant="ghost" size="sm" className="h-8 gap-1 text-xs">
            <ArrowLeft className="h-3.5 w-3.5" />
            Back to Fleet Overview
          </Button>
        </Link>
      </div>
      <div>
        <h1 className="text-xl font-bold tracking-tight uppercase">
          {engineType} Engine Telemetry
        </h1>
        <p className="text-xs text-muted-foreground mt-0.5">
          Dedicated saturation, percentiles, error rate and domain metrics
        </p>
      </div>

      <PerEngineDashboard initialEngine={engineType} />
    </div>
  );
}
