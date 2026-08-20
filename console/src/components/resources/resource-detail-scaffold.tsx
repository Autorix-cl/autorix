"use client";

import * as React from "react";
import {
  FileCode2,
  Terminal,
  Trash2,
  Edit,
  ArrowLeft,
  Copy,
  Check,
  AlertTriangle,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from "@/components/ui/card";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs";
import { Input } from "@/components/ui/input";
import type { ResourceDescriptor } from "@/lib/resources/descriptor";

export interface ResourceDetailScaffoldProps<T extends Record<string, unknown> & { id: string }> {
  descriptor: ResourceDescriptor<T>;
  record: T;
  title: string;
  subtitle?: string;
  onBack?: () => void;
  onEdit?: () => void;
  onDelete?: () => void;
  children?: React.ReactNode;
}

export function ResourceDetailScaffold<T extends Record<string, unknown> & { id: string }>({
  descriptor,
  record,
  title,
  subtitle,
  onBack,
  onEdit,
  onDelete,
  children,
}: ResourceDetailScaffoldProps<T>) {
  const [copiedCurl, setCopiedCurl] = React.useState(false);
  const [deleteConfirmOpen, setDeleteConfirmOpen] = React.useState(false);
  const [deleteInput, setDeleteInput] = React.useState("");

  const curlCommand = `curl -X GET "https://api.autorix.internal${descriptor.basePath}/${record.id}" \\
  -H "Authorization: Bearer <YOUR_API_KEY>" \\
  -H "Content-Type: application/json"`;

  const handleCopyCurl = () => {
    navigator.clipboard.writeText(curlCommand);
    setCopiedCurl(true);
    setTimeout(() => setCopiedCurl(false), 2000);
  };

  const isDeleteConfirmed = deleteInput === record.id || deleteInput === title;

  return (
    <div className="space-y-6 max-w-5xl mx-auto">
      {/* Top Header */}
      <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between border-b border-border/60 pb-4">
        <div className="flex items-center gap-3">
          {onBack && (
            <Button variant="ghost" size="icon-sm" onClick={onBack} className="h-8 w-8">
              <ArrowLeft className="h-4 w-4" />
            </Button>
          )}
          <div>
            <div className="flex items-center gap-2">
              <h1 className="text-xl font-bold tracking-tight text-foreground">{title}</h1>
              <Badge variant="outline" className="font-mono text-xs uppercase">
                {descriptor.name}
              </Badge>
            </div>
            {subtitle && <p className="text-xs text-muted-foreground font-mono mt-0.5">{subtitle}</p>}
          </div>
        </div>

        <div className="flex items-center gap-2">
          {onEdit && (
            <Button variant="outline" size="sm" onClick={onEdit} className="h-8 gap-1.5 text-xs">
              <Edit className="h-3.5 w-3.5" />
              Edit
            </Button>
          )}
          {onDelete && (
            <Button
              variant="destructive"
              size="sm"
              onClick={() => setDeleteConfirmOpen(true)}
              className="h-8 gap-1.5 text-xs"
            >
              <Trash2 className="h-3.5 w-3.5" />
              Delete
            </Button>
          )}
        </div>
      </div>

      {/* Main Tabs */}
      <Tabs defaultValue="overview" className="space-y-4">
        <TabsList className="grid grid-cols-3 w-full max-w-sm">
          <TabsTrigger value="overview" className="text-xs">Overview</TabsTrigger>
          <TabsTrigger value="json" className="text-xs gap-1.5">
            <FileCode2 className="w-3.5 h-3.5" /> Raw JSON
          </TabsTrigger>
          <TabsTrigger value="api" className="text-xs gap-1.5">
            <Terminal className="w-3.5 h-3.5" /> API / cURL
          </TabsTrigger>
        </TabsList>

        <TabsContent value="overview" className="space-y-4">
          {children}
        </TabsContent>

        <TabsContent value="json" className="space-y-4">
          <Card className="border-border/80 bg-card">
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-semibold">Raw Resource Payload</CardTitle>
              <CardDescription className="text-xs">Direct JSON representation from engine store</CardDescription>
            </CardHeader>
            <CardContent>
              <pre className="p-4 rounded-lg bg-muted/60 border border-border text-xs font-mono overflow-x-auto text-emerald-400">
                {JSON.stringify(record, null, 2)}
              </pre>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="api" className="space-y-4">
          <Card className="border-border/80 bg-card">
            <CardHeader className="pb-2">
              <div className="flex items-center justify-between">
                <div>
                  <CardTitle className="text-sm font-semibold">API Equivalence (P4-S2-T4)</CardTitle>
                  <CardDescription className="text-xs">Equivalent CLI and REST invocation to fetch or manipulate this resource</CardDescription>
                </div>
                <Button variant="outline" size="sm" onClick={handleCopyCurl} className="h-7 gap-1.5 text-xs">
                  {copiedCurl ? <Check className="h-3.5 w-3.5 text-emerald-400" /> : <Copy className="h-3.5 w-3.5" />}
                  {copiedCurl ? "Copied" : "Copy cURL"}
                </Button>
              </div>
            </CardHeader>
            <CardContent>
              <pre className="p-4 rounded-lg bg-muted/60 border border-border text-xs font-mono overflow-x-auto text-amber-300">
                {curlCommand}
              </pre>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>

      {/* Destructive Action Modal (P4-S2-T5) */}
      {deleteConfirmOpen && (
        <div className="fixed inset-0 z-50 bg-black/60 backdrop-blur-sm flex items-center justify-center p-4">
          <Card className="w-full max-w-md border-red-500/40 bg-card shadow-2xl animate-in fade-in zoom-in-95">
            <CardHeader>
              <div className="flex items-center gap-2 text-rose-500">
                <AlertTriangle className="h-5 w-5" />
                <CardTitle className="text-base font-semibold">Confirm Irreversible Deletion</CardTitle>
              </div>
              <CardDescription className="text-xs text-muted-foreground mt-1">
                This operation cannot be undone. To confirm, type{" "}
                <span className="font-mono text-foreground font-bold">{record.id}</span> below.
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <Input
                placeholder={`Type ${record.id} to confirm`}
                value={deleteInput}
                onChange={(e) => setDeleteInput(e.target.value)}
                className="text-xs font-mono"
              />

              <div className="flex items-center justify-end gap-2">
                <Button variant="outline" size="sm" onClick={() => setDeleteConfirmOpen(false)} className="text-xs">
                  Cancel
                </Button>
                <Button
                  variant="destructive"
                  size="sm"
                  disabled={!isDeleteConfirmed}
                  onClick={() => {
                    setDeleteConfirmOpen(false);
                    onDelete?.();
                  }}
                  className="text-xs"
                >
                  Delete Resource
                </Button>
              </div>
            </CardContent>
          </Card>
        </div>
      )}
    </div>
  );
}
