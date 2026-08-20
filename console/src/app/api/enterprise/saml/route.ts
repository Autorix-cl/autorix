import { NextRequest } from "next/server";
import { proxyRequest } from "@/lib/api/proxy";
import { samlProviderListSchema, samlProviderSchema } from "@/lib/api/schemas/hermes";

export async function GET() {
  return proxyRequest("hermes", "/admin/saml/providers", samlProviderListSchema);
}

export async function POST(req: NextRequest) {
  const body = await req.json();
  const payload = {
    id: body.id,
    display_name: body.name || body.id,
    idp_entity_id: body.idpEntityId || `https://sts.example.com/${body.id}`,
    idp_sso_url: body.ssoUrl,
    idp_certificate_pem:
      body.certificatePem || "-----BEGIN CERTIFICATE-----\nMIID...AUTORIX...CERT\n-----END CERTIFICATE-----",
    enabled: true,
    attribute_mapping: {},
  };

  return proxyRequest("hermes", "/admin/saml/providers", samlProviderSchema, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
}
