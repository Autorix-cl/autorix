# Evaluate Fine-Grained Permissions with Autorix Nexus

**Autorix Nexus** is an ultra-low latency, fine-grained authorization engine combining **Google Zanzibar** relation graphs with **Attribute-Based Access Control (ABAC)** powered by Google Common Expression Language (CEL). It evaluates complex hierarchical permissions across millions of entities with sub-5ms latency.

## Quick path

Evaluate whether a user has permission to access a specific resource, factoring in ABAC context:

```bash
curl -X POST http://localhost:8080/check \
  -H "Content-Type: application/json" \
  -d '{
    "namespace": "documents",
    "object": "roadmap_2026",
    "relation": "editor",
    "subject_namespace": "user",
    "subject_id": "usr_alice",
    "request_context": {
      "hour": 14,
      "ip": "10.0.4.100"
    },
    "explain": true
  }'
```

## Details

### The Zanzibar Tuple Structure & Graph

Nexus represents permissions as a directed graph of relationships stored in PostgreSQL with B-tree indexing:
`namespace:object#relation@subject_namespace:subject_id[#subject_relation] [with caveat]`

Subject Sets allow infinite hierarchical inheritance (e.g., users within groups) without duplicating permissions.

### Namespace Schemas & Rewrite Rules

Namespace schemas define inheritance rules:
- **Union:** Grants permission if a subject holds *any* specified relation (e.g., an `owner` is also an `editor`).
- **TupleToUserset:** Inherits permissions from a linked parent resource (e.g., inheriting `viewer` from a parent folder).

Instead of JSON schemas, you can use **Autorix Permission Language (APL)**, a built-in TypeScript-like syntax that compiles directly into Zanzibar rewrite rules.

### Dynamic ABAC Conditions (Caveats)

Caveats validate runtime environments using Google CEL expressions (e.g., `request_context.hour >= 9`). You can register caveats (`POST /admin/caveats`) and bind them when inserting relation tuples.

### Core API Reference (REST `:8080` & gRPC `:50051`)

- **Check Permission:** `POST /check` (traverses graph and evaluates CEL caveats)
- **Manage Tuples:** `POST /tuples`, `GET /tuples`, `DELETE /tuples`
- **Expand Permission Tree:** `POST /expand`
- **Reverse Lookup (Subjects):** `POST /lookup/subjects` (Who can access this?)
- **Reverse Lookup (Resources):** `POST /lookup/resources` (What can they access?)
- **Admin Namespaces & Caveats:** `GET`, `POST`, `DELETE` on `/admin/namespaces` and `/admin/caveats`

## Checklist

- [ ] Connect Nexus to a PostgreSQL database (`autorix_nexus`).
- [ ] Define namespace schemas using JSON or Autorix Permission Language (APL).
- [ ] Register required ABAC caveats for context-aware policies.
- [ ] Write foundational relationship tuples (e.g., group memberships, parent hierarchies).
- [ ] Configure depth-limited recursive traversal parameters to prevent infinite loops.

## Next step

Deploy the Autorix Suite to your cluster following the [Production Kubernetes Deployment Guide](./production_k8s_guide.md).
