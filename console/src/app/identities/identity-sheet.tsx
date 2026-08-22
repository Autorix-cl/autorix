"use client";

import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetDescription } from "@/components/ui/sheet";
import { IdentityItem } from "./columns";
import { CodeBlock } from "@/components/ui/code-block";
import { Button } from "@/components/ui/button";
import { ShieldAlert, UserX, Loader2 } from "lucide-react";
import React from "react";
import { toast } from "sonner";

interface IdentitySheetProps {
  identity: IdentityItem | null;
  isOpen: boolean;
  onOpenChange: (open: boolean) => void;
}

export function IdentitySheet({ identity, isOpen, onOpenChange }: IdentitySheetProps) {
  const [isSuspending, setIsSuspending] = React.useState(false);

  const handleSuspend = () => {
    setIsSuspending(true);
    // Mocking an API call
    setTimeout(() => {
      setIsSuspending(false);
      toast.success("Account suspended successfully.");
      onOpenChange(false);
    }, 1000);
  };

  if (!identity) return null;

  return (
    <Sheet open={isOpen} onOpenChange={onOpenChange}>
      <SheetContent className="sm:max-w-md overflow-y-auto">
        <SheetHeader className="mb-6">
          <SheetTitle className="text-lg font-semibold flex items-center gap-2">
            Identity Details
          </SheetTitle>
          <SheetDescription>
            View and manage the 360 profile of this user.
          </SheetDescription>
        </SheetHeader>

        <div className="space-y-6">
          <div className="space-y-2">
            <h3 className="text-sm font-medium text-foreground">Traits (JSON)</h3>
            <CodeBlock
              code={JSON.stringify(identity.original.traits, null, 2)}
              language="json"
              title="traits.json"
              className="max-h-96"
            />
          </div>

          <div className="space-y-2">
            <h3 className="text-sm font-medium text-foreground flex items-center gap-2">
              <ShieldAlert className="h-4 w-4 text-orange-500" />
              Security Actions
            </h3>
            <div className="rounded-md border border-destructive/20 bg-destructive/10 p-4">
              <h4 className="text-sm font-medium text-destructive mb-1">Suspend Account</h4>
              <p className="text-xs text-muted-foreground mb-4">
                Instantly revoke all active sessions and block new logins.
              </p>
              <Button 
                variant="destructive" 
                size="sm" 
                onClick={handleSuspend}
                disabled={isSuspending}
                className="w-full sm:w-auto"
              >
                {isSuspending ? (
                  <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                ) : (
                  <UserX className="h-4 w-4 mr-2" />
                )}
                Suspend User
              </Button>
            </div>
          </div>
        </div>
      </SheetContent>
    </Sheet>
  );
}
