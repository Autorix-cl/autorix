import * as React from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Loader2, Play, CheckCircle2, XCircle, ChevronRight, ShieldAlert, Sparkles } from "lucide-react";
import type { DecisionNode } from "@/lib/api/schemas/nexus";

export interface CheckQuery {
  subject: string;
  relation: string;
  object: string;
  context?: Record<string, unknown>;
}

export interface CheckResult {
  allowed: boolean;
  reason?: string;
  trace?: string[] | DecisionNode;
}

export interface CheckSimulatorProps {
  onCheck: (query: CheckQuery) => Promise<CheckResult>;
}

function DecisionNodeView({ node, depth = 0 }: { node: DecisionNode; depth?: number }) {
  const [expanded, setExpanded] = React.useState(true);
  const hasChildren = Boolean(node.children && node.children.length > 0);

  return (
    <div className={`space-y-1.5 ${depth > 0 ? "ml-4 pl-3 border-l border-border/60" : ""}`}>
      <div className="flex items-center gap-2 text-xs font-mono py-1 px-2 rounded bg-muted/30 hover:bg-muted/50 transition-colors">
        {hasChildren ? (
          <button
            type="button"
            onClick={() => setExpanded(!expanded)}
            className="p-0.5 hover:bg-muted rounded"
            aria-label={expanded ? "Collapse node" : "Expand node"}
          >
            <ChevronRight className={`w-3.5 h-3.5 transition-transform ${expanded ? "rotate-90" : ""}`} />
          </button>
        ) : (
          <span className="w-3.5 h-3.5 inline-block" />
        )}

        {node.allowed ? (
          <CheckCircle2 className="w-3.5 h-3.5 text-emerald-500 flex-shrink-0" />
        ) : (
          <XCircle className="w-3.5 h-3.5 text-rose-500 flex-shrink-0" />
        )}

        <span className="font-semibold text-foreground">
          {node.namespace}:{node.object}#{node.relation}
        </span>

        {node.rewrite_type && (
          <Badge variant="outline" className="text-[10px] uppercase font-sans">
            {node.rewrite_type}
          </Badge>
        )}

        {node.caveat && (
          <Badge
            variant={node.caveat.allowed ? "secondary" : "destructive"}
            className="text-[10px] font-sans flex items-center gap-1"
          >
            <ShieldAlert className="w-3 h-3" />
            {node.caveat.caveat_name} ({node.caveat.allowed ? "Passed" : "Failed"})
          </Badge>
        )}

        {node.reason && (
          <span className="text-[11px] text-muted-foreground ml-auto truncate max-w-[200px]">
            {node.reason}
          </span>
        )}
      </div>

      {hasChildren && expanded && (
        <div className="space-y-1">
          {node.children?.map((child, idx) => (
            <DecisionNodeView key={child.node_id || idx} node={child} depth={depth + 1} />
          ))}
        </div>
      )}
    </div>
  );
}

export function CheckSimulator({ onCheck }: CheckSimulatorProps) {
  const [subject, setSubject] = React.useState("");
  const [relation, setRelation] = React.useState("");
  const [object, setObject] = React.useState("");
  const [contextJson, setContextJson] = React.useState("");
  const [showContext, setShowContext] = React.useState(false);
  
  const [isChecking, setIsChecking] = React.useState(false);
  const [result, setResult] = React.useState<CheckResult | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsChecking(true);
    setResult(null);
    try {
      let parsedContext: Record<string, unknown> | undefined;
      if (contextJson.trim()) {
        parsedContext = JSON.parse(contextJson);
      }
      const res = await onCheck({ subject, relation, object, context: parsedContext });
      setResult(res);
    } catch (err) {
      console.error(err);
    } finally {
      setIsChecking(false);
    }
  };

  return (
    <div className="flex flex-col h-full bg-card rounded-md border text-card-foreground shadow-sm">
      <div className="p-4 border-b flex items-center justify-between">
        <h3 className="font-semibold text-sm flex items-center gap-2">
          <Sparkles className="w-4 h-4 text-primary" />
          Check Simulator (Playground)
        </h3>
      </div>

      
      <div className="p-4 flex flex-col gap-4">
        <form onSubmit={handleSubmit} className="space-y-3">
          <div className="flex items-end gap-3">
            <div className="space-y-1 flex-1">
              <label htmlFor="check-subject" className="text-xs font-medium text-muted-foreground">Subject</label>
              <Input id="check-subject" value={subject} onChange={e => setSubject(e.target.value)} placeholder="user:alice" required className="h-8 text-sm font-mono" />
            </div>
            <div className="space-y-1 flex-1">
              <label htmlFor="check-relation" className="text-xs font-medium text-muted-foreground">Relation / Permission</label>
              <Input id="check-relation" value={relation} onChange={e => setRelation(e.target.value)} placeholder="edit" required className="h-8 text-sm font-mono border-dashed" />
            </div>
            <div className="space-y-1 flex-1">
              <label htmlFor="check-object" className="text-xs font-medium text-muted-foreground">Object</label>
              <Input id="check-object" value={object} onChange={e => setObject(e.target.value)} placeholder="document:1" required className="h-8 text-sm font-mono" />
            </div>
            <Button type="submit" disabled={isChecking} size="sm" className="h-8">
              {isChecking ? <Loader2 className="h-4 w-4 mr-2 animate-spin" /> : <Play className="h-4 w-4 mr-2" />}
              Check Access
            </Button>
          </div>

          <div>
            <button
              type="button"
              onClick={() => setShowContext(!showContext)}
              className="text-xs text-muted-foreground hover:text-foreground transition-colors underline underline-offset-2"
            >
              {showContext ? "- Hide ABAC Context" : "+ Add Request Context (JSON)"}
            </button>
            {showContext && (
              <div className="mt-2">
                <textarea
                  value={contextJson}
                  onChange={(e) => setContextJson(e.target.value)}
                  placeholder='{"current_hour": 14, "ip_address": "10.0.0.1"}'
                  rows={2}
                  className="w-full text-xs font-mono p-2 border rounded-md bg-muted/20"
                />
              </div>
            )}
          </div>
        </form>

        {/* Results Area */}
        <div className="flex-1 bg-muted/20 border border-dashed rounded-md p-4 min-h-[150px] flex flex-col">
          {!result && !isChecking && (
            <div className="m-auto text-sm text-muted-foreground text-center">
              Enter a tuple query to simulate the graph resolution and decision trace.
            </div>
          )}
          
          {isChecking && (
            <div className="m-auto flex items-center text-sm text-muted-foreground">
              <Loader2 className="h-4 w-4 mr-2 animate-spin" /> Traversing graph and evaluating rewrites...
            </div>
          )}

          {result && !isChecking && (
            <div className="space-y-4 animate-in fade-in slide-in-from-bottom-2 duration-300">
              <div className="flex items-center gap-2">
                <span className="text-sm font-medium">Result:</span>
                {result.allowed ? (
                  <span className="inline-flex items-center text-sm font-bold text-green-600 dark:text-green-500 bg-green-100 dark:bg-green-900/30 px-2 py-1 rounded">
                    <CheckCircle2 className="w-4 h-4 mr-1" /> ALLOWED
                  </span>
                ) : (
                  <span className="inline-flex items-center text-sm font-bold text-red-600 dark:text-red-500 bg-red-100 dark:bg-red-900/30 px-2 py-1 rounded">
                    <XCircle className="w-4 h-4 mr-1" /> DENIED
                  </span>
                )}
                {result.reason && (
                  <span className="text-xs text-muted-foreground ml-2">({result.reason})</span>
                )}
              </div>

              {result.trace && (
                <div className="space-y-2">
                  <h4 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
                    Walkable Decision Trace
                  </h4>

                  {Array.isArray(result.trace) ? (
                    <ul className="space-y-2 font-mono text-xs border-l-2 border-primary/20 pl-4 py-1">
                      {result.trace.map((step, idx) => (
                        <li key={idx} className="text-foreground/80">{step}</li>
                      ))}
                    </ul>
                  ) : (
                    <div className="border rounded-md p-3 bg-background/60">
                      <DecisionNodeView node={result.trace} />
                    </div>
                  )}
                </div>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

