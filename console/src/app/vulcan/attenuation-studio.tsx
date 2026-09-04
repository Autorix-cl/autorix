"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Button } from "@/components/ui/button";
import { Plus, Trash2, Wand2, Key, ShieldAlert, ArrowDown, Shield, Clock, Globe, FileCode } from "lucide-react";
import { KeyMetadata } from "@/lib/schemas/vulcan";
import { toast } from "sonner";
import { CodeEditor } from "@/components/ui/code-editor";
import { Badge } from "@/components/ui/badge";

interface Caveat {
  id: string;
  value: string;
}

export function AttenuationStudio() {
  const [keyId, setKeyId] = useState<string>("");
  const [secret, setSecret] = useState<string>("");
  const [caveats, setCaveats] = useState<Caveat[]>([
    { id: "1", value: "time_before = 2026-12-31T23:59:59Z" },
    { id: "2", value: "ip = 192.168.1.100" },
  ]);
  const [resultToken, setResultToken] = useState<string>("");
  const [decodedMacaroon, setDecodedMacaroon] = useState<string>("");
  const [isGenerating, setIsGenerating] = useState(false);

  const { data: keys = [] } = useQuery<KeyMetadata[]>({
    queryKey: ["vulcan-keys"],
    queryFn: async () => {
      const res = await fetch("/api/vulcan/keys");
      if (!res.ok) throw new Error("Failed to fetch keys");
      return res.json();
    },
  });

  const activeKeys = keys.filter((k) => !k.revoked);
  const selectedKey = keys.find((k) => k.id === keyId);

  const addCaveat = (presetValue?: string) => {
    setCaveats([
      ...caveats,
      { id: Math.random().toString(36).substring(7), value: presetValue || "" },
    ]);
  };

  const removeCaveat = (id: string) => {
    setCaveats(caveats.filter((c) => c.id !== id));
  };

  const updateCaveat = (id: string, value: string) => {
    setCaveats(caveats.map((c) => (c.id === id ? { ...c, value } : c)));
  };

  const handleGenerate = async () => {
    if (!keyId || !secret) {
      toast.error("Root Key and Secret are required");
      return;
    }

    const validCaveats = caveats.map((c) => c.value.trim()).filter(Boolean);

    setIsGenerating(true);
    try {
      const res = await fetch("/api/vulcan/macaroons/attenuate", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          keyId,
          secret,
          caveats: validCaveats,
        }),
      });

      const data = await res.json();

      if (!res.ok) {
        throw new Error(data.error || "Failed to generate macaroon");
      }

      setResultToken(data.token);
      setDecodedMacaroon(JSON.stringify(data.macaroon, null, 2));
      toast.success("Macaroon generated successfully!");
    } catch (err) {
      toast.error(err instanceof Error ? err.message : String(err));
    } finally {
      setIsGenerating(false);
    }
  };

  return (
    <div className="space-y-6">
      {/* Visual Capability Narrowing Chain (P6-S6-T5) */}
      <Card className="border-border bg-card/60">
        <CardHeader className="pb-3">
          <CardTitle className="text-sm font-semibold flex items-center gap-2">
            <Shield className="w-4 h-4 text-primary" />
            Macaroon Attenuation Provenance (Narrowing Chain)
          </CardTitle>
          <CardDescription className="text-xs">
            Visualizes how successive caveats cryptographically narrow authority without requiring coordination with the issuer.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex flex-col md:flex-row items-center gap-2 overflow-x-auto py-2">
            {/* Base Capability */}
            <div className="p-3 rounded-lg border bg-background shrink-0 w-full md:w-56 text-xs space-y-1">
              <div className="font-semibold text-foreground flex items-center gap-1.5">
                <Key className="w-3.5 h-3.5 text-primary" />
                Root Authority
              </div>
              <div className="text-[11px] text-muted-foreground truncate">
                {selectedKey ? selectedKey.name : "Unrestricted Root Key"}
              </div>
              <div className="flex flex-wrap gap-1 pt-1">
                {(selectedKey?.scopes || ["*"]).map((s) => (
                  <Badge key={s} variant="secondary" className="text-[10px] font-mono px-1">
                    {s}
                  </Badge>
                ))}
              </div>
            </div>

            {/* Caveats in chain */}
            {caveats.map((c, idx) => (
              <div key={c.id} className="flex flex-col md:flex-row items-center gap-2 shrink-0">
                <ArrowDown className="w-4 h-4 text-muted-foreground md:-rotate-90 shrink-0" />
                <div className="p-3 rounded-lg border bg-primary/5 border-primary/20 shrink-0 w-full md:w-56 text-xs space-y-1">
                  <div className="font-semibold text-primary flex items-center gap-1.5">
                    {c.value.includes("time") ? (
                      <Clock className="w-3.5 h-3.5" />
                    ) : c.value.includes("ip") ? (
                      <Globe className="w-3.5 h-3.5" />
                    ) : (
                      <FileCode className="w-3.5 h-3.5" />
                    )}
                    Caveat #{idx + 1}
                  </div>
                  <div className="font-mono text-[11px] text-foreground truncate">
                    {c.value || "(empty caveat)"}
                  </div>
                  <p className="text-[10px] text-muted-foreground">
                    Attenuates scope cryptographically
                  </p>
                </div>
              </div>
            ))}

            {/* Narrowed Capability */}
            <ArrowDown className="w-4 h-4 text-muted-foreground md:-rotate-90 shrink-0" />
            <div className="p-3 rounded-lg border bg-emerald-500/10 border-emerald-500/30 shrink-0 w-full md:w-56 text-xs space-y-1">
              <div className="font-semibold text-emerald-500 flex items-center gap-1.5">
                <Shield className="w-3.5 h-3.5" />
                Delegated Token
              </div>
              <p className="text-[11px] text-muted-foreground">
                HMAC-SHA256 Bound
              </p>
              <div className="text-[10px] font-mono text-emerald-500">
                {caveats.length} restriction{caveats.length === 1 ? "" : "s"} enforced
              </div>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Main Studio Editor */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Left Pane: Controls */}
        <Card className="border-border">
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-sm font-semibold">
              <Wand2 className="w-4 h-4 text-primary" />
              Attenuation Studio
            </CardTitle>
            <CardDescription className="text-xs">
              Generate a restricted Macaroon offline by chaining caveats to a Root Key.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-5 text-xs">
            <div className="space-y-3">
              <div className="space-y-1.5">
                <Label className="text-xs">Root Key</Label>
                <Select value={keyId} onValueChange={setKeyId}>
                  <SelectTrigger className="h-8 text-xs">
                    <SelectValue placeholder="Select an active root key" />
                  </SelectTrigger>
                  <SelectContent>
                    {activeKeys.map((k) => (
                      <SelectItem key={k.id} value={k.id} className="text-xs">
                        {k.name} ({k.prefix})
                      </SelectItem>
                    ))}
                    {activeKeys.length === 0 && (
                      <SelectItem value="none" disabled className="text-xs">
                        No active keys found
                      </SelectItem>
                    )}
                  </SelectContent>
                </Select>
              </div>

              <div className="space-y-1.5">
                <Label className="flex justify-between items-center text-xs">
                  <span>Root Secret</span>
                  <Badge variant="outline" className="text-[10px] bg-amber-500/10 text-amber-500 border-amber-500/20 px-1.5 py-0">
                    <ShieldAlert className="w-2.5 h-2.5 mr-1" /> Client-side only
                  </Badge>
                </Label>
                <div className="relative">
                  <Key className="absolute left-2.5 top-2.5 h-3.5 w-3.5 text-muted-foreground" />
                  <Input
                    type="password"
                    placeholder="Paste plain-text secret (av_live_...)"
                    className="pl-8 font-mono h-8 text-xs"
                    value={secret}
                    onChange={(e) => setSecret(e.target.value)}
                  />
                </div>
              </div>
            </div>

            <div className="space-y-3 pt-3 border-t">
              <div className="flex items-center justify-between">
                <Label className="text-xs font-semibold">Caveats (Restrictions)</Label>
                <div className="flex items-center gap-1.5">
                  <Button
                    variant="outline"
                    size="sm"
                    className="h-6 text-[11px] px-2"
                    onClick={() => addCaveat("time_before = 2026-12-31T23:59:59Z")}
                  >
                    + Time
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    className="h-6 text-[11px] px-2"
                    onClick={() => addCaveat("ip = 192.168.1.100")}
                  >
                    + IP
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    className="h-6 text-[11px] px-2"
                    onClick={() => addCaveat("method = GET")}
                  >
                    + Method
                  </Button>
                  <Button variant="outline" size="sm" className="h-6 text-[11px] px-2" onClick={() => addCaveat()}>
                    <Plus className="w-3 h-3 mr-1" /> Custom
                  </Button>
                </div>
              </div>

              <div className="space-y-2">
                {caveats.map((caveat, i) => (
                  <div key={caveat.id} className="flex gap-2 items-center">
                    <div className="bg-muted text-muted-foreground px-2 py-1.5 rounded text-[11px] font-mono border">
                      {i + 1}
                    </div>
                    <Input
                      placeholder="e.g. ip = 192.168.1.100"
                      className="font-mono text-xs h-8"
                      value={caveat.value}
                      onChange={(e) => updateCaveat(caveat.id, e.target.value)}
                    />
                    <Button
                      variant="ghost"
                      size="icon"
                      className="text-destructive shrink-0 h-8 w-8"
                      onClick={() => removeCaveat(caveat.id)}
                    >
                      <Trash2 className="w-3.5 h-3.5" />
                    </Button>
                  </div>
                ))}
                {caveats.length === 0 && (
                  <div className="text-xs text-muted-foreground text-center p-3 border border-dashed rounded-md">
                    No caveats added. The macaroon will possess unrestricted root capability.
                  </div>
                )}
              </div>
            </div>

            <Button className="w-full h-8 text-xs" onClick={handleGenerate} disabled={isGenerating}>
              {isGenerating ? "Computing HMAC chain..." : "Generate Attenuated Macaroon"}
            </Button>
          </CardContent>
        </Card>

        {/* Right Pane: Output */}
        <Card className="bg-card border-border flex flex-col overflow-hidden">
          <CardHeader className="bg-muted/30 border-b py-3">
            <CardTitle className="text-xs font-semibold">Output Macaroon</CardTitle>
          </CardHeader>
          <CardContent className="p-0 flex flex-col h-full min-h-[350px]">
            {resultToken ? (
              <div className="flex flex-col h-full divide-y text-xs">
                <div className="p-4 space-y-1.5">
                  <Label className="text-[11px] text-muted-foreground">Bearer Token (Base64URL)</Label>
                  <div className="p-2.5 bg-muted rounded-md border break-all font-mono text-[11px] text-primary">
                    {resultToken}
                  </div>
                </div>
                <div className="p-4 flex-1 flex flex-col space-y-1.5">
                  <Label className="text-[11px] text-muted-foreground">Decoded Macaroon Payload</Label>
                  <div className="flex-1 rounded-md overflow-hidden border min-h-[160px]">
                    <CodeEditor value={decodedMacaroon} language="json" readOnly onChange={() => {}} />
                  </div>
                </div>
              </div>
            ) : (
              <div className="flex items-center justify-center h-full text-muted-foreground text-xs p-8 text-center flex-col gap-2">
                <Wand2 className="w-6 h-6 opacity-30" />
                <p>Configure root key and caveats on the left to synthesize an attenuated token.</p>
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
