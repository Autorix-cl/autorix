# Autorix Themis: ABAC CEL Policy Engine Guide

**Autorix Themis** is a dedicated Attribute-Based Access Control (ABAC) engine built in Go. It uses Google's **Common Expression Language (CEL)** to evaluate complex, contextual authorization rules with microsecond latency.

---

## 🏛️ Architecture & Policy Model

```text
       [ Authorization Request ]
                   │
                   ▼
       ┌───────────────────────┐
       │     Autorix Themis    │ (REST :4488 / gRPC :50052)
       │                       │
       │   ┌───────────────┐   │
       │   │  CEL Compiler │   │ (AST caching & type checking)
       │   └───────┬───────┘   │
       │           │           │
       │   ┌───────▼───────┐   │
       │   │ Priority Ring │   │ (Evaluates in order: Priority 1..N)
       │   └───────┬───────┘   │
       └───────────┼───────────┘
                   │
                   ▼
         [ Decision: Pass/Deny ]
```

### Key Capabilities

* **Google CEL Standard**: Fast, non-Turing complete expression evaluation designed specifically for policy decision points.
* **Context Injection**: Attributes from caller (`request.auth`), resource (`resource`), environmental context (`environment`), and arbitrary runtime parameters (`context`).
* **Priority Ordering**: Policies execute deterministically based on priority integers (lower number = higher priority).
* **Multi-Tenant Partitioning**: Scoped by `tenant_id` for SaaS isolation.

---

## 📜 Policy Definition Schema

A Themis policy consists of:

```json
{
  "ID": "pol_finance_mfa_lockdown",
  "TenantID": "default",
  "Name": "Enforce Finance MFA & Working Hours",
  "Description": "Requires MFA authentication and business hours for wire transfers",
  "Expression": "request.auth.claims.department == 'finance' && request.auth.mfa == true && request.time.hour >= 9 && request.time.hour <= 18",
  "Priority": 1,
  "Enabled": true,
  "Labels": {
    "compliance": "soc2",
    "domain": "finance"
  }
}
```

### Evaluation Variables Environment

Themis injects standard evaluation namespaces into CEL expressions:

| Variable | Type | Description | Example |
| :--- | :--- | :--- | :--- |
| `request.auth` | `map` | Identity & authentication claims from JWT / Session | `request.auth.claims.sub`, `request.auth.mfa` |
| `request.ip` | `string` | Client IP address | `request.ip.startsWith('10.0.')` |
| `request.method` | `string` | HTTP verb or gRPC method | `request.method == 'POST'` |
| `request.time` | `timestamp` | Evaluation time (UTC) | `request.time.hour >= 9` |
| `resource` | `map` | Target resource attributes | `resource.owner_id`, `resource.amount` |
| `context` | `map` | Arbitrary custom key-value payload | `context.is_break_glass == true` |

---

## 🚀 Creating & Managing Policies (REST API)

### 1. Create a Policy

```bash
curl -X POST http://localhost:4488/v1/policies \
  -H "Content-Type: application/json" \
  -d '{
    "Name": "Block Suspicious Geos",
    "TenantID": "default",
    "Description": "Denies requests from non-whitelisted country codes",
    "Expression": "!['US', 'CA', 'CL', 'BR', 'AR', 'EU'].exists(c, c == context.country_code)",
    "Priority": 1,
    "Enabled": true,
    "Labels": { "security": "geo-fence" }
  }'
```

### 2. List Policies

```bash
curl -s http://localhost:4488/v1/policies?tenant_id=default
```

### 3. Evaluate Policies

Test expressions dynamically against sample request payloads:

```bash
curl -X POST http://localhost:4488/v1/policies/evaluate \
  -H "Content-Type: application/json" \
  -d '{
    "TenantID": "default",
    "Context": {
      "request": {
        "auth": {
          "claims": { "department": "finance" },
          "mfa": true
        },
        "time": { "hour": 14 }
      },
      "resource": {
        "type": "wire_transfer",
        "amount": 50000
      }
    }
  }'
```

**Response (`200 OK`):**
```json
{
  "AllPassed": true,
  "Results": [
    {
      "PolicyID": "pol_finance_mfa_lockdown",
      "PolicyName": "Enforce Finance MFA & Working Hours",
      "Passed": true,
      "Expression": "request.auth.claims.department == 'finance' && request.auth.mfa == true"
    }
  ],
  "TotalEvaluated": 1
}
```

---

## ⚡ gRPC Interface (`:50052`)

Themis exposes high-performance protobuf RPCs for in-cluster evaluation:

```protobuf
service ThemisService {
  rpc EvaluatePolicies (EvaluateRequest) returns (EvaluateResponse);
  rpc CreatePolicy (CreatePolicyRequest) returns (Policy);
  rpc GetPolicy (GetPolicyRequest) returns (Policy);
  rpc ListPolicies (ListPoliciesRequest) returns (ListPoliciesResponse);
  rpc DeletePolicy (DeletePolicyRequest) returns (DeletePolicyResponse);
}
```
