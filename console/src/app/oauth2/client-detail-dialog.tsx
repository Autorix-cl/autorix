"use client";

import * as React from "react";
import { KeyRound, RotateCcw, Copy, Check, AlertTriangle, Trash2, Loader2, ShieldAlert } from "lucide-react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { fetchAndParse } from "@/lib/api/schema";
import {
  rotateSecretResponseSchema,
  type OAuth2Client,
  type RotateSecretResponse,
} from "@/lib/api/schemas/oauth2";
import { z } from "zod";

interface ClientDetailDialogProps {
  client: OAuth2Client | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onDeleted?: () => void;
}

export function ClientDetailDialog({
  client,
  open,
  onOpenChange,
  onDeleted,
}: ClientDetailDialogProps) {
  const queryClient = useQueryClient();
  const [overlapSeconds, setOverlapSeconds] = React.useState("86400"); // 24 hours default
  const [rotatedSecret, setRotatedSecret] = React.useState<string | null>(null);
  const [copied, setCopied] = React.useState(false);

  // Reset rotated secret on close or change
  React.useEffect(() => {
    if (!open) {
      setRotatedSecret(null);
      setCopied(false);
    }
  }, [open]);

  const rotateSecretMutation = useMutation({
    mutationFn: async () => {
      if (!client) throw new Error("No client selected");
      const res = await fetchAndParse<RotateSecretResponse>(
        `/api/oauth2/clients/${encodeURIComponent(client.client_id)}/rotate-secret`,
        rotateSecretResponseSchema,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ overlap_seconds: parseInt(overlapSeconds, 10) }),
        }
      );
      if (!res.ok) {
        throw new Error(res.error.message || "Failed to rotate secret");
      }
      return res.data;
    },
    onSuccess: (data) => {
      setRotatedSecret(data.client_secret);
      queryClient.invalidateQueries({ queryKey: ["oauth2-clients"] });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: async () => {
      if (!client) throw new Error("No client selected");
      const res = await fetchAndParse(
        `/api/oauth2/clients/${encodeURIComponent(client.client_id)}`,
        z.unknown(),
        { method: "DELETE" }
      );
      if (!res.ok) {
        throw new Error(res.error.message || "Failed to delete client");
      }
      return res.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["oauth2-clients"] });
      onOpenChange(false);
      onDeleted?.();
    },
  });

  const handleCopy = () => {
    if (rotatedSecret) {
      navigator.clipboard.writeText(rotatedSecret);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  };

  if (!client) return null;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <div className="flex items-center justify-between">
            <DialogTitle className="text-lg font-bold flex items-center gap-2">
              <KeyRound className="w-5 h-5 text-primary" />
              {client.client_name || client.client_id}
            </DialogTitle>
            <Badge variant={client.is_public ? "secondary" : "default"}>
              {client.is_public ? "Public Client" : "Confidential"}
            </Badge>
          </div>
          <DialogDescription className="font-mono text-xs text-muted-foreground">
            {client.client_id}
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-2 text-sm">
          {client.previous_secret_expires_at && (
            <div className="p-3 bg-amber-500/10 border border-amber-500/20 text-amber-400 rounded-md text-xs flex items-start gap-2">
              <AlertTriangle className="w-4 h-4 mt-0.5 shrink-0" />
              <div>
                <strong>Secret Rollover Active:</strong> The previous secret remains valid until{" "}
                {new Date(client.previous_secret_expires_at).toLocaleString()}. Update your client application before expiry.
              </div>
            </div>
          )}

          {rotatedSecret && (
            <div className="p-4 bg-emerald-500/10 border border-emerald-500/20 rounded-md space-y-2">
              <div className="flex items-center gap-2 text-emerald-400 font-medium text-xs">
                <ShieldAlert className="w-4 h-4" />
                New Secret Generated — Store it now!
              </div>
              <div className="flex items-center gap-2 bg-background border p-2 rounded">
                <code className="font-mono text-xs flex-1 break-all select-all">
                  {rotatedSecret}
                </code>
                <Button size="icon" variant="ghost" aria-label="Copy to clipboard" className="h-7 w-7" onClick={handleCopy}>
                  {copied ? <Check className="w-3.5 h-3.5 text-emerald-400" /> : <Copy className="w-3.5 h-3.5" />}
                </Button>
              </div>
              <p className="text-xs text-muted-foreground">
                This secret will not be displayed again.
              </p>
            </div>
          )}

          <div className="grid grid-cols-2 gap-2 text-xs">
            <div className="p-2.5 rounded bg-muted/40 border">
              <span className="text-muted-foreground block mb-1">Grant Types</span>
              <div className="flex flex-wrap gap-1">
                {client.grant_types?.map((g) => (
                  <Badge key={g} variant="outline" className="text-[10px]">
                    {g}
                  </Badge>
                ))}
              </div>
            </div>
            <div className="p-2.5 rounded bg-muted/40 border">
              <span className="text-muted-foreground block mb-1">Scopes</span>
              <div className="flex flex-wrap gap-1">
                {client.scopes?.map((s) => (
                  <Badge key={s} variant="outline" className="text-[10px]">
                    {s}
                  </Badge>
                ))}
              </div>
            </div>
          </div>

          <div className="p-2.5 rounded bg-muted/40 border text-xs">
            <span className="text-muted-foreground block mb-1">Redirect URIs</span>
            {client.redirect_uris && client.redirect_uris.length > 0 ? (
              <ul className="list-disc list-inside font-mono text-[11px] space-y-0.5">
                {client.redirect_uris.map((uri) => (
                  <li key={uri}>{uri}</li>
                ))}
              </ul>
            ) : (
              <span className="text-muted-foreground text-xs">—</span>
            )}
          </div>

          {!client.is_public && (
            <div className="p-3 border rounded-md space-y-3 bg-muted/10">
              <div className="flex items-center justify-between">
                <div className="space-y-0.5">
                  <h4 className="text-xs font-semibold">Zero-Downtime Secret Rotation</h4>
                  <p className="text-[11px] text-muted-foreground">
                    Rotate secret with a dual-secret grace overlap window.
                  </p>
                </div>
              </div>

              <div className="flex items-center gap-2">
                <Label htmlFor="overlap-duration" className="text-xs text-muted-foreground whitespace-nowrap">
                  Overlap Window:
                </Label>
                <Select value={overlapSeconds} onValueChange={setOverlapSeconds}>
                  <SelectTrigger id="overlap-duration" className="h-8 text-xs flex-1">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="3600">1 hour</SelectItem>
                    <SelectItem value="86400">24 hours (1 day)</SelectItem>
                    <SelectItem value="604800">7 days</SelectItem>
                  </SelectContent>
                </Select>
                <Button
                  size="sm"
                  onClick={() => rotateSecretMutation.mutate()}
                  disabled={rotateSecretMutation.isPending}
                >
                  {rotateSecretMutation.isPending ? (
                    <Loader2 className="w-3.5 h-3.5 mr-1 animate-spin" />
                  ) : (
                    <RotateCcw className="w-3.5 h-3.5 mr-1" />
                  )}
                  Rotate Secret
                </Button>
              </div>
            </div>
          )}
        </div>

        <DialogFooter className="flex items-center justify-between border-t pt-3">
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={() => {
              if (confirm(`Delete client "${client.client_id}" permanently?`)) {
                deleteMutation.mutate();
              }
            }}
            className="text-destructive hover:bg-destructive/10"
            disabled={deleteMutation.isPending}
          >
            {deleteMutation.isPending ? (
              <Loader2 className="w-4 h-4 mr-2 animate-spin" />
            ) : (
              <Trash2 className="w-4 h-4 mr-2" />
            )}
            Delete Client
          </Button>

          <Button type="button" variant="outline" size="sm" onClick={() => onOpenChange(false)}>
            Close
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
