import { NextRequest, NextResponse } from "next/server";
import { proxyRequest } from "@/lib/api/proxy";
import { registrationResponseSchema } from "@/lib/api/schemas/identity";

interface BulkRow {
  email: string;
  password?: string;
  firstName?: string;
  lastName?: string;
  traits?: Record<string, unknown>;
}

export async function POST(req: NextRequest) {
  try {
    const body = await req.json();
    const rows: BulkRow[] = Array.isArray(body) ? body : body.rows || [];
    const dryRun = Boolean(body.dry_run);

    let succeeded = 0;
    let failed = 0;
    const errors: { row: number; email?: string; error: string }[] = [];

    for (let i = 0; i < rows.length; i++) {
      const row = rows[i];
      const rowNum = i + 1;

      if (!row.email || !row.email.includes("@")) {
        failed++;
        errors.push({
          row: rowNum,
          email: row.email,
          error: "Missing or invalid email address",
        });
        continue;
      }

      if (dryRun) {
        succeeded++;
        continue;
      }

      try {
        const payload = {
          password: row.password || "TempPassword#123",
          traits: {
            email: row.email,
            name: {
              first: row.firstName || "",
              last: row.lastName || "",
            },
            ...(row.traits || {}),
          },
        };

        const res = await proxyRequest("ego", "/self-service/registration", registrationResponseSchema, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(payload),
        });

        if (!res.ok) {
          failed++;
          errors.push({
            row: rowNum,
            email: row.email,
            error: "Engine registration rejected",
          });
        } else {
          succeeded++;
        }
      } catch (err: unknown) {
        failed++;
        const message = err instanceof Error ? err.message : "Creation failed";
        errors.push({
          row: rowNum,
          email: row.email,
          error: message,
        });
      }
    }

    return NextResponse.json({
      total: rows.length,
      succeeded,
      failed,
      errors,
      dry_run: dryRun,
    });
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : "Failed to process bulk import";
    return NextResponse.json({ error: message }, { status: 400 });
  }
}
