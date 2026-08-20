"use client";

import * as React from "react";
import {
  Palette,
  Type,
  Layers,
  Sparkles,
  ShieldCheck,
  AlertTriangle,
  XCircle,
  CheckCircle2,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs";

export default function DesignSystemPage() {
  const [density, setDensity] = React.useState<"comfortable" | "compact">("comfortable");

  return (
    <div className="space-y-8 max-w-6xl mx-auto pb-16">
      {/* Header */}
      <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between border-b border-border/60 pb-6">
        <div>
          <div className="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full border border-primary/20 bg-primary/5 text-primary text-xs font-mono mb-2">
            <Sparkles className="w-3 h-3" />
            DESIGN SYSTEM & COMPONENT GALLERY (P4-S1)
          </div>
          <h1 className="text-3xl font-bold tracking-tight text-foreground">Autorix Design Tokens & Primitives</h1>
          <p className="text-sm text-muted-foreground mt-1">
            Standard design tokens, typography, semantic status scales, and UI primitives for the Autorix Control Plane.
          </p>
        </div>

        <div className="flex items-center gap-2">
          <span className="text-xs font-medium text-muted-foreground font-mono">Density:</span>
          <div className="inline-flex rounded-lg border border-border bg-card p-0.5">
            <button
              onClick={() => setDensity("comfortable")}
              className={`px-3 py-1 text-xs rounded-md font-medium transition-colors ${
                density === "comfortable" ? "bg-primary text-primary-foreground" : "text-muted-foreground hover:text-foreground"
              }`}
            >
              Comfortable
            </button>
            <button
              onClick={() => setDensity("compact")}
              className={`px-3 py-1 text-xs rounded-md font-medium transition-colors ${
                density === "compact" ? "bg-primary text-primary-foreground" : "text-muted-foreground hover:text-foreground"
              }`}
            >
              Compact
            </button>
          </div>
        </div>
      </div>

      <Tabs defaultValue="tokens" className="space-y-6">
        <TabsList className="grid grid-cols-4 w-full max-w-md">
          <TabsTrigger value="tokens" className="text-xs gap-1.5">
            <Palette className="w-3.5 h-3.5" /> Tokens
          </TabsTrigger>
          <TabsTrigger value="semantic" className="text-xs gap-1.5">
            <ShieldCheck className="w-3.5 h-3.5" /> Status Scales
          </TabsTrigger>
          <TabsTrigger value="typography" className="text-xs gap-1.5">
            <Type className="w-3.5 h-3.5" /> Typography
          </TabsTrigger>
          <TabsTrigger value="primitives" className="text-xs gap-1.5">
            <Layers className="w-3.5 h-3.5" /> Primitives
          </TabsTrigger>
        </TabsList>

        {/* 1. Tokens Tab */}
        <TabsContent value="tokens" className="space-y-6">
          <Card className="border-border/80 bg-card/60">
            <CardHeader>
              <CardTitle className="text-base font-semibold">Core Color Scales</CardTitle>
              <CardDescription className="text-xs">Dynamic HSL CSS custom properties responsive to light/dark themes</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                <div className="p-4 rounded-lg bg-background border border-border space-y-2">
                  <div className="h-10 rounded bg-background border border-border" />
                  <div className="text-xs font-mono font-medium">--background</div>
                  <div className="text-[10px] text-muted-foreground font-mono">hsl(var(--background))</div>
                </div>
                <div className="p-4 rounded-lg bg-card border border-border space-y-2">
                  <div className="h-10 rounded bg-card border border-border" />
                  <div className="text-xs font-mono font-medium">--card</div>
                  <div className="text-[10px] text-muted-foreground font-mono">hsl(var(--card))</div>
                </div>
                <div className="p-4 rounded-lg bg-card border border-border space-y-2">
                  <div className="h-10 rounded bg-primary text-primary-foreground flex items-center justify-center font-bold text-xs">
                    Primary
                  </div>
                  <div className="text-xs font-mono font-medium">--primary</div>
                  <div className="text-[10px] text-muted-foreground font-mono">hsl(var(--primary))</div>
                </div>
                <div className="p-4 rounded-lg bg-card border border-border space-y-2">
                  <div className="h-10 rounded bg-secondary text-secondary-foreground flex items-center justify-center font-bold text-xs">
                    Secondary
                  </div>
                  <div className="text-xs font-mono font-medium">--secondary</div>
                  <div className="text-[10px] text-muted-foreground font-mono">hsl(var(--secondary))</div>
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* 2. Semantic Status Tab */}
        <TabsContent value="semantic" className="space-y-6">
          <Card className="border-border/80 bg-card/60">
            <CardHeader>
              <CardTitle className="text-base font-semibold">Semantic Status Scales (P4-S1-T3)</CardTitle>
              <CardDescription className="text-xs">Accessible status indicators paired with iconography for color-blind clarity</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                <div className="p-4 rounded-xl border border-emerald-500/30 bg-emerald-500/10 flex items-center gap-3">
                  <CheckCircle2 className="w-5 h-5 text-emerald-400 shrink-0" />
                  <div>
                    <h4 className="text-xs font-semibold text-emerald-300 font-mono">HEALTHY / ALLOW</h4>
                    <p className="text-[11px] text-emerald-400/80">Engine active, policy permitted, 200 OK</p>
                  </div>
                </div>
                <div className="p-4 rounded-xl border border-amber-500/30 bg-amber-500/10 flex items-center gap-3">
                  <AlertTriangle className="w-5 h-5 text-amber-400 shrink-0" />
                  <div>
                    <h4 className="text-xs font-semibold text-amber-300 font-mono">DEGRADED / WARNING</h4>
                    <p className="text-[11px] text-amber-400/80">Partial failure, high latency, drift detected</p>
                  </div>
                </div>
                <div className="p-4 rounded-xl border border-rose-500/30 bg-rose-500/10 flex items-center gap-3">
                  <XCircle className="w-5 h-5 text-rose-400 shrink-0" />
                  <div>
                    <h4 className="text-xs font-semibold text-rose-300 font-mono">UNREACHABLE / DENY</h4>
                    <p className="text-[11px] text-rose-400/80">Engine stopped, policy forbidden, 502/403</p>
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* 3. Typography Tab */}
        <TabsContent value="typography" className="space-y-6">
          <Card className="border-border/80 bg-card/60">
            <CardHeader>
              <CardTitle className="text-base font-semibold">Typography Scale & Tabular Numerals (P4-S1-T4)</CardTitle>
              <CardDescription className="text-xs">Standard type hierarchy with proportional lining numbers and monospace fonts</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-1 pb-3 border-b border-border/60">
                <span className="text-xs font-mono text-muted-foreground">Heading 1 / 30px Bold</span>
                <h1 className="text-3xl font-bold">Autorix IAM Platform Orchestration</h1>
              </div>
              <div className="space-y-1 pb-3 border-b border-border/60">
                <span className="text-xs font-mono text-muted-foreground">Heading 2 / 24px Bold</span>
                <h2 className="text-2xl font-bold">Fleet Engine Topologies & Clusters</h2>
              </div>
              <div className="space-y-1 pb-3 border-b border-border/60">
                <span className="text-xs font-mono text-muted-foreground">Body Text / 14px Regular</span>
                <p className="text-sm text-foreground">
                  The control plane provides unified orchestration across the six specialized IAM engines.
                </p>
              </div>
              <div className="space-y-1">
                <span className="text-xs font-mono text-muted-foreground">Tabular Numerals (tabular-nums)</span>
                <div className="font-mono text-sm tabular-nums space-y-1">
                  <div>Latency: 0.124ms | Requests: 1,420,593 | Success: 99.98%</div>
                  <div>Latency: 1.849ms | Requests: 0,082,104 | Success: 98.40%</div>
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* 4. Primitives Tab */}
        <TabsContent value="primitives" className="space-y-6">
          <Card className="border-border/80 bg-card/60">
            <CardHeader>
              <CardTitle className="text-base font-semibold">UI Primitives & Density Demo</CardTitle>
              <CardDescription className="text-xs">Button variants, Badge scales, and Inputs</CardDescription>
            </CardHeader>
            <CardContent className="space-y-6">
              <div className="space-y-2">
                <span className="text-xs font-mono text-muted-foreground">Buttons</span>
                <div className="flex flex-wrap gap-2">
                  <Button variant="default" size={density === "compact" ? "sm" : "default"}>Default</Button>
                  <Button variant="secondary" size={density === "compact" ? "sm" : "default"}>Secondary</Button>
                  <Button variant="outline" size={density === "compact" ? "sm" : "default"}>Outline</Button>
                  <Button variant="destructive" size={density === "compact" ? "sm" : "default"}>Destructive</Button>
                  <Button variant="ghost" size={density === "compact" ? "sm" : "default"}>Ghost</Button>
                </div>
              </div>

              <div className="space-y-2">
                <span className="text-xs font-mono text-muted-foreground">Badges</span>
                <div className="flex flex-wrap gap-2">
                  <Badge variant="default">Primary</Badge>
                  <Badge variant="secondary">Secondary</Badge>
                  <Badge variant="outline">Outline</Badge>
                  <Badge variant="destructive">Destructive</Badge>
                </div>
              </div>

              <div className="space-y-2 max-w-sm">
                <span className="text-xs font-mono text-muted-foreground">Form Inputs</span>
                <Input placeholder="Enter token identifier or principal name..." />
              </div>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  );
}
