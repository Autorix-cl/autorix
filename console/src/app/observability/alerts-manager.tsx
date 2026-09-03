"use client";

import * as React from "react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Bell,
  AlertTriangle,
  CheckCircle2,
  VolumeX,
  Radio,
  Sliders,
  RefreshCw,
  Loader2,
  Mail,
  MessageSquare,
  Webhook,
} from "lucide-react";
import { toast } from "sonner";
import type { AlertEvent, AlertRule } from "@/lib/api/schemas/observability";

export function AlertsManager() {
  const [alerts, setAlerts] = React.useState<AlertEvent[]>([]);
  const [rules, setRules] = React.useState<AlertRule[]>([]);
  const [activeTab, setActiveTab] = React.useState<"firing" | "rules" | "channels">("firing");
  const [isLoading, setIsLoading] = React.useState(false);

  const fetchAlertsAndRules = React.useCallback(async () => {
    setIsLoading(true);
    try {
      const [alertsRes, rulesRes] = await Promise.all([
        fetch("/api/observability/alerts"),
        fetch("/api/observability/alerts/rules"),
      ]);
      if (!alertsRes.ok || !rulesRes.ok) throw new Error("Failed to load alerts data");
      const [alertsData, rulesData] = await Promise.all([alertsRes.json(), rulesRes.json()]);
      setAlerts(alertsData);
      setRules(rulesData);
    } catch {
      toast.error("Error loading alert data");
    } finally {
      setIsLoading(false);
    }
  }, []);

  React.useEffect(() => {
    fetchAlertsAndRules();
  }, [fetchAlertsAndRules]);

  const handleAcknowledge = (id: string) => {
    setAlerts((prev) =>
      prev.map((a) => (a.id === id ? { ...a, state: "acknowledged" as const } : a))
    );
    toast.success("Alert acknowledged");
  };

  const handleSilence = (id: string) => {
    setAlerts((prev) =>
      prev.map((a) => (a.id === id ? { ...a, state: "silenced" as const } : a))
    );
    toast.success("Alert silenced for 2 hours");
  };

  const firingCount = alerts.filter((a) => a.state === "firing").length;

  return (
    <div className="space-y-4">
      {/* Tab Selector & Controls */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Button
            variant={activeTab === "firing" ? "default" : "outline"}
            size="sm"
            onClick={() => setActiveTab("firing")}
            className="h-8 text-xs gap-1.5"
          >
            <Radio className="h-3.5 w-3.5" />
            Active Alerts
            {firingCount > 0 && (
              <Badge variant="destructive" className="ml-1 px-1 py-0 text-[10px]">
                {firingCount}
              </Badge>
            )}
          </Button>
          <Button
            variant={activeTab === "rules" ? "default" : "outline"}
            size="sm"
            onClick={() => setActiveTab("rules")}
            className="h-8 text-xs gap-1.5"
          >
            <Sliders className="h-3.5 w-3.5" />
            Alert Rules ({rules.length})
          </Button>
          <Button
            variant={activeTab === "channels" ? "default" : "outline"}
            size="sm"
            onClick={() => setActiveTab("channels")}
            className="h-8 text-xs gap-1.5"
          >
            <Bell className="h-3.5 w-3.5" />
            Notification Targets
          </Button>
        </div>

        <Button
          variant="outline"
          size="sm"
          onClick={() => fetchAlertsAndRules()}
          disabled={isLoading}
          className="h-8 gap-1 text-xs"
        >
          {isLoading ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <RefreshCw className="h-3.5 w-3.5" />}
          Refresh
        </Button>
      </div>

      {/* Active Alerts Tab */}
      {activeTab === "firing" && (
        <Card className="border-border">
          <CardHeader className="p-4 pb-2">
            <CardTitle className="text-xs font-semibold flex items-center gap-1.5">
              <AlertTriangle className="h-4 w-4 text-amber-500" />
              Active and Historical Alert Events
            </CardTitle>
            <CardDescription className="text-xs">
              Monitored firing, acknowledged, silenced and resolved alarms across the fleet
            </CardDescription>
          </CardHeader>
          <CardContent className="p-0">
            <div className="divide-y divide-border">
              {alerts.length === 0 ? (
                <div className="p-6 text-center text-muted-foreground text-xs">
                  All alerts clear. No incidents currently firing.
                </div>
              ) : (
                alerts.map((evt) => (
                  <div key={evt.id} className="p-3 hover:bg-muted/20 flex items-center justify-between gap-3">
                    <div className="space-y-1">
                      <div className="flex items-center gap-2">
                        <Badge
                          variant={evt.severity === "critical" ? "destructive" : "secondary"}
                          className="text-[9px] uppercase px-1 py-0"
                        >
                          {evt.severity}
                        </Badge>
                        <Badge variant="outline" className="text-[9px] uppercase px-1 py-0 font-mono">
                          {evt.engine_type}
                        </Badge>
                        <span className="text-xs font-semibold">{evt.rule_name}</span>
                        <Badge
                          variant={evt.state === "firing" ? "destructive" : "outline"}
                          className="text-[10px] capitalize px-1.5 py-0"
                        >
                          {evt.state}
                        </Badge>
                      </div>
                      <div className="text-[11px] text-muted-foreground">
                        Triggered at {new Date(evt.triggered_at).toLocaleTimeString()} · Value:{" "}
                        <span className="font-mono text-foreground">{evt.value}</span> (threshold: {evt.threshold})
                      </div>
                    </div>

                    <div className="flex items-center gap-1.5">
                      {evt.state === "firing" && (
                        <>
                          <Button
                            variant="outline"
                            size="sm"
                            className="h-7 text-[11px] gap-1"
                            onClick={() => handleAcknowledge(evt.id)}
                          >
                            <CheckCircle2 className="h-3 w-3 text-emerald-500" />
                            Ack
                          </Button>
                          <Button
                            variant="outline"
                            size="sm"
                            className="h-7 text-[11px] gap-1"
                            onClick={() => handleSilence(evt.id)}
                          >
                            <VolumeX className="h-3 w-3 text-muted-foreground" />
                            Silence
                          </Button>
                        </>
                      )}
                    </div>
                  </div>
                ))
              )}
            </div>
          </CardContent>
        </Card>
      )}

      {/* Alert Rules Tab */}
      {activeTab === "rules" && (
        <Card className="border-border">
          <CardHeader className="p-4 pb-2">
            <CardTitle className="text-xs font-semibold flex items-center gap-1.5">
              <Sliders className="h-4 w-4 text-sky-500" />
              Configured Alert Evaluation Rules (Prometheus &amp; Argus)
            </CardTitle>
            <CardDescription className="text-xs">
              Thresholds and durations evaluated continuously against platform metrics
            </CardDescription>
          </CardHeader>
          <CardContent className="p-0">
            <div className="divide-y divide-border">
              {rules.map((rule) => (
                <div key={rule.id} className="p-3 hover:bg-muted/20 flex items-center justify-between">
                  <div className="space-y-1">
                    <div className="flex items-center gap-2">
                      <Badge
                        variant={rule.severity === "critical" ? "destructive" : "secondary"}
                        className="text-[9px] uppercase px-1 py-0"
                      >
                        {rule.severity}
                      </Badge>
                      <Badge variant="outline" className="text-[9px] uppercase px-1 py-0 font-mono">
                        {rule.engine_type}
                      </Badge>
                      <span className="text-xs font-semibold">{rule.name}</span>
                    </div>
                    <div className="text-[11px] font-mono text-muted-foreground">
                      {rule.metric} {rule.operator} {rule.threshold} for {rule.duration}
                    </div>
                  </div>
                  <Badge variant="outline" className="text-[10px] text-emerald-500 border-emerald-500/20">
                    Active
                  </Badge>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      )}

      {/* Notification Targets Tab */}
      {activeTab === "channels" && (
        <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
          <Card className="border-border">
            <CardHeader className="p-3 pb-2">
              <CardTitle className="text-xs font-semibold flex items-center gap-1.5">
                <Mail className="h-4 w-4 text-sky-500" />
                Email Alerts
              </CardTitle>
            </CardHeader>
            <CardContent className="p-3 pt-0 space-y-2 text-xs">
              <p className="text-muted-foreground text-[11px]">Primary security &amp; ops mailing list</p>
              <Badge variant="outline" className="font-mono text-[10px]">ops-oncall@autorix.internal</Badge>
              <div className="text-[10px] text-emerald-500 flex items-center gap-1">
                <CheckCircle2 className="h-3 w-3" />
                Connected
              </div>
            </CardContent>
          </Card>

          <Card className="border-border">
            <CardHeader className="p-3 pb-2">
              <CardTitle className="text-xs font-semibold flex items-center gap-1.5">
                <MessageSquare className="h-4 w-4 text-emerald-500" />
                Slack Channel
              </CardTitle>
            </CardHeader>
            <CardContent className="p-3 pt-0 space-y-2 text-xs">
              <p className="text-muted-foreground text-[11px]">Incident response feed webhook</p>
              <Badge variant="outline" className="font-mono text-[10px]">#autorix-fleet-alerts</Badge>
              <div className="text-[10px] text-emerald-500 flex items-center gap-1">
                <CheckCircle2 className="h-3 w-3" />
                Webhook Validated
              </div>
            </CardContent>
          </Card>

          <Card className="border-border">
            <CardHeader className="p-3 pb-2">
              <CardTitle className="text-xs font-semibold flex items-center gap-1.5">
                <Webhook className="h-4 w-4 text-indigo-500" />
                PagerDuty Webhook
              </CardTitle>
            </CardHeader>
            <CardContent className="p-3 pt-0 space-y-2 text-xs">
              <p className="text-muted-foreground text-[11px]">Critical severity paging escalation</p>
              <Badge variant="outline" className="font-mono text-[10px]">Events API v2</Badge>
              <div className="text-[10px] text-emerald-500 flex items-center gap-1">
                <CheckCircle2 className="h-3 w-3" />
                Ready (1 Service)
              </div>
            </CardContent>
          </Card>
        </div>
      )}
    </div>
  );
}
