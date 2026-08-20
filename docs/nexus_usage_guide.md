# Autorix Nexus: Zanzibar ReBAC & Authorization Engine Manual

**Autorix Nexus** is an ultra-low latency authorization engine combining **Google Zanzibar** relation graphs with **Attribute-Based Access Control (ABAC)** powered by Google Common Expression Language (CEL). It evaluates complex hierarchical permissions across millions of entities in sub-5ms latency.

---

## 🏛️ 1. Architectural Concepts & Data Model

Unlike legacy RBAC systems that assign flat roles to users (`alice -> role:admin`), Nexus represents permissions as a **directed graph of relationships**:

$$\text{namespace:object\#relation@subject\_namespace:subject\_id[\#subject\_relation] [with caveat]}$$

### 1.1 The Zanzibar Tuple Structure

| Field | Type | Description | Example |
| :--- | :--- | :--- | :--- |
| `namespace` | `string` | The resource category or domain boundary. | `documents`, `organizations`, `projects` |
| `object` | `string` | Unique identifier of the resource instance. | `roadmap_2026_q3`, `org_enterprise_corp` |
| `relation` | `string` | The named edge linking subject to object. | `owner`, `editor`, `viewer`, `parent` |
| `subject_namespace` | `string` | Namespace of the subject (defaults to `user`). | `user`, `group`, `service_account` |
| `subject_id` | `string` | Unique identifier of the subject. | `usr_4455`, `grp_engineering` |
| `subject_relation` | `string` *(optional)* | Subject set indirection (for group membership). | `member`, `admin` |
| `caveat_name` | `string` *(optional)* | Name of a registered CEL condition for ABAC. | `require_work_hours`, `ip_in_subnet` |
| `caveat_context` | `map` *(optional)* | Static binding parameters for the caveat. | `{"allowed_ip": "10.0.0.0/8"}` |

### 1.2 Subject Sets & Transitive Inheritance

Nexus natively supports **Subject Sets** (users-in-usersets) allowing infinite hierarchical inheritance without duplicating permissions:

```text
  [ user:alice ] ──(member)──► [ group:engineering ] ──(editor)──► [ document:roadmap_2026 ]
```

* Tuple 1: `group:engineering#member@user:alice`
* Tuple 2: `document:roadmap_2026#editor@group:engineering#member`

When querying `Check(document:roadmap_2026, editor, user:alice)`, Nexus traverses the graph recursively and resolves `allowed: true`.

---

## ⚙️ 2. PostgreSQL Storage Engine & Optimization

Nexus stores tuples in the `relation_tuples` table in `autorix_nexus`:

```sql
CREATE TABLE relation_tuples (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    namespace VARCHAR(128) NOT NULL,
    object VARCHAR(256) NOT NULL,
    relation VARCHAR(128) NOT NULL,
    subject_namespace VARCHAR(128) NOT NULL DEFAULT 'user',
    subject_object VARCHAR(256) NOT NULL,
    subject_relation VARCHAR(128) NOT NULL DEFAULT '',
    caveat_name VARCHAR(128),
    caveat_context JSONB,
    commit_time TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT uq_relation_tuple UNIQUE(namespace, object, relation, subject_namespace, subject_object, subject_relation)
);

CREATE INDEX idx_tuples_object_lookup ON relation_tuples(namespace, object, relation);
CREATE INDEX idx_tuples_subject_lookup ON relation_tuples(subject_namespace, subject_object, subject_relation);
```

### Recursive Graph Traversal Engine

Nexus executes graph expansions using depth-limited recursive traversal with cycle detection:
* **Max Depth**: Configurable (default: 10 levels).
* **Cycle Guard**: Visited set $(N, O, R, S)$ tracking prevents infinite loops in circular group memberships (`A -> B -> A`).

---

## 📜 3. Namespace Schemas & Rewrite Rules

Namespaces define permission inheritance rules via schema definitions.

### 3.1 Defining a Namespace Schema

```bash
curl -X POST http://localhost:8080/admin/namespaces \
  -H "Content-Type: application/json" \
  -d '{
    "name": "documents",
    "relations": {
      "owner": {},
      "editor": {
        "union": ["owner"]
      },
      "viewer": {
        "union": ["editor", "parent.viewer"]
      }
    }
  }'
```

* **Union**: Grants permission if the subject holds any of the specified relations (e.g. `owner` automatically implies `editor` and `viewer`).
* **TupleToUserset (`parent.viewer`)**: Inherits permissions from a linked parent resource (e.g. all viewers of a folder can view documents inside it).

---

## ⚡ 4. Dynamic ABAC Conditions (Caveats)

Caveats allow runtime environmental validation using Google CEL expressions.

### 4.1 Registering a Caveat Definition

```bash
curl -X POST http://localhost:8080/admin/caveats \
  -H "Content-Type: application/json" \
  -d '{
    "name": "is_business_hours_and_corp_ip",
    "cel_expression": "request_context.hour >= 9 && request_context.hour <= 18 && request_context.ip.startsWith(\"10.0.\")"
  }'
```

### 4.2 Binding a Caveat to a Relation Tuple

```bash
curl -X POST http://localhost:8080/tuples \
  -H "Content-Type: application/json" \
  -d '{
    "tuples": [
      {
        "namespace": "databases",
        "object": "prod_customer_db",
        "relation": "operator",
        "subject_namespace": "user",
        "subject_id": "usr_bob",
        "caveat_name": "is_business_hours_and_corp_ip"
      }
    ]
  }'
```

---

## 📡 5. Complete REST & gRPC API Reference

### 5.1 Evaluate Permission (`POST /check` or gRPC `Check`)

```bash
curl -X POST http://localhost:8080/check \
  -H "Content-Type: application/json" \
  -d '{
    "namespace": "databases",
    "object": "prod_customer_db",
    "relation": "operator",
    "subject_namespace": "user",
    "subject_id": "usr_bob",
    "request_context": {
      "hour": 14,
      "ip": "10.0.4.100"
    },
    "explain": true
  }'
```

**Response (`200 OK`):**
```json
{
  "allowed": true,
  "reason": "caveat_passed",
  "trace": {
    "node": "databases:prod_customer_db#operator@user:usr_bob",
    "depth": 1,
    "caveat_evaluated": "is_business_hours_and_corp_ip",
    "result": true
  }
}
```

### 5.2 Expand Permission Tree (`POST /expand` or gRPC `Expand`)

Visualizes the complete tree of users and groups that have a given permission:

```bash
curl -X POST http://localhost:8080/expand \
  -H "Content-Type: application/json" \
  -d '{
    "namespace": "documents",
    "object": "roadmap_2026",
    "relation": "viewer"
  }'
```

### 5.3 Reverse Lookup: Find Accessible Resources (`POST /lookup/resources`)

Returns all resources of a given type that a user can access:

```bash
curl -X POST http://localhost:8080/lookup/resources \
  -H "Content-Type: application/json" \
  -d '{
    "namespace": "documents",
    "relation": "editor",
    "subject_namespace": "user",
    "subject_id": "usr_alice"
  }'
```

**Response:**
```json
{
  "resources": [
    "roadmap_2026",
    "architecture_spec_v2",
    "q3_financial_model"
  ]
}
```

### 5.4 Reverse Lookup: Find Authorized Subjects (`POST /lookup/subjects`)

Returns all subjects who hold a relationship with an object:

```bash
curl -X POST http://localhost:8080/lookup/subjects \
  -H "Content-Type: application/json" \
  -d '{
    "namespace": "documents",
    "object": "roadmap_2026",
    "relation": "viewer"
  }'
```

---

## 🛠️ 6. Production Recipes

### Multi-Tenant Document Hierarchy Recipe

```text
Organization (org:acme)
  └── Team (group:backend) -> member: user:alice
        └── Folder (folder:eng_specs) -> parent: org:acme
              └── Document (doc:iam_architecture) -> parent: folder:eng_specs
```

**Tuples required:**
```json
[
  {"namespace": "organizations", "object": "acme", "relation": "member", "subject_namespace": "group", "subject_id": "backend", "subject_relation": "member"},
  {"namespace": "groups", "object": "backend", "relation": "member", "subject_namespace": "user", "subject_id": "alice"},
  {"namespace": "folders", "object": "eng_specs", "relation": "organization", "subject_namespace": "organizations", "subject_id": "acme"},
  {"namespace": "documents", "object": "iam_architecture", "relation": "parent_folder", "subject_namespace": "folders", "subject_id": "eng_specs"}
]
```
