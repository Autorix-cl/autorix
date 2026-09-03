"use client";

import * as React from "react";
import { ShieldAlert, Plus, Trash2, Code2, AlertCircle, CheckCircle2, Loader2 } from "lucide-react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog";
import { fetchAndParse } from "@/lib/api/schema";
import { caveatListSchema, caveatSchema, type CaveatDefinition } from "@/lib/api/schemas/nexus";


export function CaveatRegistry() {
  const queryClient = useQueryClient();
  const [isAddOpen, setIsAddOpen] = React.useState(false);
  const [name, setName] = React.useState("");
  const [celExpression, setCelExpression] = React.useState("");
  const [formError, setFormError] = React.useState<string | null>(null);

  const { data: caveats = [], isLoading, isError, refetch } = useQuery({
    queryKey: ["nexus-caveats"],
    queryFn: async () => {
      const res = await fetchAndParse<CaveatDefinition[]>("/api/nexus/caveats", caveatListSchema);
      if (!res.ok) {
        throw new Error(res.error.message || "Failed to load caveats");
      }
      return res.data;
    },
  });

  const createMutation = useMutation({
    mutationFn: async (payload: { name: string; cel_expression: string }) => {
      const res = await fetchAndParse<CaveatDefinition>("/api/nexus/caveats", caveatSchema, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      });
      if (!res.ok) {
        throw new Error(res.error.message || "Failed to create caveat");
      }
      return res.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["nexus-caveats"] });
      setIsAddOpen(false);
      setName("");
      setCelExpression("");
      setFormError(null);
    },
    onError: (err: Error) => {
      setFormError(err.message);
    },
  });

  const deleteMutation = useMutation({
    mutationFn: async (caveatName: string) => {
      const res = await fetch(`/api/nexus/caveats?name=${encodeURIComponent(caveatName)}`, {
        method: "DELETE",
      });
      if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        throw new Error(body.error || "Failed to delete caveat");
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["nexus-caveats"] });
    },
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim() || !celExpression.trim()) return;
    createMutation.mutate({ name: name.trim(), cel_expression: celExpression.trim() });
  };

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader className="flex flex-row items-center justify-between pb-4">
          <div>
            <CardTitle className="text-base font-semibold flex items-center gap-2">
              <ShieldAlert className="w-4 h-4 text-primary" />
              CEL Caveats Registry
            </CardTitle>
            <CardDescription className="text-xs">
              Contextual ABAC conditions compiled and evaluated in Common Expression Language (CEL).
            </CardDescription>
          </div>

          <Dialog open={isAddOpen} onOpenChange={setIsAddOpen}>
            <DialogTrigger asChild>
              <Button size="sm">
                <Plus className="w-4 h-4 mr-1.5" /> Define Caveat
              </Button>
            </DialogTrigger>
            <DialogContent className="max-w-lg">
              <DialogHeader>
                <DialogTitle>Define CEL Caveat</DialogTitle>
              </DialogHeader>

              <form onSubmit={handleSubmit} className="space-y-4 pt-2">
                {formError && (
                  <div className="p-2.5 rounded bg-destructive/10 border border-destructive/20 text-destructive text-xs flex items-center gap-2">
                    <AlertCircle className="w-4 h-4 flex-shrink-0" />
                    <span>{formError}</span>
                  </div>
                )}

                <div className="space-y-1.5">
                  <label htmlFor="caveat-name" className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
                    Caveat Name
                  </label>
                  <Input
                    id="caveat-name"
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                    placeholder="e.g. is_business_hours"
                    required
                    className="font-mono text-xs"
                  />
                </div>

                <div className="space-y-1.5">
                  <label htmlFor="caveat-cel" className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
                    CEL Expression
                  </label>
                  <textarea
                    id="caveat-cel"
                    value={celExpression}
                    onChange={(e) => setCelExpression(e.target.value)}
                    placeholder="request.time.hour >= 9 && request.time.hour < 18"
                    rows={4}
                    required
                    className="w-full text-xs font-mono p-3 rounded-md border bg-muted/20"
                  />
                  <p className="text-[11px] text-muted-foreground">
                    Expressions have access to <code>request</code> (time, IP, headers) and tuple context variables.
                  </p>
                </div>

                <div className="flex justify-end gap-2 pt-2">
                  <Button variant="ghost" size="sm" type="button" onClick={() => setIsAddOpen(false)}>
                    Cancel
                  </Button>
                  <Button size="sm" type="submit" disabled={createMutation.isPending}>
                    {createMutation.isPending && <Loader2 className="w-4 h-4 mr-2 animate-spin" />}
                    Compile & Save
                  </Button>
                </div>
              </form>
            </DialogContent>
          </Dialog>
        </CardHeader>

        <CardContent>
          {isLoading ? (
            <div className="py-12 flex justify-center items-center text-muted-foreground">
              <Loader2 className="w-6 h-6 animate-spin mr-2" /> Loading caveats...
            </div>
          ) : isError ? (
            <div className="py-8 text-center text-sm text-destructive">
              Failed to load caveats from Nexus.
              <Button variant="link" size="sm" onClick={() => refetch()} className="block mx-auto mt-1">
                Try again
              </Button>
            </div>
          ) : caveats.length === 0 ? (
            <div className="py-12 border border-dashed rounded-lg text-center text-muted-foreground text-sm">
              No caveats registered. Click &quot;Define Caveat&quot; to attach ABAC conditions to relation tuples.
            </div>
          ) : (
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              {caveats.map((c) => (
                <div
                  key={c.name}
                  className="p-4 rounded-lg border bg-card/60 flex flex-col justify-between gap-3 hover:border-primary/40 transition-colors"
                >
                  <div className="space-y-2">
                    <div className="flex items-center justify-between">
                      <span className="font-mono font-semibold text-sm text-foreground">
                        {c.name}
                      </span>
                      <Badge variant="outline" className="text-[10px] font-mono flex items-center gap-1">
                        <Code2 className="w-3 h-3" /> CEL
                      </Badge>
                    </div>

                    <pre className="text-xs font-mono p-2.5 rounded bg-muted/40 border overflow-x-auto text-emerald-400">
                      {c.cel_expression}
                    </pre>
                  </div>

                  <div className="flex items-center justify-between pt-2 border-t text-[11px] text-muted-foreground">
                    <span className="flex items-center gap-1 text-emerald-500">
                      <CheckCircle2 className="w-3 h-3" /> Validated Syntax
                    </span>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => deleteMutation.mutate(c.name)}
                      disabled={deleteMutation.isPending}
                      className="h-7 px-2 text-destructive hover:bg-destructive/10 text-xs"
                    >
                      <Trash2 className="w-3.5 h-3.5 mr-1" /> Delete
                    </Button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
