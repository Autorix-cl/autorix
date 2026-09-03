"use client";

import * as React from "react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import {
  User,
  Search,
  Key,
  Layers,
  Network,
  Laptop,
  Building2,
  Loader2,
} from "lucide-react";
import { toast } from "sonner";
import type { UnifiedSubject } from "@/lib/api/schemas/explorer";

export function UnifiedSubjectView() {
  const [subjectQuery, setSubjectQuery] = React.useState("alice");
  const [isLoading, setIsLoading] = React.useState(false);
  const [subject, setSubject] = React.useState<UnifiedSubject | null>(null);

  const fetchSubject = React.useCallback(async (id: string) => {
    if (!id.trim()) return;
    setIsLoading(true);
    try {
      const res = await fetch(`/api/explorer/subject/${encodeURIComponent(id.trim())}`);
      if (!res.ok) throw new Error("Failed to resolve subject");
      const data = await res.json();
      setSubject(data);
    } catch {
      toast.error("Failed to resolve subject across engines");
    } finally {
      setIsLoading(false);
    }
  }, []);

  React.useEffect(() => {
    fetchSubject("alice");
  }, [fetchSubject]);

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault();
    fetchSubject(subjectQuery);
  };

  return (
    <div className="space-y-4">
      {/* Search Input */}
      <form onSubmit={handleSearch} className="flex gap-2 max-w-md">
        <div className="relative flex-1">
          <Search className="absolute left-2.5 top-2.5 h-3.5 w-3.5 text-muted-foreground" />
          <Input
            placeholder="Search Subject ID or Username (e.g. alice)"
            value={subjectQuery}
            onChange={(e) => setSubjectQuery(e.target.value)}
            className="pl-8 h-8 text-xs font-mono"
          />
        </div>
        <Button type="submit" size="sm" disabled={isLoading} className="h-8 text-xs gap-1.5">
          {isLoading ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : <Search className="w-3.5 h-3.5" />}
          Resolve
        </Button>
      </form>

      {isLoading ? (
        <div className="py-12 text-center text-muted-foreground flex items-center justify-center gap-2">
          <Loader2 className="w-4 h-4 animate-spin text-primary" /> Resolving cross-engine identity graph...
        </div>
      ) : !subject ? (
        <div className="py-12 text-center text-muted-foreground border rounded-lg border-dashed">
          No subject selected. Enter a subject identifier above.
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          {/* Column 1: Identity & Sessions (Ego) */}
          <div className="space-y-4">
            <Card className="border-border">
              <CardHeader className="pb-3">
                <CardTitle className="text-xs font-semibold flex items-center justify-between">
                  <span className="flex items-center gap-1.5">
                    <User className="w-3.5 h-3.5 text-primary" />
                    Ego Identity Profile
                  </span>
                  <Badge variant="outline" className="text-[10px] uppercase">
                    {subject.identity?.state || "active"}
                  </Badge>
                </CardTitle>
                <CardDescription className="text-[11px]">Primary authentication entity</CardDescription>
              </CardHeader>
              <CardContent className="space-y-2 text-xs">
                <div className="font-mono text-primary font-bold text-sm truncate">{subject.id}</div>
                <div className="p-2 rounded bg-muted/40 font-mono text-[11px] space-y-1">
                  <div>
                    <span className="text-muted-foreground">email: </span>
                    <span>{String(subject.identity?.traits?.email || "—")}</span>
                  </div>
                  <div>
                    <span className="text-muted-foreground">name: </span>
                    <span>{String(subject.identity?.traits?.name || subject.id)}</span>
                  </div>
                </div>

                {subject.enterprise_linkage && (
                  <div className="pt-2 border-t flex items-center gap-1.5 text-[11px] text-muted-foreground">
                    <Building2 className="w-3.5 h-3.5 text-rose-400" />
                    <span>SCIM Linked: {subject.enterprise_linkage.email}</span>
                  </div>
                )}
              </CardContent>
            </Card>

            <Card className="border-border">
              <CardHeader className="pb-2">
                <CardTitle className="text-xs font-semibold flex items-center gap-1.5">
                  <Laptop className="w-3.5 h-3.5 text-primary" />
                  Active Sessions ({subject.sessions.length})
                </CardTitle>
              </CardHeader>
              <CardContent className="text-xs space-y-2">
                {subject.sessions.length === 0 ? (
                  <div className="text-muted-foreground text-[11px]">No active sessions</div>
                ) : (
                  subject.sessions.map((s) => (
                    <div key={s.id} className="p-2 rounded border bg-card/60 text-[11px] space-y-1 font-mono">
                      <div className="flex items-center justify-between">
                        <span className="font-semibold text-foreground truncate">{s.ip_address}</span>
                        <Badge variant="default" className="bg-emerald-500 text-[9px] px-1 py-0">
                          Active
                        </Badge>
                      </div>
                      <div className="text-[10px] text-muted-foreground truncate">{s.user_agent}</div>
                    </div>
                  ))
                )}
              </CardContent>
            </Card>
          </div>

          {/* Column 2: Permissions & ReBAC Relations (Nexus) */}
          <div className="space-y-4">
            <Card className="border-border h-full">
              <CardHeader className="pb-3">
                <CardTitle className="text-xs font-semibold flex items-center gap-1.5">
                  <Network className="w-3.5 h-3.5 text-primary" />
                  Nexus Relation Tuples ({subject.relations.length})
                </CardTitle>
                <CardDescription className="text-[11px]">
                  Graph relations where subject is direct or indirect member
                </CardDescription>
              </CardHeader>
              <CardContent className="text-xs space-y-2 max-h-96 overflow-y-auto pr-1">
                {subject.relations.length === 0 ? (
                  <div className="text-muted-foreground text-[11px] py-4 text-center">
                    No Zanzibar relation tuples bound to this subject.
                  </div>
                ) : (
                  subject.relations.map((r, i) => (
                    <div key={i} className="p-2 rounded border bg-card/60 text-[11px] font-mono space-y-1">
                      <div className="flex items-center justify-between">
                        <span className="text-primary font-semibold">
                          {r.namespace}:{r.object}
                        </span>
                        <Badge variant="secondary" className="text-[9px] px-1 py-0">
                          {r.relation}
                        </Badge>
                      </div>
                      {r.caveat && (
                        <div className="text-[10px] text-amber-500">
                          caveat: {r.caveat}
                        </div>
                      )}
                    </div>
                  ))
                )}
              </CardContent>
            </Card>
          </div>

          {/* Column 3: Machine Credentials & API Keys (Vulcan & Janus) */}
          <div className="space-y-4">
            <Card className="border-border">
              <CardHeader className="pb-2">
                <CardTitle className="text-xs font-semibold flex items-center gap-1.5">
                  <Layers className="w-3.5 h-3.5 text-primary" />
                  Vulcan API Keys ({subject.api_keys.length})
                </CardTitle>
              </CardHeader>
              <CardContent className="text-xs space-y-2">
                {subject.api_keys.length === 0 ? (
                  <div className="text-muted-foreground text-[11px]">No active machine credentials</div>
                ) : (
                  subject.api_keys.map((k) => (
                    <div key={k.id} className="p-2 rounded border bg-card/60 text-[11px] font-mono space-y-1">
                      <div className="flex items-center justify-between">
                        <span className="font-semibold text-foreground truncate">{k.name}</span>
                        <span className="text-[10px] text-muted-foreground">{k.call_count} calls</span>
                      </div>
                      <div className="text-[10px] text-muted-foreground truncate">{k.prefix}</div>
                    </div>
                  ))
                )}
              </CardContent>
            </Card>

            <Card className="border-border">
              <CardHeader className="pb-2">
                <CardTitle className="text-xs font-semibold flex items-center gap-1.5">
                  <Key className="w-3.5 h-3.5 text-primary" />
                  Janus OAuth2 Grants
                </CardTitle>
              </CardHeader>
              <CardContent className="text-xs space-y-2">
                {subject.oauth_grants.length === 0 ? (
                  <div className="text-muted-foreground text-[11px]">No active OAuth2 authorizations</div>
                ) : (
                  subject.oauth_grants.map((g) => (
                    <div key={g.id} className="p-2 rounded border bg-card/60 text-[11px] font-mono">
                      <div className="font-semibold text-foreground">{g.client_id}</div>
                      <div className="text-[10px] text-muted-foreground">{g.scope || "openid"}</div>
                    </div>
                  ))
                )}
              </CardContent>
            </Card>
          </div>
        </div>
      )}
    </div>
  );
}
