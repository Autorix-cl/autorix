import { NextRequest, NextResponse } from "next/server";
import crypto from "crypto";
import { z } from "zod";
import { proxyRequest } from "@/lib/api/proxy";
import { requireOperator } from "@/lib/auth/session";
import { identitySchema, recoveryLinkResultSchema, registrationResponseSchema } from "@/lib/api/schemas/identity";

const inviteInputSchema = z.object({
  email: z.string().email().optional(),
  schema_id: z.string().optional(),
  traits: z.record(z.string(), z.unknown()).optional(),
  expires_in: z.string().optional(),
});

const createIdentityResponseSchema = z.union([identitySchema, registrationResponseSchema]);

export async function POST(req: NextRequest) {
  const auth = await requireOperator("identities:write");
  if (!auth.ok) return auth.response;

  let rawBody: unknown;
  try {
    rawBody = await req.json();
  } catch {
    return NextResponse.json({ error: "Invalid JSON payload" }, { status: 400 });
  }

  const parsed = inviteInputSchema.safeParse(rawBody);
  if (!parsed.success) {
    return NextResponse.json({ error: "Invalid invite payload" }, { status: 400 });
  }

  const traits: Record<string, unknown> = { ...(parsed.data.traits || {}) };
  const email =
    (typeof parsed.data.email === "string" && parsed.data.email.trim()) ||
    (typeof traits.email === "string" ? traits.email.trim() : "");

  if (!email || !email.includes("@")) {
    return NextResponse.json({ error: "Valid email address is required" }, { status: 400 });
  }
  traits.email = email;

  const generatedPassword = crypto.randomBytes(24).toString("base64url");
  const createPayload = {
    schema_id: parsed.data.schema_id || "default",
    password: generatedPassword,
    traits,
  };

  let createRes = await proxyRequest("ego", "/admin/identities", createIdentityResponseSchema, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(createPayload),
    requiredPermission: "identities:write",
  });

  // Fallback to self-service registration if this Ego build only exposes /self-service/registration
  if (createRes.status === 404 || createRes.status === 405) {
    createRes = await proxyRequest("ego", "/self-service/registration", createIdentityResponseSchema, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        password: generatedPassword,
        traits,
      }),
      requiredPermission: "identities:write",
    });
  }

  if (!createRes.ok) {
    return createRes;
  }

  const createdData = await createRes.json();
  const identity = "identity" in createdData && createdData.identity ? createdData.identity : createdData;

  if (!identity?.id || typeof identity.id !== "string") {
    return NextResponse.json({ error: "Ego did not return a created identity ID" }, { status: 502 });
  }

  const recoveryRes = await proxyRequest(
    "ego",
    `/admin/identities/${encodeURIComponent(identity.id)}/recovery-link`,
    recoveryLinkResultSchema,
    {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ expires_in: parsed.data.expires_in || "24h" }),
      requiredPermission: "identities:write",
    },
  );

  if (!recoveryRes.ok) {
    return recoveryRes;
  }

  const recoveryData = await recoveryRes.json();
  return NextResponse.json(
    {
      ...identity,
      recovery_link: recoveryData.recovery_link,
      invite_link: recoveryData.recovery_link,
      expires_at: recoveryData.expires_at,
    },
    { status: 201 },
  );
}
