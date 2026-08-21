# Autorix Themis: ABAC CEL Policy Engine Manual

**Autorix Themis** is an enterprise Attribute-Based Access Control (ABAC) engine built in Go. Powered by Google's **Common Expression Language (CEL)**, Themis evaluates complex, contextual authorization policies with sub-millisecond latency.

---

## 🏛️ 1. Architecture & Policy Execution Model

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

### 1.1 Key Capabilities

* **Google CEL Standard**: Fast, non-Turing complete expression evaluation designed specifically for policy decision points.
* **Context Injection**: Attributes from caller (`request.auth`), resource (`resource`), environmental context (`environment`), and arbitrary runtime parameters (`context`).
* **Deterministic Priority Ring**: Policies execute in strict order based on priority integers (lower number = evaluated first).
* **Multi-Tenant SaaS Partitioning**: Scoped by `tenant_id` for complete organizational isolation.
* **Policy Versioning & Zero-Downtime Rollback**: Keeps an immutable history of every policy revision with instant rollback.
* **Integrated Test Fixtures**: Attach unit-test fixtures to any policy and execute CI/CD-style test suites before promotion.

---

## 📜 2. Policy Schema & Context Variables

A policy consists of metadata, priority, labels, and the CEL expression:

```json
{
  "id": "pol_finance_mfa_lockdown",
  "tenant_id": "default",
  "name": "Enforce Finance MFA & Working Hours",
  "description": "Requires MFA authentication and business hours for wire transfers",
  "expression": "request.auth.claims.department == 'finance' && request.auth.mfa == true && request.time.hour >= 9 && request.time.hour <= 18",
  "priority": 1,
  "enabled": true,
  "labels": {
    "compliance": "soc2",
    "domain": "finance"
  }
}
```

### Evaluation Variables Environment

Themis injects standard evaluation namespaces into CEL expressions:

| Variable | Type | Description | Example Expression |
| :--- | :--- | :--- | :--- |
| `request.auth` | `map` | Identity & authentication claims from JWT or Session. | `request.auth.claims.department == "finance"` |
| `request.ip` | `string` | Client IP address. | `request.ip.startsWith("10.0.")` |
| `request.method` | `string` | HTTP verb or RPC method. | `request.method in ["POST", "PUT"]` |
| `request.time` | `timestamp` | UTC timestamp of the evaluation. | `request.time.hour >= 9 && request.time.hour <= 18` |
| `resource` | `map` | Target resource attributes. | `resource.amount < 50000.0` |
| `context` | `map` | Arbitrary custom key-value runtime parameters. | `context.is_break_glass == true` |

---

## 📡 3. Complete REST & gRPC API Reference

Themis provides a REST API on port `4488` and a high-performance gRPC interface on port `50052`.

### 3.1 List Policies (`GET /policies`)

Retrieves paginated policies for a tenant.

* **Method**: `GET`
* **Path**: `/policies`
* **Query Parameters**:
  - `tenant_id` *(string, optional, default: "default")*: Tenant partition.
  - `enabled_only` *(boolean, optional)*: Filter enabled policies only.
  - `limit` *(integer, optional, default: 50)*: Page size.
  - `cursor` *(string, optional)*: Pagination cursor.

#### Response (`200 OK`)
```json
{
  "data": [
    {
      "id": "pol_finance_mfa_lockdown",
      "tenant_id": "default",
      "name": "Enforce Finance MFA & Working Hours",
      "expression": "request.auth.claims.department == 'finance' && request.auth.mfa == true",
      "priority": 1,
      "enabled": true,
      "labels": { "domain": "finance" },
      "created_at": "2026-08-20T08:00:00Z"
    }
  ],
  "next_cursor": "MXwyMDI2LTA4LTIwVDA4OjAwOjAwWnxyb290",
  "has_more": false
}
```

---

### 3.2 Create Policy (`POST /policies`)

Creates a new CEL ABAC policy.

* **Method**: `POST`
* **Path**: `/policies`
* **Headers**: `Content-Type: application/json`

#### Request Body
```json
{
  "tenant_id": "default",
  "name": "Block Suspicious Geos",
  "description": "Denies requests from non-whitelisted country codes",
  "expression": "!['US', 'CA', 'CL', 'BR', 'AR', 'EU'].exists(c, c == context.country_code)",
  "priority": 1,
  "enabled": true,
  "labels": { "security": "geo-fence" }
}
```

#### Response (`201 Created`)
```json
{
  "id": "pol_9a8b7c6d-5e4f-3a2b-1c0d-9e8f7a6b5c4d",
  "tenant_id": "default",
  "name": "Block Suspicious Geos",
  "description": "Denies requests from non-whitelisted country codes",
  "expression": "!['US', 'CA', 'CL', 'BR', 'AR', 'EU'].exists(c, c == context.country_code)",
  "priority": 1,
  "enabled": true,
  "labels": { "security": "geo-fence" },
  "created_at": "2026-08-20T08:30:00Z"
}
```

---

### 3.3 Get Policy by ID (`GET /policies/{id}`)

* **Method**: `GET`
* **Path**: `/policies/{id}`
* **Query Parameters**: `tenant_id` *(optional)*.

---

### 3.4 Update Policy (`PUT /policies/{id}`)

* **Method**: `PUT`
* **Path**: `/policies/{id}`
* **Headers**: `Content-Type: application/json`
* **Request Body**: Same structure as Create Policy. Automatically snapshots the previous version.

---

### 3.5 Delete Policy (`DELETE /policies/{id}`)

* **Method**: `DELETE`
* **Path**: `/policies/{id}`
* **Query Parameters**: `tenant_id` *(optional)*.

---

### 3.6 Evaluate Policy Context (`POST /policies/evaluate`)

Evaluates the active policy ring against a runtime request payload.

* **Method**: `POST`
* **Path**: `/policies/evaluate`
* **Headers**: `Content-Type: application/json`

#### Request Body
```json
{
  "tenant_id": "default",
  "policy_id": "pol_finance_mfa_lockdown",
  "label_filter": { "domain": "finance" },
  "payload": {
    "request": {
      "auth": {
        "mfa": true,
        "claims": { "department": "finance" }
      },
      "ip": "10.0.4.50",
      "time": "2026-08-20T14:30:00Z"
    },
    "resource": {
      "amount": 25000.00
    },
    "context": {
      "country_code": "CL"
    }
  }
}
```

#### Response (`200 OK`)
```json
{
  "allowed": true,
  "matched_policy_id": "pol_finance_mfa_lockdown",
  "reason": "policy_passed",
  "evaluations": [
    {
      "policy_id": "pol_finance_mfa_lockdown",
      "result": true,
      "duration_ns": 42500
    }
  ]
}
```

---

### 3.7 Validate CEL Expression (`POST /policies/validate`)

Compiles and validates a CEL expression without persisting it.

* **Method**: `POST`
* **Path**: `/policies/validate` or `/admin/policies/validate`

#### Request Body
```json
{
  "expression": "request.auth.claims.role == 'admin'"
}
```

#### Response (`200 OK`)
```json
{
  "valid": true
}
```

---

### 3.8 Dry-Run Evaluation (`POST /policies/dry-run`)

Executes a transient CEL expression against a payload without creating a policy record.

* **Method**: `POST`
* **Path**: `/policies/dry-run` or `/admin/policies/dry-run`

#### Request Body
```json
{
  "expression": "resource.price * 1.19 <= 100.0",
  "payload": {
    "resource": { "price": 80.0 }
  }
}
```

#### Response (`200 OK`)
```json
{
  "result": true,
  "output_type": "bool"
}
```

---

### 3.9 Policy Versions & Rollback (`GET /policies/{id}/versions` & `POST /policies/{id}/rollback/{version}`)

- `GET /policies/{id}/versions`: List all historical versions of a policy.
- `POST /policies/{id}/rollback/{version}`: Restores the policy configuration to a specific version number.

---

### 3.10 Test Fixtures & Automated Test Suite

- `GET /policies/{id}/fixtures`: List test fixtures attached to a policy.
- `POST /policies/{id}/fixtures`: Create a fixture with expected outcome.
- `DELETE /policies/{id}/fixtures/{fixture_id}`: Remove a fixture.
- `POST /policies/{id}/test-suite`: Executes all test fixtures against the current expression.

#### Run Test Suite Response (`200 OK`)
```json
{
  "total": 3,
  "passed": 3,
  "failed": 0,
  "results": [
    {
      "fixture_id": "fix_01",
      "name": "Finance Operator with MFA in business hours",
      "passed": true,
      "expected": true,
      "actual": true
    }
  ]
}
```

---

### 3.11 gRPC Interface (`:50052`)

Themis exposes high-performance protobuf RPCs in `themis/proto/themis.proto`:
- `Evaluate(EvaluateRequest) returns (EvaluateResponse)`
- `ValidateExpression(ValidateRequest) returns (ValidateResponse)`
- `DryRun(DryRunRequest) returns (DryRunResponse)`
- `ListPolicies(ListPoliciesRequest) returns (ListPoliciesResponse)`
- `GetPolicy(GetPolicyRequest) returns (GetPolicyResponse)`
- `CreatePolicy(CreatePolicyRequest) returns (CreatePolicyResponse)`
- `UpdatePolicy(UpdatePolicyRequest) returns (UpdatePolicyResponse)`
- `DeletePolicy(DeletePolicyRequest) returns (DeletePolicyResponse)`

---

## 🛠️ 4. Production Recipes & CEL Patterns

### 1. Emergency Break-Glass Rule
```javascript
context.is_break_glass == true && request.auth.claims.security_clearance >= 3
```

### 2. Time-Bound Wire Transfer Authorization
```javascript
request.auth.claims.role == "finance_officer" &&
request.auth.mfa == true &&
request.time.hour >= 9 && request.time.hour <= 17 &&
resource.amount <= 1000000.0
```

### 3. Subnet Isolation & Geo-Fencing
```javascript
request.ip.startsWith("10.240.") &&
['US', 'CL', 'BR'].exists(c, c == context.geo_country)
```
