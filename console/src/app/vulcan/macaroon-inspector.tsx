"use client";

import * as React from "react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  SearchCode,
  CheckCircle2,
  XCircle,
  ShieldCheck,
  Play,
  Loader2,
  Lock,
} from "lucide-react";
import { toast } from "sonner";
import type { Macaroon, VerifyKeyResponse } from "@/lib/api/schemas/vulcan";

export function MacaroonInspector() {
  const [tokenInput, setTokenInput] = React.useState("");
  const [parsedMacaroon, setParsedMacaroon] = React.useState<Macaroon | null>(null);

  // Verification context
  const [testIp, setTestIp] = React.useState("192.168.1.50");
  const [testMethod, setTestMethod] = React.useState("GET");
  const [testPath, setTestPath] = React.useState("/api/v1/resources");

  const [isVerifying, setIsVerifying] = React.useState(false);
  const [verificationResult, setVerificationResult] = React.useState<VerifyKeyResponse | null>(null);

  const handleParse = () => {
    setVerificationResult(null);
    if (!tokenInput.trim()) {
      setParsedMacaroon(null);
      return;
    }

    try {
      // 1. Try parsing JSON directly
      const parsed = JSON.parse(tokenInput.trim());
      if (parsed.location && parsed.key_id && parsed.signature) {
        setParsedMacaroon(parsed);
        toast.success("Macaroon decoded successfully");
        return;
      }
    } catch {
      // Not raw JSON, try base64 decode
    }

    try {
      const decoded = atob(tokenInput.trim());
      const parsed = JSON.parse(decoded);
      if (parsed.location && parsed.key_id && parsed.signature) {
        setParsedMacaroon(parsed);
        toast.success("Base64 Macaroon decoded successfully");
        return;
      }
    } catch {
      // Not base64
    }

    toast.error("Could not parse token as valid JSON or Base64 Macaroon");
  };

  const handleVerify = async () => {
    if (!parsedMacaroon) {
      toast.error("Please parse a valid Macaroon first");
      return;
    }

    setIsVerifying(true);
    try {
      const res = await fetch("/api/vulcan/verify", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          macaroon: parsedMacaroon,
          context: {
            now: new Date().toISOString(),
            ip_address: testIp,
            method: testMethod,
            path: testPath,
          },
        }),
      });

      if (!res.ok) {
        const err = await res.json();
        throw new Error(err.error || "Verification failed");
      }

      const data: VerifyKeyResponse = await res.json();
      setVerificationResult(data);
      if (data.valid) {
        toast.success("Macaroon signature and caveats verified!");
      } else {
        toast.error(`Verification failed: ${data.error || "Caveat mismatch"}`);
      }
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Verification request failed");
    } finally {
      setIsVerifying(false);
    }
  };

  return (
    <Card className="border-border">
      <CardHeader>
        <CardTitle className="flex items-center gap-2 text-sm font-semibold">
          <SearchCode className="w-4 h-4 text-primary" />
          Macaroon & Key Inspector
        </CardTitle>
        <CardDescription className="text-xs">
          Inspect caveats, attenuation provenance, and test live verification against simulated request environments.
        </CardDescription>
      </CardHeader>

      <CardContent className="space-y-4">
        {/* Token Input Bar */}
        <div className="space-y-2 text-xs">
          <div className="flex items-center justify-between font-medium">
            <span>Paste Macaroon (JSON or Base64)</span>
            <button
              type="button"
              onClick={() => {
                const sample = JSON.stringify(
                  {
                    location: "https://api.autorix.io",
                    key_id: "00000000-0000-0000-0000-000000000001",
                    caveats: [
                      { predicate: "time_before = 2026-12-31T23:59:59Z" },
                      { predicate: "ip = 192.168.1.50" },
                      { predicate: "method = GET" },
                    ],
                    signature: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
                  },
                  null,
                  2
                );
                setTokenInput(sample);
              }}
              className="underline hover:text-foreground text-[11px]"
            >
              Load Sample Macaroon
            </button>
          </div>

          <div className="flex gap-2">
            <textarea
              value={tokenInput}
              onChange={(e) => setTokenInput(e.target.value)}
              placeholder='{"location":"https://api.autorix.io","key_id":"...","caveats":[...],"signature":"..."}'
              rows={3}
              className="w-full text-xs font-mono p-2 rounded-md border bg-muted/20"
            />
            <Button onClick={handleParse} size="sm" className="h-auto shrink-0 px-4 text-xs">
              Decode
            </Button>
          </div>
        </div>

        {/* Decoded Inspector & Verification Grid */}
        {parsedMacaroon && (
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 pt-2 border-t text-xs">
            {/* Left: Decoded Structure */}
            <div className="space-y-2.5 p-3 rounded-lg border bg-card/60">
              <div className="font-semibold text-foreground flex items-center gap-1.5">
                <Lock className="w-3.5 h-3.5 text-primary" />
                Decoded Structure
              </div>

              <div className="space-y-1.5 font-mono text-[11px]">
                <div>
                  <span className="text-muted-foreground">Location: </span>
                  <span className="text-foreground">{parsedMacaroon.location}</span>
                </div>
                <div>
                  <span className="text-muted-foreground">Key ID: </span>
                  <span className="text-foreground">{parsedMacaroon.key_id}</span>
                </div>
                <div>
                  <span className="text-muted-foreground">Signature: </span>
                  <span className="text-foreground truncate block">{parsedMacaroon.signature}</span>
                </div>
              </div>

              <div className="space-y-1 pt-1">
                <span className="font-semibold text-[11px] text-muted-foreground uppercase tracking-wider">
                  Caveats ({parsedMacaroon.caveats.length})
                </span>
                {parsedMacaroon.caveats.length === 0 ? (
                  <div className="text-muted-foreground text-[11px] italic">No caveats (Unrestricted root capability)</div>
                ) : (
                  <div className="space-y-1 max-h-28 overflow-y-auto pr-1">
                    {parsedMacaroon.caveats.map((c, i) => (
                      <div key={i} className="p-1.5 rounded border bg-muted/30 font-mono text-[11px]">
                        {c.predicate}
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </div>

            {/* Right: Verification Playground */}
            <div className="space-y-2.5 p-3 rounded-lg border bg-card/60 flex flex-col justify-between">
              <div className="space-y-2.5">
                <div className="font-semibold text-foreground flex items-center gap-1.5">
                  <ShieldCheck className="w-3.5 h-3.5 text-primary" />
                  Live Verification Context
                </div>

                <div className="grid grid-cols-3 gap-2">
                  <div className="space-y-1">
                    <Label className="text-[10px]">Client IP</Label>
                    <Input
                      value={testIp}
                      onChange={(e) => setTestIp(e.target.value)}
                      className="h-7 text-xs font-mono"
                    />
                  </div>
                  <div className="space-y-1">
                    <Label className="text-[10px]">Method</Label>
                    <Input
                      value={testMethod}
                      onChange={(e) => setTestMethod(e.target.value)}
                      className="h-7 text-xs font-mono"
                    />
                  </div>
                  <div className="space-y-1">
                    <Label className="text-[10px]">Path</Label>
                    <Input
                      value={testPath}
                      onChange={(e) => setTestPath(e.target.value)}
                      className="h-7 text-xs font-mono"
                    />
                  </div>
                </div>

                <Button onClick={handleVerify} disabled={isVerifying} size="sm" className="w-full h-7 text-xs gap-1.5">
                  {isVerifying ? <Loader2 className="w-3 h-3 animate-spin" /> : <Play className="w-3 h-3" />}
                  Verify Against Context
                </Button>
              </div>

              {/* Outcome Banner */}
              {verificationResult && (
                <div
                  className={`p-2.5 rounded-md border flex items-center justify-between text-xs mt-2 ${
                    verificationResult.valid
                      ? "bg-emerald-500/10 border-emerald-500/30 text-emerald-500"
                      : "bg-destructive/10 border-destructive/30 text-destructive"
                  }`}
                >
                  <div className="flex items-center gap-2 font-medium">
                    {verificationResult.valid ? (
                      <CheckCircle2 className="w-4 h-4" />
                    ) : (
                      <XCircle className="w-4 h-4" />
                    )}
                    <span>
                      {verificationResult.valid ? "Cryptographically Valid" : "Verification Failed"}
                    </span>
                  </div>
                  {verificationResult.error && (
                    <span className="text-[10px] opacity-80 truncate max-w-[140px]">
                      {verificationResult.error}
                    </span>
                  )}
                </div>
              )}
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
