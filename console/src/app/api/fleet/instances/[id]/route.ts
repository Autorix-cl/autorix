import { z } from "zod";
import { proxyRequest } from "@/lib/api/proxy";

export async function GET(
  request: Request,
  props: { params: Promise<{ id: string }> }
) {
  const params = await props.params;
  return proxyRequest("argus", `/v1/instances/${params.id}`, z.unknown(), {
    method: "GET",
    requiredPermission: "fleet:read",
  });
}

export async function DELETE(
  request: Request,
  props: { params: Promise<{ id: string }> }
) {
  const params = await props.params;
  return proxyRequest("argus", `/v1/instances/${params.id}`, z.unknown(), {
    method: "DELETE",
    requiredPermission: "fleet:admin",
  });
}
