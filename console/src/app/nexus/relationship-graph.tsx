"use client";

import * as React from "react";
import {
  GitGraph,
  Search,
  ZoomIn,
  ZoomOut,
  RotateCcw,
  User,
  FileText,
  Users,
  ArrowRight,
  ShieldCheck,
  Loader2,
  RefreshCw,
  AlertCircle,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { useNexusTuples } from "@/hooks/use-nexus";
import { getCsrfHeaders } from "@/lib/api/client";
import type { Tuple as GridTuple } from "./tuple-grid";
import type { ExpandNode } from "@/lib/api/schemas/nexus";

export type GraphNodeType = "user" | "document" | "organization" | "group" | "resource";

export interface GraphRelationEdge {
  relation: string;
  targetId: string;
  targetType: GraphNodeType;
  direction?: "outgoing" | "incoming";
}

export interface GraphNode {
  id: string;
  type: GraphNodeType;
  label: string;
  relations: GraphRelationEdge[];
}

interface RelationshipGraphProps {
  initialObjectId?: string;
}

function classifyType(namespace: string): GraphNodeType {
  const ns = namespace.toLowerCase();
  if (ns === "user" || ns === "users" || ns === "identity") return "user";
  if (ns === "document" || ns === "doc" || ns === "file") return "document";
  if (ns === "organization" || ns === "org" || ns === "tenant") return "organization";
  if (ns === "group" || ns === "team" || ns === "role") return "group";
  return "resource";
}

function buildGraphFromTuples(tuples: GridTuple[]): Record<string, GraphNode> {
  const graph: Record<string, GraphNode> = {};

  const ensureNode = (id: string, ns: string): GraphNode => {
    if (!graph[id]) {
      graph[id] = {
        id,
        type: classifyType(ns),
        label: id,
        relations: [],
      };
    }
    return graph[id];
  };

  for (const t of tuples) {
    if (!t.objectType || !t.objectId || !t.subjectId) continue;
    const objId = `${t.objectType}:${t.objectId}`;
    const subjNs = t.subjectType || "user";
    const subjId = `${subjNs}:${t.subjectId}${t.subjectRelation ? `#${t.subjectRelation}` : ""}`;

    const objNode = ensureNode(objId, t.objectType);
    ensureNode(subjId, subjNs);

    const alreadyLinked = objNode.relations.some(
      (r) => r.relation === t.relation && r.targetId === subjId,
    );
    if (!alreadyLinked) {
      objNode.relations.push({
        relation: t.relation,
        targetId: subjId,
        targetType: classifyType(subjNs),
        direction: "outgoing",
      });
    }
  }

  return graph;
}

function collectExpandSubjects(node: ExpandNode | undefined, out: string[] = []): string[] {
  if (!node) return out;
  if (Array.isArray(node.subjects)) {
    for (const s of node.subjects) {
      if (typeof s === "string" && s.trim()) out.push(s.trim());
    }
  }
  if (Array.isArray(node.children)) {
    for (const child of node.children) {
      collectExpandSubjects(child, out);
    }
  }
  return out;
}

export function RelationshipGraph({ initialObjectId }: RelationshipGraphProps) {
  const { data: tuples = [], isLoading, isError, error, refetch, isFetching } = useNexusTuples();

  const [searchTarget, setSearchTarget] = React.useState(initialObjectId ?? "");
  const [zoom, setZoom] = React.useState(1);
  const [selectedNode, setSelectedNode] = React.useState<string | null>(initialObjectId ?? null);
  const [expandedEdges, setExpandedEdges] = React.useState<Record<string, GraphRelationEdge[]>>({});
  const [isExpanding, setIsExpanding] = React.useState(false);
  const [expandError, setExpandError] = React.useState<string | null>(null);

  const nodes = React.useMemo(() => {
    const base = buildGraphFromTuples(tuples);
    for (const [nodeId, extras] of Object.entries(expandedEdges)) {
      const [ns = "resource"] = nodeId.split(":");
      if (!base[nodeId]) {
        base[nodeId] = { id: nodeId, type: classifyType(ns), label: nodeId, relations: [] };
      }
      for (const edge of extras) {
        if (!base[nodeId].relations.some((r) => r.relation === edge.relation && r.targetId === edge.targetId)) {
          base[nodeId].relations.push(edge);
        }
        const [targetNs = "resource"] = edge.targetId.split(":");
        if (!base[edge.targetId]) {
          base[edge.targetId] = {
            id: edge.targetId,
            type: classifyType(targetNs),
            label: edge.targetId,
            relations: [],
          };
        }
      }
    }
    return base;
  }, [tuples, expandedEdges]);

  const nodeIds = React.useMemo(() => Object.keys(nodes), [nodes]);

  // Select the first live node automatically when tuples load and nothing valid is selected
  React.useEffect(() => {
    if (selectedNode && nodes[selectedNode]) return;
    if (initialObjectId && nodes[initialObjectId]) {
      setSelectedNode(initialObjectId);
      setSearchTarget(initialObjectId);
      return;
    }
    if (nodeIds.length > 0) {
      setSelectedNode(nodeIds[0]);
      setSearchTarget(nodeIds[0]);
    }
  }, [initialObjectId, nodeIds, nodes, selectedNode]);

  const currentNode = selectedNode ? nodes[selectedNode] : undefined;

  // Incoming references to the focused node so users can also navigate upward
  const incomingLinks = React.useMemo(() => {
    if (!currentNode) return [];
    const results: { sourceId: string; sourceType: GraphNodeType; relation: string }[] = [];
    for (const node of Object.values(nodes)) {
      for (const rel of node.relations) {
        if (rel.targetId === currentNode.id) {
          results.push({ sourceId: node.id, sourceType: node.type, relation: rel.relation });
        }
      }
    }
    return results;
  }, [currentNode, nodes]);

  const getNodeIcon = (type: GraphNodeType) => {
    switch (type) {
      case "user":
        return <User className="w-4 h-4 text-emerald-400" aria-hidden="true" />;
      case "document":
        return <FileText className="w-4 h-4 text-blue-400" aria-hidden="true" />;
      case "group":
        return <Users className="w-4 h-4 text-amber-400" aria-hidden="true" />;
      case "organization":
        return <ShieldCheck className="w-4 h-4 text-purple-400" aria-hidden="true" />;
      default:
        return <GitGraph className="w-4 h-4 text-muted-foreground" aria-hidden="true" />;
    }
  };

  const handleSearch = async (e: React.FormEvent) => {
    e.preventDefault();
    const trimmed = searchTarget.trim();
    if (!trimmed) return;
    setExpandError(null);

    // If user typed "namespace:object#relation", call real /api/nexus/expand
    const hashIdx = trimmed.indexOf("#");
    if (hashIdx > 0 && trimmed.includes(":")) {
      const objPart = trimmed.slice(0, hashIdx);
      const relation = trimmed.slice(hashIdx + 1);
      const colonIdx = objPart.indexOf(":");
      const namespace = objPart.slice(0, colonIdx);
      const object = objPart.slice(colonIdx + 1);

      if (namespace && object && relation) {
        setIsExpanding(true);
        try {
          const res = await fetch("/api/nexus/expand", {
            method: "POST",
            headers: { "Content-Type": "application/json", ...getCsrfHeaders() },
            body: JSON.stringify({ namespace, object, relation }),
          });
          if (!res.ok) {
            const errBody = await res.json().catch(() => ({}));
            throw new Error(errBody.error || `Expand failed (HTTP ${res.status})`);
          }
          const data = (await res.json()) as { tree?: ExpandNode };
          const subjects = collectExpandSubjects(data.tree);
          const edges: GraphRelationEdge[] = subjects.map((sub) => {
            const [subNs = "user"] = sub.split(":");
            return {
              relation,
              targetId: sub,
              targetType: classifyType(subNs),
              direction: "outgoing",
            };
          });
          setExpandedEdges((prev) => ({ ...prev, [objPart]: edges }));
          setSelectedNode(objPart);
        } catch (err) {
          setExpandError(err instanceof Error ? err.message : "Failed to expand relation from Nexus");
        } finally {
          setIsExpanding(false);
        }
        return;
      }
    }

    if (nodes[trimmed]) {
      setSelectedNode(trimmed);
    } else {
      setExpandError(
        `No node "${trimmed}" found in current relation tuples. Use "namespace:object#relation" to expand directly from Nexus.`,
      );
    }
  };

  return (
    <div className="flex flex-col h-full bg-card rounded-md border text-card-foreground shadow-sm">
      <div className="p-4 border-b flex flex-wrap items-center justify-between gap-3">
        <div>
          <h3 className="font-semibold text-sm flex items-center gap-2">
            <GitGraph className="w-4 h-4 text-primary" aria-hidden="true" />
            Relationship Graph Explorer
          </h3>
          <p className="text-xs text-muted-foreground mt-0.5">
            Live Zanzibar object neighborhoods built from Nexus tuples. Enter <code className="font-mono">ns:id#rel</code> to expand a relation tree.
          </p>
        </div>

        <div className="flex flex-wrap items-center gap-2">
          <form onSubmit={handleSearch} className="flex items-center gap-2">
            <Input
              aria-label="Inspect node or expand relation (namespace:object or namespace:object#relation)"
              value={searchTarget}
              onChange={(e) => setSearchTarget(e.target.value)}
              placeholder="e.g. document:doc_1#viewer"
              className="h-8 text-xs font-mono w-[220px]"
            />
            <Button type="submit" size="sm" variant="secondary" className="h-8" disabled={isExpanding}>
              {isExpanding ? (
                <Loader2 className="w-3.5 h-3.5 mr-1 animate-spin" />
              ) : (
                <Search className="w-3.5 h-3.5 mr-1" />
              )}
              Inspect
            </Button>
          </form>

          <div className="flex items-center gap-1 border-l pl-2">
            <Button
              type="button"
              variant="outline"
              size="icon"
              className="h-8 w-8"
              aria-label="Refresh graph from Nexus"
              title="Refresh graph"
              onClick={() => refetch()}
              disabled={isFetching}
            >
              <RefreshCw className={`w-3.5 h-3.5 ${isFetching ? "animate-spin" : ""}`} />
            </Button>
            <Button
              type="button"
              variant="outline"
              size="icon"
              className="h-8 w-8"
              aria-label="Zoom in"
              title="Zoom in"
              onClick={() => setZoom((z) => Math.min(1.5, Number((z + 0.1).toFixed(2))))}
            >
              <ZoomIn className="w-3.5 h-3.5" />
            </Button>
            <Button
              type="button"
              variant="outline"
              size="icon"
              className="h-8 w-8"
              aria-label="Zoom out"
              title="Zoom out"
              onClick={() => setZoom((z) => Math.max(0.6, Number((z - 0.1).toFixed(2))))}
            >
              <ZoomOut className="w-3.5 h-3.5" />
            </Button>
            <Button
              type="button"
              variant="outline"
              size="icon"
              className="h-8 w-8"
              aria-label="Reset zoom"
              title="Reset zoom"
              onClick={() => setZoom(1)}
            >
              <RotateCcw className="w-3.5 h-3.5" />
            </Button>
          </div>
        </div>
      </div>

      {expandError && (
        <div
          role="alert"
          className="mx-4 mt-3 p-2.5 rounded-md border border-destructive/30 bg-destructive/10 text-destructive text-xs flex items-center gap-2"
        >
          <AlertCircle className="w-4 h-4 shrink-0" />
          <span>{expandError}</span>
        </div>
      )}

      {/* Quick node selector when live nodes exist */}
      {nodeIds.length > 0 && (
        <div className="px-4 py-2 border-b bg-muted/20 flex items-center gap-1.5 overflow-x-auto">
          <span className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground shrink-0 mr-1">
            Nodes ({nodeIds.length}):
          </span>
          {nodeIds.slice(0, 12).map((id) => (
            <button
              key={id}
              type="button"
              onClick={() => {
                setSelectedNode(id);
                setSearchTarget(id);
                setExpandError(null);
              }}
              className={`px-2 py-0.5 rounded font-mono text-[11px] border shrink-0 transition-colors ${
                selectedNode === id
                  ? "bg-primary/15 border-primary text-primary font-semibold"
                  : "bg-card border-border text-muted-foreground hover:text-foreground"
              }`}
            >
              {id}
            </button>
          ))}
        </div>
      )}

      {/* Graph Canvas View */}
      <div className="flex-1 p-6 overflow-auto relative bg-muted/10 min-h-[360px] flex items-center justify-center">
        {isLoading ? (
          <div className="flex items-center gap-2 text-xs text-muted-foreground">
            <Loader2 className="w-4 h-4 animate-spin" />
            Loading relationship tuples from Nexus...
          </div>
        ) : isError ? (
          <div role="alert" className="text-center space-y-2 max-w-sm">
            <AlertCircle className="w-7 h-7 text-destructive mx-auto" />
            <p className="text-xs font-medium text-destructive">
              {error instanceof Error ? error.message : "Failed to load relationship graph from Nexus."}
            </p>
            <Button type="button" variant="outline" size="sm" onClick={() => refetch()}>
              Retry
            </Button>
          </div>
        ) : !currentNode ? (
          <div className="text-center space-y-2 max-w-sm border border-dashed rounded-lg p-6">
            <GitGraph className="w-8 h-8 text-muted-foreground/50 mx-auto" />
            <p className="text-xs font-medium text-foreground">No Relationship Tuples in Nexus</p>
            <p className="text-[11px] text-muted-foreground">
              Add relation tuples in the Tuples tab or enter <code className="font-mono">namespace:object#relation</code>{" "}
              above to expand a live Zanzibar tree.
            </p>
          </div>
        ) : (
          <div
            className="transition-transform duration-200 flex flex-col items-center gap-6"
            style={{ transform: `scale(${zoom})` }}
          >
            {/* Incoming parent relations */}
            {incomingLinks.length > 0 && (
              <div className="flex flex-wrap justify-center gap-4">
                {incomingLinks.map((inc, idx) => (
                  <div key={`${inc.sourceId}-${inc.relation}-${idx}`} className="flex flex-col items-center gap-1.5">
                    <button
                      type="button"
                      onClick={() => {
                        setSelectedNode(inc.sourceId);
                        setSearchTarget(inc.sourceId);
                      }}
                      className="px-3 py-2 rounded-lg border bg-card hover:bg-muted/40 hover:border-primary/50 transition-all text-left shadow-sm flex items-center gap-2 group"
                    >
                      {getNodeIcon(inc.sourceType)}
                      <span className="text-xs font-mono font-medium text-foreground group-hover:text-primary">
                        {inc.sourceId}
                      </span>
                    </button>
                    <Badge variant="outline" className="font-mono text-[10px]">
                      #{inc.relation}
                    </Badge>
                    <ArrowRight className="w-3 h-3 text-muted-foreground rotate-90" aria-hidden="true" />
                  </div>
                ))}
              </div>
            )}

            {/* Focused Root Node */}
            <div className="p-4 rounded-xl border-2 border-primary bg-card shadow-lg flex items-center gap-3">
              {getNodeIcon(currentNode.type)}
              <div>
                <span className="text-[10px] text-muted-foreground block uppercase font-mono">
                  Active Neighborhood Focus
                </span>
                <span className="text-sm font-semibold font-mono text-foreground">{currentNode.label}</span>
              </div>
              <Badge variant="outline" className="text-xs uppercase ml-2">
                {currentNode.type}
              </Badge>
            </div>

            {/* Outgoing Relations / Edges */}
            {currentNode.relations.length > 0 ? (
              <div className="flex flex-wrap justify-center gap-6">
                {currentNode.relations.map((rel, idx) => (
                  <div key={`${rel.relation}-${rel.targetId}-${idx}`} className="flex flex-col items-center gap-2">
                    <div className="h-6 w-px bg-primary/40" />
                    <Badge variant="secondary" className="font-mono text-xs shadow-sm">
                      {rel.relation}
                    </Badge>
                    <ArrowRight className="w-3.5 h-3.5 text-primary rotate-90" aria-hidden="true" />

                    <button
                      type="button"
                      onClick={() => {
                        setSelectedNode(rel.targetId);
                        setSearchTarget(rel.targetId);
                      }}
                      className="p-3 rounded-lg border bg-card hover:bg-muted/40 hover:border-primary/50 transition-all text-left shadow-sm flex items-center gap-2 group cursor-pointer"
                    >
                      {getNodeIcon(rel.targetType)}
                      <div>
                        <span className="text-xs font-mono font-medium block text-foreground group-hover:text-primary transition-colors">
                          {rel.targetId}
                        </span>
                        <span className="text-[10px] text-muted-foreground uppercase">{rel.targetType}</span>
                      </div>
                    </button>
                  </div>
                ))}
              </div>
            ) : (
              <div className="text-xs text-muted-foreground border border-dashed rounded p-3">
                Leaf node — no outgoing relations found in Nexus tuples.
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
