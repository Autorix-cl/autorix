# Autorix SDKs

Autorix ships source SDKs for Go, TypeScript/React, and Python. They are typed HTTP clients for the supported public-runtime operations below; they are not generated clients for every Autorix service.

## Capability matrix

| Capability | Go | TypeScript / React | Python |
| --- | --- | --- | --- |
| Ego session | Current session | Current session and logout | Registration, login, current session, logout |
| Nexus | Check, batch check, expand, resource lookup | Checks, tuple read/write/delete, expand, subject/resource lookup | Same as TypeScript |
| Themis | Evaluate | Evaluate, policy CRUD, versions, validate, dry run | Same, plus fixtures and test suite |
| Vulcan | Verify and attenuate | Create, list, verify, attenuate, revoke | Same |
| Janus | Discovery, authorization URL, token exchange, introspection, revoke, JWKS | Discovery, authorization URL, code exchange, refresh, introspection, revoke, JWKS | Same, plus client credentials |
| Argus | Audit-chain verification | No | No |
| Framework helpers | `net/http` | React hooks | FastAPI, Django, Flask |

No SDK currently implements Ego WebAuthn. Argus administrative APIs are not exposed through TypeScript or Python.

## Packages and prerequisites

| SDK | Source package | Runtime requirement |
| --- | --- | --- |
| Go | `github.com/autorix-cl/autorix/sdk/go` | Go 1.25.6 (module declaration) |
| TypeScript | `@autorix/sdk-js` | Node.js 18+ or a runtime with `fetch`; React is optional unless React exports are used |
| Python | `autorix` | Python 3.9+ |

Confirm that a release is published in your package registry before depending on an external install command; source development uses the corresponding `sdk/<language>` directory.

## Security boundary: browser and backend

The TypeScript SDK can build an OAuth authorization URL with PKCE and exchange an authorization code. It does not persist tokens or generate/store the PKCE verifier; the application owns both. Never pass `clientSecret`, an API key, an operator token, or another privileged credential to browser code. Use a backend for confidential OAuth operations, key management, tuple and policy writes.

React permission hooks are presentation helpers, not an authorization boundary. The backend must re-check and enforce the requested action. Proxy identity headers must be stripped from external requests and injected only after authentication.

## Common behavior

- Go and TypeScript retry only safe/idempotent reads by default. OAuth code and refresh-token exchanges are deliberately not retried.
- Python supports explicit retries for its read operations and Nexus checks. Its write operations are not retried.
- Nexus decision caching is in-process and defaults to 10 seconds. It is not distributed or invalidated by relationship changes.
- TypeScript has `AutorixApiError` and `AutorixTimeoutError`; Python has `AutorixError` and `AutorixHTTPError`. Go returns standard errors.
- The SDKs do not currently provide OpenTelemetry instrumentation or generated API contracts.

## Choose an SDK

- [Go SDK](/sdk/go)
- [TypeScript and React SDK](/sdk/typescript)
- [Python SDK](/sdk/python)
- [CLI and direct HTTP/gRPC](/sdk/cli)
