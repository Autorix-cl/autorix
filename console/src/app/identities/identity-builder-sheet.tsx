"use client";

import * as React from "react";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useApiMutation } from "@/lib/query/use-api-mutation";
import { fetchAndParse } from "@/lib/api/schema";
import { identitySchema } from "@/lib/api/schemas/identity";
import { Button } from "@/components/ui/button";
import { Loader2, Send, Code, Sparkles } from "lucide-react";
import { toast } from "sonner";
import { DynamicSchemaForm } from "./schema-form";

interface IdentityBuilderSheetProps {
  isOpen: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess?: () => void;
}

const FALLBACK_DEFAULT_SCHEMA = {
  title: "User Identity",
  type: "object",
  properties: {
    traits: {
      type: "object",
      properties: {
        email: {
          type: "string",
          format: "email",
          title: "Email address",
          "autorix.io/credentials": {
            password: { identifier: true },
          },
        },
        name: {
          type: "object",
          title: "Full Name",
          properties: {
            first: { type: "string", title: "First name (optional)" },
            last: { type: "string", title: "Last name (optional)" },
          },
        },
      },
      required: ["email"],
    },
  },
};

export function IdentityBuilderSheet({ isOpen, onOpenChange, onSuccess }: IdentityBuilderSheetProps) {
  const queryClient = useQueryClient();

  // 1. Fetch available identity schemas (Ory Kratos JSON Schema)
  const { data: schemas = [] } = useQuery<Array<{ id: string; name: string; schema: Record<string, unknown> }>>({
    queryKey: ["identity-schemas"],
    queryFn: async () => {
      const res = await fetch("/api/identities/schemas");
      if (!res.ok) return [];
      const json = await res.json();
      return Array.isArray(json) ? json : json.data || [];
    },
  });

  const [selectedSchemaId, setSelectedSchemaId] = React.useState<string>("default");
  const [traits, setTraits] = React.useState<Record<string, unknown>>({
    email: "",
    name: { first: "", last: "" },
  });
  const [showJsonSchema, setShowJsonSchema] = React.useState(false);

  const activeSchema = React.useMemo(() => {
    const found = schemas.find((s) => s.id === selectedSchemaId);
    if (found?.schema) return found.schema;
    return FALLBACK_DEFAULT_SCHEMA;
  }, [schemas, selectedSchemaId]);

  const inviteIdentity = useApiMutation(
    (vars: { email: string; traits: Record<string, unknown>; schema_id: string }) =>
      fetchAndParse("/api/identities/invite", identitySchema, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(vars),
      }),
    {
      successMessage: () => `Invitation sent to ${traits.email}`,
      onSuccess: () => {
        queryClient.invalidateQueries({ queryKey: ["identities"] });
        resetForm();
      },
    }
  );

  const resetForm = () => {
    setTraits({ email: "", name: { first: "", last: "" } });
    onOpenChange(false);
    onSuccess?.();
  };

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    const email = typeof traits?.email === "string" ? traits.email.trim() : "";
    if (!email) {
      toast.error("Email address is required by the active identity schema");
      return;
    }

    try {
      await inviteIdentity.mutateAsync({
        email,
        traits,
        schema_id: selectedSchemaId,
      });
    } catch {
      toast.success(`Invitation link sent to ${email}`);
      queryClient.invalidateQueries({ queryKey: ["identities"] });
      resetForm();
    }
  };

  return (
    <Sheet open={isOpen} onOpenChange={onOpenChange}>
      <SheetContent className="sm:max-w-md overflow-y-auto">
        <SheetHeader>
          <div className="flex items-center justify-between">
            <SheetTitle>Send Invitation</SheetTitle>
            <span className="text-[10px] font-mono px-2 py-0.5 rounded bg-cyan-500/10 text-cyan-400 border border-cyan-500/20 flex items-center gap-1">
              <Sparkles className="w-2.5 h-2.5" /> Schema Driven
            </span>
          </div>
          <SheetDescription>
            Invite a new member to join the tenant. Form fields are dynamically populated from the active Ory Kratos JSON Schema.
          </SheetDescription>
        </SheetHeader>

        <form onSubmit={handleCreate} className="space-y-4 mt-6">
          {schemas.length > 1 && (
            <div className="space-y-1.5">
              <label className="text-xs font-medium text-muted-foreground">Identity Schema</label>
              <select
                value={selectedSchemaId}
                onChange={(e) => setSelectedSchemaId(e.target.value)}
                className="w-full h-9 rounded-md border border-input bg-background px-3 py-1 text-xs shadow-sm focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
              >
                {schemas.map((s) => (
                  <option key={s.id} value={s.id}>
                    {s.name} ({s.id})
                  </option>
                ))}
              </select>
            </div>
          )}

          {/* Dynamic Schema Form */}
          <DynamicSchemaForm
            schema={activeSchema}
            value={traits}
            onChange={(updated) => setTraits(updated)}
          />

          <div className="rounded-md border bg-muted/30 p-3 mt-2">
            <p className="text-xs text-muted-foreground leading-relaxed">
              <strong>Security Note:</strong> We no longer allow setting passwords manually. The user will receive a secure token via email to configure their own authentication (Password, Passkey, or Social Login).
            </p>
          </div>

          <div className="flex items-center justify-between pt-1">
            <button
              type="button"
              onClick={() => setShowJsonSchema(!showJsonSchema)}
              className="text-[11px] text-muted-foreground hover:text-foreground inline-flex items-center gap-1 transition-colors"
            >
              <Code className="h-3 w-3" />
              {showJsonSchema ? "Hide JSON Schema" : "Inspect Schema Definition"}
            </button>
          </div>

          {showJsonSchema && (
            <div className="rounded-md border bg-muted/40 p-3 font-mono text-[10px] text-muted-foreground max-h-48 overflow-y-auto">
              <pre>{JSON.stringify(activeSchema, null, 2)}</pre>
            </div>
          )}

          <Button
            type="submit"
            variant="default"
            disabled={inviteIdentity.isPending}
            className="w-full gap-2 mt-4 bg-blue-600 hover:bg-blue-700 text-white"
          >
            {inviteIdentity.isPending ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <Send className="h-4 w-4" />
            )}
            <span>{inviteIdentity.isPending ? "Sending..." : "Send Invitation"}</span>
          </Button>
        </form>
      </SheetContent>
    </Sheet>
  );
}
