"use client";

import * as React from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Plus, Check, ArrowRight, ArrowLeft } from "lucide-react";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "@/components/ui/sheet";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { CodeBlock } from "@/components/ui/code-block";
import { toast } from "sonner";
import { useApiQuery } from "@/lib/query/use-api-query";
import { fetchAndParse } from "@/lib/api/schema";
import { getCsrfHeaders } from "@/lib/api/client";
import { handlerCatalogueSchema, type Rule } from "@/lib/api/schemas/aegis";

const HTTP_METHODS = ["GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "HEAD"];

interface RuleBuilderSheetProps {
  onSuccess?: () => void;
}

export function RuleBuilderSheet({ onSuccess }: RuleBuilderSheetProps) {
  const [open, setOpen] = React.useState(false);
  const [currentStep, setCurrentStep] = React.useState(1);

  // Form state
  const [ruleId, setRuleId] = React.useState("");
  const [description, setDescription] = React.useState("");
  const [urlPattern, setUrlPattern] = React.useState("/api/v1/<.*>");
  const [selectedMethods, setSelectedMethods] = React.useState<string[]>(["GET", "POST"]);

  const [authHandler, setAuthHandler] = React.useState("jwt");
  const [authConfig, setAuthConfig] = React.useState('{\n  "jwks_url": "http://janus:4444/.well-known/jwks.json"\n}');

  const [authorizerHandler, setAuthorizerHandler] = React.useState("allow");
  const [authorizerConfig, setAuthorizerConfig] = React.useState("{}");

  const [mutatorHandler, setMutatorHandler] = React.useState("header");
  const [mutatorConfig, setMutatorConfig] = React.useState('{\n  "headers": {\n    "X-Forwarded-User": "{subject}"\n  }\n}');

  const [upstreamUrl, setUpstreamUrl] = React.useState("http://backend-service:8080");
  const [stripPrefix, setStripPrefix] = React.useState("/api/v1");
  const [rewritePath, setRewritePath] = React.useState("");

  const queryClient = useQueryClient();

  const { data: catalogue } = useApiQuery(["proxy-rules-handlers"], () =>
    fetchAndParse("/api/proxy-rules/handlers", handlerCatalogueSchema)
  );

  const toggleMethod = (method: string) => {
    setSelectedMethods((prev) =>
      prev.includes(method) ? prev.filter((m) => m !== method) : [...prev, method]
    );
  };

  const authJsonState = React.useMemo(() => {
    try {
      const value = JSON.parse(authConfig);
      if (value === null || typeof value !== "object" || Array.isArray(value)) {
        return { value: {}, error: "Authenticator config must be a JSON object." };
      }
      return { value: value as Record<string, unknown>, error: null as string | null };
    } catch {
      return { value: {}, error: "Invalid JSON syntax in authenticator configuration." };
    }
  }, [authConfig]);

  const authorizerJsonState = React.useMemo(() => {
    try {
      const value = JSON.parse(authorizerConfig);
      if (value === null || typeof value !== "object" || Array.isArray(value)) {
        return { value: {}, error: "Authorizer config must be a JSON object." };
      }
      return { value: value as Record<string, unknown>, error: null as string | null };
    } catch {
      return { value: {}, error: "Invalid JSON syntax in authorizer configuration." };
    }
  }, [authorizerConfig]);

  const mutatorJsonState = React.useMemo(() => {
    try {
      const value = JSON.parse(mutatorConfig);
      if (value === null || typeof value !== "object" || Array.isArray(value)) {
        return { value: {}, error: "Mutator config must be a JSON object." };
      }
      return { value: value as Record<string, unknown>, error: null as string | null };
    } catch {
      return { value: {}, error: "Invalid JSON syntax in mutator configuration." };
    }
  }, [mutatorConfig]);

  const previewRule: Rule = React.useMemo(() => ({
    id: ruleId.trim() || "rule-sample",
    description: description.trim(),
    match: {
      url: urlPattern.trim() || "/<.*>",
      methods: selectedMethods.length > 0 ? selectedMethods : ["GET"],
    },
    authenticators: authHandler !== "none" ? [{ handler: authHandler, config: authJsonState.value }] : [],
    authorizer: {
      handler: authorizerHandler,
      config: authorizerJsonState.value,
    },
    mutators: mutatorHandler !== "none" ? [{ handler: mutatorHandler, config: mutatorJsonState.value }] : [],
    upstream: {
      url: upstreamUrl.trim() || "http://localhost:8080",
      ...(stripPrefix.trim() ? { strip_prefix: stripPrefix.trim() } : {}),
      ...(rewritePath.trim() ? { rewrite: rewritePath.trim() } : {}),
    },
  }), [
    ruleId,
    description,
    urlPattern,
    selectedMethods,
    authHandler,
    authJsonState.value,
    authorizerHandler,
    authorizerJsonState.value,
    mutatorHandler,
    mutatorJsonState.value,
    upstreamUrl,
    stripPrefix,
    rewritePath,
  ]);

  const createMutation = useMutation({
    mutationFn: async (rule: Rule) => {
      const res = await fetch("/api/proxy-rules", {
        method: "POST",
        headers: { "Content-Type": "application/json", ...getCsrfHeaders() },
        body: JSON.stringify(rule),
      });
      const data = await res.json();
      if (!res.ok) {
        throw new Error(data.error || "Failed to create proxy rule");
      }
      return data;
    },
    onSuccess: () => {
      toast.success(`Rule '${ruleId}' created successfully`);
      queryClient.invalidateQueries({ queryKey: ["proxy-rules"] });
      setOpen(false);
      resetForm();
      if (onSuccess) onSuccess();
    },
    onError: (err) => {
      toast.error(err instanceof Error ? err.message : "Error creating rule");
    },
  });

  const resetForm = () => {
    setCurrentStep(1);
    setRuleId("");
    setDescription("");
    setUrlPattern("/api/v1/<.*>");
    setSelectedMethods(["GET", "POST"]);
    setUpstreamUrl("http://backend-service:8080");
    setStripPrefix("/api/v1");
    setRewritePath("");
  };

  const handleFinish = (e: React.FormEvent) => {
    e.preventDefault();
    if (!ruleId.trim()) {
      toast.error("Rule ID is required");
      setCurrentStep(1);
      return;
    }
    if (selectedMethods.length === 0) {
      toast.error("Select at least one HTTP method");
      setCurrentStep(1);
      return;
    }
    if (authHandler !== "none" && authJsonState.error) {
      toast.error(authJsonState.error);
      setCurrentStep(2);
      return;
    }
    if (authorizerJsonState.error) {
      toast.error(authorizerJsonState.error);
      setCurrentStep(3);
      return;
    }
    if (mutatorHandler !== "none" && mutatorJsonState.error) {
      toast.error(mutatorJsonState.error);
      setCurrentStep(4);
      return;
    }
    createMutation.mutate(previewRule);
  };

  return (
    <Sheet open={open} onOpenChange={setOpen}>
      <SheetTrigger asChild>
        <Button>
          <Plus className="w-4 h-4 mr-2" />
          New Proxy Rule
        </Button>
      </SheetTrigger>
      <SheetContent className="sm:max-w-[700px] w-[95vw] overflow-y-auto">
        <SheetHeader>
          <SheetTitle>Pipeline Rule Wizard</SheetTitle>
          <SheetDescription>
            Configure an Oathkeeper-style 5-stage declarative Zero-Trust enforcement pipeline.
          </SheetDescription>
        </SheetHeader>

        {/* Stepper Progress */}
        <div className="flex items-center justify-between border-b pb-4 mt-4">
          {[
            { step: 1, title: "1. Match" },
            { step: 2, title: "2. Authenticate" },
            { step: 3, title: "3. Authorize" },
            { step: 4, title: "4. Mutate" },
            { step: 5, title: "5. Upstream" },
          ].map((s) => (
            <button
              key={s.step}
              type="button"
              onClick={() => setCurrentStep(s.step)}
              className={`text-xs font-semibold flex items-center gap-1 transition-colors ${
                currentStep === s.step
                  ? "text-primary"
                  : currentStep > s.step
                  ? "text-muted-foreground"
                  : "text-muted-foreground/50"
              }`}
            >
              {currentStep > s.step && <Check className="w-3 h-3 text-emerald-500" />}
              {s.title}
            </button>
          ))}
        </div>

        <form onSubmit={handleFinish} className="space-y-6 mt-6">
          {/* Step 1: Match */}
          {currentStep === 1 && (
            <div className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="rule-id">Rule ID *</Label>
                <Input
                  id="rule-id"
                  value={ruleId}
                  onChange={(e) => setRuleId(e.target.value)}
                  placeholder="e.g. protect-documents-api"
                  required
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="rule-desc">Description</Label>
                <Input
                  id="rule-desc"
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  placeholder="e.g. Enforce JWT authentication and Themis ABAC on documents"
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="url-pattern">URL Pattern *</Label>
                <Input
                  id="url-pattern"
                  value={urlPattern}
                  onChange={(e) => setUrlPattern(e.target.value)}
                  placeholder="e.g. /api/v1/documents/<.*>"
                  required
                />
                <p className="text-xs text-muted-foreground">Supports Oathkeeper globs like <code>/path/&lt;.*&gt;</code> or <code>/path/*</code>.</p>
              </div>

              <div className="space-y-2">
                <span className="text-sm font-medium leading-none">HTTP Methods *</span>
                <div className="flex flex-wrap gap-2" role="group" aria-label="HTTP Methods">
                  {HTTP_METHODS.map((method) => {
                    const active = selectedMethods.includes(method);
                    return (
                      <button
                        key={method}
                        type="button"
                        aria-pressed={active}
                        onClick={() => toggleMethod(method)}
                        className={`inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-semibold transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring ${
                          active
                            ? "border-transparent bg-primary text-primary-foreground hover:bg-primary/90"
                            : "border-border bg-transparent text-foreground hover:bg-muted/50"
                        }`}
                      >
                        {method}
                      </button>
                    );
                  })}
                </div>
              </div>
            </div>
          )}

          {/* Step 2: Authenticate */}
          {currentStep === 2 && (
            <div className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="auth-handler">Authenticator Handler</Label>
                <Select value={authHandler} onValueChange={setAuthHandler}>
                  <SelectTrigger id="auth-handler">
                    <SelectValue placeholder="Select authenticator" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="jwt">jwt (JSON Web Token RS256/HS256)</SelectItem>
                    <SelectItem value="bearer_token">bearer_token (Opaque Bearer Token)</SelectItem>
                    <SelectItem value="anonymous">anonymous (Public Access)</SelectItem>
                    <SelectItem value="cookie_session">cookie_session (Ego Session Cookie)</SelectItem>
                    <SelectItem value="none">none (Skip Authentication)</SelectItem>
                  </SelectContent>
                </Select>
                {catalogue?.authenticators && catalogue.authenticators.length > 0 && (
                  <p className="text-xs text-muted-foreground">
                    Available in engine: {catalogue.authenticators.map((a) => a.name).join(", ")}
                  </p>
                )}
              </div>

              {authHandler !== "none" && (
                <div className="space-y-2">
                  <Label htmlFor="auth-config-json">Configuration JSON</Label>
                  <textarea
                    id="auth-config-json"
                    rows={4}
                    value={authConfig}
                    onChange={(e) => setAuthConfig(e.target.value)}
                    aria-invalid={Boolean(authJsonState.error)}
                    className="w-full font-mono text-xs p-2 border rounded-md bg-muted/40"
                  />
                  {authJsonState.error && (
                    <p role="alert" className="text-xs text-destructive">
                      {authJsonState.error}
                    </p>
                  )}
                </div>
              )}
            </div>
          )}

          {/* Step 3: Authorize */}
          {currentStep === 3 && (
            <div className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="authorizer-handler">Authorizer Handler</Label>
                <Select value={authorizerHandler} onValueChange={setAuthorizerHandler}>
                  <SelectTrigger id="authorizer-handler">
                    <SelectValue placeholder="Select authorizer" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="allow">allow (Permit all authenticated)</SelectItem>
                    <SelectItem value="deny">deny (Explicit deny)</SelectItem>
                    <SelectItem value="themis">themis (CEL-based ABAC policy engine)</SelectItem>
                    <SelectItem value="nexus">nexus (Zanzibar ReBAC Check)</SelectItem>
                    <SelectItem value="remote_json">remote_json (Webhook authorization)</SelectItem>
                  </SelectContent>
                </Select>
              </div>

              <div className="space-y-2">
                <Label htmlFor="authorizer-config-json">Configuration JSON</Label>
                <textarea
                  id="authorizer-config-json"
                  rows={4}
                  value={authorizerConfig}
                  onChange={(e) => setAuthorizerConfig(e.target.value)}
                  aria-invalid={Boolean(authorizerJsonState.error)}
                  className="w-full font-mono text-xs p-2 border rounded-md bg-muted/40"
                />
                {authorizerJsonState.error && (
                  <p role="alert" className="text-xs text-destructive">
                    {authorizerJsonState.error}
                  </p>
                )}
              </div>
            </div>
          )}

          {/* Step 4: Mutate */}
          {currentStep === 4 && (
            <div className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="mutator-handler">Mutator Handler</Label>
                <Select value={mutatorHandler} onValueChange={setMutatorHandler}>
                  <SelectTrigger id="mutator-handler">
                    <SelectValue placeholder="Select mutator" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="header">header (Inject headers into upstream request)</SelectItem>
                    <SelectItem value="id_token">id_token (Mint signed identity JWT)</SelectItem>
                    <SelectItem value="noop">noop (No mutation)</SelectItem>
                    <SelectItem value="none">none (Skip mutation stage)</SelectItem>
                  </SelectContent>
                </Select>
              </div>

              {mutatorHandler !== "none" && (
                <div className="space-y-2">
                  <Label htmlFor="mutator-config-json">Configuration JSON</Label>
                  <textarea
                    id="mutator-config-json"
                    rows={4}
                    value={mutatorConfig}
                    onChange={(e) => setMutatorConfig(e.target.value)}
                    aria-invalid={Boolean(mutatorJsonState.error)}
                    className="w-full font-mono text-xs p-2 border rounded-md bg-muted/40"
                  />
                  {mutatorJsonState.error && (
                    <p role="alert" className="text-xs text-destructive">
                      {mutatorJsonState.error}
                    </p>
                  )}
                </div>
              )}
            </div>
          )}

          {/* Step 5: Upstream */}
          {currentStep === 5 && (
            <div className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="upstream-url">Upstream Backend URL *</Label>
                <Input
                  id="upstream-url"
                  value={upstreamUrl}
                  onChange={(e) => setUpstreamUrl(e.target.value)}
                  placeholder="e.g. http://documents-service:8080"
                  required
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="strip-prefix">Strip Prefix (Optional)</Label>
                <Input
                  id="strip-prefix"
                  value={stripPrefix}
                  onChange={(e) => setStripPrefix(e.target.value)}
                  placeholder="e.g. /api/v1"
                />
                <p className="text-xs text-muted-foreground">Strips matching prefix before forwarding request to backend.</p>
              </div>

              <div className="space-y-2">
                <Label htmlFor="rewrite-path">Regex Path Rewrite (Optional)</Label>
                <Input
                  id="rewrite-path"
                  value={rewritePath}
                  onChange={(e) => setRewritePath(e.target.value)}
                  placeholder="e.g. /v2/documents/$1"
                />
                <p className="text-xs text-muted-foreground">Rewrites inbound capture groups to target upstream format.</p>
              </div>
            </div>
          )}

          {/* Live Synthesized Rule Preview */}
          <div className="space-y-2 border-t pt-4">
            <div className="flex items-center justify-between">
              <Label className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
                Live Rule Payload Preview
              </Label>
              <Badge variant="outline" className="text-[10px]">JSON</Badge>
            </div>
            <CodeBlock code={JSON.stringify(previewRule, null, 2)} language="json" />
          </div>

          {/* Navigation Controls */}
          <div className="flex items-center justify-between pt-4 border-t">
            {currentStep > 1 ? (
              <Button type="button" variant="outline" onClick={() => setCurrentStep((s) => s - 1)}>
                <ArrowLeft className="w-4 h-4 mr-2" />
                Previous
              </Button>
            ) : <div />}

            {currentStep < 5 ? (
              <Button type="button" onClick={() => setCurrentStep((s) => s + 1)}>
                Next
                <ArrowRight className="w-4 h-4 ml-2" />
              </Button>
            ) : (
              <Button type="submit" disabled={createMutation.isPending} className="bg-emerald-600 hover:bg-emerald-700 text-white">
                {createMutation.isPending ? "Creating..." : "Save Rule"}
              </Button>
            )}
          </div>
        </form>
      </SheetContent>
    </Sheet>
  );
}
