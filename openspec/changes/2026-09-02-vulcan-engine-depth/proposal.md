# Change Proposal: Vulcan API Keys, Attenuation & Macaroons Depth (P6-S6)

## Context
Phase 6 specifies deep engine management for all engines. For Vulcan (API Keys, Caveats & Macaroon-based decentralized capabilities), the Go engine already implements:
- Full key lifecycle: detail, name & scopes update, rotation with grace period overlap (`/admin/keys/{id}`, `rotate`).
- Usage tracking: call counts, last used timestamp, last source IP.
- Scope catalogue: registered scope management (`/admin/scopes`).
- Macaroon verification: contextual evaluation against request environment (`/keys/verify`).

The Console needs:
1. Expanded Zod schemas in `console/src/lib/api/schemas/vulcan.ts` matching Vulcan's updated models (`APIKey`, `RotateKeyResponse`, `Scope`, `VerifyResponse`).
2. Next.js BFF routes in `console/src/app/api/vulcan/`:
   - Key lifecycle & rotation: `GET/PATCH /api/vulcan/keys/[id]`, `POST /api/vulcan/keys/[id]/rotate`.
   - Scope catalogue: `GET/POST /api/vulcan/scopes`, `DELETE /api/vulcan/scopes/[name]`.
   - Verification endpoint: `POST /api/vulcan/verify`.
3. Enhanced UI in `console/src/app/vulcan/`:
   - Display usage telemetry (call counts, stale key indicators, last IP) and Key Rotation action with grace period in `KeysTable`.
   - Scope Catalogue manager sheet/modal to manage platform scopes.
   - Enhanced `AttenuationStudio` visual caveat chain showing capabilities before and after attenuation.
   - Macaroon / Key Inspector allowing operators to paste a token or macaroon and inspect caveats and live validity.
4. Comprehensive verification: Vitest unit tests, Next.js production build, and roadmap update.
