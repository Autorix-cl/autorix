import { proxyRequest } from "@/lib/api/proxy";
import { operatorsListSchema } from "@/lib/api/schemas/operator";

export async function GET() {
  return proxyRequest("argus", "/v1/operators", operatorsListSchema, {
    requiredPermission: "identities:read",
  });
}
