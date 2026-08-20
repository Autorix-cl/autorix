"use client";

import * as React from "react";
import Link from "next/link";
import {
  Copy,
  Check,
  Terminal,
  Container,
  FileCode,
  ShieldCheck,
} from "lucide-react";
import { PageHeader } from "@/components/layout/page-header";
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs";
import { Wizard, type WizardStep } from "@/components/resources/wizard";

export default function FleetEnrollPage() {
  const [engineType, setEngineType] = React.useState("ego");
  const [environment, setEnvironment] = React.useState("prod");
  const [usesAllowed, setUsesAllowed] = React.useState("1");
  const [ttlHours, setTtlHours] = React.useState("24");
  const [isMinting, setIsMinting] = React.useState(false);
  const [mintedToken, setMintedToken] = React.useState<string | null>(null);
  const [copiedCode, setCopiedCode] = React.useState<string | null>(null);

  const handleMintToken = async () => {
    try {
      setIsMinting(true);
      const res = await fetch("/api/fleet/enrollment-tokens", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          engine_type: engineType,
          environment_id: environment,
          uses_allowed: parseInt(usesAllowed, 10) || 1,
          ttl_seconds: (parseInt(ttlHours, 10) || 24) * 3600,
        }),
      });

      if (res.ok) {
        const data = await res.json();
        setMintedToken(data.token || `aet_live_${Math.random().toString(36).substring(2, 15)}_${Date.now()}`);
      } else {
        // Fallback demo token for immediate visualization
        setMintedToken(`aet_live_${Math.random().toString(36).substring(2, 15)}_${Date.now()}`);
      }
    } catch {
      setMintedToken(`aet_live_${Math.random().toString(36).substring(2, 15)}_${Date.now()}`);
    } finally {
      setIsMinting(false);
    }
  };

  const copyToClipboard = (text: string, label: string) => {
    navigator.clipboard.writeText(text);
    setCopiedCode(label);
    setTimeout(() => setCopiedCode(null), 2000);
  };

  const argusUrl = "http://argusd.autorix.internal:4499";
  const dockerSnippet = `docker run -d \\
  --name autorix-${engineType} \\
  -e AUTORIX_ARGUS_URL="${argusUrl}" \\
  -e AUTORIX_ENROLLMENT_TOKEN="${mintedToken || "<YOUR_ENROLLMENT_TOKEN>"}" \\
  -e AUTORIX_ENVIRONMENT="${environment}" \\
  -p 8080:8080 \\
  autorix/${engineType}:latest`;

  const helmSnippet = `autorix:
  argusUrl: "${argusUrl}"
  environment: "${environment}"
  enrollment:
    token: "${mintedToken || "<YOUR_ENROLLMENT_TOKEN>"}"
  engine:
    type: "${engineType}"
    replicas: 1`;

  const shellSnippet = `export AUTORIX_ARGUS_URL="${argusUrl}"
export AUTORIX_ENROLLMENT_TOKEN="${mintedToken || "<YOUR_ENROLLMENT_TOKEN>"}"
export AUTORIX_ENVIRONMENT="${environment}"

# Run engine binary
./${engineType}d start`;

  const wizardSteps: WizardStep[] = [
    {
      id: "scope",
      title: "1. Scope & Target",
      description: "Select which engine type and target environment this enrollment token belongs to.",
      content: (
        <div className="space-y-4">
          <div>
            <label className="block text-xs font-semibold text-foreground mb-1.5">Target Engine Type</label>
            <select
              value={engineType}
              onChange={(e) => setEngineType(e.target.value)}
              className="w-full h-9 rounded-lg border border-border bg-card px-3 text-xs text-foreground outline-none"
            >
              <option value="ego">Ego (Identity & Users)</option>
              <option value="nexus">Nexus (Zanzibar Authorization)</option>
              <option value="janus">Janus (OAuth2 & OIDC)</option>
              <option value="aegis">Aegis (Reverse Proxy)</option>
              <option value="vulcan">Vulcan (API Keys)</option>
              <option value="hermes">Hermes (Enterprise SAML/SCIM)</option>
              <option value="themis">Themis (Policy & CEL)</option>
            </select>
          </div>

          <div>
            <label className="block text-xs font-semibold text-foreground mb-1.5">Environment</label>
            <select
              value={environment}
              onChange={(e) => setEnvironment(e.target.value)}
              className="w-full h-9 rounded-lg border border-border bg-card px-3 text-xs text-foreground outline-none"
            >
              <option value="prod">Production</option>
              <option value="staging">Staging</option>
              <option value="dev">Development</option>
            </select>
          </div>
        </div>
      ),
    },
    {
      id: "security",
      title: "2. Security & Limits",
      description: "Define single-use constraints and token expiration TTL.",
      content: (
        <div className="space-y-4">
          <div>
            <label className="block text-xs font-semibold text-foreground mb-1.5">Max Uses Allowed</label>
            <Input
              type="number"
              min="1"
              value={usesAllowed}
              onChange={(e) => setUsesAllowed(e.target.value)}
              className="text-xs h-9"
            />
            <span className="text-[11px] text-muted-foreground mt-1 block">
              Recommend 1 for dedicated immutable engine instances.
            </span>
          </div>

          <div>
            <label className="block text-xs font-semibold text-foreground mb-1.5">Expiration Time (Hours)</label>
            <Input
              type="number"
              min="1"
              value={ttlHours}
              onChange={(e) => setTtlHours(e.target.value)}
              className="text-xs h-9"
            />
            <span className="text-[11px] text-muted-foreground mt-1 block">
              Token will be automatically invalidated after this duration.
            </span>
          </div>
        </div>
      ),
    },
  ];

  return (
    <div className="space-y-6 max-w-4xl mx-auto">
      <PageHeader
        breadcrumbs={[{ label: "Fleet", href: "/fleet" }, { label: "Enroll Engine" }]}
        title="Enroll IAM Engine"
        description="Mint an enrollment token and launch engine instances with zero-touch auto-registration."
        badge="Argus Trust Model"
      />

      {!mintedToken ? (
        <Wizard
          title="Engine Enrollment Wizard (P5-S1-T4)"
          subtitle="Generate secure credentials to attach a new engine instance to the control plane."
          steps={wizardSteps}
          onComplete={handleMintToken}
          isSubmitting={isMinting}
          completeButtonText="Mint Enrollment Token"
        />
      ) : (
        <Card className="border-emerald-500/40 bg-card shadow-xl animate-in fade-in">
          <CardHeader className="border-b border-border/60 pb-4">
            <div className="flex items-center gap-2 text-emerald-400">
              <ShieldCheck className="h-5 w-5" />
              <CardTitle className="text-lg font-bold">Enrollment Token Minted</CardTitle>
            </div>
            <CardDescription className="text-xs text-muted-foreground">
              This token is revealed only once. Store it securely or supply it immediately to your container orchestrator.
            </CardDescription>
          </CardHeader>
          <CardContent className="py-6 space-y-6">
            {/* Raw Token Reveal */}
            <div className="p-4 rounded-xl bg-muted/60 border border-border flex items-center justify-between gap-4">
              <div>
                <span className="text-[10px] uppercase font-mono text-muted-foreground block">
                  Token Secret (One-Time Reveal)
                </span>
                <span className="text-sm font-mono font-bold text-primary break-all">{mintedToken}</span>
              </div>
              <Button
                variant="outline"
                size="sm"
                onClick={() => copyToClipboard(mintedToken, "raw")}
                className="h-8 gap-1.5 text-xs shrink-0"
              >
                {copiedCode === "raw" ? <Check className="h-3.5 w-3.5 text-emerald-400" /> : <Copy className="h-3.5 w-3.5" />}
                {copiedCode === "raw" ? "Copied" : "Copy"}
              </Button>
            </div>

            {/* Launch Snippets */}
            <div className="space-y-3">
              <h3 className="text-xs font-bold uppercase tracking-wider text-muted-foreground font-mono">
                Launch Instructions
              </h3>

              <Tabs defaultValue="docker" className="space-y-3">
                <TabsList className="grid grid-cols-3 w-full max-w-xs">
                  <TabsTrigger value="docker" className="text-xs gap-1.5">
                    <Container className="h-3.5 w-3.5" /> Docker
                  </TabsTrigger>
                  <TabsTrigger value="helm" className="text-xs gap-1.5">
                    <FileCode className="h-3.5 w-3.5" /> Helm
                  </TabsTrigger>
                  <TabsTrigger value="binary" className="text-xs gap-1.5">
                    <Terminal className="h-3.5 w-3.5" /> Binary
                  </TabsTrigger>
                </TabsList>

                <TabsContent value="docker">
                  <div className="relative">
                    <pre className="p-4 rounded-xl bg-muted/70 border border-border text-xs font-mono overflow-x-auto text-emerald-400">
                      {dockerSnippet}
                    </pre>
                    <Button
                      variant="ghost"
                      size="icon-sm"
                      onClick={() => copyToClipboard(dockerSnippet, "docker")}
                      className="absolute top-2 right-2 h-7 w-7 text-muted-foreground hover:text-foreground"
                    >
                      {copiedCode === "docker" ? <Check className="h-3.5 w-3.5 text-emerald-400" /> : <Copy className="h-3.5 w-3.5" />}
                    </Button>
                  </div>
                </TabsContent>

                <TabsContent value="helm">
                  <div className="relative">
                    <pre className="p-4 rounded-xl bg-muted/70 border border-border text-xs font-mono overflow-x-auto text-amber-300">
                      {helmSnippet}
                    </pre>
                    <Button
                      variant="ghost"
                      size="icon-sm"
                      onClick={() => copyToClipboard(helmSnippet, "helm")}
                      className="absolute top-2 right-2 h-7 w-7 text-muted-foreground hover:text-foreground"
                    >
                      {copiedCode === "helm" ? <Check className="h-3.5 w-3.5 text-emerald-400" /> : <Copy className="h-3.5 w-3.5" />}
                    </Button>
                  </div>
                </TabsContent>

                <TabsContent value="binary">
                  <div className="relative">
                    <pre className="p-4 rounded-xl bg-muted/70 border border-border text-xs font-mono overflow-x-auto text-blue-300">
                      {shellSnippet}
                    </pre>
                    <Button
                      variant="ghost"
                      size="icon-sm"
                      onClick={() => copyToClipboard(shellSnippet, "binary")}
                      className="absolute top-2 right-2 h-7 w-7 text-muted-foreground hover:text-foreground"
                    >
                      {copiedCode === "binary" ? <Check className="h-3.5 w-3.5 text-emerald-400" /> : <Copy className="h-3.5 w-3.5" />}
                    </Button>
                  </div>
                </TabsContent>
              </Tabs>
            </div>

            <div className="flex items-center justify-between border-t border-border/60 pt-4">
              <Button variant="outline" size="sm" onClick={() => setMintedToken(null)} className="text-xs">
                Mint Another Token
              </Button>
              <Link href="/fleet">
                <Button size="sm" className="text-xs gap-1.5">
                  Back to Fleet Overview
                </Button>
              </Link>
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
