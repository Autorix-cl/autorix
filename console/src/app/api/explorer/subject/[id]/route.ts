import { NextRequest, NextResponse } from "next/server";
import { z } from "zod";
import { proxyRequest } from "@/lib/api/proxy";
import type { UnifiedSubject } from "@/lib/api/schemas/explorer";

type Params = { params: Promise<{ id: string }> };

export async function GET(req: NextRequest, { params }: Params) {
  const { id } = await params;
  const cleanId = decodeURIComponent(id).replace(/^user:/, "");

  // Initialize unified subject container
  const unified: UnifiedSubject = {
    id: cleanId,
    identity: undefined,
    sessions: [],
    relations: [],
    oauth_grants: [],
    api_keys: [],
    enterprise_linkage: undefined,
  };

  // Run multi-engine resolution in parallel
  const [egoRes, sessionsRes, nexusRes, vulcanRes, scimRes] = await Promise.allSettled([
    // 1. Ego Identity
    proxyRequest("ego", `/identities/${cleanId}`, z.any()).then(async (r) => (r.ok ? r.json() : null)),
    // 2. Ego Sessions
    proxyRequest("ego", `/identities/${cleanId}/sessions`, z.any()).then(async (r) => (r.ok ? r.json() : null)),
    // 3. Nexus Tuples
    proxyRequest("nexus", `/tuples?limit=100`, z.any()).then(async (r) => (r.ok ? r.json() : null)),
    // 4. Vulcan API Keys
    proxyRequest("vulcan", `/keys?limit=100`, z.any()).then(async (r) => (r.ok ? r.json() : null)),
    // 5. Hermes SCIM Users
    proxyRequest("hermes", `/scim/v2/Users`, z.any()).then(async (r) => (r.ok ? r.json() : null)),
  ]);

  // Extract Identity
  if (egoRes.status === "fulfilled" && egoRes.value) {
    const data = egoRes.value;
    unified.identity = {
      id: data.id || cleanId,
      state: data.state || "active",
      traits: data.traits || {},
      created_at: data.created_at,
      schema_id: data.schema_id,
    };
  } else {
    // Basic fallback subject identity if Ego is not connected or user is pure subject ID
    unified.identity = {
      id: cleanId,
      state: "active",
      traits: { email: `${cleanId}@autorix.io`, name: cleanId },
    };
  }

  // Extract Sessions
  if (sessionsRes.status === "fulfilled" && sessionsRes.value) {
    const list = (Array.isArray(sessionsRes.value) ? sessionsRes.value : sessionsRes.value.data || []) as Record<string, unknown>[];
    unified.sessions = list.map((s) => ({
      id: String(s.id),
      identity_id: String(s.identity_id || cleanId),
      active: Boolean(s.active ?? true),
      ip_address: typeof s.ip_address === "string" ? s.ip_address : "127.0.0.1",
      user_agent: typeof s.user_agent === "string" ? s.user_agent : "Browser / API Client",
      created_at: typeof s.created_at === "string" ? s.created_at : undefined,
      expires_at: typeof s.expires_at === "string" ? s.expires_at : undefined,
    }));
  }

  // Extract Relations (filtering where subject is user:cleanId or cleanId)
  if (nexusRes.status === "fulfilled" && nexusRes.value) {
    const rawTuples = (Array.isArray(nexusRes.value) ? nexusRes.value : nexusRes.value.data || []) as Record<string, unknown>[];
    unified.relations = rawTuples
      .filter((t) => {
        const subId = String(t.subject_id || t.subject || "");
        return subId === cleanId || subId === `user:${cleanId}` || subId.includes(cleanId);
      })
      .map((t) => ({
        namespace: String(t.namespace),
        object: String(t.object),
        relation: String(t.relation),
        subject: `${String(t.subject_namespace || "user")}:${String(t.subject_id || cleanId)}`,
        caveat: typeof t.caveat_name === "string" ? t.caveat_name : undefined,
      }));
  }

  // Extract Vulcan API Keys
  if (vulcanRes.status === "fulfilled" && vulcanRes.value) {
    const rawKeys = (Array.isArray(vulcanRes.value) ? vulcanRes.value : vulcanRes.value.data || []) as Record<string, unknown>[];
    unified.api_keys = rawKeys
      .filter((k) => k.owner_id === cleanId || k.owner_id === "system" || !k.owner_id)
      .map((k) => ({
        id: String(k.id),
        name: String(k.name),
        prefix: String((k.key_prefix || "av_live_")) + String((k.key_hint || "****")),
        scopes: Array.isArray(k.scopes) ? (k.scopes as string[]) : [],
        call_count: typeof k.call_count === "number" ? k.call_count : 0,
        last_used_at: typeof k.last_used_at === "string" ? k.last_used_at : undefined,
      }));
  }

  // Extract Hermes SCIM Linkage
  if (scimRes.status === "fulfilled" && scimRes.value) {
    const users = (scimRes.value.Resources || []) as Record<string, unknown>[];
    const matched = users.find((u) => {
      const emails = u.emails as { value?: string }[] | undefined;
      return (
        u.userName === cleanId ||
        u.externalId === cleanId ||
        emails?.[0]?.value?.startsWith(cleanId)
      );
    });
    if (matched) {
      const emails = matched.emails as { value?: string }[] | undefined;
      const meta = matched.meta as { lastModified?: string } | undefined;
      unified.enterprise_linkage = {
        provider_id: "hermes-scim-v2",
        external_id: String(matched.externalId || matched.id),
        email: emails?.[0]?.value,
        active: Boolean(matched.active ?? true),
        last_sync_at: meta?.lastModified,
      };
    }
  }


  return NextResponse.json(unified);
}
