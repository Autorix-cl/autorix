"use client";

import * as React from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import {
  FileText,
  Search,
  RefreshCw,
  ExternalLink,
  ChevronDown,
  ChevronRight,
  Loader2,
} from "lucide-react";
import { toast } from "sonner";
import type { LogEntry } from "@/lib/api/schemas/observability";

interface LogViewerProps {
  onSelectTrace?: (traceId: string) => void;
}

export function LogViewer({ onSelectTrace }: LogViewerProps) {
  const [logs, setLogs] = React.useState<LogEntry[]>([]);
  const [engine, setEngine] = React.useState("all");
  const [level, setLevel] = React.useState("all");
  const [search, setSearch] = React.useState("");
  const [isLoading, setIsLoading] = React.useState(false);
  const [expandedLogId, setExpandedLogId] = React.useState<string | null>(null);

  const fetchLogs = React.useCallback(async () => {
    setIsLoading(true);
    try {
      const params = new URLSearchParams();
      if (engine !== "all") params.set("engine", engine);
      if (level !== "all") params.set("level", level);
      if (search) params.set("q", search);

      const res = await fetch(`/api/observability/logs?${params.toString()}`);
      if (!res.ok) throw new Error("Failed to fetch logs");
      const data = await res.json();
      setLogs(data);
    } catch {
      toast.error("Error fetching structured logs");
    } finally {
      setIsLoading(false);
    }
  }, [engine, level, search]);

  React.useEffect(() => {
    fetchLogs();
  }, [fetchLogs]);

  const getLevelBadge = (lvl: string) => {
    switch (lvl) {
      case "error":
        return <Badge variant="destructive" className="text-[9px] uppercase px-1 py-0">ERR</Badge>;
      case "warn":
        return <Badge variant="secondary" className="bg-amber-500/10 text-amber-500 text-[9px] uppercase px-1 py-0 border-amber-500/20">WARN</Badge>;
      case "debug":
        return <Badge variant="outline" className="text-[9px] uppercase px-1 py-0 text-muted-foreground">DBG</Badge>;
      default:
        return <Badge variant="secondary" className="text-[9px] uppercase px-1 py-0 text-sky-500">INFO</Badge>;
    }
  };

  return (
    <div className="space-y-4">
      {/* Filter Bar */}
      <Card className="border-border">
        <CardContent className="p-3">
          <div className="flex flex-wrap items-center gap-2">
            <div className="relative flex-1 min-w-[200px]">
              <Search className="absolute left-2.5 top-2.5 h-3.5 w-3.5 text-muted-foreground" />
              <Input
                placeholder="Search messages, request IDs, endpoints..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="pl-8 h-8 text-xs"
              />
            </div>

            <Select value={engine} onValueChange={setEngine}>
              <SelectTrigger className="w-[130px] h-8 text-xs">
                <SelectValue placeholder="All Engines" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All Engines</SelectItem>
                <SelectItem value="aegis">Aegis</SelectItem>
                <SelectItem value="nexus">Nexus</SelectItem>
                <SelectItem value="themis">Themis</SelectItem>
                <SelectItem value="ego">Ego</SelectItem>
                <SelectItem value="janus">Janus</SelectItem>
                <SelectItem value="vulcan">Vulcan</SelectItem>
                <SelectItem value="hermes">Hermes</SelectItem>
              </SelectContent>
            </Select>

            <Select value={level} onValueChange={setLevel}>
              <SelectTrigger className="w-[110px] h-8 text-xs">
                <SelectValue placeholder="All Levels" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All Levels</SelectItem>
                <SelectItem value="info">Info</SelectItem>
                <SelectItem value="warn">Warning</SelectItem>
                <SelectItem value="error">Error</SelectItem>
                <SelectItem value="debug">Debug</SelectItem>
              </SelectContent>
            </Select>

            <Button
              variant="outline"
              size="sm"
              onClick={() => fetchLogs()}
              disabled={isLoading}
              className="h-8 gap-1 text-xs"
            >
              {isLoading ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <RefreshCw className="h-3.5 w-3.5" />}
              Fetch
            </Button>
          </div>
        </CardContent>
      </Card>

      {/* Log Feed Table */}
      <Card className="border-border font-mono text-xs">
        <CardHeader className="p-3 pb-2 font-sans border-b border-border">
          <CardTitle className="text-xs font-semibold flex items-center justify-between">
            <span className="flex items-center gap-1.5">
              <FileText className="h-4 w-4 text-sky-500" />
              Structured Fleet Logs ({logs.length})
            </span>
            <span className="text-[10px] text-muted-foreground font-normal">
              Shared vocabulary &amp; correlation IDs
            </span>
          </CardTitle>
        </CardHeader>
        <CardContent className="p-0">
          <div className="divide-y divide-border">
            {logs.length === 0 ? (
              <div className="p-6 text-center text-muted-foreground font-sans text-xs">
                No log entries match your filter criteria.
              </div>
            ) : (
              logs.map((log) => {
                const isExpanded = expandedLogId === log.id;
                return (
                  <div key={log.id} className="hover:bg-muted/30 transition-colors">
                    <div
                      className="p-2.5 flex items-start gap-2.5 cursor-pointer select-none"
                      onClick={() => setExpandedLogId(isExpanded ? null : log.id)}
                    >
                      <button className="mt-0.5 text-muted-foreground">
                        {isExpanded ? <ChevronDown className="h-3 w-3" /> : <ChevronRight className="h-3 w-3" />}
                      </button>
                      <span className="text-[10px] text-muted-foreground whitespace-nowrap">
                        {new Date(log.timestamp).toLocaleTimeString()}
                      </span>
                      {getLevelBadge(log.level)}
                      <Badge variant="outline" className="text-[9px] uppercase px-1 py-0 whitespace-nowrap font-mono">
                        {log.engine}
                      </Badge>
                      <span className="flex-1 truncate text-[11px] text-foreground">
                        {log.message}
                      </span>
                      {log.trace_id && (
                        <Button
                          variant="ghost"
                          size="sm"
                          className="h-5 text-[10px] px-1.5 gap-1 font-sans text-sky-400 hover:text-sky-300"
                          onClick={(e) => {
                            e.stopPropagation();
                            onSelectTrace?.(log.trace_id!);
                          }}
                        >
                          Trace
                          <ExternalLink className="h-2.5 w-2.5" />
                        </Button>
                      )}
                    </div>

                    {isExpanded && (
                      <div className="px-8 pb-3 pt-1 bg-muted/10 border-t border-border/50 text-[11px] space-y-1.5 font-sans">
                        <div className="grid grid-cols-2 md:grid-cols-4 gap-2 text-xs">
                          <div>
                            <span className="text-muted-foreground text-[10px]">Request ID: </span>
                            <span className="font-mono text-[10px] select-all">{log.request_id || "none"}</span>
                          </div>
                          <div>
                            <span className="text-muted-foreground text-[10px]">Correlation ID: </span>
                            <span className="font-mono text-[10px] select-all">{log.correlation_id || "none"}</span>
                          </div>
                          <div>
                            <span className="text-muted-foreground text-[10px]">Trace ID: </span>
                            <span className="font-mono text-[10px] select-all">{log.trace_id || "none"}</span>
                          </div>
                          <div>
                            <span className="text-muted-foreground text-[10px]">Instance: </span>
                            <span className="font-mono text-[10px]">{log.instance_id || "none"}</span>
                          </div>
                        </div>
                        <div>
                          <span className="text-muted-foreground text-[10px]">Structured Attributes:</span>
                          <pre className="mt-1 p-2 rounded bg-muted/40 text-[10px] font-mono overflow-x-auto">
                            {JSON.stringify(log.attributes, null, 2)}
                          </pre>
                        </div>
                      </div>
                    )}
                  </div>
                );
              })
            )}
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
