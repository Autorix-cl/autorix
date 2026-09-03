"use client";

import * as React from "react";
import { Search, ShieldAlert, ShieldCheck, Trash2, Loader2, AlertCircle } from "lucide-react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { fetchAndParse } from "@/lib/api/schema";
import { introspectResponseSchema, type IntrospectResponse } from "@/lib/api/schemas/oauth2";
import { z } from "zod";

interface TokenInspectorProps {
  initialToken?: string;
}

export function TokenInspector({ initialToken = "" }: TokenInspectorProps = {}) {
  const [token, setToken] = React.useState(initialToken);
  const [tokenTypeHint, setTokenTypeHint] = React.useState("access_token");
  const [result, setResult] = React.useState<IntrospectResponse | null>(null);

  const queryClient = useQueryClient();


  const introspectMutation = useMutation({
    mutationFn: async () => {
      const res = await fetchAndParse<IntrospectResponse>(
        "/api/oauth2/tokens/introspect",
        introspectResponseSchema,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ token: token.trim(), token_type_hint: tokenTypeHint }),
        }
      );
      if (!res.ok) {
        throw new Error(res.error.message || "Failed to introspect token");
      }
      return res.data;
    },
    onSuccess: (data) => {
      setResult(data);
      queryClient.invalidateQueries({ queryKey: ["oauth2-tokens"] });
    },
  });

  const revokeMutation = useMutation({
    mutationFn: async () => {
      const res = await fetchAndParse(
        "/api/oauth2/tokens/revoke",
        z.unknown(),
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ token: token.trim(), token_type_hint: tokenTypeHint }),
        }
      );
      if (!res.ok) {
        throw new Error(res.error.message || "Failed to revoke token");
      }
      return res.data;
    },
    onSuccess: () => {
      if (result) {
        setResult({ ...result, active: false });
      }
      queryClient.invalidateQueries({ queryKey: ["oauth2-tokens"] });
    },
  });


  const handleRevoke = () => {
    if (!token.trim()) return;
    revokeMutation.mutate();
  };


  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle className="text-base font-semibold flex items-center gap-2">
            <Search className="w-4 h-4 text-primary" />
            RFC 7662 Token Introspection & Revocation
          </CardTitle>
          <CardDescription>
            Inspect opaque tokens or signed JWTs against Janus to verify signatures, expiration, and active grant status.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="token-input">Token String</Label>
              <Textarea
                id="token-input"
                placeholder="Paste Bearer JWT or opaque token here..."
                rows={4}
                value={token}
                onChange={(e) => setToken(e.target.value)}
                className="font-mono text-xs"
              />
            </div>

            <div className="flex flex-col sm:flex-row gap-4 items-start sm:items-center justify-between">
              <div className="flex items-center gap-2">
                <Label htmlFor="token-type-hint" className="text-xs text-muted-foreground whitespace-nowrap">
                  Type Hint:
                </Label>
                <Select value={tokenTypeHint} onValueChange={setTokenTypeHint}>
                  <SelectTrigger id="token-type-hint" className="w-[160px] h-8 text-xs">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="access_token">Access Token</SelectItem>
                    <SelectItem value="refresh_token">Refresh Token</SelectItem>
                  </SelectContent>
                </Select>
              </div>

              <div className="flex items-center gap-2">
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={handleRevoke}
                  disabled={!token.trim() || revokeMutation.isPending}
                  className="text-destructive hover:bg-destructive/10"
                >
                  {revokeMutation.isPending ? (
                    <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                  ) : (
                    <Trash2 className="w-4 h-4 mr-2" />
                  )}
                  Revoke Token
                </Button>
                <Button
                  type="button"
                  size="sm"
                  data-testid="introspect-btn"
                  onClick={() => introspectMutation.mutate()}
                  disabled={introspectMutation.isPending}
                >
                  {introspectMutation.isPending ? (
                    <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                  ) : (
                    <Search className="w-4 h-4 mr-2" />
                  )}
                  Introspect Token
                </Button>
              </div>
            </div>
          </div>

          {revokeMutation.isSuccess && (
            <div className="mt-4 p-3 bg-rose-500/10 border border-rose-500/20 rounded-md text-rose-400 text-sm flex items-center gap-2">
              <ShieldAlert className="w-4 h-4 text-rose-500" />
              Token successfully revoked.
            </div>
          )}

          {introspectMutation.isError && (
            <div className="mt-4 p-3 bg-destructive/10 border border-destructive/20 rounded-md text-destructive text-sm flex items-center gap-2">
              <AlertCircle className="w-4 h-4" />
              {introspectMutation.error.message}
            </div>
          )}
        </CardContent>
      </Card>

      {result && (





        <Card className="border-t-2 border-t-primary">
          <CardHeader>
            <div className="flex items-center justify-between">
              <CardTitle className="text-base font-semibold flex items-center gap-2">
                {result.active ? (
                  <ShieldCheck className="w-5 h-5 text-emerald-500" />
                ) : (
                  <ShieldAlert className="w-5 h-5 text-rose-500" />
                )}
                Introspection Result
              </CardTitle>
              <Badge variant={result.active ? "default" : "destructive"}>
                {result.active ? "Active" : "Inactive / Revoked"}
              </Badge>
            </div>
          </CardHeader>
          <CardContent className="space-y-4">
            {result.active ? (
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4 text-sm">
                <div className="p-3 rounded bg-muted/40 border">
                  <span className="text-xs text-muted-foreground block">Subject (sub)</span>
                  <span className="font-mono font-medium">{result.sub || "—"}</span>
                </div>
                <div className="p-3 rounded bg-muted/40 border">
                  <span className="text-xs text-muted-foreground block">Client ID (aud/client_id)</span>
                  <span className="font-mono font-medium">{result.client_id || "—"}</span>
                </div>
                <div className="p-3 rounded bg-muted/40 border">
                  <span className="text-xs text-muted-foreground block">Expires At (exp)</span>
                  <span className="font-mono font-medium">
                    {result.exp ? new Date(result.exp * 1000).toLocaleString() : "—"}
                  </span>
                </div>
                <div className="p-3 rounded bg-muted/40 border">
                  <span className="text-xs text-muted-foreground block">Token Type</span>
                  <span className="font-mono font-medium">{result.token_type || "Bearer"}</span>
                </div>
                <div className="p-3 rounded bg-muted/40 border md:col-span-2">
                  <span className="text-xs text-muted-foreground block mb-1">Scopes</span>
                  <div className="flex flex-wrap gap-1.5">
                    {result.scope ? (
                      result.scope.split(" ").map((s) => (
                        <Badge key={s} variant="secondary" className="font-mono text-xs">
                          {s}
                        </Badge>
                      ))
                    ) : (
                      <span className="text-muted-foreground text-xs">None</span>
                    )}
                  </div>
                </div>
              </div>
            ) : (
              <p className="text-sm text-muted-foreground">
                This token is either expired, revoked, or has invalid cryptographic signatures.
              </p>
            )}

            <div className="mt-4">
              <span className="text-xs text-muted-foreground block mb-1">Raw Payload</span>
              <pre className="font-mono text-xs p-3 bg-muted/40 rounded border overflow-x-auto">
                {JSON.stringify(result, null, 2)}
              </pre>
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
}

