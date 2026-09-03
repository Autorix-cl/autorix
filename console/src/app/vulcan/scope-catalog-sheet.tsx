"use client";

import * as React from "react";
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetDescription } from "@/components/ui/sheet";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { ListFilter, Plus, Trash2, Loader2, ShieldCheck } from "lucide-react";
import { toast } from "sonner";
import type { Scope } from "@/lib/api/schemas/vulcan";

interface ScopeCatalogSheetProps {
  isOpen: boolean;
  onOpenChange: (open: boolean) => void;
}

export function ScopeCatalogSheet({ isOpen, onOpenChange }: ScopeCatalogSheetProps) {
  const [scopes, setScopes] = React.useState<Scope[]>([]);
  const [isLoading, setIsLoading] = React.useState(false);
  const [isCreating, setIsCreating] = React.useState(false);

  const [newScopeName, setNewScopeName] = React.useState("");
  const [newScopeDesc, setNewScopeDesc] = React.useState("");

  const loadScopes = React.useCallback(async () => {
    setIsLoading(true);
    try {
      const res = await fetch("/api/vulcan/scopes");
      if (!res.ok) throw new Error("Failed to load scopes");
      const data = await res.json();
      setScopes(Array.isArray(data) ? data : []);
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
      const res = await fetch("/api/vulcan/scopes", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
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

  const handleDelete = async (scopeName: string) => {
    try {
      const res = await fetch(`/api/vulcan/scopes/${encodeURIComponent(scopeName)}`, {
        method: "DELETE",
      });
      if (!res.ok) throw new Error("Failed to delete scope");
      toast.success(`Scope '${scopeName}' deleted`);
      loadScopes();
    } catch {
      toast.error("Failed to remove scope");
    }
  };

  return (
    <Sheet open={isOpen} onOpenChange={onOpenChange}>
      <SheetContent className="sm:max-w-md flex flex-col">
        <SheetHeader>
          <SheetTitle className="flex items-center gap-2">
            <ListFilter className="w-5 h-5 text-primary" />
            Scope Catalogue
          </SheetTitle>
          <SheetDescription className="text-xs">
            Manage authorized machine scopes available for key issuance and attenuation.
          </SheetDescription>
        </SheetHeader>

        <div className="flex-1 overflow-y-auto space-y-4 pt-4 pr-1">
          {/* Create Scope Form */}
          <form onSubmit={handleCreate} className="p-3 border rounded-lg bg-card/60 space-y-3 text-xs">
            <div className="font-semibold text-foreground flex items-center gap-1.5">
              <Plus className="w-3.5 h-3.5 text-primary" />
              Register New Scope
            </div>

            <div className="space-y-2">
              <div className="space-y-1">
                <Label className="text-[11px]">Scope Identifier</Label>
                <Input
                  placeholder="e.g., billing:invoices:read"
                  value={newScopeName}
                  onChange={(e) => setNewScopeName(e.target.value)}
                  className="h-7 text-xs font-mono"
                  required
                />
              </div>

              <div className="space-y-1">
                <Label className="text-[11px]">Description</Label>
                <Input
                  placeholder="e.g., Allows reading customer invoices"
                  value={newScopeDesc}
                  onChange={(e) => setNewScopeDesc(e.target.value)}
                  className="h-7 text-xs"
                />
              </div>
            </div>

            <Button type="submit" size="sm" disabled={isCreating || !newScopeName.trim()} className="h-7 text-xs w-full">
              {isCreating ? <Loader2 className="w-3.5 h-3.5 animate-spin mr-1" /> : null}
              Add Scope
            </Button>
          </form>

          {/* Scopes List */}
          <div className="space-y-2">
            <div className="flex items-center justify-between text-xs font-semibold text-muted-foreground">
              <span>Managed Scopes ({scopes.length})</span>
            </div>

            {isLoading ? (
              <div className="flex items-center justify-center py-6 text-xs text-muted-foreground gap-2">
                <Loader2 className="w-4 h-4 animate-spin" />
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
                        <ShieldCheck className="w-3.5 h-3.5 text-primary shrink-0" />
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

                    <Button
                      variant="ghost"
                      size="icon"
                      className="h-6 w-6 text-muted-foreground hover:text-destructive shrink-0"
                      onClick={() => handleDelete(s.name)}
                    >
                      <Trash2 className="w-3.5 h-3.5" />
                    </Button>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      </SheetContent>
    </Sheet>
  );
}
