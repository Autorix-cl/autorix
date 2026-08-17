# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Autorix is a Zero-Trust IAM (Identity & Access Management) suite, architected as independent Go microservices inspired by the ORY stack (Keto, Kratos, Hydra, Oathkeeper, Talos, Polis), plus a Next.js admin console.

| Service | Port | Protocol | ORY equivalent | Purpose |
| :--- | :--- | :--- | :--- | :--- |
| **nexus** | 50051 | gRPC | Keto/Zanzibar | Hybrid ReBAC (Zanzibar) + ABAC (Google CEL) authorization engine |
| **ego** | 4433 | REST | Kratos | Identity lifecycle, sessions, Argon2id credential hashing |
| **janus** | 4444 | REST/OIDC | Hydra | OAuth 2.0 / OIDC server with PKCE, asymmetric JWKS |
| **aegis** | 4455 | HTTP proxy | Oathkeeper | Zero-Trust Access Proxy / perimeter PEP |
| **vulcan** | 4466 | REST | Talos | API keys (`av_live_...` prefix) + locally-attenuable Macaroons |
| **hermes** | 4477 | REST/XML | Polis | SAML 2.0 → OIDC bridge, SCIM 2.0 sync server |
| **themis** | — | gRPC+HTTP | — | ABAC/CEL policy evaluation engine (newer addition, not yet in README diagram) |

`console/` is the Next.js 15 (App Router) admin UI for all engines. `sdk/` holds official Go, TypeScript, and Python client SDKs. `aegis` sits in front of all other engines as the perimeter proxy.

## Architecture Pattern (per Go service)

Every Go microservice (`aegis`, `ego`, `hermes`, `janus`, `nexus`, `themis`, `vulcan`) follows the same hexagonal layout and Go module `github.com/autorix/<service>`:

```
<service>/
  cmd/<service>d/main.go          # entrypoint, wiring
  internal/core/                  # domain logic, framework-free
  internal/<domain>/              # domain-specific packages (e.g. ego/internal/credential, ego/internal/session)
  internal/storage/postgres/repository.go   # Postgres persistence adapter
  internal/transport/http/server.go         # REST handlers (and /transport/grpc/server.go for nexus & themis)
  migrations/                     # SQL migrations, applied via scripts/init-databases.sh on container init
  api/autorix/<service>/v1/*.proto  # protobuf/gRPC contracts (nexus, themis) — generated via buf
```

Each service owns an isolated Postgres database (`autorix_<service>`), provisioned by `scripts/init-databases.sh`. There is no shared database or ORM across services — cross-service calls go through each service's public API/gRPC contract, never direct DB access.

themis additionally uses Google CEL (`internal/engine/cel.go`) to evaluate ABAC policy expressions against dynamic JSON payloads (`internal/core/engine.go`, `internal/core/service.go`).

## Common Commands

### Run the whole stack
```bash
docker compose up --build
```
Spins up Postgres (with 6 isolated databases via `scripts/init-databases.sh`), all seven Go engines, and the console.

### Go services — test / build / run
Run from inside a service directory (e.g. `cd ego`):
```bash
go test -v ./...              # all tests
go test -v ./internal/core/...    # single package
go test -run TestName ./...       # single test
go build -o bin/<service>d ./cmd/<service>d/main.go
go run ./cmd/<service>d/main.go
```
Run across all engines from repo root:
```bash
(cd nexus && go test -v ./...)
(cd ego && go test -v ./...)
(cd janus && go test -v ./...)
(cd aegis && go test -v ./...)
(cd vulcan && go test -v ./...)
(cd hermes && go test -v ./...)
(cd themis && go test -v ./...)
```

### themis-specific (has a Makefile; the pattern for services with gRPC/protobuf)
```bash
cd themis
make proto    # regenerate .pb.go via buf (requires protoc-gen-go, protoc-gen-go-grpc, buf — `make setup`)
make build
make run
make test
make bench    # benchmarks for internal/engine
```
`nexus` follows the same buf/protobuf pattern for its `.proto` contract under `nexus/api/autorix/nexus/v1/`.

### Console (Next.js)
```bash
cd console
npm run dev      # localhost:3000
npm run build
npm run start
npm run lint
```

## Documentation

Per-service usage guides and the technical roadmap live in `docs/`:
- `docs/api_reference_and_integration_guide.md` — master API integration manual
- `docs/<service>_usage_guide.md` — one guide per engine (nexus, ego, janus, aegis, vulcan, hermes)
- `docs/roadmap.md` — technical roadmap and overall architecture

## SDD (Spec-Driven Development)

This repo uses SDD workflows (`openspec/`) with **strict TDD mode enabled**: write a failing test before implementation code for any SDD-tracked change.
