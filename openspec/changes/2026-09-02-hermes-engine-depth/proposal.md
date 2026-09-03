# Change Proposal: Hermes Enterprise SAML 2.0 & SCIM 2.0 Engine Depth (P6-S7)

## Context
Phase 6 requires deep engine management across all 7 engines. For Hermes (Enterprise Federation & Directory Synchronization), the Go engine already implements:
- Full SAML provider lifecycle: `/admin/providers/{id}`, enable, disable, delete.
- Certificate parsing and expiration tracking: `CertificateInfo` with subject, issuer, serial, expiry date, and warning calculation.
- Attribute mapping between SAML assertions and Ego identity traits.
- SCIM 2.0 Groups resource (`/scim/v2/Groups`) alongside Users.
- SCIM sync history and run recording (`/admin/scim/sync-history`).

The Console needs:
1. Expanded Zod schemas in `console/src/lib/api/schemas/hermes.ts` for `CertificateInfo`, enhanced `SAMLProvider` with certificate diagnostics, `SCIMGroup`, and `SCIMSyncHistory`.
2. Next.js BFF routes in `console/src/app/api/enterprise/`:
   - `GET/PATCH/DELETE /api/enterprise/saml/[id]`: provider detail, update, deletion.
   - `POST /api/enterprise/saml/[id]/enable` and `disable`: hot state toggling.
   - `GET/POST /api/enterprise/scim/groups` and `[id]`: SCIM group management.
   - `GET/POST /api/enterprise/scim/sync`: SCIM directory sync history and manual trigger.
3. Enterprise UI upgrades in `console/src/app/enterprise/`:
   - SAML Providers Table with certificate expiry badges and enable/disable switches.
   - SAML Connection Wizard & Attribute Mapper dialog for configuring assertion-to-trait mapping.
   - SCIM Directory View with tabs for Users, Groups, and Sync History monitoring with error diagnostics.
4. Comprehensive verification: Vitest unit tests, Next.js production build, and roadmap update.
