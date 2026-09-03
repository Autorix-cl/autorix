"use client";

import * as React from "react";
import { User, ShieldCheck, Zap, ShieldAlert } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { UnifiedSubjectView } from "./unified-subject-view";
import { EffectiveAccessExplorer } from "./effective-access-explorer";
import { RequestSimulator } from "./request-simulator";
import { ConsistencyChecker } from "./consistency-checker";

export default function ExplorerPage() {
  const [activeTab, setActiveTab] = React.useState<"subject" | "access" | "simulator" | "consistency">("subject");

  return (
    <div className="space-y-6 p-8">
      {/* Page Header */}
      <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
        <div>
          <div className="flex items-center gap-2">
            <h1 className="text-xl font-bold tracking-tight text-foreground sm:text-2xl">
              Cross-Engine Intelligence & Explorer
            </h1>
            <Badge variant="outline" className="text-[10px] font-mono border-primary/40 text-primary">
              PHASE 6 DEPTH
            </Badge>
          </div>
          <p className="text-xs text-muted-foreground mt-1">
            Correlate identity, authorization, proxies, and policies across all 7 Autorix Zero-Trust engines.
          </p>
        </div>

        {/* View Switcher Tabs */}
        <div className="flex items-center gap-1.5 bg-muted/40 p-1 rounded-lg border">
          <Button
            variant={activeTab === "subject" ? "default" : "ghost"}
            size="sm"
            onClick={() => setActiveTab("subject")}
            className="h-7 text-xs gap-1.5"
          >
            <User className="w-3.5 h-3.5" />
            Unified Subject
          </Button>
          <Button
            variant={activeTab === "access" ? "default" : "ghost"}
            size="sm"
            onClick={() => setActiveTab("access")}
            className="h-7 text-xs gap-1.5"
          >
            <ShieldCheck className="w-3.5 h-3.5" />
            Effective Access
          </Button>
          <Button
            variant={activeTab === "simulator" ? "default" : "ghost"}
            size="sm"
            onClick={() => setActiveTab("simulator")}
            className="h-7 text-xs gap-1.5"
          >
            <Zap className="w-3.5 h-3.5" />
            Request Simulator
          </Button>
          <Button
            variant={activeTab === "consistency" ? "default" : "ghost"}
            size="sm"
            onClick={() => setActiveTab("consistency")}
            className="h-7 text-xs gap-1.5"
          >
            <ShieldAlert className="w-3.5 h-3.5" />
            Consistency Checks
          </Button>
        </div>
      </div>

      {/* Tab Panels */}
      {activeTab === "subject" && <UnifiedSubjectView />}
      {activeTab === "access" && <EffectiveAccessExplorer />}
      {activeTab === "simulator" && <RequestSimulator />}
      {activeTab === "consistency" && <ConsistencyChecker />}
    </div>
  );
}
