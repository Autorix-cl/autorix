"use client";

import * as React from "react";
import { useTranslation } from "@/lib/i18n";
import { useQueryClient } from "@tanstack/react-query";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "@/components/ui/sheet";
import { Plus, Lock, Loader2 } from "lucide-react";
import { useApiMutation } from "@/lib/query/use-api-mutation";
import { fetchAndParse } from "@/lib/api/schema";
import { createKeyResponseSchema, macaroonSchema, type APIKey } from "@/lib/api/schemas/vulcan";
import type { z } from "zod";

type Macaroon = z.infer<typeof macaroonSchema>;

interface KeyBuilderSheetProps {
  onKeyCreated: (data: { api_key: APIKey; raw_token?: string; macaroon: Macaroon; isLive: boolean }) => void;
}

export function KeyBuilderSheet({ onKeyCreated }: KeyBuilderSheetProps) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();

  const [open, setOpen] = React.useState(false);
  const [keyName, setKeyName] = React.useState("");
  const [ownerId, setOwnerId] = React.useState("");
  const [isLive, setIsLive] = React.useState(true);

  const createKey = useApiMutation(
    (vars: { name: string; ownerId: string; isLive: boolean }) =>
      fetchAndParse("/api/keys", createKeyResponseSchema, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(vars),
      }),
    {
      successMessage: "API key generated",
      onSuccess: (data, vars) => {
        queryClient.invalidateQueries({ queryKey: ["api-keys"] });
        onKeyCreated({ ...data, isLive: vars.isLive });
        setKeyName("");
        setOwnerId("");
        setIsLive(true);
        setOpen(false);
      },
    }
  );

  const handleCreate = (e: React.FormEvent) => {
    e.preventDefault();
    if (!keyName) return;
    createKey.mutate({ name: keyName, ownerId, isLive });
  };

  return (
    <Sheet open={open} onOpenChange={setOpen}>
      <SheetTrigger asChild>
        <Button variant="cyan" size="sm" className="gap-2">
          <Plus className="h-4 w-4" />
          <span>{t("apiKeys.generateBtn")}</span>
        </Button>
      </SheetTrigger>
      <SheetContent>
        <SheetHeader>
          <SheetTitle>{t("apiKeys.generateTitle")}</SheetTitle>
          <SheetDescription>{t("apiKeys.generateDesc")}</SheetDescription>
        </SheetHeader>
        <div className="py-6">
          <form onSubmit={handleCreate} className="space-y-4">
            <div className="space-y-1.5">
              <Label htmlFor="keyName">{t("apiKeys.keyNameLabel")}</Label>
              <Input
                id="keyName"
                placeholder={t("apiKeys.keyNamePlaceholder")}
                value={keyName}
                onChange={(e) => setKeyName(e.target.value)}
                required
              />
            </div>

            <div className="space-y-1.5">
              <Label htmlFor="ownerId">{t("apiKeys.ownerLabel")}</Label>
              <Input
                id="ownerId"
                placeholder={t("apiKeys.ownerPlaceholder")}
                value={ownerId}
                onChange={(e) => setOwnerId(e.target.value)}
              />
            </div>

            <div className="space-y-1.5">
              <Label htmlFor="env">{t("apiKeys.envLabel")}</Label>
              <Select value={isLive ? "live" : "test"} onValueChange={(val) => setIsLive(val === "live")}>
                <SelectTrigger id="env">
                  <SelectValue placeholder="Select Environment" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="live">{t("apiKeys.liveOption")}</SelectItem>
                  <SelectItem value="test">{t("apiKeys.testOption")}</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <Button type="submit" variant="cyan" disabled={createKey.isPending} className="w-full gap-2 mt-4">
              {createKey.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : <Lock className="h-4 w-4" />}
              <span>{createKey.isPending ? t("common.loading") : t("apiKeys.generateBtn")}</span>
            </Button>
          </form>
        </div>
      </SheetContent>
    </Sheet>
  );
}
