import { describe, it, expect, vi, beforeEach } from "vitest";
import { NextRequest } from "next/server";
import { GET as getProvider, PATCH as updateProvider, DELETE as deleteProvider } from "./saml/[id]/route";
import { POST as enableProvider } from "./saml/[id]/enable/route";
import { POST as disableProvider } from "./saml/[id]/disable/route";
import { GET as listGroups, POST as createGroup } from "./scim/groups/route";
import {
  GET as getGroup,
  PUT as putGroup,
  PATCH as patchGroup,
  DELETE as deleteGroup,
} from "./scim/groups/[id]/route";
import { GET as listSyncHistory, POST as triggerSync } from "./scim/sync/route";

describe("Hermes Enterprise BFF Routes", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    global.fetch = vi.fn();
  });

  it("handles SAML provider lifecycle (GET, PATCH, DELETE)", async () => {
    // 1. GET
    vi.mocked(global.fetch).mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          id: "okta-test",
          display_name: "Okta Test",
          idp_entity_id: "https://idp.example.com",
          idp_sso_url: "https://idp.example.com/sso",
          idp_certificate_pem: "cert",
          sp_entity_id: "https://autorix.io/sp",
          attribute_mapping: {},
          enabled: true,
          created_at: "2026-09-02T10:00:00Z",
          updated_at: "2026-09-02T10:00:00Z",
        }),
        { status: 200, headers: { "Content-Type": "application/json" } }
      )
    );

    const getRes = await getProvider(new NextRequest("http://localhost/api/enterprise/saml/okta-test"), {
      params: Promise.resolve({ id: "okta-test" }),
    });
    expect(getRes.status).toBe(200);
    const getJson = await getRes.json();
    expect(getJson.display_name).toBe("Okta Test");

    // 2. PATCH
    vi.mocked(global.fetch).mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          id: "okta-test",
          display_name: "Okta Updated",
          idp_entity_id: "https://idp.example.com",
          idp_sso_url: "https://idp.example.com/sso",
          idp_certificate_pem: "cert",
          sp_entity_id: "https://autorix.io/sp",
          attribute_mapping: { email: "mail" },
          enabled: true,
          created_at: "2026-09-02T10:00:00Z",
          updated_at: "2026-09-02T10:00:00Z",
        }),
        { status: 200, headers: { "Content-Type": "application/json" } }
      )
    );

    const patchRes = await updateProvider(
      new NextRequest("http://localhost/api/enterprise/saml/okta-test", {
        method: "PATCH",
        body: JSON.stringify({ display_name: "Okta Updated", attribute_mapping: { email: "mail" } }),
      }),
      { params: Promise.resolve({ id: "okta-test" }) }
    );
    expect(patchRes.status).toBe(200);
    const patchJson = await patchRes.json();
    expect(patchJson.display_name).toBe("Okta Updated");

    // 3. DELETE
    vi.mocked(global.fetch).mockResolvedValueOnce(
      new Response(JSON.stringify({ status: "deleted" }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      })
    );

    const delRes = await deleteProvider(new NextRequest("http://localhost/api/enterprise/saml/okta-test", { method: "DELETE" }), {
      params: Promise.resolve({ id: "okta-test" }),
    });
    expect(delRes.status).toBe(200);
  });

  it("handles enable and disable SAML provider", async () => {
    vi.mocked(global.fetch).mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          id: "okta-test",
          display_name: "Okta Test",
          idp_entity_id: "https://idp.example.com",
          idp_sso_url: "https://idp.example.com/sso",
          idp_certificate_pem: "cert",
          sp_entity_id: "https://autorix.io/sp",
          attribute_mapping: {},
          enabled: true,
          created_at: "2026-09-02T10:00:00Z",
          updated_at: "2026-09-02T10:00:00Z",
        }),
        { status: 200, headers: { "Content-Type": "application/json" } }
      )
    );

    const enableRes = await enableProvider(new NextRequest("http://localhost/api/enterprise/saml/okta-test/enable", { method: "POST" }), {
      params: Promise.resolve({ id: "okta-test" }),
    });
    expect(enableRes.status).toBe(200);

    vi.mocked(global.fetch).mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          id: "okta-test",
          display_name: "Okta Test",
          idp_entity_id: "https://idp.example.com",
          idp_sso_url: "https://idp.example.com/sso",
          idp_certificate_pem: "cert",
          sp_entity_id: "https://autorix.io/sp",
          attribute_mapping: {},
          enabled: false,
          created_at: "2026-09-02T10:00:00Z",
          updated_at: "2026-09-02T10:00:00Z",
        }),
        { status: 200, headers: { "Content-Type": "application/json" } }
      )
    );

    const disableRes = await disableProvider(new NextRequest("http://localhost/api/enterprise/saml/okta-test/disable", { method: "POST" }), {
      params: Promise.resolve({ id: "okta-test" }),
    });
    expect(disableRes.status).toBe(200);
  });

  it("handles SCIM groups management (GET, POST, PUT, DELETE)", async () => {
    // 1. List
    vi.mocked(global.fetch).mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          schemas: ["urn:ietf:params:scim:api:messages:2.0:ListResponse"],
          totalResults: 1,
          startIndex: 1,
          itemsPerPage: 1,
          Resources: [
            {
              schemas: ["urn:ietf:params:scim:schemas:core:2.0:Group"],
              id: "g-1",
              displayName: "Security Ops",
              members: [],
              meta: { resourceType: "Group", created: "2026-09-02T10:00:00Z", lastModified: "2026-09-02T10:00:00Z", location: "/scim/v2/Groups/g-1" },
            },
          ],
        }),
        { status: 200, headers: { "Content-Type": "application/json" } }
      )
    );

    const listRes = await listGroups();
    expect(listRes.status).toBe(200);
    const listJson = await listRes.json();
    expect(listJson.Resources).toHaveLength(1);

    // 2. Create
    vi.mocked(global.fetch).mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          schemas: ["urn:ietf:params:scim:schemas:core:2.0:Group"],
          id: "g-2",
          displayName: "DevOps",
          members: [{ value: "u-1" }],
          meta: { resourceType: "Group", created: "2026-09-02T10:00:00Z", lastModified: "2026-09-02T10:00:00Z", location: "/scim/v2/Groups/g-2" },
        }),
        { status: 201, headers: { "Content-Type": "application/json" } }
      )
    );

    const createRes = await createGroup(
      new NextRequest("http://localhost/api/enterprise/scim/groups", {
        method: "POST",
        body: JSON.stringify({ displayName: "DevOps", members: [{ value: "u-1" }] }),
      })
    );
    expect(createRes.status).toBe(201);

    // 3. Detail
    vi.mocked(global.fetch).mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          schemas: ["urn:ietf:params:scim:schemas:core:2.0:Group"],
          id: "g-2",
          displayName: "DevOps",
          members: [{ value: "u-1" }],
          meta: { resourceType: "Group", created: "2026-09-02T10:00:00Z", lastModified: "2026-09-02T10:00:00Z", location: "/scim/v2/Groups/g-2" },
        }),
        { status: 200, headers: { "Content-Type": "application/json" } }
      )
    );

    const getGroupRes = await getGroup(new NextRequest("http://localhost/api/enterprise/scim/groups/g-2"), {
      params: Promise.resolve({ id: "g-2" }),
    });
    expect(getGroupRes.status).toBe(200);

    // PUT
    vi.mocked(global.fetch).mockResolvedValueOnce(
      new Response(JSON.stringify({ id: "g-2", displayName: "DevOps Updated" }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      })
    );
    const putRes = await putGroup(
      new NextRequest("http://localhost/api/enterprise/scim/groups/g-2", {
        method: "PUT",
        body: JSON.stringify({ displayName: "DevOps Updated" }),
      }),
      { params: Promise.resolve({ id: "g-2" }) }
    );
    expect(putRes.status).toBe(200);

    // PATCH
    vi.mocked(global.fetch).mockResolvedValueOnce(
      new Response(JSON.stringify({ id: "g-2", displayName: "DevOps Patched" }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      })
    );
    const patchRes = await patchGroup(
      new NextRequest("http://localhost/api/enterprise/scim/groups/g-2", {
        method: "PATCH",
        body: JSON.stringify({ displayName: "DevOps Patched" }),
      }),
      { params: Promise.resolve({ id: "g-2" }) }
    );
    expect(patchRes.status).toBe(200);

    // 4. Delete
    vi.mocked(global.fetch).mockResolvedValueOnce(
      new Response(JSON.stringify({ status: "deleted" }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      })
    );

    const delGroupRes = await deleteGroup(new NextRequest("http://localhost/api/enterprise/scim/groups/g-2", { method: "DELETE" }), {
      params: Promise.resolve({ id: "g-2" }),
    });
    expect(delGroupRes.status).toBe(200);
  });

  it("handles SCIM sync history (GET, POST)", async () => {
    vi.mocked(global.fetch).mockResolvedValueOnce(
      new Response(
        JSON.stringify([
          {
            id: "sync-1",
            provider_id: "okta",
            resource_type: "Users",
            status: "success",
            total_records: 100,
            created_count: 5,
            updated_count: 95,
            deleted_count: 0,
            error_count: 0,
            errors: [],
            started_at: "2026-09-02T10:00:00Z",
            completed_at: "2026-09-02T10:01:00Z",
          },
        ]),
        { status: 200, headers: { "Content-Type": "application/json" } }
      )
    );

    const listRes = await listSyncHistory();
    expect(listRes.status).toBe(200);
    const listJson = await listRes.json();
    expect(listJson).toHaveLength(1);

    vi.mocked(global.fetch).mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          id: "sync-2",
          provider_id: "azure-ad",
          resource_type: "Groups",
          status: "success",
          total_records: 10,
          created_count: 2,
          updated_count: 8,
          deleted_count: 0,
          error_count: 0,
          errors: [],
          started_at: "2026-09-02T10:00:00Z",
          completed_at: "2026-09-02T10:01:00Z",
        }),
        { status: 200, headers: { "Content-Type": "application/json" } }
      )
    );

    const triggerRes = await triggerSync(
      new NextRequest("http://localhost/api/enterprise/scim/sync", {
        method: "POST",
        body: JSON.stringify({ resource_type: "Groups" }),
      })
    );
    expect(triggerRes.status).toBe(200);
  });
});
