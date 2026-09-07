# Autorix: Next-Generation Zero-Trust IAM

Autorix is a high-performance, modular Identity and Access Management (IAM) suite. It provides authentication, authorization (ReBAC/ABAC), OAuth2/OIDC, and Zero-Trust proxy enforcement, all backed by cryptographic audit logging.

## Quick start (local development only)

> **Docker Compose is for local development and evaluation only.** It uses
> development credentials, local ports, HTTP service traffic, and PostgreSQL
> with `sslmode=disable`. Do not expose this stack to the Internet or use it as
> a production deployment.

Requirements: Docker Engine with Compose v2, 8 GB RAM, and ports `3000`,
`4444`, `4455`, and `5432` available.

```bash
./scripts/generate-local-secrets.sh
docker compose --profile core up -d --build
docker compose --profile core ps
```

1. Run `./scripts/generate-local-secrets.sh` before every fresh local setup. It creates untracked development secrets; do not commit its output.
2. Open **[http://localhost:3000](http://localhost:3000)**.
3. Retrieve Argus's one-time bootstrap token locally:
   ```bash
   docker logs autorix-argus | grep -i "bootstrap token"
   ```
4. Complete the Console setup flow. Never commit or share bootstrap tokens or
   credentials.

Verify the public development endpoints:

```bash
curl -fsS http://localhost:4444/health/ready
curl -fsS http://localhost:4455/health/ready
curl -fsS http://localhost:4400/health/ready
```

Janus administration (`:4445`) and Aegis administration (`:4456`) are private;
Compose does not publish them. See the [onboarding guide](docs/getting-started.md)
before changing security settings.

## Architecture at a glance

```text
                                         [ EXTERNAL TRAFFIC ]
                                                   │
                                                   ▼
                                        ┌─────────────────────┐
                                        │    Autorix Aegis    │  (Zero Trust PEP Reverse Proxy :4455)
                                        └──────────┬──────────┘
                                                   │
        ┌──────────────────────┬───────────────────┼───────────────────┬──────────────────────┐
        ▼                      ▼                   ▼                   ▼                      ▼
 ┌──────────────┐       ┌──────────────┐    ┌──────────────┐    ┌──────────────┐       ┌──────────────┐
 │Autorix Janus │       │Autorix Nexus │    │ Autorix Ego  │    │Autorix Vulcan│       │Autorix Hermes│
 │(OAuth2/OIDC) │       │(ReBAC/Zanzibar)   │(Identity/MFA)│    │ (API Keys)   │       │ (SAML/SCIM)  │
 │    :4444     │       │:50051 / :8080│    │    :4433     │    │    :4466     │       │    :4477     │
 └──────────────┘       └──────────────┘    └──────────────┘    └──────────────┘       └──────────────┘
        │                      │                   │                   │                      │
        └──────────────────────┼───────────────────┼───────────────────┼──────────────────────┘
                               │                   │                   │
                               ▼                   ▼                   ▼
                        ┌──────────────┐    ┌──────────────┐    ┌──────────────┐
                        │Autorix Themis│    │Autorix Argus │    │Autorix Console
                        │ (ABAC / CEL) │    │(Control Plane│    │(Next.js 15 UI│
                        │:50052 / :4488│    │:50053 / :4400│    │    :3000     │
                        └──────────────┘    └──────────────┘    └──────────────┘
```

## Core Engines

| Component | Role | Protocols |
|-----------|------|-----------|
| **Argus** | Fleet Control Plane, Governance & Audit | REST / gRPC |
| **Console** | Admin UI (Next.js 15) | HTTP |
| **Nexus** | Fine-Grained ReBAC (Google Zanzibar) | gRPC / REST |
| **Ego** | Identity, Traits & MFA | REST |
| **Janus** | OAuth 2.0 & OIDC Server | REST |
| **Aegis** | Zero-Trust Proxy (PEP) | HTTP |
| **Vulcan** | API Keys & Macaroons | REST |
| **Hermes** | SAML/SCIM Federation | REST / XML |
| **Themis** | ABAC Policy Engine (CEL) | gRPC / REST |

## Security Guarantees

- [x] **Zero Standing Privileges**: Validated at ingress (Aegis) and engine level (Nexus/Themis).
- [x] **Tamper-Evident Logs**: SHA-256 linked audit records verifiable via `/v1/audit/verify`.
- [x] **Memory-Hard Crypto**: Argon2id for passwords (64MB memory cost).
- [x] **Decentralized Attenuation**: API keys use HMAC-SHA256 Macaroons.

## Testing

<details>
<summary>Click to view testing commands</summary>

```bash
# Run backend tests
for dir in argus nexus ego janus aegis vulcan hermes themis; do (cd $dir && go test -v ./...); done

# Run console unit tests and production build
(cd console && npm test && npm run build)

# Run console Playwright E2E acceptance tests
(cd console && npx playwright test)
```
</details>

## Next steps

- 📘 [Master API Reference](docs/api_reference_and_integration_guide.md)
- 🛠️ [Operations & Runbook](docs/operations_and_runbook.md)
- ☸️ [Kubernetes Deployment](docs/production_k8s_guide.md)
- 📦 [SDK Integration](docs/sdks_integration_guide.md)

*(For detailed documentation on specific engines like Ego, Nexus, or Aegis, browse the [`docs/`](docs/) directory).*
