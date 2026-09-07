"use client";

import * as React from "react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Server } from "lucide-react";

interface PerEngineDashboardProps {
  initialEngine?: string;
}
const engines = ["aegis", "ego", "janus", "nexus", "themis", "vulcan", "hermes"];

export function PerEngineDashboard({ initialEngine = "nexus" }: PerEngineDashboardProps) {
  const [selectedEngine, setSelectedEngine] = React.useState(initialEngine);
  React.useEffect(() => setSelectedEngine(initialEngine), [initialEngine]);
  return (
    <div className="space-y-4">
      <Select value={selectedEngine} onValueChange={setSelectedEngine}>
        <SelectTrigger className="w-[180px] h-8 text-xs font-semibold uppercase">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {engines.map((engine) => (
            <SelectItem key={engine} value={engine}>
              {engine.toUpperCase()}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      <Card className="border-border">
        <CardHeader>
          <CardTitle className="text-sm flex gap-2">
            <Server className="h-4 w-4" />
            {selectedEngine.toUpperCase()} telemetry
          </CardTitle>
          <CardDescription>Per-engine metric recording rules are not configured.</CardDescription>
        </CardHeader>
        <CardContent className="text-xs text-muted-foreground">
          No values are displayed until Prometheus exposes engine-labelled metrics. The Fleet Overview shows only
          observed metrics.
        </CardContent>
      </Card>
    </div>
  );
}
