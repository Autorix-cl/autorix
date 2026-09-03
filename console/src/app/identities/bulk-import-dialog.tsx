"use client";

import * as React from "react";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Loader2, CheckCircle2, AlertCircle, UploadCloud, FileSpreadsheet } from "lucide-react";
import { toast } from "sonner";
import type { BulkImportResult } from "@/lib/api/schemas/identity";

interface BulkImportDialogProps {
  isOpen: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess?: () => void;
}

export function BulkImportDialog({ isOpen, onOpenChange, onSuccess }: BulkImportDialogProps) {
  const [csvText, setCsvText] = React.useState("");
  const [isValidating, setIsValidating] = React.useState(false);
  const [isCommitting, setIsCommitting] = React.useState(false);
  const [validationResult, setValidationResult] = React.useState<BulkImportResult | null>(null);

  const parseCsvToRows = (text: string) => {
    const lines = text.trim().split("\n").filter(Boolean);
    const rows = [];
    for (const line of lines) {
      // ignore header if present
      if (line.toLowerCase().startsWith("email")) continue;
      const parts = line.split(",").map((s) => s.trim());
      if (parts[0]) {
        rows.push({
          email: parts[0],
          firstName: parts[1] || "",
          lastName: parts[2] || "",
        });
      }
    }
    return rows;
  };

  const handleDryRun = async () => {
    const rows = parseCsvToRows(csvText);
    if (rows.length === 0) {
      toast.error("Please enter at least one identity row (email,firstName,lastName)");
      return;
    }

    setIsValidating(true);
    setValidationResult(null);
    try {
      const res = await fetch("/api/identities/bulk-import", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ rows, dry_run: true }),
      });
      if (!res.ok) throw new Error("Validation request failed");
      const data: BulkImportResult = await res.json();
      setValidationResult(data);
      if (data.failed === 0) {
        toast.success(`Dry-run passed: ${data.succeeded} rows ready for import.`);
      } else {
        toast.warning(`Dry-run finished with ${data.failed} invalid rows.`);
      }
    } catch {
      toast.error("Failed to run dry-run validation");
    } finally {
      setIsValidating(false);
    }
  };

  const handleCommit = async () => {
    const rows = parseCsvToRows(csvText);
    if (rows.length === 0) return;

    setIsCommitting(true);
    try {
      const res = await fetch("/api/identities/bulk-import", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ rows, dry_run: false }),
      });
      if (!res.ok) throw new Error("Commit import failed");
      const data: BulkImportResult = await res.json();
      toast.success(`Import completed: ${data.succeeded} identities created.`);
      onSuccess?.();
      onOpenChange(false);
      setCsvText("");
      setValidationResult(null);
    } catch {
      toast.error("Failed to commit identities import");
    } finally {
      setIsCommitting(false);
    }
  };

  return (
    <Dialog open={isOpen} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-xl">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <UploadCloud className="w-5 h-5 text-primary" />
            Bulk Import Identities
          </DialogTitle>
          <DialogDescription className="text-xs">
            Import users via CSV with pre-commit dry-run validation and automatic schema assignment.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 pt-2">
          <div className="space-y-1.5">
            <div className="flex items-center justify-between text-xs text-muted-foreground font-mono">
              <span>Format: email,firstName,lastName</span>
              <button
                type="button"
                onClick={() =>
                  setCsvText(
                    "email,firstName,lastName\nalice@autorix.io,Alice,Smith\nbob@autorix.io,Bob,Jones\ncarol@autorix.io,Carol,White"
                  )
                }
                className="underline hover:text-foreground text-[11px]"
              >
                Insert sample
              </button>
            </div>
            <textarea
              value={csvText}
              onChange={(e) => {
                setCsvText(e.target.value);
                setValidationResult(null);
              }}
              placeholder="user@example.com,John,Doe"
              rows={6}
              className="w-full text-xs font-mono p-3 rounded-md border bg-muted/20"
            />
          </div>

          {/* Validation Results Drawer */}
          {validationResult && (
            <div className="p-3 rounded-md border bg-card/60 space-y-2 text-xs">
              <div className="flex items-center justify-between font-medium">
                <span className="flex items-center gap-1.5">
                  <FileSpreadsheet className="w-4 h-4 text-primary" />
                  Validation Summary
                </span>
                <div className="flex items-center gap-2">
                  <Badge variant="outline" className="text-emerald-500 border-emerald-500/30">
                    <CheckCircle2 className="w-3 h-3 mr-1" /> {validationResult.succeeded} Valid
                  </Badge>
                  {validationResult.failed > 0 && (
                    <Badge variant="destructive">
                      <AlertCircle className="w-3 h-3 mr-1" /> {validationResult.failed} Errors
                    </Badge>
                  )}
                </div>
              </div>

              {validationResult.errors.length > 0 && (
                <div className="max-h-24 overflow-y-auto space-y-1 pt-1 border-t text-[11px] text-destructive">
                  {validationResult.errors.map((err, idx) => (
                    <div key={idx} className="flex items-center gap-2">
                      <span className="font-mono font-semibold">Row {err.row}:</span>
                      <span>{err.error}</span>
                      {err.email && <span className="opacity-70">({err.email})</span>}
                    </div>
                  ))}
                </div>
              )}
            </div>
          )}

          <div className="flex justify-between items-center pt-2">
            <Button
              variant="outline"
              size="sm"
              onClick={handleDryRun}
              disabled={isValidating || isCommitting || !csvText.trim()}
            >
              {isValidating && <Loader2 className="w-3.5 h-3.5 mr-1.5 animate-spin" />}
              Dry-Run Validation
            </Button>

            <div className="flex items-center gap-2">
              <Button variant="ghost" size="sm" onClick={() => onOpenChange(false)}>
                Cancel
              </Button>
              <Button
                size="sm"
                onClick={handleCommit}
                disabled={isCommitting || !csvText.trim() || (validationResult !== null && validationResult.succeeded === 0)}
              >
                {isCommitting && <Loader2 className="w-3.5 h-3.5 mr-1.5 animate-spin" />}
                Commit Import
              </Button>
            </div>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
