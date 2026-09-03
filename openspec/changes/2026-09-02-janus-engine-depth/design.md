# Design: Janus OAuth2 & OIDC Depth Management (P6-S2)

## Architectural Seams
- **Janus Go Admin API**: Listening on port `:4444`. Handlers in `janus/internal/transport/http/server.go`.
- **Next.js BFF**: Route handlers under `console/src/app/api/oauth2/*` proxying requests via `proxyRequest("janus", path, schema, options)`.
- **Console Frontend**: Next.js 15 Client components located in `console/src/app/oauth2/`.

## Data Models & Schemas (`console/src/lib/api/schemas/oauth2.ts`)
```typescript
export const oauth2ClientSchema = z.object({
  client_id: z.string(),
  client_name: z.string(),
  grant_types: z.array(z.string()),
  response_types: z.array(z.string()),
  redirect_uris: z.array(z.string()),
  scopes: z.array(z.string()),
  is_public: z.boolean(),
  previous_secret_expires_at: z.string().nullable().optional(),
  has_previous_secret: z.boolean().optional(),
  created_at: z.string(),
  updated_at: z.string(),
});

export const rotateSecretResponseSchema = z.object({
  client_id: z.string(),
  client_secret: z.string(),
  previous_secret_expires_at: z.string().nullable().optional(),
});

export const oauth2ScopeSchema = z.object({
  name: z.string(),
  description: z.string().default(""),
  claims: z.array(z.string()).default([]),
  created_at: z.string().optional(),
});
export const oauth2ScopeListSchema = pagedListSchema(oauth2ScopeSchema);

export const introspectRequestSchema = z.object({
  token: z.string(),
  token_type_hint: z.string().optional(),
});

export const introspectResponseSchema = z.object({
  active: z.boolean(),
  scope: z.string().optional(),
  client_id: z.string().optional(),
  sub: z.string().optional(),
  exp: z.number().optional(),
  iat: z.number().optional(),
  iss: z.string().optional(),
  token_type: z.string().optional(),
});

export const revokeTokenRequestSchema = z.object({
  token: z.string(),
  token_type_hint: z.string().optional(),
});

export const revokeTokenResponseSchema = z.object({
  status: z.string(),
});

export const rotateKeysResponseSchema = z.object({
  status: z.string(),
  new_kid: z.string().optional(),
  active_keys_count: z.number().optional(),
});

export const oauth2GrantSchema = z.object({
  grant_id: z.string(),
  client_id: z.string(),
  subject: z.string(),
  scopes: z.array(z.string()),
  created_at: z.string(),
  expires_at: z.string().nullable().optional(),
});
export const oauth2GrantListSchema = pagedListSchema(oauth2GrantSchema);
```

## BFF Endpoints Hierarchy
1. `GET /api/oauth2/clients/[id]` -> proxies `GET /admin/clients/{id}`
2. `PATCH /api/oauth2/clients/[id]` -> proxies `PATCH /admin/clients/{id}`
3. `DELETE /api/oauth2/clients/[id]` -> proxies `DELETE /admin/clients/{id}`
4. `POST /api/oauth2/clients/[id]/rotate-secret` -> proxies `POST /admin/clients/{id}/rotate-secret`
5. `GET /api/oauth2/scopes` -> proxies `GET /admin/scopes`
6. `POST /api/oauth2/scopes` -> proxies `POST /admin/scopes`
7. `DELETE /api/oauth2/scopes/[name]` -> proxies `DELETE /admin/scopes/{name}`
8. `POST /api/oauth2/tokens/introspect` -> proxies `POST /oauth2/introspect`
9. `POST /api/oauth2/tokens/revoke` -> proxies `POST /oauth2/revoke`
10. `POST /api/oauth2/keys/rotate` -> proxies `POST /admin/keys/rotate`
11. `GET /api/oauth2/grants` -> proxies `GET /admin/grants`

## UI Component Layout (`console/src/app/oauth2/`)
Tabs layout:
- **Clients Tab**: Client table, registration sheet/form, client detail modal with Secret Rotation (showing overlap duration and revealed secret warning).
- **Token Inspector Tab**: Textarea for token, execute introspection button, visual breakdown of claims, active state badge, and Revoke action.
- **Scopes Catalogue Tab**: List of scopes with descriptions & associated claims tags, and dialog to add new scopes.
- **Keys & JWKS Tab**: Active keys counter, list of keys with Key ID (`kid`), algorithm, usage, and "Rotate Keys" action button.
- **Consents & Grants Tab**: List of active grants per subject and client.
