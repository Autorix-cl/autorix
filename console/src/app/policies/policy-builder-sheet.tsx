"use client";

import * as React from "react";
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
import { Textarea } from "@/components/ui/textarea";
import { Loader2, Sparkles, Plus } from "lucide-react";
import { useTranslation } from "@/lib/i18n";
import { useApiMutation } from "@/lib/query/use-api-mutation";
import { fetchAndParse } from "@/lib/api/schema";
import { policySchema } from "@/lib/api/schemas/themis";
import { useQueryClient } from "@tanstack/react-query";

interface PolicyBuilderSheetProps {
  tenantId: string;
}

export function PolicyBuilderSheet({ tenantId }: PolicyBuilderSheetProps) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [open, setOpen] = React.useState(false);

  const [name, setName] = React.useState("");
  const [description, setDescription] = React.useState("");
  const [expression, setExpression] = React.useState('request.auth.claims.department == "finance"');
  const [priority, setPriority] = React.useState("1");

  const createPolicy = useApiMutation(
    (vars: { tenantId: string; name: string; description: string; expression: string; priority: number }) =>
      fetchAndParse("/api/policies", policySchema, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ ...vars, enabled: true }),
      }),
    {
      successMessage: (data) => t("themis.successMsg", { name: data.Name }),
      onSuccess: () => {
        queryClient.invalidateQueries({ queryKey: ["policies", tenantId] });
        setName("");
        setDescription("");
        setOpen(false);
      },
      errorFix: "Fix the CEL expression syntax and try again.",
    },
  );

  const handleCreatePolicy = (e: React.FormEvent) => {
    e.preventDefault();
    if (!name || !expression) return;
    createPolicy.mutate({ tenantId, name, description, expression, priority: Number(priority) || 1 });
  };

  const isSubmitting = createPolicy.isPending;

  const setTemplate = (expr: string, templateName: string) => {
    setExpression(expr);
    if (!name) setName(templateName);
  };

  return (
    <Sheet open={open} onOpenChange={setOpen}>
      <SheetTrigger asChild>
        <Button variant="purple" size="sm" className="gap-2">
          <Plus className="h-4 w-4" />
          <span>{t("themis.createTitle")}</span>
        </Button>
      </SheetTrigger>
      <SheetContent className="sm:max-w-md w-full overflow-y-auto">
        <SheetHeader className="mb-6">
          <SheetTitle className="flex items-center gap-2">
            <Sparkles className="h-4 w-4 text-purple-400" />
            {t("themis.createTitle")}
          </SheetTitle>
          <SheetDescription>{t("themis.createDesc")}</SheetDescription>
        </SheetHeader>

        <form onSubmit={handleCreatePolicy} className="space-y-4">
          <div className="space-y-1.5">
            <Label htmlFor="name">{t("themis.policyNameLabel")}</Label>
            <Input
              id="name"
              placeholder={t("themis.policyNamePlaceholder")}
              value={name}
              onChange={(e) => setName(e.target.value)}
              required
            />
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="priority">{t("themis.priorityLabel")}</Label>
            <Input
              id="priority"
              type="number"
              min="1"
              max="100"
              value={priority}
              onChange={(e) => setPriority(e.target.value)}
            />
          </div>

          <div className="space-y-1.5">
            <div className="flex items-center justify-between">
              <Label htmlFor="expression">{t("themis.expressionLabel")}</Label>
              <span className="text-[10px] font-mono text-purple-400">Google CEL AST</span>
            </div>
            <div className="font-mono">
              <Textarea
                id="expression"
                rows={4}
                className="font-mono text-xs text-purple-300 resize-none bg-black/40 border-border/70"
                placeholder={t("themis.expressionPlaceholder")}
                value={expression}
                onChange={(e) => setExpression(e.target.value)}
                required
              />
            </div>
          </div>

          {/* Template Chips */}
          <div className="space-y-1.5">
            <span className="text-[11px] font-medium text-muted-foreground">Quick CEL Templates:</span>
            <div className="flex flex-wrap gap-1.5">
              <button
                type="button"
                onClick={() => setTemplate('request.auth.claims.department == "finance"', "Finance Dept Only")}
                className="rounded border border-border/80 bg-muted/40 px-2 py-0.5 text-[10px] font-mono text-muted-foreground hover:text-foreground hover:border-purple-500/50 transition-colors"
              >
                Department Match
              </button>
              <button
                type="button"
                onClick={() => setTemplate("resource.owner_id == auth.claims.sub", "Resource Owner Enforce")}
                className="rounded border border-border/80 bg-muted/40 px-2 py-0.5 text-[10px] font-mono text-muted-foreground hover:text-foreground hover:border-purple-500/50 transition-colors"
              >
                Owner Check
              </button>
              <button
                type="button"
                onClick={() =>
                  setTemplate('request.amount < 10000 || auth.claims.role == "vp"', "High Value Approver")
                }
                className="rounded border border-border/80 bg-muted/40 px-2 py-0.5 text-[10px] font-mono text-muted-foreground hover:text-foreground hover:border-purple-500/50 transition-colors"
              >
                Threshold & Role
              </button>
            </div>
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="description">{t("themis.descriptionLabel")}</Label>
            <Textarea
              id="description"
              rows={2}
              className="resize-none"
              placeholder={t("themis.descriptionPlaceholder")}
              value={description}
              onChange={(e) => setDescription(e.target.value)}
            />
          </div>

          <Button type="submit" variant="purple" disabled={isSubmitting} className="w-full gap-2 mt-4">
            {isSubmitting ? <Loader2 className="h-4 w-4 animate-spin" /> : <Sparkles className="h-4 w-4" />}
            <span>{isSubmitting ? t("common.loading") : t("themis.submitBtn")}</span>
          </Button>
        </form>
      </SheetContent>
    </Sheet>
  );
}
