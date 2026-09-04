"use client";

import * as React from "react";
import {
  BookOpen,
  KeyRound,
  Layers,
  Terminal,
  ShieldCheck,
  CheckCircle,
  Database,
  Code2,
} from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { CodeBlock } from "@/components/ui/code-block";

export function TraitsGuide() {
  return (
    <div className="space-y-6">
      {/* Intro Hero Card */}
      <Card className="bg-card/80 border-border/80 shadow-xs">
        <CardHeader className="p-5 pb-3">
          <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3">
            <div className="space-y-1">
              <div className="flex items-center gap-2">
                <BookOpen className="h-5 w-5 text-purple-400" />
                <CardTitle className="text-base font-semibold">
                  Ego Traits & JSON Schema Architecture Guide
                </CardTitle>
                <Badge variant="purple" className="text-[10px] font-mono">
                  Ory Kratos Parity
                </Badge>
              </div>
              <CardDescription className="text-xs max-w-3xl leading-relaxed">
                Autorix Ego abstracts user models away from static SQL columns into declarative JSON Schema traits.
                Learn how schemas define validation rules, credential identifiers, and custom enterprise attributes.
              </CardDescription>
            </div>
          </div>
        </CardHeader>

        <CardContent className="p-5 pt-2 grid grid-cols-1 md:grid-cols-3 gap-4">
          <div className="p-3.5 rounded-lg border border-border/60 bg-muted/20 space-y-1.5">
            <div className="flex items-center gap-2 text-cyan-400 font-semibold text-xs">
              <Database className="h-4 w-4" />
              <span>Zero-Migration Extensibility</span>
            </div>
            <p className="text-[11px] text-muted-foreground leading-relaxed">
              No <code className="text-foreground">ALTER TABLE</code> migrations required. Attributes are stored as GIN-indexed <code className="text-foreground">jsonb</code> in PostgreSQL with sub-millisecond query latency.
            </p>
          </div>

          <div className="p-3.5 rounded-lg border border-border/60 bg-muted/20 space-y-1.5">
            <div className="flex items-center gap-2 text-amber-400 font-semibold text-xs">
              <KeyRound className="h-4 w-4" />
              <span>Credential Mapping</span>
            </div>
            <p className="text-[11px] text-muted-foreground leading-relaxed">
              Use <code className="text-foreground">autorix.io/credentials</code> tags to designate which traits act as login identifiers (e.g., email, username, phone).
            </p>
          </div>

          <div className="p-3.5 rounded-lg border border-border/60 bg-muted/20 space-y-1.5">
            <div className="flex items-center gap-2 text-emerald-400 font-semibold text-xs">
              <ShieldCheck className="h-4 w-4" />
              <span>Strict Runtime Validation</span>
            </div>
            <p className="text-[11px] text-muted-foreground leading-relaxed">
              Input data is strictly validated against JSON Schema Draft-07 before any identity is persisted to disk or session is minted.
            </p>
          </div>
        </CardContent>
      </Card>

      {/* Deep-Dive Grid */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Card 1: Anatomy of an Identity Schema */}
        <Card className="bg-card/80 border-border/80">
          <CardHeader className="p-4 pb-3 border-b border-border/60">
            <div className="flex items-center gap-2">
              <Layers className="h-4 w-4 text-blue-400" />
              <CardTitle className="text-xs font-semibold uppercase tracking-wider">
                Anatomy of an Identity Schema
              </CardTitle>
            </div>
            <CardDescription className="text-xs">
              Essential sections required in any Ego Draft-07 JSON schema
            </CardDescription>
          </CardHeader>

          <CardContent className="p-4 space-y-4">
            <div className="space-y-3 text-xs text-muted-foreground leading-relaxed">
              <div className="flex items-start gap-2.5">
                <CheckCircle className="h-4 w-4 text-emerald-400 shrink-0 mt-0.5" />
                <div>
                  <strong className="text-foreground font-medium">$schema & $id</strong>
                  <p className="text-[11px]">Identifies the JSON Schema specification (<code className="text-foreground">http://json-schema.org/draft-07/schema#</code>) and the unique URI identifier for caching.</p>
                </div>
              </div>

              <div className="flex items-start gap-2.5">
                <CheckCircle className="h-4 w-4 text-emerald-400 shrink-0 mt-0.5" />
                <div>
                  <strong className="text-foreground font-medium">properties.traits</strong>
                  <p className="text-[11px]">The container object holding all user attributes. Any field placed here becomes accessible in identity payloads as <code className="text-foreground">identity.traits.my_field</code>.</p>
                </div>
              </div>

              <div className="flex items-start gap-2.5">
                <CheckCircle className="h-4 w-4 text-emerald-400 shrink-0 mt-0.5" />
                <div>
                  <strong className="text-foreground font-medium">autorix.io/credentials</strong>
                  <p className="text-[11px]">Metadata extension. Placing <code className="text-foreground">{`"password": { "identifier": true }`}</code> on a field makes it a valid login handle in <code className="text-foreground">POST /self-service/login</code>.</p>
                </div>
              </div>

              <div className="flex items-start gap-2.5">
                <CheckCircle className="h-4 w-4 text-emerald-400 shrink-0 mt-0.5" />
                <div>
                  <strong className="text-foreground font-medium">required array</strong>
                  <p className="text-[11px]">Specifies mandatory traits (e.g. <code className="text-foreground">{`["email", "name"]`}</code>) that reject registration if missing.</p>
                </div>
              </div>
            </div>

            <div className="pt-2">
              <CodeBlock
                language="json"
                title="Credential Identifier Extension"
                code={`{
  "email": {
    "type": "string",
    "format": "email",
    "title": "Corporate E-Mail",
    "autorix.io/credentials": {
      "password": {
        "identifier": true
      }
    }
  }
}`}
              />
            </div>
          </CardContent>
        </Card>

        {/* Card 2: Supported Trait Types & UI Mapping */}
        <Card className="bg-card/80 border-border/80">
          <CardHeader className="p-4 pb-3 border-b border-border/60">
            <div className="flex items-center gap-2">
              <Code2 className="h-4 w-4 text-emerald-400" />
              <CardTitle className="text-xs font-semibold uppercase tracking-wider">
                Supported Types & Form Fields
              </CardTitle>
            </div>
            <CardDescription className="text-xs">
              How JSON Schema properties map to frontend form components
            </CardDescription>
          </CardHeader>

          <CardContent className="p-4 space-y-3">
            <div className="border border-border/60 rounded-md overflow-hidden text-xs">
              <table className="w-full text-left">
                <thead className="bg-muted/40 text-muted-foreground font-medium text-[11px] border-b border-border/60">
                  <tr>
                    <th className="p-2.5">JSON Type</th>
                    <th className="p-2.5">Format / Key</th>
                    <th className="p-2.5">Rendered UI Element</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-border/40 text-[11px]">
                  <tr>
                    <td className="p-2.5 font-mono text-cyan-400">string</td>
                    <td className="p-2.5 font-mono text-muted-foreground">format: email</td>
                    <td className="p-2.5 text-foreground">Email Input with RFC validation</td>
                  </tr>
                  <tr>
                    <td className="p-2.5 font-mono text-cyan-400">string</td>
                    <td className="p-2.5 font-mono text-muted-foreground">enum: [...]</td>
                    <td className="p-2.5 text-foreground">Select dropdown with fixed options</td>
                  </tr>
                  <tr>
                    <td className="p-2.5 font-mono text-cyan-400">string</td>
                    <td className="p-2.5 font-mono text-muted-foreground">format: tel</td>
                    <td className="p-2.5 text-foreground">Phone number input</td>
                  </tr>
                  <tr>
                    <td className="p-2.5 font-mono text-cyan-400">object</td>
                    <td className="p-2.5 font-mono text-muted-foreground">properties: &#123;...&#125;</td>
                    <td className="p-2.5 text-foreground">Nested grouped container (e.g. name.first, name.last)</td>
                  </tr>
                  <tr>
                    <td className="p-2.5 font-mono text-cyan-400">boolean</td>
                    <td className="p-2.5 font-mono text-muted-foreground">default: false</td>
                    <td className="p-2.5 text-foreground">Interactive Checkbox / Switch</td>
                  </tr>
                  <tr>
                    <td className="p-2.5 font-mono text-cyan-400">number</td>
                    <td className="p-2.5 font-mono text-muted-foreground">type: integer</td>
                    <td className="p-2.5 text-foreground">Numeric input stepper</td>
                  </tr>
                </tbody>
              </table>
            </div>

            <div className="pt-2">
              <CodeBlock
                language="json"
                title="Nested Object & Enum Example"
                code={`{
  "name": {
    "type": "object",
    "title": "Full Legal Name",
    "properties": {
      "first": { "type": "string", "title": "First Name" },
      "last": { "type": "string", "title": "Last Name" }
    },
    "required": ["first"]
  },
  "department": {
    "type": "string",
    "title": "Department",
    "enum": ["Security", "Engineering", "Operations"]
  }
}`}
              />
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Integration & API Quickstart */}
      <Card className="bg-card/80 border-border/80">
        <CardHeader className="p-4 pb-3 border-b border-border/60">
          <div className="flex items-center gap-2">
            <Terminal className="h-4 w-4 text-cyan-400" />
            <CardTitle className="text-xs font-semibold uppercase tracking-wider">
              API Quickstart: Registering Schemas & Identities via cURL
            </CardTitle>
          </div>
          <CardDescription className="text-xs">
            Direct commands to register custom schemas in Ego and onboard identities adhering to them
          </CardDescription>
        </CardHeader>

        <CardContent className="p-4 space-y-4">
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
            <div className="space-y-1.5">
              <span className="text-xs font-semibold text-foreground flex items-center gap-1.5">
                <span className="flex h-5 w-5 items-center justify-center rounded-full bg-blue-500/20 text-blue-400 text-[10px] font-mono">1</span>
                <span>Register a Custom Schema via Ego Admin API</span>
              </span>
              <CodeBlock
                language="bash"
                title="POST /admin/schemas"
                code={`curl -X POST http://localhost:4433/admin/schemas \\
  -H "Content-Type: application/json" \\
  -d '{
    "id": "partner_v1",
    "name": "B2B Partner Schema",
    "schema": {
      "$id": "https://schemas.autorix.io/partner.json",
      "$schema": "http://json-schema.org/draft-07/schema#",
      "type": "object",
      "properties": {
        "traits": {
          "type": "object",
          "properties": {
            "email": {
              "type": "string",
              "format": "email",
              "autorix.io/credentials": { "password": { "identifier": true } }
            },
            "company_name": { "type": "string" }
          },
          "required": ["email", "company_name"]
        }
      }
    }
  }'`}
              />
            </div>

            <div className="space-y-1.5">
              <span className="text-xs font-semibold text-foreground flex items-center gap-1.5">
                <span className="flex h-5 w-5 items-center justify-center rounded-full bg-emerald-500/20 text-emerald-400 text-[10px] font-mono">2</span>
                <span>Onboard an Identity with that Schema</span>
              </span>
              <CodeBlock
                language="bash"
                title="POST /admin/identities or POST /self-service/registration"
                code={`curl -X POST http://localhost:4433/self-service/registration \\
  -H "Content-Type: application/json" \\
  -d '{
    "schema_id": "partner_v1",
    "traits": {
      "email": "lead@acme-partner.com",
      "company_name": "Acme Partner LLC"
    },
    "password": "CorrectHorseBatteryStaple!2026"
  }'`}
              />
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
