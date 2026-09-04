"use client";

import * as React from "react";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Loader2, UserPlus, Shield, Eye, EyeOff } from "lucide-react";
import { toast } from "sonner";
import { useQueryClient } from "@tanstack/react-query";

interface OperatorBuilderSheetProps {
  isOpen: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess?: () => void;
}

const ROLE_DESCRIPTIONS: Record<string, { label: string; desc: string }> = {
  owner: {
    label: "Owner (Root Principal)",
    desc: "Wildcard (*) permissions across all 7 engines and cluster configuration.",
  },
  admin: {
    label: "Admin (Full Control)",
    desc: "Full write access to identities, policies, OAuth2 clients, and proxy rules.",
  },
  operator: {
    label: "Operator (Standard)",
    desc: "Operational access with read capabilities across all engines and identity lifecycle.",
  },
  auditor: {
    label: "Auditor (Read-Only)",
    desc: "Compliance and governance auditing across immutable hash chains and evidence.",
  },
};

export function OperatorBuilderSheet({ isOpen, onOpenChange, onSuccess }: OperatorBuilderSheetProps) {
  const queryClient = useQueryClient();
  const [name, setName] = React.useState("");
  const [email, setEmail] = React.useState("");
  const [role, setRole] = React.useState<string>("operator");
  const [password, setPassword] = React.useState("");
  const [showPassword, setShowPassword] = React.useState(false);
  const [isSubmitting, setIsSubmitting] = React.useState(false);

  const resetForm = () => {
    setName("");
    setEmail("");
    setRole("operator");
    setPassword("");
    setShowPassword(false);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!name.trim() || !email.trim() || !password) {
      toast.error("Please fill in all required fields");
      return;
    }

    if (password.length < 8) {
      toast.error("Password must be at least 8 characters");
      return;
    }

    setIsSubmitting(true);
    try {
      const res = await fetch("/api/operators", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          name: name.trim(),
          email: email.trim().toLowerCase(),
          role,
          password,
        }),
      });

      if (!res.ok) {
        const errorData = await res.json().catch(() => ({}));
        throw new Error(errorData.error || errorData.message || "Failed to create operator");
      }

      toast.success(`Operator ${name} created successfully with role ${role.toUpperCase()}!`);
      queryClient.invalidateQueries({ queryKey: ["operators"] });
      resetForm();
      onOpenChange(false);
      onSuccess?.();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Error provisioning operator");
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <Sheet open={isOpen} onOpenChange={onOpenChange}>
      <SheetContent className="w-full sm:max-w-md overflow-y-auto">
        <SheetHeader className="pb-4 border-b border-border/80">
          <div className="flex items-center gap-2 text-amber-400">
            <UserPlus className="h-5 w-5" />
            <SheetTitle className="text-base font-semibold">Provision New Operator</SheetTitle>
          </div>
          <SheetDescription className="text-xs">
            Create an administrative principal and assign RBAC role permissions for the Autorix Control Plane.
          </SheetDescription>
        </SheetHeader>

        <form onSubmit={handleSubmit} className="space-y-4 py-6">
          <div className="space-y-1.5">
            <Label htmlFor="op-name" className="text-xs font-medium">
              Full Name
            </Label>
            <Input
              id="op-name"
              placeholder="e.g. Ada Lovelace"
              value={name}
              onChange={(e) => setName(e.target.value)}
              className="text-xs"
              required
            />
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="op-email" className="text-xs font-medium">
              Operator Email
            </Label>
            <Input
              id="op-email"
              type="email"
              placeholder="e.g. ada@autorix.internal"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className="text-xs font-mono"
              required
            />
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="op-role" className="text-xs font-medium">
              RBAC Role Assignment
            </Label>
            <Select value={role} onValueChange={setRole}>
              <SelectTrigger id="op-role" className="text-xs">
                <SelectValue placeholder="Select role" />
              </SelectTrigger>
              <SelectContent>
                {Object.entries(ROLE_DESCRIPTIONS).map(([key, info]) => (
                  <SelectItem key={key} value={key} className="text-xs">
                    <span className="font-semibold">{info.label}</span>
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <div className="p-2.5 rounded-lg border border-border/60 bg-muted/40 text-[11px] text-muted-foreground mt-1.5 space-y-1">
              <span className="font-semibold text-foreground flex items-center gap-1">
                <Shield className="w-3 h-3 text-amber-400" />
                {ROLE_DESCRIPTIONS[role]?.label}
              </span>
              <p>{ROLE_DESCRIPTIONS[role]?.desc}</p>
            </div>
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="op-password" className="text-xs font-medium">
              Master Password (Argon2id Encrypted)
            </Label>
            <div className="relative">
              <Input
                id="op-password"
                type={showPassword ? "text" : "password"}
                placeholder="Minimum 8 characters"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className="text-xs font-mono pr-9"
                required
                minLength={8}
              />
              <button
                type="button"
                onClick={() => setShowPassword(!showPassword)}
                className="absolute right-2.5 top-2.5 text-muted-foreground hover:text-foreground"
              >
                {showPassword ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
              </button>
            </div>
            <p className="text-[11px] text-muted-foreground">
              Stored locally in Argus vault for break-glass resilience during upstream SSO outages.
            </p>
          </div>

          <div className="pt-4 flex items-center justify-end gap-2 border-t border-border/80">
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => onOpenChange(false)}
              disabled={isSubmitting}
              className="text-xs h-8"
            >
              Cancel
            </Button>
            <Button
              type="submit"
              size="sm"
              disabled={isSubmitting}
              className="text-xs h-8 gap-1.5 bg-amber-600 hover:bg-amber-700 text-white font-medium shadow-sm"
            >
              {isSubmitting ? (
                <>
                  <Loader2 className="w-3.5 h-3.5 animate-spin" />
                  <span>Provisioning...</span>
                </>
              ) : (
                <>
                  <UserPlus className="w-3.5 h-3.5" />
                  <span>Create Operator</span>
                </>
              )}
            </Button>
          </div>
        </form>
      </SheetContent>
    </Sheet>
  );
}
