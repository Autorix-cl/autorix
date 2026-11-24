"use client";

import * as React from "react";
import { AlertTriangle, Loader2 } from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

export interface ConfirmDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  description: React.ReactNode;
  confirmLabel?: string;
  cancelLabel?: string;
  variant?: "destructive" | "default";
  /** Optional string the user must type to enable the confirm button (e.g. a resource ID). */
  confirmText?: string;
  confirmTextAlt?: string;
  isLoading?: boolean;
  onConfirm: () => void | Promise<void>;
}

/**
 * Accessible confirmation modal built on Radix Dialog (focus trap, Escape dismiss,
 * focus return). Focuses the safe Cancel action by default for destructive confirmations.
 */
export function ConfirmDialog({
  open,
  onOpenChange,
  title,
  description,
  confirmLabel = "Confirm",
  cancelLabel = "Cancel",
  variant = "destructive",
  confirmText,
  confirmTextAlt,
  isLoading = false,
  onConfirm,
}: ConfirmDialogProps) {
  const [typedValue, setTypedValue] = React.useState("");
  const inputId = React.useId();

  React.useEffect(() => {
    if (!open) {
      setTypedValue("");
    }
  }, [open]);

  const requiresTyping = Boolean(confirmText);
  const matchesTyped =
    !requiresTyping || typedValue === confirmText || (Boolean(confirmTextAlt) && typedValue === confirmTextAlt);

  const handleConfirm = async () => {
    if (!matchesTyped || isLoading) return;
    await onConfirm();
  };

  return (
    <Dialog open={open} onOpenChange={(next) => !isLoading && onOpenChange(next)}>
      <DialogContent
        className={variant === "destructive" ? "max-w-md border-destructive/40" : "max-w-md"}
        onOpenAutoFocus={(e) => {
          // Focus stays inside the dialog on the safe Cancel button unless typed input is required
          if (!requiresTyping) {
            const cancelBtn = (e.currentTarget as HTMLElement)?.querySelector<HTMLButtonElement>(
              "[data-confirm-cancel]",
            );
            if (cancelBtn) {
              e.preventDefault();
              cancelBtn.focus();
            }
          }
        }}
      >
        <DialogHeader>
          <div className="flex items-center gap-2">
            {variant === "destructive" && (
              <AlertTriangle className="h-5 w-5 shrink-0 text-destructive" aria-hidden="true" />
            )}
            <DialogTitle>{title}</DialogTitle>
          </div>
          <DialogDescription className="pt-1">{description}</DialogDescription>
        </DialogHeader>

        {requiresTyping && confirmText && (
          <div className="space-y-1.5 pt-1">
            <Label htmlFor={inputId} className="text-xs text-muted-foreground">
              Type <span className="font-mono font-semibold text-foreground">{confirmText}</span> to confirm
            </Label>
            <Input
              id={inputId}
              value={typedValue}
              onChange={(e) => setTypedValue(e.target.value)}
              placeholder={confirmText}
              autoComplete="off"
              spellCheck={false}
              className="font-mono text-xs"
              disabled={isLoading}
            />
          </div>
        )}

        <DialogFooter className="gap-2 sm:gap-2">
          <Button
            type="button"
            variant="outline"
            size="sm"
            data-confirm-cancel=""
            disabled={isLoading}
            onClick={() => onOpenChange(false)}
          >
            {cancelLabel}
          </Button>
          <Button
            type="button"
            variant={variant === "destructive" ? "destructive" : "default"}
            size="sm"
            disabled={!matchesTyped || isLoading}
            onClick={handleConfirm}
          >
            {isLoading && <Loader2 className="mr-1.5 h-3.5 w-3.5 animate-spin" aria-hidden="true" />}
            {confirmLabel}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
