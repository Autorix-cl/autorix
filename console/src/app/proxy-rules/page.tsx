"use client";

import * as React from "react";
import { 
  Shield, 
  Play, 
  ArrowRight, 
  CheckCircle2, 
  Lock, 
  FileCode, 
  Globe,
  Sliders,
  Sparkles,
  Server
} from "lucide-react";
import { useTranslation } from "@/lib/i18n";
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { 
  Select, 
  SelectContent, 
  SelectItem, 
  SelectTrigger, 
  SelectValue 
} from "@/components/ui/select";
import { CodeBlock } from "@/components/ui/code-block";

interface RuleItem {
  id: string;
  match: {
    methods: string[];
    url: string;
  };
  authenticator: string;
  authorizer: string;
  mutator: string;
  upstream: string;
}

// apiRule mirrors aegis's core.Rule JSON shape (GET /rules).
interface ApiHandlerConfig {
  handler: string;
  config?: Record<string, unknown>;
}
interface ApiRule {
  id: string;
  description?: string;
  match: { url: string; methods: string[] };
  authenticators: ApiHandlerConfig[];
  authorizer: ApiHandlerConfig;
  mutators: ApiHandlerConfig[];
  upstream: { url: string };
}

function toRuleItem(r: ApiRule): RuleItem {
  return {
    id: r.id,
    match: { methods: r.match.methods, url: r.match.url },
    authenticator: r.authenticators.map((a) => a.handler).join(", ") || "—",
    authorizer: r.authorizer?.handler || "—",
    mutator: r.mutators.map((m) => m.handler).join(", ") || "—",
    upstream: r.upstream.url,
  };
}

export default function ProxyRulesPage() {
  const { t } = useTranslation();

  const [apiRules, setApiRules] = React.useState<ApiRule[]>([]);
  const [rules, setRules] = React.useState<RuleItem[]>([]);
  const [loadingRules, setLoadingRules] = React.useState(true);

  const fetchRules = React.useCallback(async () => {
    setLoadingRules(true);
    try {
      const res = await fetch("/api/proxy-rules");
      const data = await res.json();
      const list: ApiRule[] = Array.isArray(data) ? data : [];
      setApiRules(list);
      setRules(list.map(toRuleItem));
    } catch (err) {
      setApiRules([]);
      setRules([]);
    } finally {
      setLoadingRules(false);
    }
  }, []);

  React.useEffect(() => {
    fetchRules();
  }, [fetchRules]);

  const [testPath, setTestPath] = React.useState("/api/v1/documents/financial_report_2026");
  const [testMethod, setTestMethod] = React.useState("GET");
  const [matchedRule, setMatchedRule] = React.useState<RuleItem | null>(null);
  const [testing, setTesting] = React.useState(false);

  const handleTestMatch = async () => {
    setTesting(true);
    try {
      const res = await fetch("/api/proxy-rules/test-match", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ method: testMethod, path: testPath }),
      });
      const data = await res.json();
      setMatchedRule(data.matched ? toRuleItem(data.rule as ApiRule) : null);
    } catch (err) {
      setMatchedRule(null);
    } finally {
      setTesting(false);
    }
  };

  const rulesJSON = JSON.stringify(apiRules, null, 2);

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
        <div>
          <h1 className="text-xl font-bold tracking-tight text-foreground sm:text-2xl">
            {t("proxyRules.title")}
          </h1>
          <p className="text-xs text-muted-foreground mt-1">
            {t("proxyRules.subtitle")}
          </p>
        </div>

        <div className="flex items-center gap-2">
          <Badge variant="success" className="gap-1.5 py-1 px-3">
            <Shield className="h-3.5 w-3.5" />
            <span>{t("proxyRules.statusBadge")}</span>
          </Badge>
        </div>
      </div>

      {/* Simulator Card */}
      <Card className="bg-card/80">
        <CardHeader className="p-6 pb-4">
          <div className="flex items-center gap-2">
            <Play className="h-4 w-4 text-emerald-400" />
            <CardTitle className="text-sm font-semibold">
              {t("proxyRules.simulatorTitle")}
            </CardTitle>
          </div>
          <CardDescription className="text-xs">
            {t("proxyRules.simulatorDesc")}
          </CardDescription>
        </CardHeader>

        <CardContent className="p-6 pt-0 space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-12 gap-3 items-end">
            <div className="md:col-span-3 space-y-1.5">
              <Label>{t("proxyRules.methodLabel")}</Label>
              <Select value={testMethod} onValueChange={setTestMethod}>
                <SelectTrigger>
                  <SelectValue placeholder="Select method" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="GET">GET</SelectItem>
                  <SelectItem value="POST">POST</SelectItem>
                  <SelectItem value="PUT">PUT</SelectItem>
                  <SelectItem value="DELETE">DELETE</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div className="md:col-span-6 space-y-1.5">
              <Label>{t("proxyRules.pathLabel")}</Label>
              <Input
                value={testPath}
                onChange={(e) => setTestPath(e.target.value)}
                placeholder="/api/v1/resource"
                className="font-mono text-xs"
              />
            </div>

            <div className="md:col-span-3">
              <Button
                type="button"
                variant="success"
                onClick={handleTestMatch}
                disabled={testing}
                className="w-full gap-2"
              >
                <Play className="h-4 w-4" />
                <span>{testing ? t("proxyRules.testingBtn") : t("proxyRules.simulateBtn")}</span>
              </Button>
            </div>
          </div>

          {/* Matched Rule Result Visualization */}
          {matchedRule ? (
            <div className="rounded-xl border border-border/80 bg-muted/20 p-4 space-y-4">
              <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-2 border-b border-border/50 pb-3">
                <Badge variant="success" className="gap-1.5 py-0.5 font-mono text-xs">
                  <CheckCircle2 className="h-3.5 w-3.5" />
                  <span>{t("proxyRules.matchedBadge", { id: matchedRule.id })}</span>
                </Badge>
                <div className="flex items-center gap-2 font-mono text-xs text-muted-foreground">
                  <Server className="h-3.5 w-3.5 text-blue-400" />
                  <span>{t("proxyRules.upstreamLabel")}: <strong className="text-foreground">{matchedRule.upstream}</strong></span>
                </div>
              </div>

              {/* 3 Pipeline Stages */}
              <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
                <div className="rounded-lg border border-border/60 bg-card p-3 space-y-1.5">
                  <div className="text-[10px] font-bold uppercase text-blue-400">
                    {t("proxyRules.step1")}
                  </div>
                  <div className="text-xs font-semibold text-foreground font-mono">
                    {matchedRule.authenticator}
                  </div>
                  <div className="text-[11px] text-muted-foreground">
                    Verifies cryptographic claims & token validity
                  </div>
                </div>

                <div className="rounded-lg border border-border/60 bg-card p-3 space-y-1.5">
                  <div className="text-[10px] font-bold uppercase text-purple-400">
                    {t("proxyRules.step2")}
                  </div>
                  <div className="text-xs font-semibold text-foreground font-mono">
                    {matchedRule.authorizer}
                  </div>
                  <div className="text-[11px] text-muted-foreground">
                    Zero-latency Zanzibar relation & CEL condition
                  </div>
                </div>

                <div className="rounded-lg border border-border/60 bg-card p-3 space-y-1.5">
                  <div className="text-[10px] font-bold uppercase text-emerald-400">
                    {t("proxyRules.step3")}
                  </div>
                  <div className="text-xs font-semibold text-foreground font-mono">
                    {matchedRule.mutator}
                  </div>
                  <div className="text-[11px] text-muted-foreground">
                    Injects verified identity claims into upstream headers
                  </div>
                </div>
              </div>
            </div>
          ) : (
            <div className="rounded-xl border border-rose-500/30 bg-rose-500/10 p-4 text-xs text-rose-400 font-medium">
              {t("proxyRules.noMatch")}
            </div>
          )}
        </CardContent>
      </Card>

      {/* Declarative Rules File Viewer */}
      <Card className="bg-card/80">
        <CardHeader className="p-6 pb-4">
          <div className="flex items-center gap-2">
            <FileCode className="h-4 w-4 text-blue-400" />
            <CardTitle className="text-sm font-semibold">
              {t("proxyRules.yamlTitle")}
            </CardTitle>
          </div>
          <CardDescription className="text-xs">
            {t("proxyRules.yamlDesc")}
          </CardDescription>
        </CardHeader>

        <CardContent className="p-6 pt-0">
          {loadingRules ? (
            <div className="flex h-24 items-center justify-center text-xs text-muted-foreground">
              Loading rules from Aegis…
            </div>
          ) : (
            <CodeBlock
              code={rulesJSON}
              language="json"
              title="GET /rules (Aegis admin API, live)"
              showLineNumbers
            />
          )}
        </CardContent>
      </Card>
    </div>
  );
}
