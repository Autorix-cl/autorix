# Operations & Runbook (Day-1 & Day-2)

This guide covers provisioning, maintenance, and incident response for the Autorix platform.

## Quick Actions (Incident Response)

If you are actively responding to an incident, start here:

- **Locked out of root account?** Jump to [Emergency Account Unlock](#emergency-account-unlock).
- **Aegis proxy returning 502?** Jump to [Aegis 502 Troubleshooting](#aegis-502-bad-gateway).
- **High latency in authorization?** Jump to [Nexus Latency Spikes](#nexus-rebac-latency-spikes).
- **Audit tampering alert?** Jump to [Forensic Verification](#forensic-verification).

---

## Emergency Account Unlock

If an operator is locked out (due to 5 failed attempts), you can unlock them directly via the database.

1. Connect to the database:
   ```bash
   docker compose exec postgres psql -U autorix -d autorix_argus
   ```
2. Reset the lockout timestamp for the specific email:
   ```sql
   UPDATE operator_credentials 
   SET failed_attempts = 0, locked_until = NULL 
   WHERE operator_id = (SELECT id FROM operators WHERE email = 'admin@autorix.local');
   ```
3. Ask the operator to attempt login again.

---

## Cluster Bootstrap

When starting a fresh cluster, you need a bootstrap token to create the root owner.

1. Find the token (`abt_...`) in the Argus logs:
   ```bash
   docker logs autorix-argus | grep "Bootstrap token generated"
   ```
2. Navigate to `http://localhost:3000/setup` and input the token.
3. Complete the form to create the Master Administrator.

> **Note**: The token is invalidated permanently immediately after use.

---

## Provisioning Engines

To mint an enrollment token (`aet_...`) to register a new engine instance:

1. Retrieve a session token:
   ```bash
   SESSION_TOKEN=$(curl -s -X POST http://localhost:4400/v1/auth/login \
     -H "Content-Type: application/json" \
     -d '{"email":"admin@autorix.local","password":"SecretMasterKey#2026"}' | jq -r .session.token)
   ```
2. Mint the specific engine token:
   ```bash
   curl -X POST http://localhost:4400/v1/enrollment-tokens \
     -H "Content-Type: application/json" \
     -H "Authorization: Bearer $SESSION_TOKEN" \
     -d '{
       "engine_type": "themis", 
       "environment": "production", 
       "expires_in_seconds": 86400, 
       "max_uses": 1
     }'
   ```

---

## Database Backups

Autorix uses isolated PostgreSQL databases (e.g., `autorix_argus`, `autorix_nexus`). 

**To backup all databases:**
```bash
BACKUP_DIR="/var/backups/autorix/$(date +%Y-%m-%d_%H%M%S)"
mkdir -p "$BACKUP_DIR"

for DB in autorix_argus autorix_nexus autorix_ego autorix_janus autorix_vulcan autorix_hermes autorix_themis autorix_aegis; do
  docker compose exec -T postgres pg_dump -U autorix -Fc "$DB" > "$BACKUP_DIR/$DB.dump"
done
```

**To restore a single database:**
```bash
docker compose exec -T postgres pg_restore -U autorix -d <DATABASE_NAME> --clean --if-exists < /path/to/dump
```

---

## Forensic Verification

**1. Verify cryptographic audit integrity:**
```bash
curl -s http://localhost:4400/v1/audit/verify | jq .verified
```
*(If this returns false, the cryptographic chain has been broken. Escalate as a critical security incident).*

**2. Export CSV for SOC 2 / ISO 27001 auditors:**
```bash
curl -s http://localhost:4400/v1/audit/export?format=csv -o audit-trail-$(date +%Y%m%d).csv
```

---

## Troubleshooting Checklist

### Aegis 502 Bad Gateway
- [ ] **Check upstream health**: `curl -I http://localhost:8080/health/ready` (e.g., for Nexus).
- [ ] **Check routing rules**: `curl -s http://localhost:4456/v1/rules` to ensure Aegis is pointing to the right internal IP.

### Nexus ReBAC Latency Spikes
- [ ] **Check tuple volume**: `docker compose exec -T postgres psql -U autorix -d autorix_nexus -c "SELECT count(*) FROM relation_tuples;"`
- [ ] **Check database connections**: Review `postgres_pool_acquired_connections{engine="nexus"}` in Prometheus (`http://localhost:9090`).

---

## Optional Alertmanager (self-hosted)

Prometheus evaluates the bundled rule file. Notification delivery is **disabled**
until you intentionally configure Alertmanager with an operator-owned receiver.
No webhook URL, SMTP credential, or paging token is included in this repository.

### Docker Compose

1. Copy `deploy/monitoring/alertmanager.example.yml` outside the repository and
   replace the `discard` receiver with your reviewed webhook, email, or paging
   configuration. Keep credentials in your secret manager or an ignored file.
2. Start monitoring and notifications together:
   ```bash
   ALERTMANAGER_CONFIG_PATH=/secure/path/alertmanager.yml \
     docker compose --profile core --profile extended --profile alerts up -d prometheus alertmanager
   ```
3. Confirm Alertmanager is reachable only on the local host at
   `http://127.0.0.1:9093`. Prometheus remains usable when the `alerts` profile
   is not enabled, but it cannot deliver notifications.

### Kubernetes Helm

Create the configuration Secret outside Helm values. Its `alertmanager.yml`
key must contain the complete Alertmanager configuration:

```bash
kubectl -n autorix create secret generic autorix-alertmanager-config \
  --from-file=alertmanager.yml=/secure/path/alertmanager.yml
```

Then provide a digest-pinned values overlay:

```yaml
monitoring:
  alertmanager:
    enabled: true
    existingConfigSecret: autorix-alertmanager-config
    configKey: alertmanager.yml
    image:
      digest: sha256:<verified-alertmanager-image-digest>
```

The chart mounts that existing Secret read-only and uses ephemeral local alert
state. Configure durable state and high availability through a dedicated
Alertmanager operator if those are required by your recovery objectives.
