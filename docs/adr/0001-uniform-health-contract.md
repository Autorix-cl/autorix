# ADR 0001: Uniform health, readiness and identity contract

- **Status**: Accepted
- **Scope**: All seven Go engines (aegis, ego, hermes, janus, nexus, themis, vulcan)
- **Related**: `openspec` roadmap Phase 1, spec P1-S1

## Context

No engine exposed a consistent way to answer "am I alive" and "am I ready".
Three engines (aegis, nexus, themis) had an ad-hoc `GET /health` that always
returned `"status": "healthy"` without checking any dependency. Four engines
(ego, hermes, janus, vulcan) had no health endpoint at all. The Helm chart for
ego already probed `GET /health/alive`, an endpoint that exists nowhere in the
code — in Kubernetes this probe fails, the kubelet restarts the pod, and ego
enters a permanent CrashLoopBackOff.

This is also the precondition for the Argus control plane (Phase 2): the
registry's heartbeat carries readiness, so the contract that produces
readiness has to exist first.

## Decision

Every engine implements the same three HTTP endpoints, provided by a shared
library (`github.com/autorix/platform/health`) rather than reimplemented per
engine:

- `GET /health/alive` — process liveness. Consults zero dependencies. Answers
  as long as the process can schedule the request; never fails because a
  downstream system is unavailable.
- `GET /health/ready` — readiness. Runs every named check registered for that
  engine (today: Postgres reachability via `platform/postgres.Check`, which
  bounds the ping to 2 seconds independently of the caller's context) and
  fails closed: any one failing check makes the response `503 "not_ready"`
  with the per-check cause. No registered checks means `200 "ready"` by
  construction.
- `GET /info` — engine identity: engine name, semver, build SHA, schema
  version, declared capabilities, instance id, uptime. Version and build SHA
  are stamped at build time via `-ldflags -X` into
  `github.com/autorix/platform/version` — without this, `/info` cannot report
  what is actually running, which the Argus registry needs for upgrade
  detection.

Nexus and Themis (the two gRPC engines) additionally register the standard
`grpc.health.v1.Health` service via `github.com/autorix/platform/grpchealth`.
It evaluates the *same* `health.Checker` live on every `Check()` call, so a
gRPC-native probe and the HTTP probe can never disagree — there is no cached
or periodically-refreshed status to drift out of sync.

Aegis serves the contract only on its admin port (4456), never on the proxy
port (4455), so a wildcard proxy rule can never shadow a health check.

Ad-hoc `GET /health` on aegis, nexus and themis is kept as a deprecated alias
returning the same shape as `/health/alive` (i.e. it stops silently claiming
health it never measured), to avoid a breaking change for existing callers.

## What "ready" means per engine

Every engine registers exactly one readiness check today: Postgres
reachability via its `*pgxpool.Pool`. Aegis has no database and registers no
checks — it is ready by construction as long as the process is scheduling
requests. This can grow (e.g. a schema-version check, a warm-cache check)
without changing the endpoint shape.

## Consequences

- Fixes the live ego Helm CrashLoopBackOff by pointing the probe at an
  endpoint that exists (P1-S1-T7).
- Every `postgres.Repository` across the six database-backed engines needs to
  expose its `*pgxpool.Pool` (or a `Ping` method) so `main.go` can wire
  `postgres.Check` into the engine's `health.Checker` — previously the pool
  was a private field with no accessor.
- Each engine's Dockerfile build context moves from `./<service>` to the repo
  root, so `COPY platform ./platform` can put the shared module in reach of a
  local `replace` directive in each engine's `go.mod`
  (`replace github.com/autorix/platform => ../platform`). No `go.work` was
  introduced — replace directives keep each module's build self-contained,
  which matters because Docker still builds one engine's directory in
  isolation from the others.
- A CI job (P1-S1-T8) renders the Helm chart, extracts every probe path, and
  asserts it against the routes actually registered in Go, so this class of
  drift cannot ship silently again.

## Alternatives considered

- **Per-engine health check bodies, no shared library.** Rejected: the exact
  bug this ADR fixes (aegis/nexus/themis's three different ad-hoc `/health`
  shapes) is duplication drifting apart. A shared library made the contract
  identical by construction instead of by convention.
- **A `go.work` workspace instead of per-module `replace` directives.**
  Rejected for now: `go.work` is a local development convenience Docker
  ignores by default (it isn't copied into an isolated build context), so it
  would not remove the need for a `replace` directive in CI/Docker builds
  anyway. It can be added later purely as an editor/tooling convenience
  without affecting this decision.
