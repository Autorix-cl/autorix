# Proposal: Nexus ReBAC Zanzibar Engine Depth (P6-S3)

## Motivation
Autorix Nexus provides a high-performance Zanzibar-style relationship-based access control (ReBAC) engine with APL schema definition, gRPC, and REST admin APIs.
While the Go engine core is complete and tested, the console BFF currently uses in-memory mock endpoints, preventing operators from interacting with the real PostgreSQL backend, visualizing decision traces, managing CEL caveats, or exploring relationship graphs.

## Scope
1. **Zod Schemas**:
   - Add schemas for decision explanation traces (`decisionTraceSchema`), tree expansion (`expandTreeSchema`), subjects/resources lookup, namespaces, and caveats.
2. **Next.js BFF Routes**:
   - Replace in-memory mock stubs with real proxy calls to `nexus`:
     - `POST /api/nexus/check` (with `explain: true` for decision traces)
     - `GET /api/nexus/schema`, `PUT /api/nexus/schema` (bridging `/admin/namespaces`)
     - `GET /api/nexus/tuples`, `POST /api/nexus/tuples`, `DELETE /api/nexus/tuples`
     - `POST /api/nexus/expand` (relationship tree expansion)
     - `POST /api/nexus/lookup` (reverse lookup)
     - `GET/POST/DELETE /api/nexus/caveats` (CEL caveat registry)
3. **Console UI Enhancements**:
   - **Simulator with Walkable Trace**: Visual tree showing traversal path, matched tuples, and evaluated caveats.
   - **Tuple Browser & Bulk Import**: Filterable grid with CSV/JSON batch import.
   - **Relationship Graph**: Visual node explorer around objects and subjects.
   - **Caveat Manager**: CEL expression editor and validation.
4. **Verification**:
   - Vitest unit tests for schemas, BFF routes, and UI components.
   - Next.js production build validation.
