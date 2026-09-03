import { proxyRequest } from "@/lib/api/proxy";
import { handlerCatalogueSchema } from "@/lib/api/schemas/aegis";

export async function GET() {
  return proxyRequest("aegis", "/handlers", handlerCatalogueSchema);
}
