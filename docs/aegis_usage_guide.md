# Enforcing Edge Security with Autorix Aegis PEP Proxy

Autorix Aegis is your Zero-Trust Policy Enforcement Point (PEP) and Reverse Access Proxy. Operating at the perimeter of your microservice mesh, it intercepts inbound HTTP requests, authenticates callers, evaluates authorization decisions against Nexus and Themis, mutates upstream headers, and securely proxies traffic to backend workloads.

## Quick Path: Proxying an API Endpoint

The fastest way to secure an endpoint is to configure a rule via the Admin API (`:4456`). This rule authenticates via JWT, checks permissions in Nexus, and injects user identity headers.

```bash
curl -X POST http://localhost:4456/rules \
  -H "Content-Type: application/json" \
  -d '{
    "id": "rule_protect_customer_pii",
    "match": {
      "url": "http://api.enterprise.corp/api/v1/customers/<[0-9a-f-]+>",
      "methods": ["GET"]
    },
    "authenticators": [{"handler": "jwt", "config": {"jwks_url": "http://janus:4444/.well-known/jwks.json"}}],
    "authorizers": [{"handler": "nexus_rebac", "config": {"namespace": "customers", "relation": "viewer", "subject_from": "jwt.sub", "object_from": "url_param.0"}}],
    "mutators": [{"handler": "header", "config": {"headers": {"X-User-ID": "{{ .Subject }}"}}}],
    "upstream": {"url": "http://customer-svc.internal:8080"}
  }'
```

---

## Details

### 1. The 4-Stage Proxy Pipeline

Aegis processes every inbound request through a strict pipeline before proxying it to your backend microservice (`:4455`).

1. **Rule Matcher**: Evaluates URL path pattern, HTTP method, and optional hostname.
2. **Authenticator Stage**: Validates credentials (JWT, Macaroon API key, or Session). Aborts with `401 Unauthorized` if invalid.
3. **Authorizer Stage**: Executes fine-grained permission checks (Nexus Zanzibar ReBAC & Themis CEL ABAC). Aborts with `403 Forbidden` if denied.
4. **Mutator Stage**: Transforms the request (e.g., injects `X-User-ID` headers) before forwarding.

### 2. Admin API Reference (`:4456`)

Use the Admin API to dynamically manage your edge rules.

*   **List Rules**: `GET /rules`
*   **Create Rule**: `POST /rules` (Requires JSON body matching Rule Schema)
*   **Get/Update/Delete Rule**: `GET`, `PUT`, `DELETE /rules/{id}`
*   **Reorder Rules Precedence**: `PUT /rules/reorder` (Provide array of `rule_ids`)
*   **Bulk Import/Export**: `POST /rules/import` (JSON/YAML array) | `GET /rules/export`
*   **Versioning**: `GET /rules/versions` | `POST /rules/rollback/{version}`
*   **Handler Catalogue**: `GET /handlers` (List supported authenticators, authorizers, mutators)

### 3. Rule Match Simulator (Dry-Run)

Test which rule matches a request and review the execution trace without making a real proxy call.

```bash
curl -X POST http://localhost:4456/rules/test-match \
  -H "Content-Type: application/json" \
  -d '{
    "method": "GET",
    "path": "/api/v1/customers/1234",
    "headers": {"Authorization": "Bearer eyJhbGci..."}
  }'
```

---

## Checklist: Production Readiness

- [ ] Ensure Aegis PEP proxy port (`4455`) is exposed to external traffic (Edge).
- [ ] Ensure Aegis Admin port (`4456`) is restricted to internal administrative traffic only.
- [ ] Verify `jwks_url` in JWT authenticators points to a highly available Janus instance.
- [ ] Test rule matching priority by calling the `/rules/test-match` simulator.
- [ ] Set `preserve_host: false` or `strip_path` in upstream configs according to backend needs.

---

## Next Step

Now that edge traffic is secured, learn how Aegis integrates with the broader ecosystem in the [API Reference & Integration Guide](./api_reference_and_integration_guide.md) or explore [Nexus](./nexus_usage_guide.md) for configuring your ReBAC rules.
