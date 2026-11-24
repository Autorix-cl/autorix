"use client";

import * as React from "react";
import { Suspense } from "react";
import Link from "next/link";
import { useSearchParams } from "next/navigation";
import {
  Copy,
  Check,
  Terminal,
  Container,
  FileCode,
  ShieldCheck,
  AlertCircle,
} from "lucide-react";
import { toast } from "sonner";
import { PageHeader } from "@/components/layout/page-header";
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs";
import { Wizard, type WizardStep } from "@/components/resources/wizard";

const VALID_ENGINES = new Set(["ego", "nexus", "janus", "aegis", "vulcan", "hermes", "themis"]);
const VALID_ENVS = new Set(["prod", "staging", "dev"]);

function readCsrfCookie(): string | undefined {
  if (typeof document === "undefined") return undefined;
  const match = document.cookie.match(/(?:^|;\s*)autorix_csrf=([^;]+)/);
  return match ? decodeURIComponent(match[1]) : undefined;
}

function FleetEnrollContent() {
  const searchParams = useSearchParams();
  const initialType = searchParams.get("type")?.toLowerCase();
  const initialEnv = searchParams.get("env")?.toLowerCase();

  const [engineType, setEngineType] = React.useState(
    initialType && VALID_ENGINES.has(initialType) ? initialType : "ego",
  );
  const [environment, setEnvironment] = React.useState(
    initialEnv && VALID_ENVS.has(initialEnv) ? initialEnv : "prod",
  );
  const [usesAllowed, setUsesAllowed] = React.useState("1");
  const [ttlHours, setTtlHours] = React.useState("24");
  const [isMinting, setIsMinting] = React.useState(false);
  const [mintedToken, setMintedToken] = React.useState<string | null>(null);
  const [mintError, setMintError] = React.useState<string | null>(null);
  const [copiedCode, setCopiedCode] = React.useState<string | null>(null);

  const handleMintToken = async () => {
    setMintError(null);
    setIsMinting(true);
    try {
      const csrf = readCsrfCookie();
      const res = await fetch("/api/fleet/enrollment-tokens", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          ...(csrf ? { "X-CSRF-Token": csrf } : {}),
        },
        body: JSON.stringify({
          engine_type: engineType,
          environment_id: environment,
          uses_allowed: Math.max(1, parseInt(usesAllowed, 10) || 1),
          ttl_seconds: Math.max(1, parseInt(ttlHours, 10) || 24) * 3600,
        }),
      });

      const data = await res.json().catch(() => ({}));
      if (!res.ok) {
        throw new Error(data.error || `Failed to mint enrollment token (HTTP ${res.status})`);
      }
      if (!data.token || typeof data.token !== "string") {
        throw new Error("Argus did not return an enrollment token secret.");
      }

      setMintedToken(data.token);
      toast.success(`Enrollment token minted for ${engineType.toUpperCase()} (${environment})`);
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : "Failed to mint enrollment token";
      setMintError(msg);
      toast.error(msg);
    } finally {
      setIsMinting(false);
    }
  };

  const copyToClipboard = (text: string, label: string) => {
    navigator.clipboard.writeText(text);
    setCopiedCode(label);
    setTimeout(() => setCopiedCode(null), 2000);
  };

  const argusUrl = "http://argus:4400";
  const dockerSnippet = `docker run -d \\
  --name autorix-${engineType} \\
  -e AUTORIX_ARGUS_URL="${argusUrl}" \\
  -e AUTORIX_ENROLLMENT_TOKEN="${mintedToken || "<YOUR_ENROLLMENT_TOKEN>"}" \\
  -e AUTORIX_ENVIRONMENT="${environment}" \\
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
            <label htmlFor="enroll-engine-type" className="block text-xs font-semibold text-foreground mb-1.5">
              Target Engine Type
            </label>
            <select
              id="enroll-engine-type"
              value={engineType}
              onChange={(e) => setEngineType(e.target.value)}
              className="w-full h-9 rounded-lg border border-border bg-card px-3 text-xs text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
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
            <label htmlFor="enroll-environment" className="block text-xs font-semibold text-foreground mb-1.5">
              Environment
            </label>
            <select
              id="enroll-environment"
              value={environment}
              onChange={(e) => setEnvironment(e.target.value)}
              className="w-full h-9 rounded-lg border border-border bg-card px-3 text-xs text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
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
            <label htmlFor="enroll-max-uses" className="block text-xs font-semibold text-foreground mb-1.5">
              Max Uses Allowed
            </label>
            <Input
              id="enroll-max-uses"
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
            <label htmlFor="enroll-ttl-hours" className="block text-xs font-semibold text-foreground mb-1.5">
              Expiration Time (Hours)
            </label>
            <Input
              id="enroll-ttl-hours"
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

      {mintError && (
        <div
          role="alert"
          className="p-4 rounded-xl border border-destructive/40 bg-destructive/10 text-destructive text-xs flex items-start gap-2.5"
        >
          <AlertCircle className="h-4 w-4 shrink-0 mt-0.5" aria-hidden="true" />
          <span>{mintError}</span>
        </div>
      )}

      {!mintedToken ? (
        <Wizard
          title="Engine Enrollment Wizard"
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
              <ShieldCheck className="h-5 w-5" aria-hidden="true" />
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
                type="button"
                variant="outline"
                size="sm"
                onClick={() => copyToClipboard(mintedToken, "raw")}
                className="h-8 gap-1.5 text-xs shrink-0"
              >
                {copiedCode === "raw" ? <Check className="h-3.5 w-3.5 text-emerald-400" aria-hidden="true" /> : <Copy className="h-3.5 w-3.5" aria-hidden="true" />}
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
                    <Container className="h-3.5 w-3.5" aria-hidden="true" /> Docker
                  </TabsTrigger>
                  <TabsTrigger value="helm" className="text-xs gap-1.5">
                    <FileCode className="h-3.5 w-3.5" aria-hidden="true" /> Helm
                  </TabsTrigger>
                  <TabsTrigger value="binary" className="text-xs gap-1.5">
                    <Terminal className="h-3.5 w-3.5" aria-hidden="true" /> Binary
                  </TabsTrigger>
                </TabsList>

                <TabsContent value="docker">
                  <div className="relative">
                    <pre className="p-4 rounded-xl bg-muted/70 border border-border text-xs font-mono overflow-x-auto text-emerald-400">
                      {dockerSnippet}
                    </pre>
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon-sm"
                      aria-label="Copy Docker command"
                      onClick={() => copyToClipboard(dockerSnippet, "docker")}
                      className="absolute top-2 right-2 h-7 w-7 text-muted-foreground hover:text-foreground"
                    >
                      {copiedCode === "docker" ? <Check className="h-3.5 w-3.5 text-emerald-400" aria-hidden="true" /> : <Copy className="h-3.5 w-3.5" aria-hidden="true" />}
                    </Button>
                  </div>
                </TabsContent>

                <TabsContent value="helm">
                  <div className="relative">
                    <pre className="p-4 rounded-xl bg-muted/70 border border-border text-xs font-mono overflow-x-auto text-amber-300">
                      {helmSnippet}
                    </pre>
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon-sm"
                      aria-label="Copy Helm values"
                      onClick={() => copyToClipboard(helmSnippet, "helm")}
                      className="absolute top-2 right-2 h-7 w-7 text-muted-foreground hover:text-foreground"
                    >
                      {copiedCode === "helm" ? <Check className="h-3.5 w-3.5 text-emerald-400" aria-hidden="true" /> : <Copy className="h-3.5 w-3.5" aria-hidden="true" />}
                    </Button>
                  </div>
                </TabsContent>

                <TabsContent value="binary">
                  <div className="relative">
                    <pre className="p-4 rounded-xl bg-muted/70 border border-border text-xs font-mono overflow-x-auto text-blue-300">
                      {shellSnippet}
                    </pre>
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon-sm"
                      aria-label="Copy shell commands"
                      onClick={() => copyToClipboard(shellSnippet, "binary")}
                      className="absolute top-2 right-2 h-7 w-7 text-muted-foreground hover:text-foreground"
                    >
                      {copiedCode === "binary" ? <Check className="h-3.5 w-3.5 text-emerald-400" aria-hidden="true" /> : <Copy className="h-3.5 w-3.5" aria-hidden="true" />}
                    </Button>
                  </div>
                </TabsContent>
              </Tabs>
            </div>

            <div className="flex items-center justify-between border-t border-border/60 pt-4">
              <Button type="button" variant="outline" size="sm" onClick={() => setMintedToken(null)} className="text-xs">
                Mint Another Token
              </Button>
              <Button asChild size="sm" className="text-xs gap-1.5">
                <Link href="/fleet">Back to Fleet Overview</Link>
              </Button>
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
}

export default function FleetEnrollPage() {
  return (
    <Suspense fallback={<div className="p-8 text-xs text-muted-foreground font-mono">Loading enrollment wizard...</div>}>
      <FleetEnrollContent />
    </Suspense>
  );
}
