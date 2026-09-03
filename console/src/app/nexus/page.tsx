"use client";

import * as React from "react";
import { useCapabilities } from "@/lib/capabilities/capability-context";
import { NotConnectedEngine } from "@/components/resources/not-connected-engine";
import { SchemaEditor } from "./schema-editor";
import { TupleGrid } from "./tuple-grid";
import { CheckSimulator } from "./check-simulator";
import { toast } from "sonner";
import { Loader2, LayoutGrid, Database, GitGraph, ShieldAlert } from "lucide-react";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs";
import { RelationshipGraph } from "./relationship-graph";
import { CaveatRegistry } from "./caveat-registry";

import { 
  useNexusSchema, 
  useSaveNexusSchema,
  useNexusTuples,
  useAddNexusTuple,
  useDeleteNexusTuples,
  useNexusCheck
} from "@/hooks/use-nexus";

export default function NexusPage() {
  const { isEngineConnected } = useCapabilities();

  // Queries
  const { data: schema, isLoading: isLoadingSchema } = useNexusSchema();
  const { data: tuples, isLoading: isLoadingTuples } = useNexusTuples();

  // Mutations
  const { mutateAsync: saveSchema } = useSaveNexusSchema();
  const { mutateAsync: addTuple } = useAddNexusTuple();
  const { mutateAsync: deleteTuples } = useDeleteNexusTuples();
  const { mutateAsync: checkAccess } = useNexusCheck();

  if (!isEngineConnected("nexus")) {
    return (
      <div className="space-y-6">
        <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
          <div>
            <h1 className="text-xl font-bold tracking-tight text-foreground sm:text-2xl">Nexus (Authorization)</h1>
            <p className="text-xs text-muted-foreground mt-1">ReBAC Identity & Permissions Graph based on Zanzibar.</p>
          </div>
        </div>
        <NotConnectedEngine
          engineType="nexus"
          engineName="Autorix Nexus (Authorization)"
          description="High-performance relationship-based access control engine."
        />
      </div>
    );
  }

  return (
    <div className="flex flex-col h-[calc(100vh-120px)] overflow-hidden">
      {/* Page Header */}
      <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between mb-3 flex-shrink-0">
        <div>
          <h1 className="text-xl font-bold tracking-tight text-foreground sm:text-2xl">Nexus Playground</h1>
          <p className="text-xs text-muted-foreground mt-1">Design, test, and manage your authorization schema and Zanzibar relationship graph.</p>
        </div>
      </div>

      {/* Tabs Container */}
      <Tabs defaultValue="playground" className="flex-1 flex flex-col overflow-hidden">
        <div className="flex items-center justify-between mb-3 flex-shrink-0">
          <TabsList className="bg-muted/60 p-1">
            <TabsTrigger value="playground" className="text-xs">
              <LayoutGrid className="w-3.5 h-3.5 mr-1.5" /> Playground
            </TabsTrigger>
            <TabsTrigger value="tuples" className="text-xs">
              <Database className="w-3.5 h-3.5 mr-1.5" /> Tuples Browser
            </TabsTrigger>
            <TabsTrigger value="graph" className="text-xs">
              <GitGraph className="w-3.5 h-3.5 mr-1.5" /> Relationship Graph
            </TabsTrigger>
            <TabsTrigger value="caveats" className="text-xs">
              <ShieldAlert className="w-3.5 h-3.5 mr-1.5" /> CEL Caveats
            </TabsTrigger>
          </TabsList>
        </div>

        {/* Tab 1: Playground (Split Schema & Simulator) */}
        <TabsContent value="playground" className="flex-1 flex gap-6 overflow-hidden m-0 p-0">
          {/* Left Pane: Schema Editor */}
          <div className="w-1/2 flex flex-col min-w-[400px]">
            {isLoadingSchema ? (
              <div className="flex-1 flex items-center justify-center border rounded-md bg-muted/10">
                <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
              </div>
            ) : (
              <SchemaEditor 
                schema={schema || ""} 
                onSave={async (newSchema) => {
                  await saveSchema(newSchema);
                  toast.success("Schema validated and updated successfully.");
                }} 
              />
            )}
          </div>

          {/* Right Pane: Tuples & Simulator */}
          <div className="w-1/2 flex flex-col gap-6 overflow-y-auto pr-2 pb-6">
            <div className="h-[400px]">
              {isLoadingTuples ? (
                <div className="flex-1 h-full flex items-center justify-center border rounded-md bg-muted/10">
                  <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
                </div>
              ) : (
                <TupleGrid 
                  tuples={tuples || []} 
                  onAdd={async (data) => {
                    await addTuple(data);
                    toast.success(`Added tuple ${data.relation}`);
                  }}
                  onDelete={async (ids) => {
                    await deleteTuples(ids);
                    toast.success(`Deleted ${ids.length} tuples`);
                  }} 
                />
              )}
            </div>
            <div className="flex-1 min-h-[300px]">
              <CheckSimulator 
                onCheck={async (query) => {
                  const result = await checkAccess(query);
                  return result;
                }} 
              />
            </div>
          </div>
        </TabsContent>

        {/* Tab 2: Tuples Browser */}
        <TabsContent value="tuples" className="flex-1 overflow-y-auto m-0 p-0 pb-6">
          <TupleGrid 
            tuples={tuples || []} 
            onAdd={async (data) => {
              await addTuple(data);
              toast.success(`Added tuple ${data.relation}`);
            }}
            onDelete={async (ids) => {
              await deleteTuples(ids);
              toast.success(`Deleted ${ids.length} tuples`);
            }} 
          />
        </TabsContent>

        {/* Tab 3: Relationship Graph */}
        <TabsContent value="graph" className="flex-1 overflow-hidden m-0 p-0">
          <RelationshipGraph />
        </TabsContent>

        {/* Tab 4: CEL Caveats */}
        <TabsContent value="caveats" className="flex-1 overflow-y-auto m-0 p-0 pb-6">
          <CaveatRegistry />
        </TabsContent>
      </Tabs>
    </div>
  );
}

