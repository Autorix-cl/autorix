"use client";

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
import { toast } from "sonner";

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
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload)
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
    onError: (error) => {
      toast.error(error.message);
    }
  });

  const onSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    mutation.mutate({
      name,
      priority: parseInt(priority, 10),
      expression
    });
  };

  return (
    <Sheet open={open} onOpenChange={setOpen}>
      <SheetTrigger asChild>
        <Button>
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
            <Label>Policy Name</Label>
            <Input 
              value={name} 
              onChange={(e) => setName(e.target.value)} 
              placeholder="e.g. Admin Only Rule" 
              required
            />
          </div>
          
          <div className="space-y-2">
            <Label>Evaluation Priority</Label>
            <Input 
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
            <Label>CEL Expression</Label>
            <div className="border rounded-md overflow-hidden min-h-[150px]">
              <CodeEditor 
                value={expression}
                onChange={setExpression}
                language="javascript" // close enough syntax to CEL for basic highlighting
              />
            </div>
            <p className="text-xs text-muted-foreground">
              Evaluates to boolean. Use <code className="bg-muted px-1">request.auth.claims.role == 'admin'</code>
            </p>
          </div>
          
          <Button type="submit" disabled={mutation.isPending} className="w-full">
            {mutation.isPending ? "Validating & Saving..." : "Save Policy"}
          </Button>
        </form>
      </SheetContent>
    </Sheet>
  );
}
