# Change Proposal: Ego Identities, Sessions & Trait Management Depth (P6-S1)

## Context
Phase 6 specifies deep engine management for all core engines. For Ego (Identities & Sessions), the Go engine already implements administrative endpoints for identity lifecycle, cursor pagination, global and per-identity session revocation, administrative credential reset/recovery, MFA factor inspection and reset, and dynamic JSON schema trait management.

The Console needs:
1. Complete Zod schemas matching Ego's core models (`Session`, `Credential`, `MfaStatus`, `TraitSchema`, `RecoveryLink`).
2. Complete Next.js BFF proxy routes (`/api/identities/[id]`, `/sessions`, `/credentials`, `/mfa`, `/schemas`, `/bulk-import`).
3. Complete UI integration in `console/src/app/identities/` with live tabbed drawer actions (session revocation, administrative password reset, recovery link, MFA factor reset, schema-driven traits) and a bulk import/export modal with dry-run validation.
4. Full verification in Vitest, Next.js build, and roadmap update.

## Scope
- Expand `console/src/lib/api/schemas/identity.ts`
- Implement Next.js BFF proxy endpoints for Ego admin APIs
- Enhance `identity-sheet.tsx` with live actions and real API calls
- Add Bulk Import & Export dialog with dry-run validation to `console/src/app/identities/`
- Add tests in `identity.test.ts`, `identities-bff.test.ts`, and component tests
- Update `docs/roadmap-control-plane.html` marking P6-S1 as DONE
