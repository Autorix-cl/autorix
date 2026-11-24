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
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from "@/components/ui/card";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
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

  const curlCommand = `curl -X GET "https://api.autorix.internal${descriptor.basePath}/${record.id}" \\
  -H "Authorization: Bearer <YOUR_API_KEY>" \\
  -H "Content-Type: application/json"`;

  const handleCopyCurl = () => {
    navigator.clipboard.writeText(curlCommand);
    setCopiedCurl(true);
    setTimeout(() => setCopiedCurl(false), 2000);
  };

  return (
    <div className="space-y-6 max-w-5xl mx-auto">
      {/* Top Header */}
      <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between border-b border-border/60 pb-4">
        <div className="flex items-center gap-3">
          {onBack && (
            <Button type="button" variant="ghost" size="icon-sm" onClick={onBack} aria-label="Go back" className="h-8 w-8">
              <ArrowLeft className="h-4 w-4" aria-hidden="true" />
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
                  <CardTitle className="text-sm font-semibold">API Equivalence</CardTitle>
                  <CardDescription className="text-xs">Equivalent CLI and REST invocation to fetch or manipulate this resource</CardDescription>
                </div>
                <Button type="button" variant="outline" size="sm" onClick={handleCopyCurl} className="h-7 gap-1.5 text-xs">
                  {copiedCurl ? <Check className="h-3.5 w-3.5 text-emerald-400" aria-hidden="true" /> : <Copy className="h-3.5 w-3.5" aria-hidden="true" />}
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

      <ConfirmDialog
        open={deleteConfirmOpen}
        onOpenChange={setDeleteConfirmOpen}
        title="Confirm Irreversible Deletion"
        description={
          <>
            This operation cannot be undone. To confirm deletion of{" "}
            <span className="font-mono font-semibold text-foreground">{title}</span>, enter its identifier below.
          </>
        }
        confirmText={record.id}
        confirmTextAlt={title}
        confirmLabel="Delete Resource"
        onConfirm={() => {
          setDeleteConfirmOpen(false);
          onDelete?.();
        }}
      />
    </div>
  );
}
