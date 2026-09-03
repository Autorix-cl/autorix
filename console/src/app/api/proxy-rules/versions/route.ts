import { proxyRequest } from "@/lib/api/proxy";
import { ruleVersionListSchema } from "@/lib/api/schemas/aegis";

export async function GET() {
  return proxyRequest("aegis", "/rules/versions", ruleVersionListSchema);
}
