"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Button } from "@/components/ui/button";
import { Plus, Trash2, Wand2, Key, ShieldAlert } from "lucide-react";
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
  const [caveats, setCaveats] = useState<Caveat[]>([{ id: "1", value: "time < 2026-12-31T23:59:59Z" }]);
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

  const activeKeys = keys.filter(k => !k.revoked);

  const addCaveat = () => {
    setCaveats([...caveats, { id: Math.random().toString(36).substring(7), value: "" }]);
  };

  const removeCaveat = (id: string) => {
    setCaveats(caveats.filter(c => c.id !== id));
  };

  const updateCaveat = (id: string, value: string) => {
    setCaveats(caveats.map(c => c.id === id ? { ...c, value } : c));
  };

  const handleGenerate = async () => {
    if (!keyId || !secret) {
      toast.error("Root Key and Secret are required");
      return;
    }

    const validCaveats = caveats.map(c => c.value.trim()).filter(Boolean);

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
    <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mt-8">
      {/* Left Pane: Controls */}
      <Card className="border-border">
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Wand2 className="w-5 h-5 text-primary" />
            Attenuation Studio
          </CardTitle>
          <CardDescription>
            Generate a restricted Macaroon offline by chaining caveats to a Root Key.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-6">
          <div className="space-y-4">
            <div className="space-y-2">
              <Label>Root Key ID</Label>
              <Select value={keyId} onValueChange={setKeyId}>
                <SelectTrigger>
                  <SelectValue placeholder="Select an active root key" />
                </SelectTrigger>
                <SelectContent>
                  {activeKeys.map(k => (
                    <SelectItem key={k.id} value={k.id}>
                      {k.name} ({k.prefix})
                    </SelectItem>
                  ))}
                  {activeKeys.length === 0 && (
                    <SelectItem value="none" disabled>No active keys found</SelectItem>
                  )}
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-2">
              <Label className="flex justify-between items-center">
                Root Secret
                <Badge variant="outline" className="text-xs bg-amber-500/10 text-amber-500 border-amber-500/20">
                  <ShieldAlert className="w-3 h-3 mr-1" /> Client-side only
                </Badge>
              </Label>
              <div className="relative">
                <Key className="absolute left-3 top-3 h-4 w-4 text-muted-foreground" />
                <Input 
                  type="password" 
                  placeholder="Paste your plain-text secret here (av_live_...)" 
                  className="pl-9 font-mono"
                  value={secret}
                  onChange={(e) => setSecret(e.target.value)}
                />
              </div>
            </div>
          </div>

          <div className="space-y-4 pt-4 border-t">
            <div className="flex items-center justify-between">
              <Label>Caveats (Restrictions)</Label>
              <Button variant="outline" size="sm" onClick={addCaveat}>
                <Plus className="w-4 h-4 mr-2" />
                Add Rule
              </Button>
            </div>
            
            <div className="space-y-3">
              {caveats.map((caveat, i) => (
                <div key={caveat.id} className="flex gap-2 items-start">
                  <div className="bg-muted text-muted-foreground px-2 py-2 rounded text-xs font-mono border">
                    {i + 1}
                  </div>
                  <Input 
                    placeholder="e.g. ip = 192.168.1.100" 
                    className="font-mono text-sm"
                    value={caveat.value}
                    onChange={(e) => updateCaveat(caveat.id, e.target.value)}
                  />
                  <Button variant="ghost" size="icon" className="text-destructive shrink-0" onClick={() => removeCaveat(caveat.id)}>
                    <Trash2 className="w-4 h-4" />
                  </Button>
                </div>
              ))}
              {caveats.length === 0 && (
                <div className="text-sm text-muted-foreground text-center p-4 border border-dashed rounded-md">
                  No caveats added. The macaroon will have root privileges.
                </div>
              )}
            </div>
          </div>

          <Button className="w-full" onClick={handleGenerate} disabled={isGenerating}>
            {isGenerating ? "Computing HMAC chain..." : "Generate Attenuated Macaroon"}
          </Button>
        </CardContent>
      </Card>

      {/* Right Pane: Output */}
      <Card className="bg-card border-border flex flex-col overflow-hidden">
        <CardHeader className="bg-muted/30 border-b">
          <CardTitle className="text-sm font-medium">Output</CardTitle>
        </CardHeader>
        <CardContent className="p-0 flex flex-col h-full min-h-[400px]">
          {resultToken ? (
            <div className="flex flex-col h-full divide-y">
              <div className="p-4 space-y-2">
                <Label className="text-xs text-muted-foreground">Bearer Token (Base64URL)</Label>
                <div className="p-3 bg-muted rounded-md border break-all font-mono text-xs text-primary">
                  {resultToken}
                </div>
              </div>
              <div className="p-4 flex-1 flex flex-col space-y-2">
                <Label className="text-xs text-muted-foreground">Decoded Macaroon Payload</Label>
                <div className="flex-1 rounded-md overflow-hidden border">
                  <CodeEditor 
                    value={decodedMacaroon} 
                    language="json" 
                    readOnly
                    onChange={() => {}}
                  />
                </div>
              </div>
            </div>
          ) : (
            <div className="flex items-center justify-center h-full text-muted-foreground text-sm p-8 text-center flex-col gap-2">
              <Wand2 className="w-8 h-8 opacity-20" />
              <p>Fill in the root key and caveats on the left to generate a macaroon.</p>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
