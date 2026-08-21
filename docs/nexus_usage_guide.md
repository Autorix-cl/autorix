# Autorix Nexus: Zanzibar ReBAC & Authorization Engine Manual

**Autorix Nexus** is an ultra-low latency, fine-grained authorization engine combining **Google Zanzibar** relation graphs with **Attribute-Based Access Control (ABAC)** powered by Google Common Expression Language (CEL). It evaluates complex hierarchical permissions across millions of entities with sub-5ms latency.

---

## 🏛️ 1. Architectural Concepts & Data Model

Unlike legacy RBAC systems that assign flat roles to users (`alice -> role:admin`), Nexus represents permissions as a **directed graph of relationships**:

```text
namespace:object#relation@subject_namespace:subject_id[#subject_relation] [with caveat]
```

### 1.1 The Zanzibar Tuple Structure

| Field | Type | Required | Description | Example |
| :--- | :--- | :--- | :--- | :--- |
| `namespace` | `string` | **Yes** | The resource category or domain boundary. | `documents`, `organizations`, `projects` |
| `object` | `string` | **Yes** | Unique identifier of the resource instance. | `roadmap_2026_q3`, `org_enterprise_corp` |
| `relation` | `string` | **Yes** | The named edge linking subject to object. | `owner`, `editor`, `viewer`, `member`, `parent` |
| `subject_namespace` | `string` | No (default: `user`) | Namespace of the subject. | `user`, `group`, `service_account` |
| `subject_id` | `string` | **Yes** | Unique identifier of the subject. | `usr_4455`, `grp_engineering` |
| `subject_relation` | `string` | No | Subject set indirection (for group membership inheritance). | `member`, `admin` |
| `caveat_name` | `string` | No | Name of a registered CEL condition for contextual evaluation. | `require_work_hours`, `ip_in_subnet` |
| `caveat_context` | `object` | No | Static binding parameters for the caveat. | `{"allowed_ip": "10.0.0.0/8"}` |

### 1.2 Subject Sets & Transitive Inheritance

Nexus natively supports **Subject Sets** (users-in-usersets), allowing infinite hierarchical inheritance without duplicating permissions:

```text
  [ user:alice ] ──(member)──► [ group:engineering ] ──(editor)──► [ document:roadmap_2026 ]
```

* Tuple 1: `group:engineering#member@user:alice`
* Tuple 2: `document:roadmap_2026#editor@group:engineering#member`

When querying `Check(document:roadmap_2026, editor, user:alice)`, Nexus traverses the graph recursively and resolves `allowed: true`.

---

## ⚙️ 2. PostgreSQL Storage Engine & Optimization

Nexus stores relation tuples in PostgreSQL (`autorix_nexus` database) with specialized B-tree indexing:

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

* **Union**: Grants permission if the subject holds any of the specified relations (e.g., `owner` automatically implies `editor` and `viewer`).
* **TupleToUserset (`parent.viewer`)**: Inherits permissions from a linked parent resource (e.g., all viewers of a parent folder can view documents inside it).

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

Nexus exposes both a high-throughput REST API on port `8080` and a gRPC interface on port `50051`.

### 5.1 Evaluate Permission (`POST /check`)

Evaluates whether a subject has a relation on a resource instance, executing graph traversal and CEL caveat validation.

* **Method**: `POST`
* **Path**: `/check`
* **Headers**: `Content-Type: application/json`

#### Request Body
```json
{
  "namespace": "documents",
  "object": "roadmap_2026",
  "relation": "editor",
  "subject_namespace": "user",
  "subject_id": "usr_alice",
  "subject_relation": "",
  "request_context": {
    "hour": 14,
    "ip": "10.0.4.100"
  },
  "explain": true
}
```

| Field | Type | Required | Description |
| :--- | :--- | :--- | :--- |
| `namespace` | `string` | **Yes** | Target resource namespace. |
| `object` | `string` | **Yes** | Target resource instance ID. |
| `relation` | `string` | **Yes** | Relation or permission being checked. |
| `subject_namespace` | `string` | No (default `user`) | Namespace of the subject entity. |
| `subject_id` | `string` | **Yes** | ID of the subject entity. |
| `subject_relation` | `string` | No | Subject set relation indirection. |
| `request_context` | `object` | No | Environmental variables passed to CEL caveats. |
| `explain` | `boolean` | No | If `true`, returns the decision trace. |

#### Response (`200 OK`)
```json
{
  "allowed": true,
  "reason": "caveat_passed",
  "trace": {
    "node": "documents:roadmap_2026#editor@user:usr_alice",
    "depth": 1,
    "caveat_evaluated": "is_business_hours_and_corp_ip",
    "result": true
  }
}
```

---

### 5.2 List Relation Tuples (`GET /tuples`)

Lists relation tuples with cursor-based pagination and namespace filtering.

* **Method**: `GET`
* **Path**: `/tuples`
* **Query Parameters**:
  - `namespace` *(string, optional)*: Filter by resource namespace.
  - `limit` *(integer, optional, default: 50)*: Number of records.
  - `cursor` *(string, optional)*: Pagination cursor.

#### Response (`200 OK`)
```json
{
  "data": [
    {
      "namespace": "documents",
      "object": "roadmap_2026",
      "relation": "owner",
      "subject_namespace": "user",
      "subject_id": "usr_alice"
    }
  ],
  "next_cursor": "eyJ0IjoiMjAyNi0wOC0yMFQwODozMDowMFoiLCJucyI6ImRvY3VtZW50cyJ9",
  "has_more": false
}
```

---

### 5.3 Write Relation Tuples (`POST /tuples`)

Inserts one or more relation tuples into the graph.

* **Method**: `POST`
* **Path**: `/tuples`
* **Headers**: `Content-Type: application/json`

#### Request Body
```json
{
  "tuples": [
    {
      "namespace": "documents",
      "object": "roadmap_2026",
      "relation": "viewer",
      "subject_namespace": "group",
      "subject_id": "grp_engineering",
      "subject_relation": "member"
    }
  ]
}
```

#### Response (`201 Created`)
```json
[
  {
    "namespace": "documents",
    "object": "roadmap_2026",
    "relation": "viewer",
    "subject_namespace": "group",
    "subject_id": "grp_engineering",
    "subject_relation": "member"
  }
]
```

---

### 5.4 Delete Relation Tuples (`DELETE /tuples`)

Removes relation tuples from the graph.

* **Method**: `DELETE`
* **Path**: `/tuples`
* **Headers**: `Content-Type: application/json`

#### Request Body
```json
{
  "tuples": [
    {
      "namespace": "documents",
      "object": "roadmap_2026",
      "relation": "viewer",
      "subject_namespace": "group",
      "subject_id": "grp_engineering",
      "subject_relation": "member"
    }
  ]
}
```

#### Response (`200 OK`)
```json
{
  "status": "deleted"
}
```

---

### 5.5 Expand Permission Tree (`POST /expand` / `GET /expand`)

Traverses the graph and returns the full expansion tree of subjects holding a relationship.

* **Method**: `POST` or `GET`
* **Path**: `/expand`
* **Query Parameters / JSON Body**: `namespace`, `object`, `relation`.

#### Response (`200 OK`)
```json
{
  "tree": {
    "type": "union",
    "children": [
      {
        "type": "leaf",
        "tuple": {
          "namespace": "documents",
          "object": "roadmap_2026",
          "relation": "owner",
          "subject_namespace": "user",
          "subject_id": "usr_alice"
        }
      }
    ]
  }
}
```

---

### 5.6 Reverse Lookup: Find Authorized Subjects (`POST /lookup/subjects` / `GET /lookup/subjects`)

Returns all subjects who have a specific relation on a resource instance.

* **Method**: `POST` or `GET`
* **Path**: `/lookup/subjects`
* **Request Body / Query Params**:
```json
{
  "namespace": "documents",
  "object": "roadmap_2026",
  "relation": "viewer"
}
```

#### Response (`200 OK`)
```json
{
  "subjects": ["user:usr_alice", "user:usr_bob", "group:grp_engineering#member"]
}
```

---

### 5.7 Reverse Lookup: Find Accessible Resources (`POST /lookup/resources` / `GET /lookup/resources`)

Returns all resource IDs of a given namespace that a subject can access.

* **Method**: `POST` or `GET`
* **Path**: `/lookup/resources`
* **Request Body / Query Params**:
```json
{
  "namespace": "documents",
  "relation": "editor",
  "subject_namespace": "user",
  "subject_id": "usr_alice"
}
```

#### Response (`200 OK`)
```json
{
  "resources": [
    "roadmap_2026",
    "architecture_spec_v2",
    "q3_financial_model"
  ]
}
```

---

### 5.8 Admin: Namespace Management

- `GET /admin/namespaces`: List all namespace schemas.
- `POST /admin/namespaces`: Create or update a namespace schema.
- `GET /admin/namespaces/{name}`: Get a specific namespace schema.
- `DELETE /admin/namespaces/{name}`: Delete a namespace schema.

---

### 5.9 Admin: Caveat Management

- `GET /admin/caveats`: List registered CEL caveat definitions.
- `POST /admin/caveats`: Register or update a CEL caveat.
- `GET /admin/caveats/{name}`: Get a caveat definition.
- `DELETE /admin/caveats/{name}`: Delete a caveat definition.

---

### 5.10 gRPC Interface (`:50051`)

Nexus implements `nexus.v1.NexusService` defined in `nexus/proto/nexus.proto`:
- `Check(CheckRequest) returns (CheckResponse)`
- `Expand(ExpandRequest) returns (ExpandResponse)`
- `LookupSubjects(LookupSubjectsRequest) returns (LookupSubjectsResponse)`
- `LookupResources(LookupResourcesRequest) returns (LookupResourcesResponse)`

---

## 📜 6. Autorix Permission Language (APL)

Nexus includes a built-in Lexer/Parser for **Autorix Permission Language (APL)**, providing Ory Keto OPL parity. Instead of handcrafting complex JSON schemas for userset rewrites, you can define relationships and permissions using clean TypeScript-like class syntax:

```typescript
import { Namespace, Context } from "@autorix/nexus-types"

class Document implements Namespace {
  related: {
    owner: User[]
    editor: User[]
    viewer: User[]
    parent: Folder[]
  }

  permits = {
    view: (ctx: Context): boolean =>
      this.related.viewer.includes(ctx.subject) ||
      this.related.editor.includes(ctx.subject) ||
      this.related.owner.includes(ctx.subject) ||
      this.related.parent.traverse((p) => p.related.view),

    edit: (ctx: Context): boolean =>
      this.related.editor.includes(ctx.subject) ||
      this.related.owner.includes(ctx.subject),

    owner: (ctx: Context): boolean =>
      this.related.owner.includes(ctx.subject)
  }
}
```

The APL compiler transforms this directly into Zanzibar `computed_userset`, `tuple_to_userset`, and `union` rewrite rules stored in `namespace_schemas`.

---

## 🛠️ 7. Production Recipes

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
