import { NextRequest, NextResponse } from "next/server";
import { proxyRequest } from "@/lib/api/proxy";
import { namespaceListSchema, namespaceSchema, type NamespaceSchema } from "@/lib/api/schemas/nexus";


const defaultSchema = `definition user {}

definition document {
  relation viewer: user
  relation editor: user
  permission view = viewer + editor
  permission edit = editor
}

definition organization {
  relation admin: user
  relation member: user
}`;

let cachedSchema = defaultSchema;

export async function GET() {
  try {
    const res = await proxyRequest("nexus", "/admin/namespaces", namespaceListSchema);
    if (res.ok) {
      const namespaces: NamespaceSchema[] = await res.json();
      if (Array.isArray(namespaces) && namespaces.length > 0) {
        const aplText = namespaces
          .map((ns) => {
            const relLines = Object.entries(ns.relations || {}).map(([relName, relDef]) => {
              const rw = relDef?.rewrite;
              if (rw && rw.type === "union" && rw.children && rw.children.length > 0) {
                const terms = rw.children.map((c) => c.relation || c.type);
                return `  permission ${relName} = ${terms.join(" + ")}`;
              }
              if (rw && rw.type === "computed_userset" && rw.relation) {
                return `  permission ${relName} = ${rw.relation}`;
              }
              return `  relation ${relName}: user`;
            });
            return `definition ${ns.name} {\n${relLines.join("\n")}\n}`;
          })
          .join("\n\n");


        cachedSchema = aplText;
        return NextResponse.json({ schema: aplText, namespaces });
      }
    }
  } catch {
    // Fall back to cached schema
  }

  return NextResponse.json({ schema: cachedSchema });
}

export async function PUT(req: NextRequest) {
  try {
    const body = await req.json();
    if (typeof body.schema === "string") {
      cachedSchema = body.schema;

      // Extract definition names from schema string (e.g. definition document { ... })
      const matches = [...body.schema.matchAll(/definition\s+([a-zA-Z0-9_-]+)\s*\{([^}]*)\}/g)];
      for (const m of matches) {
        const nsName = m[1];
        const content = m[2];
        const relations: Record<string, { rewrite?: { type: string } }> = {};

        const relMatches = [...content.matchAll(/(?:relation|permission)\s+([a-zA-Z0-9_-]+)/g)];
        for (const rm of relMatches) {
          relations[rm[1]] = { rewrite: { type: "this" } };
        }

        try {
          await proxyRequest("nexus", "/admin/namespaces", namespaceSchema, {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({
              name: nsName,
              relations,
            }),
          });
        } catch {
          // Continue updating other definitions
        }
      }

      return NextResponse.json({ success: true, schema: cachedSchema });
    }
    return NextResponse.json({ error: "Invalid schema payload" }, { status: 400 });
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : "Failed to update schema";
    return NextResponse.json({ error: message }, { status: 400 });
  }
}

