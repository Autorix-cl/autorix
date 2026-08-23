import * as React from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Loader2, Play, CheckCircle2, XCircle } from "lucide-react";

export interface CheckQuery {
  subject: string;
  relation: string;
  object: string;
}

export interface CheckResult {
  allowed: boolean;
  trace?: string[];
}

export interface CheckSimulatorProps {
  onCheck: (query: CheckQuery) => Promise<CheckResult>;
}

export function CheckSimulator({ onCheck }: CheckSimulatorProps) {
  const [subject, setSubject] = React.useState("");
  const [relation, setRelation] = React.useState("");
  const [object, setObject] = React.useState("");
  
  const [isChecking, setIsChecking] = React.useState(false);
  const [result, setResult] = React.useState<CheckResult | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsChecking(true);
    setResult(null);
    try {
      const res = await onCheck({ subject, relation, object });
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
        <h3 className="font-semibold text-sm">Check Simulator (Playground)</h3>
      </div>
      
      <div className="p-4 flex flex-col gap-6">
        <form onSubmit={handleSubmit} className="flex items-end gap-3">
          <div className="space-y-1 flex-1">
            <label htmlFor="check-subject" className="text-xs font-medium text-muted-foreground">Subject</label>
            <Input id="check-subject" value={subject} onChange={e => setSubject(e.target.value)} placeholder="user:alice" required className="h-8 text-sm" />
          </div>
          <div className="space-y-1 flex-1">
            <label htmlFor="check-relation" className="text-xs font-medium text-muted-foreground">Relation / Permission</label>
            <Input id="check-relation" value={relation} onChange={e => setRelation(e.target.value)} placeholder="edit" required className="h-8 text-sm border-dashed" />
          </div>
          <div className="space-y-1 flex-1">
            <label htmlFor="check-object" className="text-xs font-medium text-muted-foreground">Object</label>
            <Input id="check-object" value={object} onChange={e => setObject(e.target.value)} placeholder="document:1" required className="h-8 text-sm" />
          </div>
          <Button type="submit" disabled={isChecking} size="sm" className="h-8">
            {isChecking ? <Loader2 className="h-4 w-4 mr-2 animate-spin" /> : <Play className="h-4 w-4 mr-2" />}
            Check Access
          </Button>
        </form>

        {/* Results Area */}
        <div className="flex-1 bg-muted/20 border border-dashed rounded-md p-4 min-h-[150px] flex flex-col">
          {!result && !isChecking && (
            <div className="m-auto text-sm text-muted-foreground text-center">
              Enter a tuple query to simulate the graph resolution.
            </div>
          )}
          
          {isChecking && (
            <div className="m-auto flex items-center text-sm text-muted-foreground">
              <Loader2 className="h-4 w-4 mr-2 animate-spin" /> Traversing graph...
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
              </div>

              {result.trace && result.trace.length > 0 && (
                <div className="space-y-2">
                  <h4 className="text-xs font-medium text-muted-foreground uppercase tracking-wider">Resolution Trace</h4>
                  <ul className="space-y-2 font-mono text-xs border-l-2 border-primary/20 pl-4 py-1">
                    {result.trace.map((step, idx) => (
                      <li key={idx} className="text-foreground/80">{step}</li>
                    ))}
                  </ul>
                </div>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
