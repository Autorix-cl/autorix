import { proxyRequest } from "@/lib/api/proxy";
import { rotateKeysResponseSchema } from "@/lib/api/schemas/oauth2";

export async function POST() {
  return proxyRequest("janus", "/admin/keys/rotate", rotateKeysResponseSchema, {
    method: "POST",
  });
}

