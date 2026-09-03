"use client";

import * as React from "react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  Play,
  CheckCircle2,
  XCircle,
  Code2,
  Sparkles,
  AlertCircle,
  FileCode2,
  Layers,
  Loader2,
} from "lucide-react";
import { toast } from "sonner";
import type { ValidationResult } from "@/lib/api/schemas/themis";

interface EvaluationTraceItem {
  policy_id?: string;
  policy_name?: string;
  passed?: boolean;
  error?: string;
  expression?: string;
}

export function DryRunPlayground() {
  const [mode, setMode] = React.useState<"tenant" | "scratchpad">("tenant");
  const [expression, setExpression] = React.useState("request.auth.role == 'admin' && request.auth.mfa == true");
  const [contextJson, setContextJson] = React.useState(
    JSON.stringify(
      {
        request: {
          auth: {
            role: "admin",
            mfa: true,
          },
        },
      },
      null,
      2
    )
  );

  const [validation, setValidation] = React.useState<ValidationResult | null>(null);
  const [isValidating, setIsValidating] = React.useState(false);
  const [isEvaluating, setIsEvaluating] = React.useState(false);
  const [result, setResult] = React.useState<{
    status: "Passed" | "Failed";
    matchedPolicy?: { name: string; expression?: string } | null;
    trace?: EvaluationTraceItem[];
  } | null>(null);

  // Debounced CEL syntax validation when in scratchpad mode
  React.useEffect(() => {
    if (mode !== "scratchpad" || !expression.trim()) {
      setValidation(null);
      return;
    }

    const timer = setTimeout(async () => {
      setIsValidating(true);
      try {
        const res = await fetch("/api/themis/validate", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ expression }),
        });
        if (res.ok) {
          const data: ValidationResult = await res.json();
          setValidation(data);
        }
      } catch {
        // silent fail on auto-validation
      } finally {
        setIsValidating(false);
      }
    }, 400);

    return () => clearTimeout(timer);
  }, [expression, mode]);

  const handleEvaluate = async () => {
    setIsEvaluating(true);
    setResult(null);
    try {
      let parsedContext;
      try {
        parsedContext = JSON.parse(contextJson);
      } catch {
        throw new Error("Invalid JSON in context payload");
      }

      const body: Record<string, unknown> = { context: parsedContext };
      if (mode === "scratchpad") {
        body.expression = expression;
      }

      const res = await fetch("/api/themis/eval", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      });

      const data = await res.json();
      if (!res.ok) throw new Error(data.error || "Evaluation failed");

      setResult({
        status: data.result,
        matchedPolicy: data.matchedPolicy,
        trace: data.results || [],
      });
    } catch (err) {
      toast.error(err instanceof Error ? err.message : String(err));
    } finally {
      setIsEvaluating(false);
    }
  };

  const applyPreset = (presetName: string) => {
    if (presetName === "admin_mfa") {
      setContextJson(
        JSON.stringify(
          {
            request: {
              auth: {
                role: "admin",
                mfa: true,
              },
              action: "delete",
              resource: "database_cluster",
            },
          },
          null,
          2
        )
      );
    } else if (presetName === "standard_user") {
      setContextJson(
        JSON.stringify(
          {
            request: {
              auth: {
                role: "user",
                mfa: false,
              },
              action: "read",
              resource: "dashboard",
            },
          },
          null,
          2
        )
      );
    }
  };

  return (
    <Card className="bg-card/80">
      <CardHeader>
        <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <CardTitle className="text-sm font-semibold flex items-center gap-2">
              <Sparkles className="w-4 h-4 text-primary" />
              ABAC Simulator & Evaluation Tracer
            </CardTitle>
            <CardDescription className="text-xs">
              Test request contexts against active policies or prototype ad-hoc CEL expressions with rule traces.
            </CardDescription>
          </div>

          <div className="flex items-center gap-1.5 p-1 bg-muted/40 rounded-lg border text-xs">
            <button
              type="button"
              onClick={() => setMode("tenant")}
              className={`px-2.5 py-1 rounded text-xs font-medium transition-colors ${
                mode === "tenant"
                  ? "bg-background text-foreground shadow-sm"
                  : "text-muted-foreground hover:text-foreground"
              }`}
            >
              <span className="flex items-center gap-1.5">
                <Layers className="w-3.5 h-3.5" />
                Active Policies
              </span>
            </button>
            <button
              type="button"
              onClick={() => setMode("scratchpad")}
              className={`px-2.5 py-1 rounded text-xs font-medium transition-colors ${
                mode === "scratchpad"
                  ? "bg-background text-foreground shadow-sm"
                  : "text-muted-foreground hover:text-foreground"
              }`}
            >
              <span className="flex items-center gap-1.5">
                <FileCode2 className="w-3.5 h-3.5" />
                Expression Scratchpad
              </span>
            </button>
          </div>
        </div>
      </CardHeader>

      <CardContent className="space-y-4">
        {mode === "scratchpad" && (
          <div className="space-y-1.5 p-3 rounded-lg border bg-muted/20 text-xs">
            <div className="flex items-center justify-between">
              <span className="font-semibold text-foreground flex items-center gap-1.5">
                <Code2 className="w-3.5 h-3.5 text-primary" />
                CEL Expression Scratchpad
              </span>

              {isValidating ? (
                <Badge variant="outline" className="text-[10px] gap-1">
                  <Loader2 className="w-3 h-3 animate-spin" /> Validating syntax...
                </Badge>
              ) : validation ? (
                validation.valid ? (
                  <Badge variant="outline" className="text-[10px] text-emerald-500 border-emerald-500/30 gap-1">
                    <CheckCircle2 className="w-3 h-3" /> Valid CEL
                  </Badge>
                ) : (
                  <Badge variant="destructive" className="text-[10px] gap-1">
                    <AlertCircle className="w-3 h-3" /> Syntax Error
                  </Badge>
                )
              ) : null}
            </div>

            <input
              type="text"
              value={expression}
              onChange={(e) => setExpression(e.target.value)}
              placeholder="e.g. request.auth.role == 'admin'"
              className="w-full font-mono text-xs p-2 rounded border bg-background text-foreground"
            />

            {validation && !validation.valid && validation.errors.length > 0 && (
              <p className="text-[11px] text-destructive pt-1 font-mono">
                {validation.errors[0].message}
              </p>
            )}
          </div>
        )}

        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {/* Left Column: Request Context */}
          <div className="flex flex-col space-y-2">
            <div className="flex items-center justify-between text-xs font-medium">
              <span>Request Context (JSON)</span>
              <div className="flex items-center gap-2 text-[11px] text-muted-foreground">
                <span>Presets:</span>
                <button
                  type="button"
                  onClick={() => applyPreset("admin_mfa")}
                  className="underline hover:text-foreground"
                >
                  Admin MFA
                </button>
                <button
                  type="button"
                  onClick={() => applyPreset("standard_user")}
                  className="underline hover:text-foreground"
                >
                  User
                </button>
              </div>
            </div>

            <textarea
              value={contextJson}
              onChange={(e) => setContextJson(e.target.value)}
              rows={10}
              className="w-full text-xs font-mono p-3 rounded-md border bg-muted/20 text-foreground resize-none"
            />

            <Button onClick={handleEvaluate} disabled={isEvaluating} className="w-full text-xs gap-1.5">
              {isEvaluating ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : <Play className="w-3.5 h-3.5" />}
              {isEvaluating ? "Evaluating..." : "Run Evaluation"}
            </Button>
          </div>

          {/* Right Column: Output & Evaluation Tracer */}
          <div className="flex flex-col space-y-2">
            <div className="text-xs font-medium">Decision & Evaluation Trace</div>

            <div className="border rounded-md flex-1 bg-muted/20 p-4 flex flex-col justify-center min-h-[260px] text-xs">
              {!result && (
                <div className="text-muted-foreground text-center py-10">
                  Hit Run Evaluation to see decision outcome and rule evaluation trace.
                </div>
              )}

              {result && (
                <div className="space-y-4">
                  <div className="flex items-center justify-between p-3 rounded-lg border bg-background">
                    <div className="flex items-center gap-3">
                      {result.status === "Passed" ? (
                        <CheckCircle2 className="w-7 h-7 text-emerald-500 shrink-0" />
                      ) : (
                        <XCircle className="w-7 h-7 text-destructive shrink-0" />
                      )}
                      <div>
                        <h4
                          className={`text-base font-bold ${
                            result.status === "Passed" ? "text-emerald-500" : "text-destructive"
                          }`}
                        >
                          {result.status}
                        </h4>
                        <p className="text-[11px] text-muted-foreground">
                          {result.status === "Passed"
                            ? "Request meets active authorization requirements."
                            : "No matching policy permitted this request context."}
                        </p>
                      </div>
                    </div>

                    {result.matchedPolicy && (
                      <Badge variant="outline" className="font-mono text-[10px]">
                        <span>Matched: </span>
                        <span>{result.matchedPolicy.name}</span>
                      </Badge>
                    )}
                  </div>

                  {/* Evaluation Trace Breakdown */}
                  {result.trace && result.trace.length > 0 && (
                    <div className="space-y-1.5">
                      <span className="font-semibold text-[11px] text-muted-foreground uppercase tracking-wider">
                        Rule Evaluation Trace ({result.trace.length})
                      </span>
                      <div className="max-h-40 overflow-y-auto space-y-1 pr-1">
                        {result.trace.map((item, idx) => (
                          <div
                            key={idx}
                            className="p-2 rounded border bg-card/60 flex items-center justify-between text-[11px]"
                          >
                            <div className="space-y-0.5 truncate max-w-[280px]">
                              <div className="font-semibold text-foreground">{item.policy_name}</div>
                              <code className="text-[10px] text-muted-foreground font-mono truncate block">
                                {item.expression}
                              </code>
                            </div>

                            <Badge
                              variant={item.passed ? "outline" : "secondary"}
                              className={`text-[10px] ${
                                item.passed ? "text-emerald-500 border-emerald-500/30" : "text-muted-foreground"
                              }`}
                            >
                              {item.passed ? "Matched" : "Denied"}
                            </Badge>
                          </div>
                        ))}
                      </div>
                    </div>
                  )}
                </div>
              )}
            </div>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
