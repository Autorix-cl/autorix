# Autorix Aegis: Zero-Trust Reverse PEP Proxy Manual

**Autorix Aegis** is a high-performance Zero-Trust Policy Enforcement Point (PEP) and Reverse Access Proxy inspired by Ory Oathkeeper. Sitting at the edge of your microservice mesh, Aegis intercepts inbound HTTP requests, authenticates callers, evaluates ReBAC/ABAC authorization decisions, mutates upstream headers, and securely proxies traffic to backend workloads.

---

## 🏛️ 1. Architecture & The 4-Stage Proxy Pipeline

```text
                               [ INBOUND REQUEST ]
                                        │
                                        ▼ (Port :4455)
   ┌────────────────────────────────────────────────────────────────────────┐
   │                          Autorix Aegis PEP                             │
   │                                                                        │
   │  ┌─────────────────┐                                                   │
   │  │ 1. Rule Matcher │ (Method, Path Pattern, Host Regex)                │
   │  └────────┬────────┘                                                   │
   │           │                                                            │
   │  ┌────────▼──────────────┐                                             │
   │  │ 2. Authenticator      │ (JWT / Bearer Token / API Key / Session)    │
   │  └────────┬──────────────┘                                             │
   │           │                                                            │
   │  ┌────────▼──────────────┐                                             │
   │  │ 3. Authorizer         │ (Nexus Zanzibar ReBAC & Themis CEL ABAC)    │
   │  └────────┬──────────────┘                                             │
   │           │                                                            │
   │  ┌────────▼──────────────┐                                             │
   │  │ 4. Header Mutator     │ (Injects X-User-ID, Claims, Custom Headers) │
   │  └────────┬──────────────┘                                             │
   └───────────┼────────────────────────────────────────────────────────────┘
               │
               ▼ (Authenticated & Authorized Request)
     [ Backend Microservice ] (e.g. http://billing-service:8080)
```

### Pipeline Execution Contract

1. **Rule Matcher**: Evaluates URL path, HTTP method, and optional hostname to find the highest-precedence matching rule.
2. **Authenticator Stage**: Validates credentials (JWT signature via JWKS, Vulcan API key, or Ego session cookie). If unauthenticated, immediately aborts with `401 Unauthorized`.
3. **Authorizer Stage**: Executes fine-grained permission checks against Nexus (gRPC Zanzibar) or Themis (CEL ABAC). If unauthorized, immediately aborts with `403 Forbidden`.
4. **Mutator Stage**: Transforms the request before forwarding to the upstream service. Can inject identity headers or generate a signed identity token.

---

## 📜 2. Rule Configuration Schema

Rules are configured dynamically via the Admin REST API (`:4456`) or loaded from YAML files.

```json
{
  "id": "rule_protect_customer_data",
  "description": "Protects customer PII endpoint with JWT auth and Zanzibar editor check",
  "match": {
    "url": "http://api.enterprise.io/api/v1/customers/<[0-9a-f-]+>",
    "methods": ["GET", "PUT", "DELETE"]
  },
  "authenticators": [
    {
      "handler": "jwt",
      "config": {
        "jwks_url": "http://janus:4444/.well-known/jwks.json",
        "required_scope": ["customers:read", "customers:write"]
      }
    }
  ],
  "authorizers": [
    {
      "handler": "nexus_rebac",
      "config": {
        "namespace": "customers",
        "relation": "editor",
        "subject_from": "jwt.sub",
        "object_from": "url_param.0"
      }
    }
  ],
  "mutators": [
    {
      "handler": "header",
      "config": {
        "headers": {
          "X-User-ID": "{{ .Subject }}",
          "X-User-Email": "{{ .Claims.email }}",
          "X-Authenticated-By": "autorix-aegis-pep"
        }
      }
    }
  ],
  "upstream": {
    "url": "http://customer-svc.internal:8080",
    "preserve_host": false,
    "strip_path": ""
  }
}
```

---

## 🛠️ 3. Handler Catalog & Capabilities

### 3.1 Authenticator Handlers

| Handler | Configuration | Description |
| :--- | :--- | :--- |
| `anonymous` | `{}` | Allows unauthenticated traffic; assigns subject `anonymous`. |
| `noop` | `{}` | Passes credentials straight through without inspection. |
| `jwt` | `{"jwks_url": "...", "required_scope": [...]}` | Validates RS256/ES256 signatures, `iss`, `aud`, and `exp` claims. |
| `api_key` | `{"vulcan_url": "http://vulcan:4466"}` | Extracts `X-API-Key` or `Authorization: Bearer av_live_...` and verifies against Vulcan. |
| `cookie_session` | `{"ego_url": "http://ego:4433", "cookie_name": "autorix_session_token"}` | Validates active user sessions against Ego. |

### 3.2 Authorizer Handlers

| Handler | Configuration | Description |
| :--- | :--- | :--- |
| `allow` | `{}` | Unconditionally permits authenticated callers. |
| `deny` | `{}` | Unconditionally rejects requests. |
| `nexus_rebac` | `{"nexus_addr": "nexus:50051", "namespace": "...", "relation": "..."}` | Queries Zanzibar graph in Nexus. |
| `themis_abac` | `{"themis_addr": "themis:50052", "tenant_id": "default"}` | Evaluates Google CEL policies against dynamic request attributes. |

### 3.3 Mutator Handlers

| Handler | Configuration | Description |
| :--- | :--- | :--- |
| `noop` | `{}` | Forwards request headers as received. |
| `header` | `{"headers": {"X-Auth-Sub": "{{ .Subject }}"}}` | Injects custom headers with Go template resolution. |
| `id_token` | `{"issuer": "http://aegis", "jwks_url": "..."}` | Mints a short-lived, signed JWT assertion for zero-trust downstream consumption. |

---

## 🚀 4. Admin REST API (`:4456`) Reference

### 4.1 Create or Update a Rule (`POST /rules` or `PUT /rules/{id}`)

```bash
curl -X POST http://localhost:4456/rules \
  -H "Content-Type: application/json" \
  -d '{
    "id": "public_health_check",
    "match": { "url": "http://localhost:4455/health", "methods": ["GET"] },
    "authenticators": [{ "handler": "anonymous" }],
    "authorizers": [{ "handler": "allow" }],
    "mutators": [{ "handler": "noop" }],
    "upstream": { "url": "http://backend-app:8080/health" }
  }'
```

### 4.2 Test Rule Match Simulator (`POST /rules/test-match`)

Simulate incoming request paths to verify which rule will intercept it before deployment:

```bash
curl -X POST http://localhost:4456/rules/test-match \
  -H "Content-Type: application/json" \
  -d '{
    "method": "GET",
    "path": "/api/v1/customers/c9a8b7-uuid"
  }'
```

**Response:**
```json
{
  "matched": true,
  "rule_id": "rule_protect_customer_data",
  "upstream": "http://customer-svc.internal:8080/api/v1/customers/c9a8b7-uuid"
}
```

### 4.3 Reordering Rule Precedence (`PUT /rules/reorder`)

```bash
curl -X PUT http://localhost:4456/rules/reorder \
  -H "Content-Type: application/json" \
  -d '{
    "rule_ids": ["public_health_check", "rule_protect_customer_data", "catchall_deny"]
  }'
```

### 4.4 Rule Version History & Rollback

* `GET /rules/versions`: Lists historical snapshots of rule sets.
* `POST /rules/rollback/3`: Instantly rolls back the active proxy configuration to version snapshot 3.
* `GET /rules/export`: Exports all rules in YAML/JSON for GitOps workflows.
* `POST /rules/import`: Replaces or merges rules from a GitOps repository payload.
