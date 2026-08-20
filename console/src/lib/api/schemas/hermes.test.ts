import { describe, expect, it } from "vitest";
import { samlProviderListSchema, samlProviderSchema, scimListResponseSchema, scimUserSchema } from "./hermes";

describe("samlProviderSchema", () => {
  it("accepts a realistic saml provider payload", () => {
    const payload = {
      id: "okta-corporate",
      display_name: "Okta Corporate",
      idp_entity_id: "https://sts.example.com/okta-corporate",
      idp_sso_url: "https://okta.example.com/sso/saml",
      idp_certificate_pem: "-----BEGIN CERTIFICATE-----\nMIID...\n-----END CERTIFICATE-----",
      sp_entity_id: "https://autorix.io/sp",
      attribute_mapping: { email: "user.email" },
      enabled: true,
      created_at: "2026-01-01T00:00:00Z",
      updated_at: "2026-01-01T00:00:00Z",
    };
    expect(samlProviderSchema.safeParse(payload).success).toBe(true);
  });

  it("accepts a list response", () => {
    expect(samlProviderListSchema.safeParse([]).success).toBe(true);
  });

  it("rejects a provider missing required fields", () => {
    const payload = { id: "okta-corporate", enabled: true };
    expect(samlProviderSchema.safeParse(payload).success).toBe(false);
  });
});

describe("scimListResponseSchema", () => {
  it("accepts a realistic SCIM list response wrapper", () => {
    const payload = {
      schemas: ["urn:ietf:params:scim:api:messages:2.0:ListResponse"],
      totalResults: 1,
      startIndex: 1,
      itemsPerPage: 1,
      Resources: [
        {
          schemas: ["urn:ietf:params:scim:schemas:core:2.0:User"],
          id: "u-1",
          userName: "alice",
          emails: [{ value: "alice@example.com", type: "work", primary: true }],
          active: true,
          meta: {
            resourceType: "User",
            created: "2026-01-01T00:00:00Z",
            lastModified: "2026-01-01T00:00:00Z",
            location: "https://hermes/scim/v2/Users/u-1",
          },
        },
      ],
    };
    expect(scimListResponseSchema.safeParse(payload).success).toBe(true);
  });

  it("rejects a bare array (not wrapped in Resources)", () => {
    const payload = [{ id: "u-1", userName: "alice" }];
    expect(scimListResponseSchema.safeParse(payload).success).toBe(false);
  });
});

describe("scimUserSchema", () => {
  it("rejects a user missing userName", () => {
    const payload = {
      schemas: ["urn:ietf:params:scim:schemas:core:2.0:User"],
      id: "u-1",
      emails: [],
      active: true,
      meta: {
        resourceType: "User",
        created: "2026-01-01T00:00:00Z",
        lastModified: "2026-01-01T00:00:00Z",
        location: "https://hermes/scim/v2/Users/u-1",
      },
    };
    expect(scimUserSchema.safeParse(payload).success).toBe(false);
  });
});
