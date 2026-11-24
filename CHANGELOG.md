# Changelog

All notable changes to Autorix are documented in this file.

The project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).
Release notes use the [Keep a Changelog](https://keepachangelog.com/en/1.1.0/)
structure.

## [Unreleased]

## [v0.2.2] - 2026-11-24

### Fixed

- Added `@types/node` to `sdk/typescript` `devDependencies` so clean TypeScript
  SDK builds resolve Node `crypto`, `Buffer`, and `process` declarations.
- Fixed Compose v2 service name filtering (`{{.Service}} != "console"`) in the
  `e2e-console` health-wait loop in `.github/workflows/ci.yml`.
- Installed `httpx` in `.github/workflows/release-supply-chain.yml` so Python
  SDK unit tests run cleanly during release validation.

## [v0.2.1] - 2026-11-24

### Fixed

- Ignored Next.js generated `next-env.d.ts` in `console/eslint.config.mjs` to
  prevent `@typescript-eslint/triple-slash-reference` lint failures after
  upgrading to Next.js `15.5.29`.
- Upgraded Node.js from `20` to `22` in
  `.github/workflows/release-supply-chain.yml` to match CI and support Vitest 4
  during release validation.

## [v0.2.0] - 2026-11-24

### Security

- Upgraded Autorix Console Next.js and `eslint-config-next` from `15.1.7` to
  `15.5.29` and ran `npm audit fix` to remediate Next.js middleware/RSC CVEs
  (`CVE-2025-29927`, `CVE-2025-66478`) and transitive high-severity advisories.
- Added cluster-wide internal service authentication (`platform/httpx.InternalAuth`
  and `platform/tls.UnaryInternalAuthInterceptor`) supporting verified mTLS
  SPIFFE client certificates and constant-time shared token verification
  (`X-Autorix-Internal-Token` / `X-Autorix-Console-Token`), wired across all 8
  Go engines (`argus`, `aegis`, `ego`, `janus`, `nexus`, `vulcan`, `hermes`,
  `themis`), Console BFF, `docker-compose.yml`, and
  `scripts/generate-local-secrets.sh`.
- Added canonical `v1` HMAC-SHA256 identity header signing
  (`X-Autorix-Identity-Signature`) in Aegis's header mutator and constant-time
  verification across the Go, Python (FastAPI/Flask), and TypeScript SDKs.

### Added

- Implemented real WebAuthn / Passkey registration and login ceremonies in Ego
  (`/self-service/webauthn/{registration,login}/{start,finish}`) using
  `github.com/go-webauthn/webauthn`, with single-use TTL challenge consumption,
  sign-count persistence, clone-warning rejection, and end-to-end cryptographic
  ECDSA P-256 tests.
- Replaced the placeholder Terraform provider stub in
  `terraform-provider-autorix/` with a real HTTP CRUD client and CLI supporting
  `autorix_environment`, `autorix_policy`, and `autorix_oauth2_client`.

### Fixed

- Aligned CI Go toolchain (`1.26.6`) and `golangci-lint-action@v9` (`v2.12.2`)
  with `.golangci.yml` v2, fixing `covdata` and linter version mismatches across
  all Go modules.
- Excluded third-party `alertmanager` probes from Go-route drift checks in
  `scripts/ci/check_helm_probes.py`.
- Fixed HTTP status code parsing in `scripts/ci/smoke_test.sh` and added
  internal auth header propagation for Compose smoke tests.
- Formatted Autorix Console with Prettier so `format:check` passes cleanly.

## [v0.1.0] - 2026-11-24

### Security

- Hardened backend engines (`argus`, `aegis`, `ego`, `janus`, `nexus`, `vulcan`,
  `hermes`, `themis`), Console BFF routes, CLI, and SDKs across authentication,
  authorization, SSRF/path-traversal protection, fail-closed defaults, and
  cryptographic hygiene.

### Added

- OSS governance baseline: Apache-2.0 license, contribution guide, code of
  conduct, support policy, issue forms, and public community documentation.
- Console UX, accessibility, RBAC gating, and real-backend wiring across
  Explorer, Diagnostics, Fleet, Audit, Policies, and Vulcan screens.

[Unreleased]: https://github.com/Autorix-cl/autorix/compare/v0.2.2...HEAD
[v0.2.2]: https://github.com/Autorix-cl/autorix/compare/v0.2.1...v0.2.2
[v0.2.1]: https://github.com/Autorix-cl/autorix/compare/v0.2.0...v0.2.1
[v0.2.0]: https://github.com/Autorix-cl/autorix/compare/v0.1.0...v0.2.0
[v0.1.0]: https://github.com/Autorix-cl/autorix/releases/tag/v0.1.0
