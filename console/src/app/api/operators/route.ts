import { proxyRequest } from "@/lib/api/proxy";
import { operatorsListSchema, operatorSchema } from "@/lib/api/schemas/operator";

export async function GET() {
  return proxyRequest("argus", "/v1/operators", operatorsListSchema, {
    requiredPermission: "identities:read",
  });
}

export async function POST(req: Request) {
  const body = await req.json();
  return proxyRequest("argus", "/v1/operators", operatorSchema, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
    requiredPermission: "identities:write",
  });
}
