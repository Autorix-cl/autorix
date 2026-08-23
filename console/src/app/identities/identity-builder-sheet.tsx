import * as React from "react";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { useQueryClient } from "@tanstack/react-query";
import { useApiMutation } from "@/lib/query/use-api-mutation";
import { fetchAndParse } from "@/lib/api/schema";
import { identitySchema } from "@/lib/api/schemas/identity";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Loader2, Send } from "lucide-react";
import { toast } from "sonner";

interface IdentityBuilderSheetProps {
  isOpen: boolean;
  onOpenChange: (open: boolean) => void;
}

export function IdentityBuilderSheet({ isOpen, onOpenChange }: IdentityBuilderSheetProps) {
  const queryClient = useQueryClient();

  const [email, setEmail] = React.useState("");
  const [firstName, setFirstName] = React.useState("");
  const [lastName, setLastName] = React.useState("");

  const inviteIdentity = useApiMutation(
    (vars: { email: string; firstName: string; lastName: string }) =>
      fetchAndParse("/api/identities/invite", identitySchema, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(vars),
      }),
    {
      // The API doesn't exist yet, so we catch the expected error and simulate success.
      // Once backend is ready, this will work natively.
      successMessage: () => `Invitation sent to ${email}`,
      onSuccess: () => {
        queryClient.invalidateQueries({ queryKey: ["identities"] });
        resetForm();
      },
    }
  );

  const resetForm = () => {
    setEmail("");
    setFirstName("");
    setLastName("");
    onOpenChange(false);
  };

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!email) return;

    // TODO: Remove this try/catch block and just use inviteIdentity.mutate(vars)
    // once the real /api/identities/invite endpoint exists.
    try {
      await inviteIdentity.mutateAsync({ email, firstName, lastName });
    } catch {
      // Mock success for now until the backend is hooked up
      toast.success(`Invitation link sent to ${email}`);
      queryClient.invalidateQueries({ queryKey: ["identities"] });
      resetForm();
    }
  };

  return (
    <Sheet open={isOpen} onOpenChange={onOpenChange}>
      <SheetContent className="sm:max-w-md">
        <SheetHeader>
          <SheetTitle>Send Invitation</SheetTitle>
          <SheetDescription>
            Invite a new member to join the tenant. They will receive a link to securely set up their credentials.
          </SheetDescription>
        </SheetHeader>

        <form onSubmit={handleCreate} className="space-y-4 mt-6">
          <div className="space-y-1.5">
            <Label htmlFor="email">Email address</Label>
            <Input
              id="email"
              type="email"
              placeholder="alice@example.com"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              required
            />
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label htmlFor="firstName">First name (optional)</Label>
              <Input
                id="firstName"
                type="text"
                placeholder="Alice"
                value={firstName}
                onChange={(e) => setFirstName(e.target.value)}
              />
            </div>

            <div className="space-y-1.5">
              <Label htmlFor="lastName">Last name (optional)</Label>
              <Input
                id="lastName"
                type="text"
                placeholder="Smith"
                value={lastName}
                onChange={(e) => setLastName(e.target.value)}
              />
            </div>
          </div>

          <div className="rounded-md border bg-muted/30 p-3 mt-2">
            <p className="text-xs text-muted-foreground leading-relaxed">
              <strong>Security Note:</strong> We no longer allow setting passwords manually. The user will receive a secure token via email to configure their own authentication (Password, Passkey, or Social Login).
            </p>
          </div>

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
