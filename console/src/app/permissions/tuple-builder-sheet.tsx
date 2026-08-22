"use client";

import * as React from "react";
import { Loader2, Plus, Database } from "lucide-react";
import { useQueryClient } from "@tanstack/react-query";
import { useApiMutation } from "@/lib/query/use-api-mutation";
import { fetchAndParse } from "@/lib/api/schema";
import { writeTuplesResponseSchema } from "@/lib/api/schemas/nexus";

import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetDescription,
} from "@/components/ui/sheet";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

interface TupleBuilderSheetProps {
  isOpen: boolean;
  onOpenChange: (open: boolean) => void;
}

export function TupleBuilderSheet({ isOpen, onOpenChange }: TupleBuilderSheetProps) {
  const queryClient = useQueryClient();

  const [namespace, setNamespace] = React.useState("document");
  const [object, setObject] = React.useState("");
  const [relation, setRelation] = React.useState("viewer");
  const [subjectNamespace, setSubjectNamespace] = React.useState("user");
  const [subjectId, setSubjectId] = React.useState("");
  const [subjectRelation, setSubjectRelation] = React.useState("");
  const [caveatName, setCaveatName] = React.useState("");

  const createMutation = useApiMutation(
    (vars: {
      namespace: string;
      object: string;
      relation: string;
      subjectNamespace: string;
      subjectId: string;
      subjectRelation?: string;
      caveatName?: string;
    }) =>
      fetchAndParse("/api/permissions", writeTuplesResponseSchema, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(vars),
      }),
    {
      successMessage: () => "Tuple created successfully.",
      onSuccess: () => {
        queryClient.invalidateQueries({ queryKey: ["nexus-tuples"] });
        onOpenChange(false);
        setObject("");
        setSubjectId("");
      },
    }
  );

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!namespace || !object || !relation || !subjectNamespace || !subjectId) return;

    createMutation.mutate({
      namespace,
      object,
      relation,
      subjectNamespace,
      subjectId,
      subjectRelation: subjectRelation || undefined,
      caveatName: caveatName || undefined,
    });
  };

  return (
    <Sheet open={isOpen} onOpenChange={onOpenChange}>
      <SheetContent className="overflow-y-auto sm:max-w-md">
        <SheetHeader className="mb-6">
          <div className="flex items-center gap-2">
            <Database className="h-5 w-5 text-purple-400" />
            <SheetTitle>Create Tuple</SheetTitle>
          </div>
          <SheetDescription>
            Add a new relationship tuple to the Zanzibar engine.
          </SheetDescription>
        </SheetHeader>

        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-1.5">
            <Label htmlFor="namespace">Namespace</Label>
            <Input
              id="namespace"
              value={namespace}
              onChange={(e) => setNamespace(e.target.value)}
              required
            />
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="object">Object ID</Label>
            <Input
              id="object"
              value={object}
              onChange={(e) => setObject(e.target.value)}
              placeholder="e.g. document_123"
              required
            />
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="relation">Relation</Label>
            <Input
              id="relation"
              value={relation}
              onChange={(e) => setRelation(e.target.value)}
              required
            />
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label htmlFor="subjectNamespace">Subject Namespace</Label>
              <Input
                id="subjectNamespace"
                value={subjectNamespace}
                onChange={(e) => setSubjectNamespace(e.target.value)}
                required
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="subjectId">Subject ID</Label>
              <Input
                id="subjectId"
                value={subjectId}
                onChange={(e) => setSubjectId(e.target.value)}
                placeholder="e.g. alice"
                required
              />
            </div>
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="subjectRelation">Subject Relation (Optional)</Label>
            <Input
              id="subjectRelation"
              value={subjectRelation}
              onChange={(e) => setSubjectRelation(e.target.value)}
              placeholder="e.g. member"
            />
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="caveatName">Caveat Name (Optional)</Label>
            <Input
              id="caveatName"
              value={caveatName}
              onChange={(e) => setCaveatName(e.target.value)}
            />
          </div>

          <Button
            type="submit"
            variant="purple"
            disabled={createMutation.isPending}
            className="w-full gap-2 mt-4"
          >
            {createMutation.isPending ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <Plus className="h-4 w-4" />
            )}
            <span>{createMutation.isPending ? "Creating..." : "Create Tuple"}</span>
          </Button>
        </form>
      </SheetContent>
    </Sheet>
  );
}
