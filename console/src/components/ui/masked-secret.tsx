"use client";

import * as React from "react";
import { Eye, EyeOff, Copy, Check } from "lucide-react";
import { Button } from "@/components/ui/button";
import { toast } from "sonner";

interface MaskedSecretProps {
  secret: string;
  label?: string;
  maskLength?: number;
  autoMaskDelayMs?: number;
  onReveal?: () => void;
  className?: string;
}

/**
 * MaskedSecret renders sensitive cryptographic material (tokens, keys, Macaroons)
 * masked by default, requiring an explicit user action to reveal. Automatically
 * re-masks after a timeout to prevent shoulder surfing and screen capture exposure.
 */
export function MaskedSecret({
  secret,
  label = "secret",
  maskLength = 24,
  autoMaskDelayMs = 30000,
  onReveal,
  className = "",
}: MaskedSecretProps) {
  const [revealed, setRevealed] = React.useState(false);
  const [copied, setCopied] = React.useState(false);

  React.useEffect(() => {
    if (!revealed) return;
    const timer = setTimeout(() => {
      setRevealed(false);
    }, autoMaskDelayMs);
    return () => clearTimeout(timer);
  }, [revealed, autoMaskDelayMs]);

  const handleToggle = () => {
    if (!revealed) {
      onReveal?.();
      setRevealed(true);
    } else {
      setRevealed(false);
    }
  };

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(secret);
      setCopied(true);
      toast.success(`Copied ${label} to clipboard`);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      toast.error(`Failed to copy ${label}`);
    }
  };

  const maskedText = "•".repeat(maskLength);

  return (
    <div
      className={`inline-flex items-center gap-2 rounded-md border border-border/60 bg-muted/30 px-2.5 py-1 font-mono text-xs ${className}`}
    >
      <span className="select-all text-muted-foreground truncate max-w-[280px]">
        {revealed ? secret : maskedText}
      </span>
      <div className="flex items-center gap-1">
        <Button
          type="button"
          variant="ghost"
          size="icon"
          className="h-6 w-6 text-muted-foreground hover:text-foreground"
          onClick={handleToggle}
          title={revealed ? "Mask secret" : "Reveal secret"}
          aria-label={revealed ? "Mask secret" : "Reveal secret"}
        >
          {revealed ? <EyeOff className="h-3.5 w-3.5" /> : <Eye className="h-3.5 w-3.5" />}
        </Button>
        {revealed && (
          <Button
            type="button"
            variant="ghost"
            size="icon"
            className="h-6 w-6 text-muted-foreground hover:text-foreground"
            onClick={handleCopy}
            title={`Copy ${label}`}
            aria-label={`Copy ${label}`}
          >
            {copied ? (
              <Check className="h-3.5 w-3.5 text-emerald-400" />
            ) : (
              <Copy className="h-3.5 w-3.5" />
            )}
          </Button>
        )}
      </div>
    </div>
  );
}

interface CopyableIdentifierProps {
  value: string;
  label?: string;
  truncateLength?: number;
  className?: string;
}

/**
 * CopyableIdentifier renders public identifiers (WAL LSNs, Zookies, UUIDs)
 * with instant 1-click clipboard micro-interaction and visual checkmark feedback.
 */
export function CopyableIdentifier({
  value = "",
  label = "identifier",
  truncateLength = 32,
  className = "",
}: CopyableIdentifierProps) {
  const [copied, setCopied] = React.useState(false);
  const safeValue = typeof value === "string" ? value : String(value ?? "");

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(safeValue);
      setCopied(true);
      toast.success(`Copied ${label}`);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      toast.error(`Failed to copy ${label}`);
    }
  };

  const displayValue =
    truncateLength && safeValue.length > truncateLength
      ? `${safeValue.slice(0, truncateLength / 2)}...${safeValue.slice(-truncateLength / 2)}`
      : safeValue;

  return (
    <button
      type="button"
      onClick={handleCopy}
      title={`Click to copy ${label}: ${safeValue}`}
      aria-label={`Copy ${label}: ${safeValue}`}
      className={`group inline-flex items-center gap-1.5 rounded border border-border/50 bg-muted/20 px-2 py-0.5 font-mono text-xs text-muted-foreground hover:border-border hover:bg-muted/40 hover:text-foreground transition-colors ${className}`}
    >
      <span className="truncate">{displayValue || "—"}</span>
      {copied ? (
        <Check className="h-3 w-3 text-emerald-400 shrink-0" />
      ) : (
        <Copy className="h-3 w-3 opacity-50 group-hover:opacity-100 transition-opacity shrink-0" />
      )}
    </button>
  );
}
