"use client";

import * as React from "react";
import { GitGraph, Search, ZoomIn, ZoomOut, RotateCcw, User, FileText, Users, ArrowRight, ShieldCheck } from "lucide-react";
import { Button } from "@/components/ui/button";

import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";

export interface GraphNode {
  id: string;
  type: "user" | "document" | "organization" | "group" | "resource";
  label: string;
  relations: {
    relation: string;
    targetId: string;
    targetType: "user" | "document" | "organization" | "group" | "resource";
  }[];
}

interface RelationshipGraphProps {
  initialObjectId?: string;
}

export function RelationshipGraph({ initialObjectId = "document:doc_123" }: RelationshipGraphProps) {
  const [searchTarget, setSearchTarget] = React.useState(initialObjectId);
  const [zoom, setZoom] = React.useState(1);
  const [selectedNode, setSelectedNode] = React.useState<string | null>(initialObjectId);

  // Mock graph data for interactive neighborhood visualization
  const [nodes, setNodes] = React.useState<Record<string, GraphNode>>({
    "document:doc_123": {
      id: "document:doc_123",
      type: "document",
      label: "document:doc_123",
      relations: [
        { relation: "viewer", targetId: "user:alice", targetType: "user" },
        { relation: "editor", targetId: "user:bob", targetType: "user" },
        { relation: "parent", targetId: "organization:org_1", targetType: "organization" },
      ],
    },
    "organization:org_1": {
      id: "organization:org_1",
      type: "organization",
      label: "organization:org_1",
      relations: [
        { relation: "member", targetId: "group:devs", targetType: "group" },
        { relation: "admin", targetId: "user:charlie", targetType: "user" },
      ],
    },
    "group:devs": {
      id: "group:devs",
      type: "group",
      label: "group:devs",
      relations: [
        { relation: "member", targetId: "user:alice", targetType: "user" },
        { relation: "member", targetId: "user:dave", targetType: "user" },
      ],
    },
    "user:alice": {
      id: "user:alice",
      type: "user",
      label: "user:alice",
      relations: [],
    },
    "user:bob": {
      id: "user:bob",
      type: "user",
      label: "user:bob",
      relations: [],
    },
    "user:charlie": {
      id: "user:charlie",
      type: "user",
      label: "user:charlie",
      relations: [],
    },
    "user:dave": {
      id: "user:dave",
      type: "user",
      label: "user:dave",
      relations: [],
    },
  });

  const currentNode = nodes[selectedNode || ""] || nodes["document:doc_123"];

  const getNodeIcon = (type: string) => {
    switch (type) {
      case "user":
        return <User className="w-4 h-4 text-emerald-400" />;
      case "document":
        return <FileText className="w-4 h-4 text-blue-400" />;
      case "group":
        return <Users className="w-4 h-4 text-amber-400" />;
      case "organization":
        return <ShieldCheck className="w-4 h-4 text-purple-400" />;
      default:
        return <GitGraph className="w-4 h-4 text-muted-foreground" />;
    }
  };

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault();
    if (nodes[searchTarget]) {
      setSelectedNode(searchTarget);
    } else {
      // Create node on the fly for exploration
      const [type] = searchTarget.split(":");
      setNodes((prev) => ({
        ...prev,
        [searchTarget]: {
          id: searchTarget,
          type: (type as "user" | "document") || "resource",
          label: searchTarget,
          relations: [],
        },
      }));
      setSelectedNode(searchTarget);
    }
  };

  return (
    <div className="flex flex-col h-full bg-card rounded-md border text-card-foreground shadow-sm">
      <div className="p-4 border-b flex flex-wrap items-center justify-between gap-3">
        <div>
          <h3 className="font-semibold text-sm flex items-center gap-2">
            <GitGraph className="w-4 h-4 text-primary" />
            Relationship Graph Explorer
          </h3>
          <p className="text-xs text-muted-foreground mt-0.5">
            Explore inherited ReBAC access paths and Zanzibar object neighborhoods.
          </p>
        </div>

        <div className="flex items-center gap-2">
          <form onSubmit={handleSearch} className="flex items-center gap-2">
            <Input
              value={searchTarget}
              onChange={(e) => setSearchTarget(e.target.value)}
              placeholder="e.g. document:doc_123"
              className="h-8 text-xs font-mono w-[200px]"
            />
            <Button type="submit" size="sm" variant="secondary" className="h-8">
              <Search className="w-3.5 h-3.5 mr-1" /> Inspect
            </Button>
          </form>

          <div className="flex items-center gap-1 border-l pl-2">
            <Button
              variant="outline"
              size="icon"
              className="h-8 w-8"
              onClick={() => setZoom((z) => Math.min(1.5, z + 0.1))}
            >
              <ZoomIn className="w-3.5 h-3.5" />
            </Button>
            <Button
              variant="outline"
              size="icon"
              className="h-8 w-8"
              onClick={() => setZoom((z) => Math.max(0.6, z - 0.1))}
            >
              <ZoomOut className="w-3.5 h-3.5" />
            </Button>
            <Button
              variant="outline"
              size="icon"
              className="h-8 w-8"
              onClick={() => setZoom(1)}
            >
              <RotateCcw className="w-3.5 h-3.5" />
            </Button>
          </div>
        </div>
      </div>

      {/* Graph Canvas View */}
      <div className="flex-1 p-6 overflow-auto relative bg-muted/10 min-h-[360px] flex items-center justify-center">
        <div
          className="transition-transform duration-200 flex flex-col items-center gap-8"
          style={{ transform: `scale(${zoom})` }}
        >
          {/* Focused Root Node */}
          <div className="p-4 rounded-xl border-2 border-primary bg-card shadow-lg flex items-center gap-3 animate-in fade-in zoom-in-95">
            {getNodeIcon(currentNode.type)}
            <div>
              <span className="text-xs text-muted-foreground block uppercase font-mono">
                Active Neighborhood Focus
              </span>
              <span className="text-sm font-semibold font-mono text-foreground">
                {currentNode.label}
              </span>
            </div>
            <Badge variant="outline" className="text-xs uppercase ml-2">
              {currentNode.type}
            </Badge>
          </div>

          {/* Outgoing Relations / Edges */}
          {currentNode.relations.length > 0 ? (
            <div className="flex flex-wrap justify-center gap-6">
              {currentNode.relations.map((rel, idx) => (
                <div key={idx} className="flex flex-col items-center gap-2">
                  <div className="h-6 w-px bg-primary/40" />
                  <Badge variant="secondary" className="font-mono text-xs shadow-sm">
                    {rel.relation}
                  </Badge>
                  <ArrowRight className="w-3.5 h-3.5 text-primary rotate-90" />

                  <button
                    type="button"
                    onClick={() => setSelectedNode(rel.targetId)}
                    className="p-3 rounded-lg border bg-card hover:bg-muted/40 hover:border-primary/50 transition-all text-left shadow-sm flex items-center gap-2 group cursor-pointer"
                  >
                    {getNodeIcon(rel.targetType)}
                    <div>
                      <span className="text-xs font-mono font-medium block text-foreground group-hover:text-primary transition-colors">
                        {rel.targetId}
                      </span>
                      <span className="text-[10px] text-muted-foreground uppercase">
                        {rel.targetType}
                      </span>
                    </div>
                  </button>
                </div>
              ))}
            </div>
          ) : (
            <div className="text-xs text-muted-foreground border border-dashed rounded p-3">
              Leaf node — no outgoing relations configured.
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
