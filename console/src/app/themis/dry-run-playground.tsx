/* eslint-disable */
"use client";

import { useState } from "react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { CodeEditor } from "@/components/ui/code-editor";
import { Button } from "@/components/ui/button";
import { Play, CheckCircle2, XCircle } from "lucide-react";
import { toast } from "sonner";

export function DryRunPlayground() {
  const [contextJson, setContextJson] = useState('{\n  "request": {\n    "auth": {\n      "claims": {\n        "role": "user"\n      }\n    }\n  }\n}');
  const [isEvaluating, setIsEvaluating] = useState(false);
  const [result, setResult] = useState<{ status: "Passed" | "Failed"; matchedPolicy?: any } | null>(null);

  const handleEvaluate = async () => {
    setIsEvaluating(true);
    setResult(null);
    try {
      let parsedContext;
      try {
        parsedContext = JSON.parse(contextJson);
      } catch (e) {
        throw new Error("Invalid JSON in context");
      }

      const res = await fetch("/api/themis/eval", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ context: parsedContext }),
      });
      
      const data = await res.json();
      if (!res.ok) throw new Error(data.error || "Evaluation failed");
      
      setResult({
        status: data.result,
        matchedPolicy: data.matchedPolicy
      });
    } catch (err) {
      toast.error(err instanceof Error ? err.message : String(err));
    } finally {
      setIsEvaluating(false);
    }
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle>Dry-Run Playground</CardTitle>
        <CardDescription>Test your active ABAC policies against a mock request payload.</CardDescription>
      </CardHeader>
      <CardContent>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div className="flex flex-col space-y-2">
            <div className="text-sm font-medium">Request Context (JSON)</div>
            <div className="border rounded-md overflow-hidden flex-1 min-h-[300px]">
              <CodeEditor 
                value={contextJson}
                onChange={setContextJson}
                language="json"
              />
            </div>
            <Button onClick={handleEvaluate} disabled={isEvaluating} className="w-full">
              <Play className="w-4 h-4 mr-2" />
              {isEvaluating ? "Evaluating..." : "Run Evaluation"}
            </Button>
          </div>
          
          <div className="flex flex-col space-y-2">
            <div className="text-sm font-medium">Evaluation Result</div>
            <div className="border rounded-md flex-1 bg-muted/30 p-6 flex flex-col items-center justify-center min-h-[300px]">
              {!result && <div className="text-muted-foreground text-sm">Hit Run to see results</div>}
              
              {result && result.status === "Passed" && (
                <div className="flex flex-col items-center text-center space-y-4">
                  <CheckCircle2 className="w-16 h-16 text-emerald-500" />
                  <h3 className="text-2xl font-bold text-emerald-500">Passed</h3>
                  <div className="text-sm text-muted-foreground">
                    Matched: <span className="font-mono text-primary">{result.matchedPolicy?.name}</span>
                  </div>
                  <code className="text-xs bg-muted p-2 rounded">
                    {result.matchedPolicy?.expression}
                  </code>
                </div>
              )}
              
              {result && result.status === "Failed" && (
                <div className="flex flex-col items-center text-center space-y-4">
                  <XCircle className="w-16 h-16 text-destructive" />
                  <h3 className="text-2xl font-bold text-destructive">Failed</h3>
                  <div className="text-sm text-muted-foreground">
                    No active policy matched this request context.
                  </div>
                </div>
              )}
            </div>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
