# Change Proposal: Phase 6 Cross-Engine Views & Request Simulator (P6-S8)

## Context
With all seven individual engines (Ego, Janus, Nexus, Themis, Aegis, Vulcan, Hermes) deeply managed, the final milestone of Phase 6 is **Cross-Engine Views (P6-S8)**. The unique superpower of Autorix as an integrated Zero-Trust control plane is synthesizing answers that no single engine can provide in isolation.

We will provide an integrated **Cross-Engine Explorer** screen (`/explorer`) and Next.js BFF endpoints that implement:
1. **Unified Subject View (P6-S8-T1)**:
   Resolves a subject across the suite in one cohesive view:
   - Ego Identity (profile, traits, state)
   - Active Sessions & Devices
   - Nexus Relation Tuples (roles, permissions, memberships)
   - Janus OAuth2 Grants & Tokens
   - Vulcan API Keys & Delegated Macaroons
   - Hermes SAML / SCIM Linkage
2. **Effective Access Explorer (P6-S8-T2)**:
   Determines whether a subject can reach a protected resource by evaluating the multi-engine chain:
   - Aegis: Route rule match & handler pipeline
   - Nexus: Relation graph check (`user:id` has `relation` on `namespace:object`)
   - Themis: CEL policy evaluation
   - Synthesizes an engine contribution breakdown with full decision attribution.
3. **End-to-End Request Simulator (P6-S8-T3)**:
   Simulates an incoming HTTP request arriving at Aegis and step-by-step traces:
   - Ingress Routing (Aegis)
   - Auth Resolution (Ego session, Janus bearer token, or Vulcan macaroon)
   - Authorization Check (Nexus ReBAC)
   - Policy Guardrails (Themis ABAC)
   - Upstream Decision & Header Mutation
4. **Configuration Consistency Checks (P6-S8-T4)**:
   Automated diagnostic sweep identifying cross-engine discrepancies:
   - Proxy rules referencing namespaces missing in Nexus
   - OAuth2 clients missing upstream routing
   - Orphaned relation tuples with non-existent Ego subjects
   - Expiring SAML IdP certificates
   - Stale API keys without active usage

## BFF API Architecture
- `GET /api/explorer/subject/[id]`: Multi-engine aggregation route pulling concurrently from Ego, Nexus, Janus, Vulcan, and Hermes.
- `POST /api/explorer/effective-access`: Multi-engine evaluation checking route, relation, and CEL policy.
- `POST /api/explorer/simulate`: Step-by-step pipeline execution tracer.
- `GET /api/explorer/consistency`: Rule and configuration anomaly detection engine.
