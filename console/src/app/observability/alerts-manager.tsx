"use client";
import * as React from "react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { AlertTriangle, RefreshCw } from "lucide-react";
import type { AlertEvent, AlertRule } from "@/lib/api/schemas/observability";

export function AlertsManager() {
  const [alerts, setAlerts] = React.useState<AlertEvent[]>([]); const [rules, setRules] = React.useState<AlertRule[]>([]); const [error, setError] = React.useState<string>();
  const load = React.useCallback(async () => { setError(undefined); const [alertsResponse, rulesResponse] = await Promise.all([fetch("/api/observability/alerts"), fetch("/api/observability/alerts/rules")]); if (!alertsResponse.ok || !rulesResponse.ok) { setError("Prometheus alerts or rules are unavailable."); return; } setAlerts(await alertsResponse.json()); setRules(await rulesResponse.json()); }, []);
  React.useEffect(() => { load(); }, [load]);
  return <div className="space-y-4"><div className="flex justify-end"><Button size="sm" variant="outline" onClick={load}><RefreshCw className="mr-1 h-3 w-3" />Refresh</Button></div>
    <Card><CardHeader><CardTitle className="text-sm flex gap-2"><AlertTriangle className="h-4 w-4" />Prometheus alerts</CardTitle><CardDescription>Read-only. Silence, acknowledgement, routing, and delivery are managed by Alertmanager.</CardDescription></CardHeader><CardContent className="space-y-2 text-xs">{error ? <p className="text-muted-foreground">{error}</p> : alerts.length === 0 ? <p className="text-muted-foreground">No firing alerts reported by Prometheus.</p> : alerts.map((alert) => <div key={alert.id} className="flex gap-2 border-b pb-2"><Badge variant={alert.severity === "critical" ? "destructive" : "secondary"}>{alert.severity}</Badge><span>{alert.rule_name}</span><span className="text-muted-foreground">{alert.engine_type}</span></div>)}</CardContent></Card>
    <Card><CardHeader><CardTitle className="text-sm">Configured alert rules</CardTitle></CardHeader><CardContent className="space-y-2 text-xs">{error ? <p className="text-muted-foreground">Rule data is unavailable.</p> : rules.length === 0 ? <p className="text-muted-foreground">No Prometheus alert rules are configured.</p> : rules.map((rule) => <div key={rule.id} className="border-b pb-2"><b>{rule.name}</b><div className="font-mono text-muted-foreground">{rule.metric}</div></div>)}</CardContent></Card>
    <Card><CardHeader><CardTitle className="text-sm">Notification delivery</CardTitle></CardHeader><CardContent className="text-xs text-muted-foreground">Configure receivers, webhooks, and email in Alertmanager. Autorix Console does not store or deliver notifications.</CardContent></Card>
  </div>;
}
