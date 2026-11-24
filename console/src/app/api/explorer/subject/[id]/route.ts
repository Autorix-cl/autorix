import { NextRequest, NextResponse } from "next/server";
import { getServiceUrl, JANUS_ADMIN_URL } from "@/lib/api-config";
import { buildInternalAuthHeaders } from "@/lib/api/internal-headers";
import { requireOperator } from "@/lib/auth/session";
import type { UnifiedSubject } from "@/lib/api/schemas/explorer";

type Params = { params: Promise<{ id: string }> };

const SUBJECT_TIMEOUT_MS = 8_000;

function internalHeaders(): Record<string, string> {
  return buildInternalAuthHeaders();
}

async function fetchJsonWithStatus(url: string): Promise<{ status: number; ok: boolean; data: unknown }> {
  const res = await fetch(url, {
    cache: "no-store",
    headers: internalHeaders(),
    signal: AbortSignal.timeout(SUBJECT_TIMEOUT_MS),
  });
  const data = res.ok ? await res.json().catch(() => null) : null;
  return { status: res.status, ok: res.ok, data };
}

export async function GET(_req: NextRequest, { params }: Params) {
  const auth = await requireOperator("explorer:read");
  if (!auth.ok) return auth.response;

  const { id } = await params;
  let decodedId: string;
  try {
    decodedId = decodeURIComponent(id);
  } catch {
    return NextResponse.json({ error: "Invalid subject identifier encoding" }, { status: 400 });
  }

  const cleanId = decodedId.replace(/^user:/, "").trim();
  if (!cleanId || cleanId.includes("..") || cleanId.includes("/") || cleanId.includes("\\") || cleanId.includes("\0")) {
    return NextResponse.json({ error: "Invalid subject identifier" }, { status: 400 });
  }

  const encodedId = encodeURIComponent(cleanId);

  // Run multi-engine resolution in parallel
  const [egoRes, sessionsRes, nexusRes, vulcanRes, scimRes, janusGrantsRes] = await Promise.allSettled([
    fetchJsonWithStatus(`${getServiceUrl("ego")}/admin/identities/${encodedId}`),
    fetchJsonWithStatus(`${getServiceUrl("ego")}/admin/identities/${encodedId}/sessions`),
    fetchJsonWithStatus(`${getServiceUrl("nexus")}/tuples?limit=100`),
    fetchJsonWithStatus(`${getServiceUrl("vulcan")}/keys?limit=100`),
    fetchJsonWithStatus(`${getServiceUrl("hermes")}/scim/v2/Users`),
    fetchJsonWithStatus(`${JANUS_ADMIN_URL}/admin/grants?subject=${encodedId}`),
  ]);

  // Never fabricate a fake identity on 404
  if (egoRes.status === "fulfilled" && egoRes.value.status === 404) {
    return NextResponse.json({ error: "Identity not found" }, { status: 404 });
  }

  const unified: UnifiedSubject = {
    id: cleanId,
    identity: undefined,
    sessions: [],
    relations: [],
    oauth_grants: [],
    api_keys: [],
    enterprise_linkage: undefined,
  };

  let identityEmail: string | undefined;

  if (egoRes.status === "fulfilled" && egoRes.value.ok && egoRes.value.data && typeof egoRes.value.data === "object") {
    const data = egoRes.value.data as Record<string, unknown>;
    const traits = (data.traits && typeof data.traits === "object" ? data.traits : {}) as Record<string, unknown>;
    if (typeof traits.email === "string" && traits.email.trim()) {
      identityEmail = traits.email.trim();
    }
    unified.identity = {
      id: String(data.id || cleanId),
      state: String(data.state || "active"),
      traits,
      created_at: typeof data.created_at === "string" ? data.created_at : undefined,
      schema_id: typeof data.schema_id === "string" ? data.schema_id : undefined,
    };
  }

  // Extract Sessions
  if (sessionsRes.status === "fulfilled" && sessionsRes.value.ok && sessionsRes.value.data) {
    const raw = sessionsRes.value.data as unknown;
    const list = (Array.isArray(raw) ? raw : (raw as Record<string, unknown>).data || []) as Record<string, unknown>[];
    unified.sessions = list.map((s) => ({
      id: String(s.id),
      identity_id: String(s.identity_id || cleanId),
      active: Boolean(s.active ?? true),
      ip_address: typeof s.ip_address === "string" ? s.ip_address : undefined,
      user_agent: typeof s.user_agent === "string" ? s.user_agent : undefined,
      created_at: typeof s.created_at === "string" ? s.created_at : undefined,
      expires_at: typeof s.expires_at === "string" ? s.expires_at : undefined,
    }));
  }

  // Extract Relations (strictly match cleanId or user:cleanId)
  if (nexusRes.status === "fulfilled" && nexusRes.value.ok && nexusRes.value.data) {
    const raw = nexusRes.value.data as unknown;
    const rawTuples = (Array.isArray(raw) ? raw : (raw as Record<string, unknown>).data || []) as Record<
      string,
      unknown
    >[];
    unified.relations = rawTuples
      .filter((t) => {
        const subId = String(t.subject_id || t.subject || "");
        return (
          subId === cleanId ||
          subId === `user:${cleanId}` ||
          (identityEmail !== undefined && (subId === identityEmail || subId === `user:${identityEmail}`))
        );
      })
      .map((t) => ({
        namespace: String(t.namespace),
        object: String(t.object),
        relation: String(t.relation),
        subject: `${String(t.subject_namespace || "user")}:${String(t.subject_id || cleanId)}`,
        caveat: typeof t.caveat_name === "string" ? t.caveat_name : undefined,
      }));
  }

  // Extract Janus OAuth2 Grants
  if (janusGrantsRes.status === "fulfilled" && janusGrantsRes.value.ok && janusGrantsRes.value.data) {
    const raw = janusGrantsRes.value.data as unknown;
    const rawGrants = (Array.isArray(raw) ? raw : (raw as Record<string, unknown>).data || []) as Record<
      string,
      unknown
    >[];
    unified.oauth_grants = rawGrants
      .filter((g) => {
        const sub = typeof g.subject === "string" ? g.subject : typeof g.sub === "string" ? g.sub : "";
        if (!sub) return true; // already filtered server-side by ?subject=
        return sub === cleanId || sub === `user:${cleanId}` || (identityEmail !== undefined && sub === identityEmail);
      })
      .map((g) => ({
        id: String(g.id || g.grant_id || ""),
        client_id: String(g.client_id || ""),
        scope: Array.isArray(g.scopes)
          ? (g.scopes as string[]).join(" ")
          : typeof g.scope === "string"
            ? g.scope
            : undefined,
        created_at: typeof g.created_at === "string" ? g.created_at : undefined,
        expires_at: typeof g.expires_at === "string" ? g.expires_at : undefined,
      }));
  }

  // Extract Vulcan API Keys (strictly match owner_id against cleanId or identity email; never leak system/unowned keys)
  if (vulcanRes.status === "fulfilled" && vulcanRes.value.ok && vulcanRes.value.data) {
    const raw = vulcanRes.value.data as unknown;
    const rawKeys = (Array.isArray(raw) ? raw : (raw as Record<string, unknown>).data || []) as Record<
      string,
      unknown
    >[];
    unified.api_keys = rawKeys
      .filter((k) => {
        const owner = typeof k.owner_id === "string" ? k.owner_id : "";
        if (!owner) return false;
        return owner === cleanId || (identityEmail !== undefined && owner === identityEmail);
      })
      .map((k) => ({
        id: String(k.id),
        name: String(k.name),
        prefix: String(k.key_prefix || "av_live_") + String(k.key_hint || "****"),
        scopes: Array.isArray(k.scopes) ? (k.scopes as string[]) : [],
        call_count: typeof k.call_count === "number" ? k.call_count : 0,
        last_used_at: typeof k.last_used_at === "string" ? k.last_used_at : undefined,
      }));
  }

  // Extract Hermes SCIM Linkage
  if (
    scimRes.status === "fulfilled" &&
    scimRes.value.ok &&
    scimRes.value.data &&
    typeof scimRes.value.data === "object"
  ) {
    const scimBody = scimRes.value.data as Record<string, unknown>;
    const users = (Array.isArray(scimBody.Resources) ? scimBody.Resources : []) as Record<string, unknown>[];
    const matched = users.find((u) => {
      const emails = u.emails as { value?: string }[] | undefined;
      const primaryEmail = emails?.[0]?.value;
      return (
        u.userName === cleanId ||
        u.externalId === cleanId ||
        (identityEmail !== undefined && (u.userName === identityEmail || primaryEmail === identityEmail))
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
