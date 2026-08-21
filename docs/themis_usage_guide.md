# Evaluate Authorization Policies with Autorix Themis

Autorix Themis evaluates complex, contextual authorization policies with sub-millisecond latency. It is an enterprise Attribute-Based Access Control (ABAC) engine powered by Google's Common Expression Language (CEL).

## Quick path

Themis exposes a REST API on port `4488` and gRPC on port `50052`. 

To quickly create and evaluate a policy:

1. **Create a policy** (`POST /policies`):
   ```json
   {
     "tenant_id": "default",
     "name": "Block Suspicious Geos",
     "expression": "!['US', 'CA', 'CL', 'BR', 'AR', 'EU'].exists(c, c == context.country_code)",
     "priority": 1,
     "enabled": true
   }
   ```

2. **Evaluate the context** (`POST /policies/evaluate`):
   ```json
   {
     "tenant_id": "default",
     "policy_id": "pol_finance_mfa_lockdown",
     "payload": {
       "request": { "auth": { "mfa": true } },
       "context": { "country_code": "CL" }
     }
   }
   ```

## Details

### Architecture & Policy Execution Model

Themis executes policies in a deterministic priority ring (lower number = evaluated first). It provides multi-tenant SaaS partitioning via `tenant_id` and supports policy versioning with zero-downtime rollback.

```text
       [ Authorization Request / Context ]
                        │
                        ▼ (REST :4488 / gRPC :50052)
       ┌────────────────────────────────────────────────────────┐
       │                     Autorix Themis                     │
       │                                                        │
       │  ┌──────────────────────┐  ┌────────────────────────┐  │
       │  │ CEL Compiler         │  │ Policy Version Vault   │  │
       │  │ & Type Checker       │  │ (Audit Trail & History)│  │
       │  └──────────┬───────────┘  └────────────────────────┘  │
       │             │                                          │
       │  ┌──────────▼───────────┐  ┌────────────────────────┐  │
       │  │ Priority Ring Engine │  │ Automated Test Suites  │  │
       │  │ (Evaluates 1..N)     │  │ & Policy Fixtures      │  │
       │  └──────────┬───────────┘  └────────────────────────┘  │
       └─────────────┼──────────────────────────────────────────┘
                     │
                     ▼
          [ Decision: Pass / Deny ]
```

### Policy Schema & Context Variables

A policy consists of metadata, priority, labels, and the CEL expression. Themis injects standard namespaces into CEL expressions:

* `request.auth` (map): Identity & authentication claims from JWT/Session (e.g., `request.auth.claims.department == "finance"`).
* `request.ip` (string): Client IP address.
* `request.method` (string): HTTP verb or RPC method.
* `request.time` (timestamp): UTC timestamp of the evaluation.
* `resource` (map): Target resource attributes (e.g., `resource.amount < 50000.0`).
* `context` (map): Arbitrary custom key-value runtime parameters.

### Complete API Reference (REST :4488)

* **`GET /policies`**: List policies for a tenant (supports pagination).
* **`POST /policies`**: Create a new CEL ABAC policy.
* **`GET /policies/{id}`**: Get policy by ID.
* **`PUT /policies/{id}`**: Update a policy (automatically snapshots the previous version).
* **`DELETE /policies/{id}`**: Delete a policy.
* **`POST /policies/evaluate`**: Evaluate the active policy ring against a runtime request payload.
* **`POST /policies/validate`**: Compile and validate a CEL expression without persisting it.
* **`POST /policies/dry-run`**: Execute a transient CEL expression against a payload.
* **`GET /policies/{id}/versions`**: List all historical versions.
* **`POST /policies/{id}/rollback/{version}`**: Restore to a specific version number.
* **Test Fixtures**: Manage and execute test fixtures via `/policies/{id}/fixtures` and `/policies/{id}/test-suite`.

### gRPC Interface (:50052)

Themis exposes high-performance protobuf RPCs in `themis/proto/themis.proto` (e.g., `Evaluate`, `ValidateExpression`, `ListPolicies`, `CreatePolicy`, etc.).

### Production Recipes & CEL Patterns

* **Emergency Break-Glass Rule**: `context.is_break_glass == true && request.auth.claims.security_clearance >= 3`
* **Time-Bound Wire Transfer**: `request.auth.claims.role == "finance_officer" && request.auth.mfa == true && request.time.hour >= 9 && request.time.hour <= 17 && resource.amount <= 1000000.0`
* **Subnet Isolation & Geo-Fencing**: `request.ip.startsWith("10.240.") && ['US', 'CL', 'BR'].exists(c, c == context.geo_country)`

## Checklist

* [ ] Construct policies using valid CEL expressions.
* [ ] Verify that injected variables (`request`, `resource`, `context`) match the evaluation payload.
* [ ] Write automated test fixtures for complex policies before deploying to production.
* [ ] Ensure policies are prioritized correctly within the deterministic ring.

## Next step

Explore the `themis/proto/themis.proto` file if you plan to integrate over gRPC, or run a `POST /policies/dry-run` to experiment with your first CEL expressions.
