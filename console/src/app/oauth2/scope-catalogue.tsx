"use client";

import * as React from "react";
import { Plus, Trash2, Tag, Loader2, RefreshCw } from "lucide-react";
import { useQueryClient } from "@tanstack/react-query";
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from "@/components/ui/table";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog";
import { useApiQuery } from "@/lib/query/use-api-query";
import { fetchAndParse } from "@/lib/api/schema";
import {
  oauth2ScopeListSchema,
  oauth2ScopeSchema,
  type OAuth2Scope,
} from "@/lib/api/schemas/oauth2";
import { LoadingState } from "@/components/state/loading-state";
import { EmptyState } from "@/components/state/empty-state";
import { z } from "zod";

export function ScopeCatalogue() {
  const queryClient = useQueryClient();
  const [isCreateOpen, setIsCreateOpen] = React.useState(false);
  const [name, setName] = React.useState("");
  const [description, setDescription] = React.useState("");
  const [claimsStr, setClaimsStr] = React.useState("");
  const [isSubmitting, setIsSubmitting] = React.useState(false);
  const [errorMsg, setErrorMsg] = React.useState<string | null>(null);

  const {
    data: scopesRaw,
    isLoading,
    isError,
    refetch,
  } = useApiQuery(["oauth2-scopes"], () =>
    fetchAndParse<OAuth2Scope[]>("/api/oauth2/scopes", oauth2ScopeListSchema)
  );

  const scopes = scopesRaw ?? [];

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim()) return;

    setIsSubmitting(true);
    setErrorMsg(null);

    const claims = claimsStr
      .split(/[,\s]+/)
      .map((c) => c.trim())
      .filter(Boolean);

    const res = await fetchAndParse(
      "/api/oauth2/scopes",
      oauth2ScopeSchema,
      {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          name: name.trim(),
          description: description.trim(),
          claims,
        }),
      }
    );

    setIsSubmitting(false);

    if (!res.ok) {
      setErrorMsg(res.error.message || "Failed to create scope");
      return;
    }

    setName("");
    setDescription("");
    setClaimsStr("");
    setIsCreateOpen(false);
    queryClient.invalidateQueries({ queryKey: ["oauth2-scopes"] });
  };

  const handleDelete = async (scopeName: string) => {
    if (!confirm(`Are you sure you want to delete scope "${scopeName}"?`)) return;

    const res = await fetchAndParse(
      `/api/oauth2/scopes/${encodeURIComponent(scopeName)}`,
      z.unknown(),
      { method: "DELETE" }
    );

    if (res.ok) {
      queryClient.invalidateQueries({ queryKey: ["oauth2-scopes"] });
    }
  };

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <div>
            <CardTitle className="text-base font-semibold flex items-center gap-2">
              <Tag className="w-4 h-4 text-primary" />
              Scope & Claims Catalogue
            </CardTitle>
            <CardDescription>
              Define authorized OAuth2 scopes and their mapped OpenID Connect claims for client issuance.
            </CardDescription>
          </div>
          <div className="flex items-center gap-2">
            <Button variant="outline" size="sm" onClick={() => refetch()}>
              <RefreshCw className="w-4 h-4" />
            </Button>
            <Button size="sm" onClick={() => setIsCreateOpen(true)}>
              <Plus className="w-4 h-4 mr-2" />
              New Scope
            </Button>
          </div>
        </CardHeader>
        <CardContent>
          {isLoading && <LoadingState label="Loading scopes catalogue..." />}

          {isError && (
            <div className="p-4 rounded border border-destructive/20 bg-destructive/10 text-destructive text-sm">
              Failed to load scope catalogue from Janus.
            </div>
          )}

          {!isLoading && !isError && scopes.length === 0 && (
            <EmptyState
              title="No scopes defined"
              description="Register scopes to specify granted privileges and identity claims."
              action={
                <Button size="sm" onClick={() => setIsCreateOpen(true)}>
                  <Plus className="w-4 h-4 mr-2" />
                  Add First Scope
                </Button>
              }
            />
          )}

          {!isLoading && scopes.length > 0 && (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-[180px]">Scope Name</TableHead>
                  <TableHead>Description</TableHead>
                  <TableHead>Mapped Claims</TableHead>
                  <TableHead className="w-[100px] text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {scopes.map((s) => (
                  <TableRow key={s.name}>
                    <TableCell className="font-mono font-medium text-sm">
                      {s.name}
                    </TableCell>
                    <TableCell className="text-sm text-muted-foreground">
                      {s.description || "—"}
                    </TableCell>
                    <TableCell>
                      <div className="flex flex-wrap gap-1">
                        {s.claims && s.claims.length > 0 ? (
                          s.claims.map((c) => (
                            <Badge key={c} variant="outline" className="text-xs font-mono">
                              {c}
                            </Badge>
                          ))
                        ) : (
                          <span className="text-xs text-muted-foreground">—</span>
                        )}
                      </div>
                    </TableCell>
                    <TableCell className="text-right">
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => handleDelete(s.name)}
                        className="text-muted-foreground hover:text-destructive h-8 w-8"
                      >
                        <Trash2 className="w-4 h-4" />
                      </Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      <Dialog open={isCreateOpen} onOpenChange={setIsCreateOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Register New Scope</DialogTitle>
            <DialogDescription>
              Create a managed scope in Janus to allow fine-grained permission requests by clients.
            </DialogDescription>
          </DialogHeader>

          <form onSubmit={handleCreate} className="space-y-4 py-2">
            {errorMsg && (
              <div className="p-3 bg-destructive/10 border border-destructive/20 text-destructive text-sm rounded-md">
                {errorMsg}
              </div>
            )}

            <div className="space-y-2">
              <Label htmlFor="scope-name">Scope Name</Label>
              <Input
                id="scope-name"
                placeholder="e.g. read:reports, billing:admin"
                value={name}
                onChange={(e) => setName(e.target.value)}
                required
                className="font-mono text-sm"
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="scope-desc">Description</Label>
              <Input
                id="scope-desc"
                placeholder="Explains what permissions this scope grants"
                value={description}
                onChange={(e) => setDescription(e.target.value)}
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="scope-claims">Mapped Claims (comma or space separated)</Label>
              <Input
                id="scope-claims"
                placeholder="e.g. org_id, department, tier"
                value={claimsStr}
                onChange={(e) => setClaimsStr(e.target.value)}
                className="font-mono text-sm"
              />
            </div>

            <DialogFooter className="pt-4">
              <Button
                type="button"
                variant="outline"
                onClick={() => setIsCreateOpen(false)}
                disabled={isSubmitting}
              >
                Cancel
              </Button>
              <Button type="submit" disabled={isSubmitting || !name.trim()}>
                {isSubmitting && <Loader2 className="w-4 h-4 mr-2 animate-spin" />}
                Create Scope
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </div>
  );
}
