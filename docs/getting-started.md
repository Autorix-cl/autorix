# Docker Compose onboarding

Docker Compose is for **local development and evaluation only**. It uses local
ports, development credentials, HTTP between services, and `sslmode=disable`.
Do not expose it to the Internet or treat it as a production deployment.

## Start the core stack

Requirements: Docker Engine with Compose v2, 8 GB RAM, and ports `3000`,
`4444`, `4455`, and `5432` available.

```bash
./scripts/generate-local-secrets.sh
docker compose --profile core up -d --build
docker compose --profile core ps
```

Run the secret generator before the Compose command on every fresh local
setup. It creates untracked development secrets; do not commit its output.

Wait until the services report healthy, then open the Console at
`http://localhost:3000`. Retrieve Argus's one-time
bootstrap token from the local logs and complete the setup flow:

```bash
docker logs autorix-argus | grep -i 'bootstrap token'
```

Do not copy a bootstrap token or password into source control, shell history,
or a shared ticket.

## Verify local health

```bash
curl -fsS http://localhost:4444/health/ready  # Janus public API
curl -fsS http://localhost:4455/health/ready  # Aegis proxy
curl -fsS http://localhost:4400/health/ready  # Argus control plane
```

Janus administration (`:4445`) and Aegis administration (`:4456`) are not
published by Compose. Use the Console or an authenticated, local-only access
path; do not add host-port mappings for those listeners.

Next: [security configuration](./security-configuration.md).
