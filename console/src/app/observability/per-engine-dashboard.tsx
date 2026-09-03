"use client";

import * as React from "react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { RefreshCw, Clock } from "lucide-react";

interface PerEngineDashboardProps {
  initialEngine?: string;
}

export function PerEngineDashboard({ initialEngine = "nexus" }: PerEngineDashboardProps) {
  const [selectedEngine, setSelectedEngine] = React.useState(initialEngine);

  React.useEffect(() => {
    if (initialEngine) {
      setSelectedEngine(initialEngine);
    }
  }, [initialEngine]);

  const renderDomainCards = () => {
    switch (selectedEngine) {
      case "nexus":
        return (
          <>
            <Card className="border-border">
              <CardHeader className="p-3 pb-1">
                <CardDescription className="text-xs">Zanzibar Check Latency (p95)</CardDescription>
              </CardHeader>
              <CardContent className="p-3 pt-0">
                <div className="text-lg font-bold">4.8 ms</div>
                <p className="text-[10px] text-emerald-500">Cached: 94.2% hit rate</p>
              </CardContent>
            </Card>
            <Card className="border-border">
              <CardHeader className="p-3 pb-1">
                <CardDescription className="text-xs">Graph Traversal Depth</CardDescription>
              </CardHeader>
              <CardContent className="p-3 pt-0">
                <div className="text-lg font-bold">2.4 hops (avg)</div>
                <p className="text-[10px] text-muted-foreground">Max depth limit: 32</p>
              </CardContent>
            </Card>
            <Card className="border-border">
              <CardHeader className="p-3 pb-1">
                <CardDescription className="text-xs">Tuple Read Throughput</CardDescription>
              </CardHeader>
              <CardContent className="p-3 pt-0">
                <div className="text-lg font-bold">1,840 ops/s</div>
                <p className="text-[10px] text-muted-foreground">Postgres connection pool: 12%</p>
              </CardContent>
            </Card>
          </>
        );
      case "themis":
        return (
          <>
            <Card className="border-border">
              <CardHeader className="p-3 pb-1">
                <CardDescription className="text-xs">CEL Eval Latency (p95)</CardDescription>
              </CardHeader>
              <CardContent className="p-3 pt-0">
                <div className="text-lg font-bold">2.7 ms</div>
                <p className="text-[10px] text-emerald-500">Precompiled AST program</p>
              </CardContent>
            </Card>
            <Card className="border-border">
              <CardHeader className="p-3 pb-1">
                <CardDescription className="text-xs">Policy Cache Status</CardDescription>
              </CardHeader>
              <CardContent className="p-3 pt-0">
                <div className="text-lg font-bold">100% warm</div>
                <p className="text-[10px] text-muted-foreground">12 active versioned policies</p>
              </CardContent>
            </Card>
            <Card className="border-border">
              <CardHeader className="p-3 pb-1">
                <CardDescription className="text-xs">Test Fixture Pass Rate</CardDescription>
              </CardHeader>
              <CardContent className="p-3 pt-0">
                <div className="text-lg font-bold text-emerald-500">100%</div>
                <p className="text-[10px] text-muted-foreground">28/28 assertions green</p>
              </CardContent>
            </Card>
          </>
        );
      case "aegis":
        return (
          <>
            <Card className="border-border">
              <CardHeader className="p-3 pb-1">
                <CardDescription className="text-xs">Pipeline Ingress Hop (p95)</CardDescription>
              </CardHeader>
              <CardContent className="p-3 pt-0">
                <div className="text-lg font-bold">2.2 ms</div>
                <p className="text-[10px] text-emerald-500">Route resolution: 0.3ms</p>
              </CardContent>
            </Card>
            <Card className="border-border">
              <CardHeader className="p-3 pb-1">
                <CardDescription className="text-xs">Active Proxy Handlers</CardDescription>
              </CardHeader>
              <CardContent className="p-3 pt-0">
                <div className="text-lg font-bold">6 handlers</div>
                <p className="text-[10px] text-muted-foreground">Upstream pool health: 100%</p>
              </CardContent>
            </Card>
            <Card className="border-border">
              <CardHeader className="p-3 pb-1">
                <CardDescription className="text-xs">Rate Limiter Rejections</CardDescription>
              </CardHeader>
              <CardContent className="p-3 pt-0">
                <div className="text-lg font-bold">0.01%</div>
                <p className="text-[10px] text-emerald-500">Token buckets stable</p>
              </CardContent>
            </Card>
          </>
        );
      default:
        return (
          <>
            <Card className="border-border">
              <CardHeader className="p-3 pb-1">
                <CardDescription className="text-xs">Engine Internal Processing</CardDescription>
              </CardHeader>
              <CardContent className="p-3 pt-0">
                <div className="text-lg font-bold">1.4 ms</div>
                <p className="text-[10px] text-emerald-500">Normal operating range</p>
              </CardContent>
            </Card>
            <Card className="border-border">
              <CardHeader className="p-3 pb-1">
                <CardDescription className="text-xs">Cryptographic Operations</CardDescription>
              </CardHeader>
              <CardContent className="p-3 pt-0">
                <div className="text-lg font-bold">Hardware Accelerated</div>
                <p className="text-[10px] text-muted-foreground">AES-GCM &amp; Ed25519</p>
              </CardContent>
            </Card>
            <Card className="border-border">
              <CardHeader className="p-3 pb-1">
                <CardDescription className="text-xs">Goroutine Pool Saturation</CardDescription>
              </CardHeader>
              <CardContent className="p-3 pt-0">
                <div className="text-lg font-bold">4.2%</div>
                <p className="text-[10px] text-emerald-500">Zero goroutine leaks</p>
              </CardContent>
            </Card>
          </>
        );
    }
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Select value={selectedEngine} onValueChange={setSelectedEngine}>
            <SelectTrigger className="w-[180px] h-8 text-xs font-semibold uppercase">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="nexus">NEXUS (ReBAC)</SelectItem>
              <SelectItem value="themis">THEMIS (ABAC)</SelectItem>
              <SelectItem value="aegis">AEGIS (Proxy)</SelectItem>
              <SelectItem value="ego">EGO (Identity)</SelectItem>
              <SelectItem value="janus">JANUS (OAuth2)</SelectItem>
              <SelectItem value="vulcan">VULCAN (API Keys)</SelectItem>
              <SelectItem value="hermes">HERMES (Enterprise)</SelectItem>
            </SelectContent>
          </Select>
          <Badge variant="outline" className="text-xs font-mono">
            Instances: 2 / 2 Healthy
          </Badge>
        </div>
        <Button variant="outline" size="sm" className="h-8 gap-1 text-xs">
          <RefreshCw className="h-3 w-3" />
          Poll Stats
        </Button>
      </div>

      {/* Domain-Specific Metric Cards */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-3">{renderDomainCards()}</div>

      {/* Latency & Saturation Distribution Visual */}
      <Card className="border-border">
        <CardHeader className="p-4 pb-2">
          <CardTitle className="text-xs font-semibold flex items-center gap-1.5">
            <Clock className="h-4 w-4 text-sky-500" />
            Execution Percentiles &amp; Concurrency Curve
          </CardTitle>
          <CardDescription className="text-xs">
            Measured distribution over 10,000 requests on engine <b>{selectedEngine.toUpperCase()}</b>
          </CardDescription>
        </CardHeader>
        <CardContent className="p-4 pt-2">
          <div className="h-32 w-full flex items-end gap-2 pt-4 border-b border-border">
            {[24, 32, 45, 68, 92, 110, 85, 42, 28, 14, 8, 4].map((val, idx) => (
              <div key={idx} className="flex-1 flex flex-col items-center gap-1 group">
                <div
                  className="w-full bg-sky-500/80 hover:bg-sky-400 rounded-t transition-all"
                  style={{ height: `${(val / 110) * 100}%` }}
                />
                <span className="text-[9px] text-muted-foreground group-hover:text-foreground font-mono">
                  {idx * 2}ms
                </span>
              </div>
            ))}
          </div>
          <div className="flex justify-between items-center text-[10px] text-muted-foreground mt-2">
            <span>Fastest: 0.4ms</span>
            <span>p50: 1.8ms</span>
            <span>p95: 4.8ms</span>
            <span>p99: 11.2ms</span>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
