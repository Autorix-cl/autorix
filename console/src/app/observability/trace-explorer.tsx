"use client";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Network } from "lucide-react";
export function TraceExplorer(props: { initialTraceId?: string }) {
  void props;
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm flex gap-2">
          <Network className="h-4 w-4" />
          Distributed traces
        </CardTitle>
        <CardDescription>No trace backend is configured.</CardDescription>
      </CardHeader>
      <CardContent className="text-xs text-muted-foreground">
        Configure an OpenTelemetry-compatible trace backend before exploring traces. Console does not generate synthetic
        traces.
      </CardContent>
    </Card>
  );
}
