"use client";

import * as React from "react";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { RotateCw, Check, Copy, AlertTriangle, Loader2 } from "lucide-react";
import { toast } from "sonner";
import type { RotateKeyResponse } from "@/lib/api/schemas/vulcan";

interface KeyRotateDialogProps {
  keyId: string | null;
  keyName: string | null;
  isOpen: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess?: () => void;
}

export function KeyRotateDialog({
  keyId,
  keyName,
  isOpen,
  onOpenChange,
  onSuccess,
}: KeyRotateDialogProps) {
  const [gracePeriod, setGracePeriod] = React.useState("24h");
  const [isRotating, setIsRotating] = React.useState(false);
  const [rotatedData, setRotatedData] = React.useState<RotateKeyResponse | null>(null);
  const [copied, setCopied] = React.useState(false);

  React.useEffect(() => {
    if (isOpen) {
      setRotatedData(null);
      setCopied(false);
    }
  }, [isOpen]);

  const handleRotate = async () => {
    if (!keyId) return;

    setIsRotating(true);
    try {
      const res = await fetch(`/api/vulcan/keys/${keyId}/rotate`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ grace_period: gracePeriod }),
      });

      if (!res.ok) {
        const err = await res.json();
        throw new Error(err.error || "Rotation failed");
      }

      const data: RotateKeyResponse = await res.json();
      setRotatedData(data);
      toast.success("API key rotated successfully. Copy the new token now.");
      onSuccess?.();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Key rotation failed");
    } finally {
      setIsRotating(false);
    }
  };

  const copyToken = () => {
    if (!rotatedData?.raw_token) return;
    navigator.clipboard.writeText(rotatedData.raw_token);
    setCopied(true);
    toast.success("Token copied to clipboard");
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <Dialog open={isOpen} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <RotateCw className="w-5 h-5 text-primary" />
            Rotate API Key · {keyName || keyId}
          </DialogTitle>
          <DialogDescription className="text-xs">
            Generate a new cryptographically secure token while maintaining an overlap grace period for zero-downtime migration.
          </DialogDescription>
        </DialogHeader>

        {!rotatedData ? (
          <div className="space-y-4 pt-2 text-xs">
            <div className="p-3 rounded-lg border bg-amber-500/10 border-amber-500/30 text-amber-500 flex items-start gap-2">
              <AlertTriangle className="w-4 h-4 shrink-0 mt-0.5" />
              <div>
                <p className="font-semibold">Zero-Downtime Overlap</p>
                <p className="text-[11px] opacity-90 mt-0.5">
                  The previous root key will remain valid during the grace period so you can deploy the new token without service disruption.
                </p>
              </div>
            </div>

            <div className="space-y-1.5">
              <Label className="text-xs">Grace Period Duration</Label>
              <Select value={gracePeriod} onValueChange={setGracePeriod}>
                <SelectTrigger className="h-8 text-xs">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="1h">1 hour</SelectItem>
                  <SelectItem value="24h">24 hours (Recommended)</SelectItem>
                  <SelectItem value="72h">72 hours (3 days)</SelectItem>
                  <SelectItem value="168h">168 hours (7 days)</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <DialogFooter className="pt-2">
              <Button variant="ghost" size="sm" onClick={() => onOpenChange(false)}>
                Cancel
              </Button>
              <Button size="sm" onClick={handleRotate} disabled={isRotating} className="gap-1.5">
                {isRotating ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : <RotateCw className="w-3.5 h-3.5" />}
                Confirm Rotation
              </Button>
            </DialogFooter>
          </div>
        ) : (
          <div className="space-y-4 pt-2 text-xs">
            <div className="p-3 rounded-lg border bg-emerald-500/10 border-emerald-500/30 space-y-2">
              <div className="flex items-center justify-between font-semibold text-emerald-500">
                <span>New Token Issued</span>
                <span className="text-[10px] text-muted-foreground font-mono">One-Time Reveal</span>
              </div>
              <p className="text-[11px] text-muted-foreground">
                Copy this token now. It will not be shown again.
              </p>
              <div className="flex items-center gap-2">
                <input
                  type="text"
                  readOnly
                  value={rotatedData.raw_token}
                  className="w-full text-xs font-mono p-2 rounded border bg-background text-foreground"
                />
                <Button size="sm" variant="outline" onClick={copyToken} className="shrink-0 h-8 gap-1">
                  {copied ? <Check className="w-3.5 h-3.5 text-emerald-500" /> : <Copy className="w-3.5 h-3.5" />}
                  {copied ? "Copied" : "Copy"}
                </Button>
              </div>
            </div>

            <DialogFooter>
              <Button size="sm" onClick={() => onOpenChange(false)} className="w-full">
                I have securely saved this token
              </Button>
            </DialogFooter>
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}
