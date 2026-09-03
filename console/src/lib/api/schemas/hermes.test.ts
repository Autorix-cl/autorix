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

describe("Hermes Engine Depth Schemas", () => {
  it("validates certificateInfoSchema and provider with certificates", async () => {
    const { certificateInfoSchema } = await import("./hermes");
    const cert = {
      subject: "CN=Okta SAML Certificate",
      issuer: "CN=Okta Root CA",
      serial_number: "1234567890",
      not_before: "2025-01-01T00:00:00Z",
      not_after: "2027-01-01T00:00:00Z",
      expired: false,
      expiring_soon: false,
      days_until_expiry: 120,
    };
    expect(certificateInfoSchema.safeParse(cert).success).toBe(true);

    const providerPayload = {
      id: "azure-ad",
      display_name: "Azure AD",
      idp_entity_id: "https://sts.windows.net/tenant-id/",
      idp_sso_url: "https://login.microsoftonline.com/sso",
      idp_certificate_pem: "-----BEGIN CERTIFICATE-----\n...\n-----END CERTIFICATE-----",
      idp_cert_expires_at: "2027-01-01T00:00:00Z",
      sp_entity_id: "https://autorix.io/sp",
      attribute_mapping: { email: "mail", name: "displayName" },
      enabled: true,
      certificates: [cert],
      warnings: ["Certificate expires in 120 days"],
      created_at: "2026-01-01T00:00:00Z",
      updated_at: "2026-01-01T00:00:00Z",
    };
    expect(samlProviderSchema.safeParse(providerPayload).success).toBe(true);
  });

  it("validates scimGroupSchema and list response", async () => {
    const { scimGroupSchema, scimGroupListResponseSchema } = await import("./hermes");
    const group = {
      schemas: ["urn:ietf:params:scim:schemas:core:2.0:Group"],
      id: "g-1",
      displayName: "Engineering",
      members: [{ value: "u-1", display: "Alice" }],
      meta: {
        resourceType: "Group",
        created: "2026-01-01T00:00:00Z",
        lastModified: "2026-01-01T00:00:00Z",
        location: "https://hermes/scim/v2/Groups/g-1",
      },
    };
    expect(scimGroupSchema.safeParse(group).success).toBe(true);

    const listPayload = {
      schemas: ["urn:ietf:params:scim:api:messages:2.0:ListResponse"],
      totalResults: 1,
      startIndex: 1,
      itemsPerPage: 1,
      Resources: [group],
    };
    expect(scimGroupListResponseSchema.safeParse(listPayload).success).toBe(true);
  });

  it("validates scimSyncHistorySchema and list", async () => {
    const { scimSyncHistorySchema, scimSyncHistoryListSchema } = await import("./hermes");
    const syncItem = {
      id: "sync-1",
      provider_id: "okta",
      resource_type: "Users",
      status: "success",
      total_records: 150,
      created_count: 5,
      updated_count: 145,
      deleted_count: 0,
      error_count: 0,
      errors: [],
      started_at: "2026-09-02T10:00:00Z",
      completed_at: "2026-09-02T10:01:00Z",
    };
    expect(scimSyncHistorySchema.safeParse(syncItem).success).toBe(true);
    expect(scimSyncHistoryListSchema.safeParse([syncItem]).success).toBe(true);
  });
});

