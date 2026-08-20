# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Autorix is a Next-Generation Zero-Trust IAM (Identity & Access Management) suite, architected as independent Go microservices inspired by the Ory stack (Keto, Kratos, Hydra, Oathkeeper, Talos, Polis) and Google Zanzibar, plus an enterprise Next.js 15 admin console.

| Service | Port | Protocol | Purpose |
| :--- | :--- | :--- | :--- |
| **argus** | 4400 / 50053 | REST / gRPC | Fleet Control Plane, enrollment tokens, operators, SHA-256 chained audit trail, compliance |
| **console** | 3000 | Next.js 15 | Zero-Trust Admin Console, standalone `/login` & `/setup`, Studio interfaces, live Merkle audit verification |
| **nexus** | 50051 / 8080 | gRPC / REST | Google Zanzibar ReBAC authorization engine with high-throughput Check API |
| **ego** | 4433 | REST | Identity lifecycle, JSON schema traits, Argon2id passwords, TOTP MFA, sessions |
| **janus** | 4444 | REST / OIDC | OAuth 2.0 / OIDC server with PKCE, asymmetric RS256 JWKS |
| **aegis** | 4455 / 4456 | HTTP proxy / REST | Zero-Trust Access Proxy (PEP), rule routing, authenticators, authorizers, header mutators |
| **vulcan** | 4466 | REST | API keys (`av_live_...` prefix) + locally-attenuable HMAC-SHA256 Macaroons |
| **hermes** | 4477 | REST / XML | SAML 2.0 → OIDC bridge, SCIM 2.0 user/group directory synchronization |
| **themis** | 50052 / 4488 | gRPC / REST | Dedicated ABAC/CEL policy evaluation engine |

`console/` is the Next.js 15 (App Router) admin UI for all engines. `sdk/` holds official Go, TypeScript, and Python client SDKs. `aegis` sits in front of all other engines as the perimeter proxy.

## Architecture Pattern (per Go service)

Every Go microservice (`argus`, `aegis`, `ego`, `hermes`, `janus`, `nexus`, `themis`, `vulcan`) follows the hexagonal layout and Go module `github.com/autorix/<service>`:

```
<service>/
  cmd/<service>d/main.go          # entrypoint, wiring
  internal/core/                  # domain logic, framework-free
  internal/<domain>/              # domain-specific packages (e.g. ego/internal/credential)
  internal/storage/postgres/      # Postgres persistence adapter
  internal/transport/http/        # REST handlers
  internal/transport/grpc/        # gRPC handlers (argus, nexus, themis)
  migrations/                     # SQL migrations, applied via scripts/init-databases.sh
  api/autorix/<service>/v1/*.proto  # protobuf/gRPC contracts
```

Each service owns an isolated Postgres database (`autorix_<service>`), provisioned by `scripts/init-databases.sh`. There is no shared database or ORM across services — cross-service calls go through each service's public API/gRPC contract, never direct DB access.

## Common Commands

### Run the whole stack
```bash
docker compose up --build
```
Spins up Postgres (with 8 isolated databases), Prometheus, all 8 Go microservices, and the Console.

### Go services — test / build / run
Run from inside a service directory (e.g. `cd argus`):
```bash
go test -v ./...                  # all tests
go test -v ./internal/core/...    # single package
go test -run TestName ./...       # single test
go build -o bin/<service>d ./cmd/<service>d/main.go
go run ./cmd/<service>d/main.go
```

Run across all engines from repo root:
```bash
(cd argus && go test -v ./...)
(cd nexus && go test -v ./...)
(cd ego && go test -v ./...)
(cd janus && go test -v ./...)
(cd aegis && go test -v ./...)
(cd vulcan && go test -v ./...)
(cd hermes && go test -v ./...)
(cd themis && go test -v ./...)
```

### Console (Next.js 15)
```bash
cd console
npm test          # Run Vitest test suite
npm run build     # Validate production build & linting
npm run start     # Run production server
npx playwright test # Run E2E test suites
```

## Documentation

Per-service usage guides and runbooks live in `docs/`:
- `docs/api_reference_and_integration_guide.md` — master API integration manual
- `docs/argus_usage_guide.md` — Argus Control Plane & cryptographic audit trail
- `docs/console_usage_guide.md` — Console UI & Studio user manual
- `docs/themis_usage_guide.md` — Themis ABAC CEL engine
- `docs/nexus_usage_guide.md`, `ego_usage_guide.md`, `janus_usage_guide.md`, `aegis_usage_guide.md`, `vulcan_usage_guide.md`, `hermes_usage_guide.md`
- `docs/operations_and_runbook.md` — Day-1 & Day-2 Operations Runbook
- `docs/production_k8s_guide.md` — Kubernetes production deployment guide
- `docs/sdks_integration_guide.md` — Official SDKs for Go, TypeScript, Python

## Commit Guidelines
- Conventional Commits only (`feat(...)`, `fix(...)`, `docs(...)`, `refactor(...)`, `test(...)`).
- Never include `Co-Authored-By` or AI attribution trailers.
