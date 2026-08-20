import { NextRequest } from "next/server";
import { proxyRequest } from "@/lib/api/proxy";
import { identityListSchema, registrationResponseSchema } from "@/lib/api/schemas/identity";

export async function GET() {
  return proxyRequest("ego", "/admin/identities", identityListSchema);
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
