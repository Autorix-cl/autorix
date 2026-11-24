"use client";

import * as React from "react";
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetDescription } from "@/components/ui/sheet";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { ListFilter, Plus, Trash2, Loader2, ShieldCheck } from "lucide-react";
import { toast } from "sonner";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { Can } from "@/lib/auth/use-permission";
import type { Scope } from "@/lib/api/schemas/vulcan";

interface ScopeCatalogSheetProps {
  isOpen: boolean;
  onOpenChange: (open: boolean) => void;
}

function readCsrfCookie(): string | undefined {
  if (typeof document === "undefined") return undefined;
  const match = document.cookie.match(/(?:^|;\s*)autorix_csrf=([^;]+)/);
  return match ? decodeURIComponent(match[1]) : undefined;
}

export function ScopeCatalogSheet({ isOpen, onOpenChange }: ScopeCatalogSheetProps) {
  const [scopes, setScopes] = React.useState<Scope[]>([]);
  const [isLoading, setIsLoading] = React.useState(false);
  const [isCreating, setIsCreating] = React.useState(false);
  const [scopeToDelete, setScopeToDelete] = React.useState<string | null>(null);
  const [isDeleting, setIsDeleting] = React.useState(false);

  const [newScopeName, setNewScopeName] = React.useState("");
  const [newScopeDesc, setNewScopeDesc] = React.useState("");

  const loadScopes = React.useCallback(async () => {
    setIsLoading(true);
    try {
      const res = await fetch("/api/vulcan/scopes");
      if (!res.ok) throw new Error("Failed to load scopes");
      const data = await res.json();
      setScopes(Array.isArray(data) ? data : data.data || []);
    } catch {
      toast.error("Failed to load scope catalogue");
    } finally {
      setIsLoading(false);
    }
  }, []);

  React.useEffect(() => {
    if (isOpen) {
      loadScopes();
    }
  }, [isOpen, loadScopes]);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newScopeName.trim()) return;

    setIsCreating(true);
    try {
      const csrf = readCsrfCookie();
      const res = await fetch("/api/vulcan/scopes", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          ...(csrf ? { "X-CSRF-Token": csrf } : {}),
        },
        body: JSON.stringify({
          name: newScopeName.trim(),
          description: newScopeDesc.trim(),
        }),
      });
      if (!res.ok) throw new Error("Failed to create scope");
      toast.success(`Scope '${newScopeName.trim()}' added`);
      setNewScopeName("");
      setNewScopeDesc("");
      loadScopes();
    } catch {
      toast.error("Failed to register scope");
    } finally {
      setIsCreating(false);
    }
  };

  const handleConfirmDelete = async () => {
    if (!scopeToDelete) return;
    setIsDeleting(true);
    try {
      const csrf = readCsrfCookie();
      const res = await fetch(`/api/vulcan/scopes/${encodeURIComponent(scopeToDelete)}`, {
        method: "DELETE",
        headers: csrf ? { "X-CSRF-Token": csrf } : undefined,
      });
      if (!res.ok) throw new Error("Failed to delete scope");
      toast.success(`Scope '${scopeToDelete}' deleted`);
      setScopeToDelete(null);
      loadScopes();
    } catch {
      toast.error("Failed to remove scope");
    } finally {
      setIsDeleting(false);
    }
  };

  return (
    <>
    <Sheet open={isOpen} onOpenChange={onOpenChange}>
      <SheetContent className="sm:max-w-md flex flex-col">
        <SheetHeader>
          <SheetTitle className="flex items-center gap-2">
            <ListFilter className="w-5 h-5 text-primary" aria-hidden="true" />
            Scope Catalogue
          </SheetTitle>
          <SheetDescription className="text-xs">
            Manage authorized machine scopes available for key issuance and attenuation.
          </SheetDescription>
        </SheetHeader>

        <div className="flex-1 overflow-y-auto space-y-4 pt-4 pr-1">
          {/* Create Scope Form */}
          <Can perform="api-keys:write">
            <form onSubmit={handleCreate} className="p-3 border rounded-lg bg-card/60 space-y-3 text-xs">
              <div className="font-semibold text-foreground flex items-center gap-1.5">
                <Plus className="w-3.5 h-3.5 text-primary" aria-hidden="true" />
                Register New Scope
              </div>

              <div className="space-y-2">
                <div className="space-y-1">
                  <Label htmlFor="vulcan-scope-name" className="text-[11px]">Scope Identifier</Label>
                  <Input
                    id="vulcan-scope-name"
                    placeholder="e.g., billing:invoices:read"
                    value={newScopeName}
                    onChange={(e) => setNewScopeName(e.target.value)}
                    className="h-7 text-xs font-mono"
                    required
                  />
                </div>

                <div className="space-y-1">
                  <Label htmlFor="vulcan-scope-desc" className="text-[11px]">Description</Label>
                  <Input
                    id="vulcan-scope-desc"
                    placeholder="e.g., Allows reading customer invoices"
                    value={newScopeDesc}
                    onChange={(e) => setNewScopeDesc(e.target.value)}
                    className="h-7 text-xs"
                  />
                </div>
              </div>

              <Button type="submit" size="sm" disabled={isCreating || !newScopeName.trim()} className="h-7 text-xs w-full">
                {isCreating ? <Loader2 className="w-3.5 h-3.5 animate-spin mr-1" aria-hidden="true" /> : null}
                Add Scope
              </Button>
            </form>
          </Can>

          {/* Scopes List */}
          <div className="space-y-2">
            <div className="flex items-center justify-between text-xs font-semibold text-muted-foreground">
              <span>Managed Scopes ({scopes.length})</span>
            </div>

            {isLoading ? (
              <div className="flex items-center justify-center py-6 text-xs text-muted-foreground gap-2">
                <Loader2 className="w-4 h-4 animate-spin" aria-hidden="true" />
                Loading catalogue...
              </div>
            ) : scopes.length === 0 ? (
              <div className="text-center py-6 text-xs text-muted-foreground border rounded-lg border-dashed">
                No custom scopes registered yet.
              </div>
            ) : (
              <div className="space-y-1.5">
                {scopes.map((s) => (
                  <div
                    key={s.name}
                    className="p-2.5 rounded-lg border bg-card/60 flex items-center justify-between gap-2 text-xs"
                  >
                    <div className="space-y-0.5 truncate max-w-[280px]">
                      <div className="flex items-center gap-1.5">
                        <ShieldCheck className="w-3.5 h-3.5 text-primary shrink-0" aria-hidden="true" />
                        <span className="font-mono font-semibold text-foreground text-[11px] truncate">
                          {s.name}
                        </span>
                      </div>
                      {s.description && (
                        <p className="text-[11px] text-muted-foreground truncate pl-5">
                          {s.description}
                        </p>
                      )}
                    </div>

                    <Can perform="api-keys:write">
                      <Button
                        type="button"
                        variant="ghost"
                        size="icon"
                        aria-label={`Delete scope ${s.name}`}
                        className="h-6 w-6 text-muted-foreground hover:text-destructive shrink-0"
                        onClick={() => setScopeToDelete(s.name)}
                      >
                        <Trash2 className="w-3.5 h-3.5" aria-hidden="true" />
                      </Button>
                    </Can>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      </SheetContent>
    </Sheet>

    <ConfirmDialog
      open={Boolean(scopeToDelete)}
      onOpenChange={(open) => !open && setScopeToDelete(null)}
      title="Delete Scope"
      description={`Remove scope "${scopeToDelete ?? ""}" from the Vulcan catalogue?`}
      confirmLabel="Delete Scope"
      isLoading={isDeleting}
      onConfirm={handleConfirmDelete}
    />
    </>
  );
}
