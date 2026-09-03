"use client";

import * as React from "react";
import { Activity, Server, FileText, Network, Bell, Target } from "lucide-react";
import { Button } from "@/components/ui/button";
import { FleetDashboard } from "./fleet-dashboard";
import { PerEngineDashboard } from "./per-engine-dashboard";
import { LogViewer } from "./log-viewer";
import { TraceExplorer } from "./trace-explorer";
import { AlertsManager } from "./alerts-manager";
import { SLODashboard } from "./slo-dashboard";

export default function ObservabilityPage() {
  const [activeTab, setActiveTab] = React.useState<
    "fleet" | "engine" | "logs" | "traces" | "alerts" | "slo"
  >("fleet");
  const [selectedEngine, setSelectedEngine] = React.useState<string>("nexus");
  const [selectedTraceId, setSelectedTraceId] = React.useState<string | undefined>(undefined);

  const handleSelectEngine = (eng: string) => {
    setSelectedEngine(eng);
    setActiveTab("engine");
  };

  const handleSelectTrace = (trId: string) => {
    setSelectedTraceId(trId);
    setActiveTab("traces");
  };

  return (
    <div className="space-y-4">
      {/* Header */}
      <div>
        <h1 className="text-xl font-bold tracking-tight">Fleet Observability &amp; Telemetry</h1>
        <p className="text-xs text-muted-foreground mt-0.5">
          Real-time RED instrumentation, structured logs, distributed trace waterfalls, alerts and SLOs
        </p>
      </div>

      {/* Navigation Sub-Tabs */}
      <div className="flex flex-wrap items-center gap-1.5 border-b border-border pb-2">
        <Button
          variant={activeTab === "fleet" ? "default" : "ghost"}
          size="sm"
          onClick={() => setActiveTab("fleet")}
          className="h-8 text-xs gap-1.5"
        >
          <Activity className="h-3.5 w-3.5" />
          Fleet Overview
        </Button>
        <Button
          variant={activeTab === "engine" ? "default" : "ghost"}
          size="sm"
          onClick={() => setActiveTab("engine")}
          className="h-8 text-xs gap-1.5"
        >
          <Server className="h-3.5 w-3.5" />
          Engine Deep Dive
        </Button>
        <Button
          variant={activeTab === "logs" ? "default" : "ghost"}
          size="sm"
          onClick={() => setActiveTab("logs")}
          className="h-8 text-xs gap-1.5"
        >
          <FileText className="h-3.5 w-3.5" />
          Structured Logs
        </Button>
        <Button
          variant={activeTab === "traces" ? "default" : "ghost"}
          size="sm"
          onClick={() => setActiveTab("traces")}
          className="h-8 text-xs gap-1.5"
        >
          <Network className="h-3.5 w-3.5" />
          Trace Explorer
        </Button>
        <Button
          variant={activeTab === "alerts" ? "default" : "ghost"}
          size="sm"
          onClick={() => setActiveTab("alerts")}
          className="h-8 text-xs gap-1.5"
        >
          <Bell className="h-3.5 w-3.5" />
          Alerts &amp; Rules
        </Button>
        <Button
          variant={activeTab === "slo" ? "default" : "ghost"}
          size="sm"
          onClick={() => setActiveTab("slo")}
          className="h-8 text-xs gap-1.5"
        >
          <Target className="h-3.5 w-3.5" />
          SLOs &amp; Budgets
        </Button>
      </div>

      {/* Tab Panels */}
      {activeTab === "fleet" && <FleetDashboard onSelectEngine={handleSelectEngine} />}
      {activeTab === "engine" && <PerEngineDashboard initialEngine={selectedEngine} />}
      {activeTab === "logs" && <LogViewer onSelectTrace={handleSelectTrace} />}
      {activeTab === "traces" && <TraceExplorer initialTraceId={selectedTraceId} />}
      {activeTab === "alerts" && <AlertsManager />}
      {activeTab === "slo" && <SLODashboard />}
    </div>
  );
}
