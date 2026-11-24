"use client";

import * as React from "react";
import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "@/components/ui/sheet";
import { Button } from "@/components/ui/button";
import { Plus } from "lucide-react";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { CodeEditor } from "@/components/ui/code-editor";
import { Can } from "@/lib/auth/use-permission";
import { toast } from "sonner";

function readCsrfHeaders(): Record<string, string> {
  if (typeof document === "undefined") return {};
  const m = document.cookie.match(/(?:^|;\s*)autorix_csrf=([^;]+)/);
  return m ? { "X-CSRF-Token": decodeURIComponent(m[1]) } : {};
}

export function PolicyBuilderSheet() {
  const [open, setOpen] = useState(false);
  const [name, setName] = useState("");
  const [priority, setPriority] = useState("10");
  const [expression, setExpression] = useState("request.role == 'admin'");

  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: async (payload: { name: string; priority: number; expression: string }) => {
      const res = await fetch("/api/themis/policies", {
        method: "POST",
        headers: { "Content-Type": "application/json", ...readCsrfHeaders() },
        body: JSON.stringify(payload),
      });
      const data = await res.json();
      if (!res.ok) throw new Error(data.error || "Failed to create policy");
      return data;
    },
    onSuccess: () => {
      toast.success("Policy created successfully");
      queryClient.invalidateQueries({ queryKey: ["themis-policies"] });
      setOpen(false);
      setName("");
    },
    onError: (error: Error) => {
      toast.error(error.message);
    },
  });

  const onSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    const parsedPriority = Number.parseInt(priority, 10);
    mutation.mutate({
      name: name.trim(),
      priority: Number.isNaN(parsedPriority) ? 10 : parsedPriority,
      expression,
    });
  };

  return (
    <Can perform="policies:write">
      <Sheet open={open} onOpenChange={setOpen}>
        <SheetTrigger asChild>
          <Button type="button">
            <Plus className="w-4 h-4 mr-2" />
            Create Policy
          </Button>
        </SheetTrigger>
        <SheetContent className="sm:max-w-[600px] w-[90vw] overflow-y-auto">
          <SheetHeader>
            <SheetTitle>Create ABAC Policy</SheetTitle>
            <SheetDescription>
              Write a CEL expression that will be evaluated against incoming requests.
            </SheetDescription>
          </SheetHeader>

          <form onSubmit={onSubmit} className="space-y-6 mt-6">
            <div className="space-y-2">
              <Label htmlFor="themis-policy-name">Policy Name</Label>
              <Input
                id="themis-policy-name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="e.g. Admin Only Rule"
                required
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="themis-policy-priority">Evaluation Priority</Label>
              <Input
                id="themis-policy-priority"
                type="number"
                value={priority}
                onChange={(e) => setPriority(e.target.value)}
                min={1}
                max={100}
                required
              />
              <p className="text-xs text-muted-foreground">Lower number = higher priority.</p>
            </div>

            <div className="space-y-2">
              <Label htmlFor="themis-policy-expression">CEL Expression</Label>
              <div id="themis-policy-expression" className="border rounded-md overflow-hidden min-h-[150px]">
                <CodeEditor
                  value={expression}
                  onChange={setExpression}
                  language="cel"
                />
              </div>
              <p className="text-xs text-muted-foreground">
                Evaluates to boolean. Use{" "}
                <code className="bg-muted px-1">request.auth.claims.role == &apos;admin&apos;</code>
              </p>
            </div>

            <Button type="submit" disabled={mutation.isPending} className="w-full">
              {mutation.isPending ? "Validating & Saving..." : "Save Policy"}
            </Button>
          </form>
        </SheetContent>
      </Sheet>
    </Can>
  );
}
