"use client";

import * as React from "react";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Checkbox } from "@/components/ui/checkbox";

export interface JSONSchemaProperty {
  type?: string;
  title?: string;
  description?: string;
  format?: string;
  required?: string[];
  properties?: Record<string, JSONSchemaProperty>;
  items?: JSONSchemaProperty;
  default?: unknown;
  "autorix.io/credentials"?: {
    password?: { identifier?: boolean };
  };
}

export interface DynamicSchemaFormProps {
  schema: {
    properties?: {
      traits?: {
        properties?: Record<string, JSONSchemaProperty>;
        required?: string[];
      };
      [key: string]: unknown;
    };
    required?: string[];
  };
  value: Record<string, unknown>;
  onChange: (value: Record<string, unknown>) => void;
  disabled?: boolean;
}

/**
 * DynamicSchemaForm dynamically translates Ory Kratos / Ego JSON Schema definitions
 * into interactive, accessible HTML form fields with full nesting support.
 */
export function DynamicSchemaForm({
  schema,
  value,
  onChange,
  disabled = false,
}: DynamicSchemaFormProps) {
  // Extract traits definition from schema
  const traitsDef = schema?.properties?.traits?.properties || {};
  const requiredFields = new Set<string>(schema?.properties?.traits?.required || schema?.required || []);

  const handleFieldChange = (path: string[], fieldValue: unknown) => {
    const updated = JSON.parse(JSON.stringify(value || {}));
    let current = updated;
    for (let i = 0; i < path.length - 1; i++) {
      const key = path[i];
      if (!current[key] || typeof current[key] !== "object") {
        current[key] = {};
      }
      current = current[key];
    }
    current[path[path.length - 1]] = fieldValue;
    onChange(updated);
  };

  const getFieldValue = (path: string[]): unknown => {
    let current: unknown = value;
    for (const key of path) {
      if (current && typeof current === "object" && key in (current as Record<string, unknown>)) {
        current = (current as Record<string, unknown>)[key];
      } else {
        return undefined;
      }
    }
    return current;
  };

  const renderProperty = (
    key: string,
    prop: JSONSchemaProperty,
    path: string[],
    isRequired: boolean
  ) => {
    const fieldId =
      key === "first"
        ? "firstName"
        : key === "last"
        ? "lastName"
        : key === "email"
        ? "email"
        : path.join(".");
    const title = prop.title || key.charAt(0).toUpperCase() + key.slice(1);

    // Object type with nested properties
    if (prop.type === "object" && prop.properties) {
      const subRequired = new Set<string>(prop.required || []);
      return (
        <div key={fieldId} className="space-y-3 rounded-md border border-border/50 bg-muted/10 p-3.5">
          <div className="space-y-0.5">
            <h4 className="text-xs font-semibold text-foreground flex items-center gap-1">
              {title}
              {isRequired && <span className="text-destructive">*</span>}
            </h4>
            {prop.description && (
              <p className="text-[11px] text-muted-foreground">{prop.description}</p>
            )}
          </div>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            {Object.entries(prop.properties).map(([subKey, subProp]) =>
              renderProperty(subKey, subProp, [...path, subKey], subRequired.has(subKey))
            )}
          </div>
        </div>
      );
    }

    // Boolean type
    if (prop.type === "boolean") {
      const val = Boolean(getFieldValue(path));
      return (
        <div key={fieldId} className="flex items-center space-x-2 py-1">
          <Checkbox
            id={fieldId}
            checked={val}
            disabled={disabled}
            onCheckedChange={(checked) => handleFieldChange(path, Boolean(checked))}
          />
          <Label htmlFor={fieldId} className="text-xs font-medium cursor-pointer">
            {title} {isRequired && <span className="text-destructive">*</span>}
          </Label>
        </div>
      );
    }

    // Number type
    if (prop.type === "number" || prop.type === "integer") {
      const val = (getFieldValue(path) as number) ?? "";
      return (
        <div key={fieldId} className="space-y-1.5">
          <Label htmlFor={fieldId} className="text-xs font-medium">
            {title} {isRequired && <span className="text-destructive">*</span>}
          </Label>
          <Input
            id={fieldId}
            type="number"
            value={val}
            disabled={disabled}
            required={isRequired}
            placeholder={`Enter ${title.toLowerCase()}`}
            onChange={(e) =>
              handleFieldChange(path, e.target.value === "" ? "" : Number(e.target.value))
            }
            className="h-9 text-xs"
          />
        </div>
      );
    }

    // String type (email, text, password)
    const inputType = prop.format === "email" ? "email" : "text";
    const val = (getFieldValue(path) as string) ?? "";
    const isIdentifier = Boolean(prop["autorix.io/credentials"]?.password?.identifier);

    return (
      <div key={fieldId} className="space-y-1.5">
        <div className="flex items-center justify-between">
          <Label htmlFor={fieldId} className="text-xs font-medium">
            {title} {isRequired && <span className="text-destructive">*</span>}
          </Label>
          {isIdentifier && (
            <span className="text-[10px] font-mono px-1.5 py-0.2 rounded bg-cyan-500/10 text-cyan-400 border border-cyan-500/20">
              Primary Identifier
            </span>
          )}
        </div>
        <Input
          id={fieldId}
          type={inputType}
          value={val}
          disabled={disabled}
          required={isRequired}
          placeholder={`Enter ${title.toLowerCase()}`}
          onChange={(e) => handleFieldChange(path, e.target.value)}
          className="h-9 text-xs"
        />
        {prop.description && (
          <p className="text-[10px] text-muted-foreground">{prop.description}</p>
        )}
      </div>
    );
  };

  const entries = Object.entries(traitsDef);

  if (entries.length === 0) {
    return (
      <div className="rounded-md border border-dashed p-4 text-center text-xs text-muted-foreground">
        No trait properties found in active schema.
      </div>
    );
  }

  return (
    <div className="space-y-3.5">
      {entries.map(([key, prop]) =>
        renderProperty(key, prop, [key], requiredFields.has(key))
      )}
    </div>
  );
}
