"use client";

import * as React from "react";
import { Key, RotateCcw, CheckCircle2, AlertCircle, Loader2, RefreshCw } from "lucide-react";
import { useQueryClient, useMutation } from "@tanstack/react-query";
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { CodeBlock } from "@/components/ui/code-block";
import { useApiQuery } from "@/lib/query/use-api-query";
import { fetchAndParse } from "@/lib/api/schema";
import {
  jwksSchema,
  rotateKeysResponseSchema,
  type JWKS,
  type RotateKeysResponse,
} from "@/lib/api/schemas/oauth2";
import { LoadingState } from "@/components/state/loading-state";

export function KeyManager() {
  const queryClient = useQueryClient();
  const [successInfo, setSuccessInfo] = React.useState<RotateKeysResponse | null>(null);

  const {
    data: jwks,
    isLoading,
    isError,
    refetch,
  } = useApiQuery(["oauth2-jwks"], () =>
    fetchAndParse<JWKS>("/api/oauth2/jwks", jwksSchema)
  );

  const rotateMutation = useMutation({
    mutationFn: async () => {
      setSuccessInfo(null);
      const res = await fetchAndParse<RotateKeysResponse>(
        "/api/oauth2/keys/rotate",
        rotateKeysResponseSchema,
        { method: "POST" }
      );
      if (!res.ok) {
        throw new Error(res.error.message || "Failed to rotate keys");
      }
      return res.data;
    },
    onSuccess: (data) => {
      setSuccessInfo(data);
      queryClient.invalidateQueries({ queryKey: ["oauth2-jwks"] });
    },
  });

  const keys = Array.isArray(jwks?.keys) ? jwks.keys : [];


  return (
    <div className="space-y-6">
      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <div>
            <CardTitle className="text-base font-semibold flex items-center gap-2">
              <Key className="w-4 h-4 text-primary" />
              JWKS Keystore & Key Rollover
            </CardTitle>
            <CardDescription>
              Public key set for token verification and asymmetric signing key rotation with grace overlap.
            </CardDescription>
          </div>
          <div className="flex items-center gap-2">
            <Button variant="outline" size="sm" onClick={() => refetch()}>
              <RefreshCw className="w-4 h-4" />
            </Button>
            <Button
              size="sm"
              onClick={() => rotateMutation.mutate()}
              disabled={rotateMutation.isPending}
            >
              {rotateMutation.isPending ? (
                <Loader2 className="w-4 h-4 mr-2 animate-spin" />
              ) : (
                <RotateCcw className="w-4 h-4 mr-2" />
              )}
              Rotate Signing Key
            </Button>
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          {successInfo && (
            <div className="p-3 rounded-md bg-emerald-500/10 border border-emerald-500/20 text-emerald-400 text-sm flex items-center gap-2">
              <CheckCircle2 className="w-4 h-4 text-emerald-500" />
              <span>
                New primary signing key generated: <strong>{successInfo.new_kid || "Success"}</strong> (Total active keys: {successInfo.active_keys_count ?? keys.length})
              </span>
            </div>
          )}

          {rotateMutation.isError && (
            <div className="p-3 rounded-md bg-destructive/10 border border-destructive/20 text-destructive text-sm flex items-center gap-2">
              <AlertCircle className="w-4 h-4" />
              {rotateMutation.error.message}
            </div>
          )}

          {isLoading && <LoadingState label="Loading JWKS keystore..." />}

          {isError && (
            <div className="p-4 rounded border border-destructive/20 bg-destructive/10 text-destructive text-sm">
              Failed to load JWKS keys from Janus.
            </div>
          )}

          {!isLoading && !isError && (
            <div className="space-y-4">
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                {keys.map((k, idx) => (
                  <Card key={k.kid || idx} className="bg-muted/30 border">
                    <CardHeader className="p-4 pb-2">
                      <div className="flex items-center justify-between">
                        <Badge variant={idx === 0 ? "default" : "secondary"}>
                          {idx === 0 ? "Primary / Active" : "Verification / Rollover"}
                        </Badge>
                        <span className="text-xs font-mono text-muted-foreground">{k.alg}</span>
                      </div>
                      <CardTitle className="text-xs font-mono break-all mt-2">
                        {k.kid}
                      </CardTitle>
                    </CardHeader>
                    <CardContent className="p-4 pt-2 text-xs space-y-1">
                      <div className="flex justify-between text-muted-foreground">
                        <span>Type:</span>
                        <span className="font-mono">{k.kty}</span>
                      </div>
                      <div className="flex justify-between text-muted-foreground">
                        <span>Usage:</span>
                        <span className="font-mono">{k.use === "sig" ? "Signature" : k.use}</span>
                      </div>
                    </CardContent>
                  </Card>
                ))}
              </div>

              <div className="mt-4">
                <span className="text-xs text-muted-foreground block mb-1">Raw JWKS Document</span>
                <CodeBlock language="json" code={JSON.stringify(jwks, null, 2)} />
              </div>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
