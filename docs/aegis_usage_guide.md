# Autorix Aegis: Zero-Trust Reverse PEP Proxy Manual

**Autorix Aegis** is a high-performance Zero-Trust Policy Enforcement Point (PEP) and Reverse Access Proxy inspired by Ory Oathkeeper. Operating at the perimeter of your microservice mesh, Aegis intercepts inbound HTTP requests, authenticates callers, evaluates ReBAC/ABAC authorization decisions against Nexus and Themis, mutates upstream headers, and securely proxies traffic to backend workloads.

---

## 🏛️ 1. Architecture & The 4-Stage Proxy Pipeline

```text
                                [ INBOUND REQUEST ]
                                         │
                                         ▼ (PEP Proxy Port :4455)
   ┌────────────────────────────────────────────────────────────────────────┐
   │                          Autorix Aegis PEP                             │
   │                                                                        │
   │  ┌─────────────────┐                                                   │
   │  │ 1. Rule Matcher │ (Method, URL Path Pattern, Hostname)              │
   │  └────────┬────────┘                                                   │
   │           │                                                            │
   │  ┌────────▼──────────────┐                                             │
   │  │ 2. Authenticator      │ (JWT / Bearer Token / Vulcan Key / Session) │
   │  └────────┬──────────────┘                                             │
   │           │                                                            │
   │  ┌────────▼──────────────┐                                             │
   │  │ 3. Authorizer         │ (Nexus Zanzibar ReBAC & Themis CEL ABAC)    │
   │  └────────┬──────────────┘                                             │
   │           │                                                            │
   │  ┌────────▼──────────────┐                                             │
   │  │ 4. Header Mutator     │ (Injects X-User-ID, Claims, Context Headers)│
   │  └────────┬──────────────┘                                             │
   └───────────┼────────────────────────────────────────────────────────────┘
               │
               ▼ (Authenticated & Authorized Request)
     [ Backend Microservice ] (e.g. http://billing-service:8080)
```

### 1.1 Pipeline Execution Contract

1. **Rule Matcher**: Evaluates URL path pattern, HTTP method, and optional hostname to find the matching rule.
2. **Authenticator Stage**: Validates credentials (JWT via Janus JWKS, Vulcan Macaroon API key, or Ego session cookie). If unauthenticated, immediately aborts with `401 Unauthorized`.
3. **Authorizer Stage**: Executes fine-grained permission checks against Nexus (gRPC Zanzibar) or Themis (CEL ABAC). If unauthorized, immediately aborts with `403 Forbidden`.
4. **Mutator Stage**: Transforms the request before forwarding to the upstream service (e.g. injects `X-User-ID`, `X-User-Email`).
5. **Upstream Proxy**: Forwards request to backend workload.

---

## 📜 2. Rule Configuration Schema

Rules are configured dynamically via the Admin REST API (`:4456`) or imported from JSON/YAML files.

```json
{
  "id": "rule_protect_customer_pii",
  "description": "Protects customer PII endpoint with JWT auth and Zanzibar editor check",
  "match": {
    "url": "http://api.enterprise.corp/api/v1/customers/<[0-9a-f-]+>",
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

## 📡 3. Complete Admin API Reference

The Aegis Admin REST API operates on port `4456`.

### 3.1 List Rules (`GET /rules` or `GET /admin/rules`)

* **Method**: `GET`
* **Path**: `/rules`
* **Query Parameters**: `limit` *(integer)*, `cursor` *(string)*.

#### Response (`200 OK`)
```json
{
  "data": [
    {
      "id": "rule_protect_customer_pii",
      "match": {
        "url": "http://api.enterprise.corp/api/v1/customers/<[0-9a-f-]+>",
        "methods": ["GET", "PUT", "DELETE"]
      },
      "upstream": {
        "url": "http://customer-svc.internal:8080"
      }
    }
  ],
  "next_cursor": "cnVsZV9wcm90ZWN0X2N1c3RvbWVyX3BpaQ==",
  "has_more": false
}
```

---

### 3.2 Create Rule (`POST /rules` or `POST /admin/rules`)

* **Method**: `POST`
* **Path**: `/rules`
* **Headers**: `Content-Type: application/json`

---

### 3.3 Get Rule by ID (`GET /rules/{id}`)

* **Method**: `GET`
* **Path**: `/rules/{id}`

---

### 3.4 Update Rule (`PUT /rules/{id}`)

* **Method**: `PUT`
* **Path**: `/rules/{id}`

---

### 3.5 Delete Rule (`DELETE /rules/{id}`)

* **Method**: `DELETE`
* **Path**: `/rules/{id}`

---

### 3.6 Reorder Rules Precedence (`PUT /rules/reorder`)

Updates the evaluation sequence of proxy rules:

```bash
curl -X PUT http://localhost:4456/rules/reorder \
  -H "Content-Type: application/json" \
  -d '{
    "rule_ids": ["rule_high_priority_admin", "rule_general_api", "rule_public_health"]
  }'
```

---

### 3.7 Import Rules (`POST /rules/import`)

Bulk imports rules from JSON or YAML.

```bash
curl -X POST http://localhost:4456/rules/import \
  -H "Content-Type: application/json" \
  -d '[ ... array of rules ... ]'
```

---

### 3.8 Export Rules (`GET /rules/export`)

Exports all proxy rules formatted as YAML.

---

### 3.9 Versioning & Rollback

- `GET /rules/versions`: Lists historical snapshots of proxy routing rules.
- `POST /rules/rollback/{version}`: Rolls back the entire active rule set to a previous version number.

---

### 3.10 Handler Catalogue (`GET /handlers`)

Returns all registered authenticator, authorizer, and mutator handlers supported by the proxy.

---

### 3.11 Rule Match Simulator / Dry-Run (`POST /rules/test-match`)

Tests which rule matches an incoming method and URL path, returning the full dry-run execution trace.

* **Method**: `POST`
* **Path**: `/rules/test-match`
* **Request Body**:
```json
{
  "method": "GET",
  "path": "/api/v1/customers/c1a2b3d4-e5f6",
  "headers": {
    "Authorization": "Bearer eyJhbGciOi..."
  }
}
```

#### Response (`200 OK`)
```json
{
  "matched": true,
  "rule": {
    "id": "rule_protect_customer_pii"
  },
  "trace": {
    "matched_rule_id": "rule_protect_customer_pii",
    "authenticator": { "handler": "jwt", "status": "authenticated" },
    "authorizer": { "handler": "nexus_rebac", "status": "allowed" }
  }
}
```

---

## 🛠️ 4. Production Recipes

### Zero-Trust Reverse Proxy Integration
* Intercept all inbound edge traffic on port `4455`.
* Forward authenticated and header-mutated traffic to backend clusters on internal network tiers.
