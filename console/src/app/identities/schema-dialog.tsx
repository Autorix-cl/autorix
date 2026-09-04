"use client";

import * as React from "react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { useQuery } from "@tanstack/react-query";
import { Button } from "@/components/ui/button";
import { CopyableIdentifier } from "@/components/ui/masked-secret";
import { FileCode, Loader2, Sparkles } from "lucide-react";

interface SchemaDialogProps {
  isOpen: boolean;
  onOpenChange: (open: boolean) => void;
}

export function SchemaDialog({ isOpen, onOpenChange }: SchemaDialogProps) {
  const { data: schemas = [], isLoading } = useQuery<
    Array<{ id: string; name: string; schema: Record<string, unknown>; version: number }>
  >({
    queryKey: ["identity-schemas"],
    queryFn: async () => {
      const res = await fetch("/api/identities/schemas");
      if (!res.ok) return [];
      const json = await res.json();
      return Array.isArray(json) ? json : json.data || [];
    },
    enabled: isOpen,
  });

  const [activeSchemaId, setActiveSchemaId] = React.useState<string>("default");

  const currentSchema = React.useMemo(() => {
    return schemas.find((s) => s.id === activeSchemaId) || schemas[0];
  }, [schemas, activeSchemaId]);

  return (
    <Dialog open={isOpen} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl max-h-[85vh] flex flex-col">
        <DialogHeader>
          <div className="flex items-center justify-between pr-6">
            <DialogTitle className="flex items-center gap-2 text-base">
              <FileCode className="h-5 w-5 text-cyan-400" />
              <span>Identity Schemas (Ory Kratos Model)</span>
            </DialogTitle>
            <span className="text-[10px] font-mono px-2 py-0.5 rounded bg-cyan-500/10 text-cyan-400 border border-cyan-500/20 flex items-center gap-1">
              <Sparkles className="w-2.5 h-2.5" /> Draft-07 JSON Schema
            </span>
          </div>
          <DialogDescription className="text-xs">
            Declarative trait validation schemas for user registration, profile updates, and authentication credentials.
          </DialogDescription>
        </DialogHeader>

        {isLoading ? (
          <div className="flex items-center justify-center py-12">
            <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
          </div>
        ) : schemas.length === 0 ? (
          <div className="text-center py-8 text-xs text-muted-foreground">
            No schemas currently registered in Ego.
          </div>
        ) : (
          <div className="space-y-4 flex-1 overflow-hidden flex flex-col pt-2">
            <div className="flex items-center justify-between border-b pb-2">
              <div className="flex items-center gap-2">
                {schemas.map((s) => (
                  <Button
                    key={s.id}
                    variant={currentSchema?.id === s.id ? "secondary" : "ghost"}
                    size="sm"
                    className="text-xs h-7"
                    onClick={() => setActiveSchemaId(s.id)}
                  >
                    {s.name} ({s.id})
                  </Button>
                ))}
              </div>
              {currentSchema && (
                <CopyableIdentifier
                  value={currentSchema.id}
                  label="Schema ID"
                />
              )}
            </div>

            {currentSchema && (
              <div className="flex-1 overflow-y-auto rounded-md border bg-muted/40 p-3 font-mono text-xs text-foreground">
                <pre>{JSON.stringify(currentSchema.schema, null, 2)}</pre>
              </div>
            )}
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}
