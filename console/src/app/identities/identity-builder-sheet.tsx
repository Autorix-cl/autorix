import * as React from "react";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { useTranslation } from "@/lib/i18n";
import { useQueryClient } from "@tanstack/react-query";
import { useApiMutation } from "@/lib/query/use-api-mutation";
import { fetchAndParse } from "@/lib/api/schema";
import { registrationResponseSchema } from "@/lib/api/schemas/identity";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Loader2, Fingerprint } from "lucide-react";

interface IdentityBuilderSheetProps {
  isOpen: boolean;
  onOpenChange: (open: boolean) => void;
}

export function IdentityBuilderSheet({ isOpen, onOpenChange }: IdentityBuilderSheetProps) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();

  const [email, setEmail] = React.useState("");
  const [firstName, setFirstName] = React.useState("");
  const [lastName, setLastName] = React.useState("");
  const [password, setPassword] = React.useState("");

  const createIdentity = useApiMutation(
    (vars: { email: string; firstName: string; lastName: string; password: string }) =>
      fetchAndParse("/api/identities", registrationResponseSchema, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(vars),
      }),
    {
      successMessage: (data) => t("identities.successMsg", { email: (data.identity.traits?.email as string) ?? email }),
      onSuccess: () => {
        queryClient.invalidateQueries({ queryKey: ["identities"] });
        setEmail("");
        setFirstName("");
        setLastName("");
        setPassword("");
        onOpenChange(false);
      },
    },
  );

  const handleCreate = (e: React.FormEvent) => {
    e.preventDefault();
    if (!email || !password) return;
    createIdentity.mutate({ email, firstName, lastName, password });
  };

  return (
    <Sheet open={isOpen} onOpenChange={onOpenChange}>
      <SheetContent className="sm:max-w-md">
        <SheetHeader>
          <SheetTitle>{t("identities.registerTitle")}</SheetTitle>
          <SheetDescription>{t("identities.registerDesc")}</SheetDescription>
        </SheetHeader>

        <form onSubmit={handleCreate} className="space-y-4 mt-6">
          <div className="space-y-1.5">
            <Label htmlFor="email">{t("identities.emailLabel")}</Label>
            <Input
              id="email"
              type="email"
              placeholder={t("identities.emailPlaceholder")}
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              required
            />
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label htmlFor="firstName">{t("identities.firstNameLabel")}</Label>
              <Input
                id="firstName"
                type="text"
                placeholder={t("identities.firstNamePlaceholder")}
                value={firstName}
                onChange={(e) => setFirstName(e.target.value)}
              />
            </div>

            <div className="space-y-1.5">
              <Label htmlFor="lastName">{t("identities.lastNameLabel")}</Label>
              <Input
                id="lastName"
                type="text"
                placeholder={t("identities.lastNamePlaceholder")}
                value={lastName}
                onChange={(e) => setLastName(e.target.value)}
              />
            </div>
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="password">{t("identities.passwordLabel")}</Label>
            <Input
              id="password"
              type="password"
              placeholder={t("identities.passwordPlaceholder")}
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
              minLength={8}
            />
          </div>

          <Button
            type="submit"
            variant="default"
            disabled={createIdentity.isPending}
            className="w-full gap-2 mt-4 bg-blue-600 hover:bg-blue-700 text-white"
          >
            {createIdentity.isPending ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <Fingerprint className="h-4 w-4" />
            )}
            <span>{createIdentity.isPending ? t("common.loading") : t("identities.submitBtn")}</span>
          </Button>
        </form>
      </SheetContent>
    </Sheet>
  );
}
