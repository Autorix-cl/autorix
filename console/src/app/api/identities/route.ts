import { NextRequest } from "next/server";
import { proxyRequest } from "@/lib/api/proxy";
import { paginatedIdentityListSchema, registrationResponseSchema } from "@/lib/api/schemas/identity";

export async function GET(req: NextRequest) {
  const { searchParams } = new URL(req.url);
  const q = searchParams.get("q") || "";
  const cursor = searchParams.get("cursor") || "";
  const limit = searchParams.get("limit") || "50";
  const state = searchParams.get("state") || "";

  const query = new URLSearchParams();
  if (q) query.set("q", q);
  if (cursor) query.set("cursor", cursor);
  if (limit) query.set("limit", limit);
  if (state) query.set("state", state);

  const path = `/admin/identities?${query.toString()}`;
  return proxyRequest("ego", path, paginatedIdentityListSchema);
}

export async function POST(req: NextRequest) {
  const body = await req.json();
  const payload = {
    password: body.password,
    traits: {
      email: body.email,
      name: {
        first: body.firstName || "",
        last: body.lastName || "",
      },
    },
  };

  return proxyRequest("ego", "/self-service/registration", registrationResponseSchema, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
}
