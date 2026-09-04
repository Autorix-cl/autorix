"use client";

import * as React from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import {
  FileCode,
  Sparkles,
  Plus,
  Save,
  Trash2,
  CheckCircle2,
  AlertCircle,
  Copy,
  Check,
  RotateCcw,
  Layers,
  Eye,
  Sliders,
  Code2,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { CodeEditor } from "@/components/ui/code-editor";
import { DynamicSchemaForm, type DynamicSchemaFormProps } from "./schema-form";
import { toast } from "sonner";

export interface SchemaStudioProps {
  onSchemaCreated?: (schemaId: string) => void;
}

interface IdentitySchemaItem {
  id: string;
  name: string;
  schema: Record<string, unknown>;
  version?: number;
  created_at?: string;
  updated_at?: string;
}

const PRESET_TEMPLATES: Record<
  string,
  { name: string; description: string; schema: Record<string, unknown> }
> = {
  customer: {
    name: "SaaS Customer / End-User",
    description: "Standard end-user with email, username login identifiers, billing country, and consent",
    schema: {
      $id: "https://schemas.autorix.io/customer.identity.schema.json",
      $schema: "http://json-schema.org/draft-07/schema#",
      title: "Customer Identity",
      type: "object",
      properties: {
        traits: {
          type: "object",
          additionalProperties: false,
          properties: {
            email: {
              type: "string",
              format: "email",
              title: "Email Address",
              "autorix.io/credentials": {
                password: { identifier: true },
              },
            },
            username: {
              type: "string",
              title: "Username Handle",
              description: "Unique public identifier for login",
              "autorix.io/credentials": {
                password: { identifier: true },
              },
            },
            name: {
              type: "object",
              title: "Full Name",
              properties: {
                first: { type: "string", title: "First Name" },
                last: { type: "string", title: "Last Name" },
              },
              required: ["first"],
            },
            billing_country: {
              type: "string",
              title: "Billing Country",
              enum: ["US", "CA", "GB", "DE", "FR", "ES", "BR", "MX", "AR", "CL"],
            },
            marketing_consent: {
              type: "boolean",
              title: "Receive product updates & security alerts",
            },
          },
          required: ["email"],
        },
      },
    },
  },
  b2b_partner: {
    name: "B2B Partner / Vendor Workload",
    description: "Enterprise tenant or partner with company tax ID, SLA tier, and webhook URL",
    schema: {
      $id: "https://schemas.autorix.io/partner.identity.schema.json",
      $schema: "http://json-schema.org/draft-07/schema#",
      title: "B2B Partner Identity",
      type: "object",
      properties: {
        traits: {
          type: "object",
          additionalProperties: false,
          properties: {
            email: {
              type: "string",
              format: "email",
              title: "Technical Lead Email",
              "autorix.io/credentials": {
                password: { identifier: true },
              },
            },
            company_name: {
              type: "string",
              title: "Organization Name",
            },
            tax_identifier: {
              type: "string",
              title: "Tax ID / VAT Registration",
            },
            sla_tier: {
              type: "string",
              title: "SLA Tier",
              enum: ["Standard Support", "Gold 99.9%", "Platinum 24/7 Dedicated"],
            },
            webhook_url: {
              type: "string",
              format: "uri",
              title: "Security Event Webhook URL",
            },
            production_access: {
              type: "boolean",
              title: "Production Mesh Authorization Granted",
            },
          },
          required: ["email", "company_name", "sla_tier"],
        },
      },
    },
  },
  employee: {
    name: "Corporate Employee / Staff",
    description: "Internal staff member with employee badge ID, organizational department, and operator role",
    schema: {
      $id: "https://schemas.autorix.io/employee.identity.schema.json",
      $schema: "http://json-schema.org/draft-07/schema#",
      title: "Corporate Employee",
      type: "object",
      properties: {
        traits: {
          type: "object",
          additionalProperties: false,
          properties: {
            email: {
              type: "string",
              format: "email",
              title: "Corporate Email Address",
              "autorix.io/credentials": {
                password: { identifier: true },
              },
            },
            employee_id: {
              type: "string",
              title: "Employee Badge ID",
            },
            name: {
              type: "object",
              title: "Legal Name",
              properties: {
                first: { type: "string", title: "First Name" },
                last: { type: "string", title: "Last Name" },
              },
              required: ["first"],
            },
            department: {
              type: "string",
              title: "Division / Department",
              enum: ["Infrastructure", "Security Operations", "Core Engineering", "Compliance", "Executive"],
            },
            role: {
              type: "string",
              title: "IAM Operator Role",
              enum: ["admin", "operator", "auditor", "viewer"],
            },
            security_clearance: {
              type: "boolean",
              title: "Elevated Security Clearance",
            },
          },
          required: ["email", "employee_id"],
        },
      },
    },
  },
  iot_machine: {
    name: "IoT Device / Edge Workload",
    description: "Cryptographically authenticated edge device, hardware serial, and environment",
    schema: {
      $id: "https://schemas.autorix.io/iot.identity.schema.json",
      $schema: "http://json-schema.org/draft-07/schema#",
      title: "IoT / Edge Machine Subject",
      type: "object",
      properties: {
        traits: {
          type: "object",
          additionalProperties: false,
          properties: {
            device_id: {
              type: "string",
              title: "Device Serial / Hardware UUID",
              "autorix.io/credentials": {
                password: { identifier: true },
              },
            },
            model_name: {
              type: "string",
              title: "Hardware Model",
            },
            firmware_version: {
              type: "string",
              title: "Firmware Revision",
            },
            deployment_zone: {
              type: "string",
              title: "Deployment Zone",
              enum: ["edge-gateway", "industrial-floor", "datacenter-rack", "field-roaming"],
            },
            telemetry_streaming: {
              type: "boolean",
              title: "Telemetry Ingestion Enabled",
            },
          },
          required: ["device_id", "model_name"],
        },
      },
    },
  },
};

export function SchemaStudio({ onSchemaCreated }: SchemaStudioProps) {
  const queryClient = useQueryClient();

  // 1. Fetch registered schemas from Ego
  const { data: schemas = [] } = useQuery<IdentitySchemaItem[]>({
    queryKey: ["identity-schemas"],
    queryFn: async () => {
      const res = await fetch("/api/identities/schemas");
      if (!res.ok) return [];
      const json = await res.json();
      return Array.isArray(json) ? json : json.data || [];
    },
  });

  const [selectedSchemaId, setSelectedSchemaId] = React.useState<string>("default");
  const [isCreatingNew, setIsCreatingNew] = React.useState<boolean>(false);
  const lastLoadedSchemaIdRef = React.useRef<string | null>(null);

  // Schema form state
  const [schemaId, setSchemaId] = React.useState<string>("");
  const [schemaName, setSchemaName] = React.useState<string>("");
  const [rawJson, setRawJson] = React.useState<string>("");
  const [parsedSchema, setParsedSchema] = React.useState<DynamicSchemaFormProps["schema"] | null>(null);
  const [jsonError, setJsonError] = React.useState<string | null>(null);

  // Form preview sandbox state
  const [previewValues, setPreviewValues] = React.useState<Record<string, unknown>>({});
  const [isSaving, setIsSaving] = React.useState<boolean>(false);
  const [isDeleting, setIsDeleting] = React.useState<boolean>(false);
  const [copied, setCopied] = React.useState<boolean>(false);

  // Synchronize when selected schema changes
  React.useEffect(() => {
    if (isCreatingNew) return;
    if (schemas.length === 0) return;

    if (lastLoadedSchemaIdRef.current === selectedSchemaId) {
      return;
    }

    const current = schemas.find((s) => s.id === selectedSchemaId) || schemas[0];
    if (current) {
      lastLoadedSchemaIdRef.current = current.id;
      setSelectedSchemaId(current.id);
      setSchemaId(current.id);
      setSchemaName(current.name);
      const formatted = JSON.stringify(current.schema, null, 2);
      setRawJson(formatted);
      setParsedSchema(current.schema);
      setJsonError(null);
      setPreviewValues({});
    }
  }, [schemas, selectedSchemaId, isCreatingNew]);

  // Real-time JSON validation
  const handleJsonChange = (val: string) => {
    setRawJson(val);
    try {
      const parsed = JSON.parse(val);
      if (typeof parsed !== "object" || parsed === null) {
        setJsonError("Schema root must be a valid JSON object");
        return;
      }
      if (!parsed.properties?.traits) {
        setJsonError("Warning: Ory Kratos requires 'properties.traits' object for identity attributes");
      } else {
        setJsonError(null);
      }
      setParsedSchema(parsed);
    } catch (err: unknown) {
      setJsonError(err instanceof Error ? err.message : "Invalid JSON syntax");
    }
  };

  const handleApplyPreset = (key: string) => {
    const preset = PRESET_TEMPLATES[key];
    if (!preset) return;

    setIsCreatingNew(true);
    setSchemaId(key);
    setSchemaName(preset.name);
    const formatted = JSON.stringify(preset.schema, null, 2);
    setRawJson(formatted);
    setParsedSchema(preset.schema);
    setJsonError(null);
    setPreviewValues({});
    toast.info(`Loaded preset template: ${preset.name}`);
  };

  const handleStartNew = () => {
    setIsCreatingNew(true);
    setSchemaId("");
    setSchemaName("");
    handleApplyPreset("customer");
  };

  const handleFormatJson = () => {
    try {
      const parsed = JSON.parse(rawJson);
      const formatted = JSON.stringify(parsed, null, 2);
      setRawJson(formatted);
      setJsonError(null);
      toast.success("JSON formatted successfully");
    } catch {
      toast.error("Cannot format invalid JSON");
    }
  };

  const handleCopyJson = async () => {
    try {
      await navigator.clipboard.writeText(rawJson);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
      toast.success("Schema copied to clipboard");
    } catch {
      toast.error("Failed to copy schema");
    }
  };

  const handleSaveSchema = async () => {
    if (!schemaId.trim()) {
      toast.error("Schema ID is required (e.g., 'customer_v1')");
      return;
    }
    if (!schemaName.trim()) {
      toast.error("Schema Name is required");
      return;
    }
    if (jsonError && !jsonError.startsWith("Warning")) {
      toast.error(`Please fix JSON errors before saving: ${jsonError}`);
      return;
    }

    let schemaObj: Record<string, unknown>;
    try {
      schemaObj = JSON.parse(rawJson);
    } catch {
      toast.error("Invalid JSON content");
      return;
    }

    setIsSaving(true);
    try {
      const existing = schemas.find((s) => s.id === schemaId);
      const url = existing ? `/api/identities/schemas/${schemaId}` : "/api/identities/schemas";
      const method = existing ? "PATCH" : "POST";

      const res = await fetch(url, {
        method,
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          id: schemaId,
          name: schemaName,
          schema: schemaObj,
        }),
      });

      if (!res.ok) {
        const errorData = await res.json().catch(() => ({}));
        throw new Error(errorData.error || `Failed to save schema (${res.status})`);
      }

      toast.success(`Schema '${schemaId}' saved successfully!`);
      await queryClient.invalidateQueries({ queryKey: ["identity-schemas"] });
      setIsCreatingNew(false);
      setSelectedSchemaId(schemaId);
      if (onSchemaCreated) {
        onSchemaCreated(schemaId);
      }
    } catch (err: unknown) {
      toast.error(err instanceof Error ? err.message : "Failed to register schema in Ego");
    } finally {
      setIsSaving(false);
    }
  };

  const handleDeleteSchema = async () => {
    if (schemaId === "default") {
      toast.error("The 'default' schema is the engine system fallback and cannot be deleted");
      return;
    }

    if (!confirm(`Are you sure you want to permanently delete identity schema '${schemaId}'?`)) {
      return;
    }

    setIsDeleting(true);
    try {
      const res = await fetch(`/api/identities/schemas/${schemaId}`, {
        method: "DELETE",
      });
      if (!res.ok) {
        const errorData = await res.json().catch(() => ({}));
        throw new Error(errorData.error || `Failed to delete schema (${res.status})`);
      }

      toast.success(`Schema '${schemaId}' deleted`);
      await queryClient.invalidateQueries({ queryKey: ["identity-schemas"] });
      setSelectedSchemaId("default");
      setIsCreatingNew(false);
    } catch (err: unknown) {
      toast.error(err instanceof Error ? err.message : "Failed to delete schema");
    } finally {
      setIsDeleting(false);
    }
  };

  return (
    <div className="space-y-6">
      {/* Schema Catalog Bar */}
      <Card className="bg-card/80 border-border/80 shadow-xs">
        <CardHeader className="p-4 pb-3">
          <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3">
            <div className="space-y-0.5">
              <CardTitle className="text-sm font-semibold flex items-center gap-2">
                <FileCode className="h-4 w-4 text-blue-400" />
                <span>Ego Identity Schemas Catalog</span>
              </CardTitle>
              <CardDescription className="text-xs">
                Declarative Ory Kratos Draft-07 JSON Schemas governing user profiles and credential identifiers
              </CardDescription>
            </div>

            <div className="flex items-center gap-2 flex-wrap">
              <Button
                variant={isCreatingNew ? "secondary" : "outline"}
                size="sm"
                onClick={handleStartNew}
                className="h-8 gap-1.5 text-xs border-dashed"
              >
                <Plus className="h-3.5 w-3.5 text-cyan-400" />
                <span>New Schema</span>
              </Button>
            </div>
          </div>
        </CardHeader>

        <CardContent className="p-4 pt-0">
          <div className="flex items-center gap-2 overflow-x-auto pb-1">
            {schemas.map((s) => {
              const isActive = !isCreatingNew && selectedSchemaId === s.id;
              return (
                <Button
                  key={s.id}
                  variant={isActive ? "secondary" : "ghost"}
                  size="sm"
                  onClick={() => {
                    setIsCreatingNew(false);
                    setSelectedSchemaId(s.id);
                  }}
                  className={`h-8 text-xs font-mono shrink-0 gap-1.5 ${
                    isActive ? "border border-blue-500/30 bg-blue-500/10 text-blue-400" : ""
                  }`}
                >
                  <Layers className="h-3.5 w-3.5 text-muted-foreground" />
                  <span>{s.name}</span>
                  <span className="text-[10px] text-muted-foreground/70">({s.id})</span>
                </Button>
              );
            })}
            {isCreatingNew && (
              <Badge variant="cyan" className="h-7 px-2.5 text-xs font-mono shrink-0 gap-1">
                <Sparkles className="h-3 w-3" />
                <span>New Draft ({schemaId || "untitled"})</span>
              </Badge>
            )}
          </div>
        </CardContent>
      </Card>

      {/* Schema Editor & Preview Workbench */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
        {/* Left Column: Schema Metadata & JSON Editor (7 cols) */}
        <div className="lg:col-span-7 space-y-4">
          <Card className="bg-card/80 border-border/80">
            <CardHeader className="p-4 pb-3">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <Sliders className="h-4 w-4 text-cyan-400" />
                  <h3 className="text-xs font-semibold text-foreground uppercase tracking-wider">
                    Schema Definition & Identifiers
                  </h3>
                </div>

                {/* Preset Templates Quick Menu */}
                <div className="flex items-center gap-1.5">
                  <span className="text-[11px] text-muted-foreground hidden sm:inline">Templates:</span>
                  {Object.keys(PRESET_TEMPLATES).map((key) => (
                    <Button
                      key={key}
                      variant="outline"
                      size="sm"
                      onClick={() => handleApplyPreset(key)}
                      className="h-6 px-2 text-[10px] font-medium"
                      title={PRESET_TEMPLATES[key].description}
                    >
                      {key === "b2b_partner"
                        ? "B2B"
                        : key === "iot_machine"
                        ? "IoT"
                        : key.charAt(0).toUpperCase() + key.slice(1)}
                    </Button>
                  ))}
                </div>
              </div>
            </CardHeader>

            <CardContent className="p-4 pt-0 space-y-4">
              {/* Metadata Inputs */}
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                <div className="space-y-1.5">
                  <Label htmlFor="schemaId" className="text-xs font-medium">
                    Schema Identifier (ID)
                  </Label>
                  <Input
                    id="schemaId"
                    value={schemaId}
                    onChange={(e) => setSchemaId(e.target.value.toLowerCase().replace(/[^a-z0-9_-]/g, ""))}
                    placeholder="e.g. customer_v1"
                    disabled={!isCreatingNew && schemaId === "default"}
                    className="h-8 text-xs font-mono"
                  />
                </div>

                <div className="space-y-1.5">
                  <Label htmlFor="schemaName" className="text-xs font-medium">
                    Schema Display Name
                  </Label>
                  <Input
                    id="schemaName"
                    value={schemaName}
                    onChange={(e) => setSchemaName(e.target.value)}
                    placeholder="e.g. Customer Profile"
                    className="h-8 text-xs"
                  />
                </div>
              </div>

              {/* JSON Editor Header Toolbar */}
              <div className="flex items-center justify-between pt-1">
                <div className="flex items-center gap-2">
                  <Code2 className="h-3.5 w-3.5 text-muted-foreground" />
                  <span className="text-xs font-medium text-foreground">JSON Schema Draft-07</span>
                  {jsonError ? (
                    <Badge variant="rose" className="h-5 text-[10px] font-mono gap-1 py-0 px-1.5">
                      <AlertCircle className="h-2.5 w-2.5" />
                      <span>{jsonError.startsWith("Warning") ? "Warning" : "Syntax Error"}</span>
                    </Badge>
                  ) : (
                    <Badge variant="success" className="h-5 text-[10px] font-mono gap-1 py-0 px-1.5">
                      <CheckCircle2 className="h-2.5 w-2.5" />
                      <span>Valid Schema</span>
                    </Badge>
                  )}
                </div>

                <div className="flex items-center gap-1.5">
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={handleFormatJson}
                    className="h-7 px-2 text-xs gap-1 text-muted-foreground hover:text-foreground"
                    title="Prettify JSON indentation"
                  >
                    <RotateCcw className="h-3 w-3" />
                    <span>Format</span>
                  </Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={handleCopyJson}
                    className="h-7 px-2 text-xs gap-1 text-muted-foreground hover:text-foreground"
                    title="Copy schema JSON to clipboard"
                  >
                    {copied ? <Check className="h-3 w-3 text-emerald-400" /> : <Copy className="h-3 w-3" />}
                    <span>{copied ? "Copied" : "Copy"}</span>
                  </Button>
                </div>
              </div>

              {/* Real Code Editor */}
              <div className="relative">
                <CodeEditor
                  value={rawJson}
                  onChange={handleJsonChange}
                  language="json"
                  height="340px"
                  className="font-mono text-xs border-border/80 shadow-inner"
                />
              </div>

              {/* Error / Warning Alert Banner */}
              {jsonError && (
                <div className="p-2.5 rounded-md border border-rose-500/20 bg-rose-500/10 text-rose-300 text-xs flex items-start gap-2">
                  <AlertCircle className="h-4 w-4 shrink-0 mt-0.5 text-rose-400" />
                  <div className="space-y-0.5 font-mono text-[11px] break-all">
                    <span>{jsonError}</span>
                  </div>
                </div>
              )}

              {/* Action Buttons */}
              <div className="flex items-center justify-between pt-2 border-t border-border/60">
                <div>
                  {!isCreatingNew && schemaId !== "default" && (
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={handleDeleteSchema}
                      disabled={isDeleting}
                      className="h-8 text-xs text-rose-400 hover:text-rose-300 hover:bg-rose-500/10 gap-1.5"
                    >
                      <Trash2 className="h-3.5 w-3.5" />
                      <span>{isDeleting ? "Deleting..." : "Delete Schema"}</span>
                    </Button>
                  )}
                </div>

                <div className="flex items-center gap-2">
                  <Button
                    size="sm"
                    onClick={handleSaveSchema}
                    disabled={isSaving || !!(jsonError && !jsonError.startsWith("Warning"))}
                    className="h-8 text-xs gap-1.5 bg-blue-600 hover:bg-blue-700 text-white font-medium shadow-xs"
                  >
                    <Save className="h-3.5 w-3.5" />
                    <span>{isSaving ? "Saving..." : isCreatingNew ? "Register Schema" : "Update Schema"}</span>
                  </Button>
                </div>
              </div>
            </CardContent>
          </Card>
        </div>

        {/* Right Column: Live Interactive Dynamic Form Preview (5 cols) */}
        <div className="lg:col-span-5 space-y-4">
          <Card className="bg-card/80 border-border/80 flex flex-col h-full">
            <CardHeader className="p-4 pb-3 border-b border-border/60">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <Eye className="h-4 w-4 text-emerald-400" />
                  <CardTitle className="text-xs font-semibold uppercase tracking-wider">
                    Interactive Form Preview
                  </CardTitle>
                </div>
                <Badge variant="cyan" className="text-[10px] font-mono">
                  Live Preview
                </Badge>
              </div>
              <CardDescription className="text-xs">
                How identities rendered with this schema appear during registration & trait updates
              </CardDescription>
            </CardHeader>

            <CardContent className="p-4 space-y-4 flex-1 flex flex-col">
              {parsedSchema && parsedSchema.properties?.traits ? (
                <div className="space-y-4 flex-1">
                  <DynamicSchemaForm
                    schema={parsedSchema}
                    value={previewValues}
                    onChange={setPreviewValues}
                  />

                  {/* Simulated Traits State */}
                  <div className="space-y-1.5 pt-2 border-t border-border/60">
                    <div className="flex items-center justify-between text-[11px] text-muted-foreground">
                      <span className="font-mono">Simulated Trait Payload</span>
                      <span className="text-[10px]">{Object.keys(previewValues).length} attributes populated</span>
                    </div>
                    <pre className="p-3 rounded-md bg-muted/40 border border-border/60 font-mono text-[11px] text-foreground max-h-36 overflow-y-auto leading-relaxed">
                      {JSON.stringify(previewValues, null, 2)}
                    </pre>
                  </div>
                </div>
              ) : (
                <div className="flex flex-col items-center justify-center py-16 text-center space-y-2 text-muted-foreground">
                  <AlertCircle className="h-6 w-6 text-muted-foreground/60" />
                  <p className="text-xs">
                    Define <code className="text-cyan-400 font-mono">properties.traits</code> in the editor to preview the generated dynamic form.
                  </p>
                </div>
              )}
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}
