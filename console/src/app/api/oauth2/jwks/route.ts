import { proxyRequest } from "@/lib/api/proxy";
import { jwksSchema } from "@/lib/api/schemas/oauth2";

export async function GET() {
  return proxyRequest("janus", "/.well-known/jwks.json", jwksSchema);
}
