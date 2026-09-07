# Python SDK

Package: `autorix` · Python 3.9+

The Python SDK covers the public runtime APIs for Ego, Janus, Nexus, Themis, and Vulcan. It exposes typed results, `AutorixHTTPError`, cursor pages, and framework adapters for FastAPI, Flask, and Django.

## Install

```bash
pip install autorix
# framework extras
pip install 'autorix[fastapi]' 'autorix[flask]' 'autorix[django]'
```

## Quick start

```python
from autorix import AutorixClient

with AutorixClient(nexus_url="http://localhost:8080") as client:
    decision = client.check("document", "42", "viewer", "user-7")
    if not decision.allowed:
        raise PermissionError(decision.reason)
```

## Public runtime coverage

- **Nexus:** checks, batch checks, tuple read/write/delete, expand, and subject/resource lookup.
- **Themis:** evaluation, policy CRUD, validation, dry runs, versions, fixtures, and test suites.
- **Vulcan:** key create/list/revoke, macaroon verification and attenuation. Verification expects the server macaroon shape; unsupported string input is denied locally.
- **Ego:** registration, login, `whoami`, and logout.
- **Janus:** OIDC discovery/JWKS, authorization URL construction, authorization-code exchange, refresh, client credentials, introspection, and revocation.

`Page` contains `data`, `next_cursor`, and `has_more`. Public administrative endpoints and Argus, Aegis, and Hermes are not part of this SDK.

## OAuth safety

`authorization_url` requires caller-provided state and PKCE challenge. The SDK never creates or persists token material. `exchange_code` and `refresh_token` are not retried because their credentials can be single-use. Pass a client secret only in a controlled backend, never to browser code.

## Framework adapters

`AutorixSecurity` provides FastAPI dependencies. `autorix.flask.require_permission` and `autorix.django.require_permission` provide route decorators. They consume trusted identity headers; deploy them only behind a proxy that strips client-supplied `X-User-*` headers. The backend must remain the authorization enforcement point.

## Reliability

Requests accept the configured timeout and return typed HTTP failures. Only idempotent reads are retried. Permission checks fail closed on transport or service failures; they return `allowed=False` rather than granting access.
