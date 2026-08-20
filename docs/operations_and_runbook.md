# Autorix Operations & Runbook Guide

This runbook provides Day-1 (provisioning, initialization) and Day-2 (maintenance, troubleshooting, disaster recovery, security incident response) operating procedures for the Autorix Zero-Trust IAM platform.

---

## 📋 Table of Contents

1. [Cluster Bootstrap & Initialization](#1-cluster-bootstrap--initialization)
2. [Operator Account Management & Lockout Recovery](#2-operator-account-management--lockout-recovery)
3. [Engine Provisioning & Token Minting](#3-engine-provisioning--token-minting)
4. [Cryptographic Audit Trail Integrity & Forensic Verification](#4-cryptographic-audit-trail-integrity--forensic-verification)
5. [Database Backup & Disaster Recovery](#5-database-backup--disaster-recovery)
6. [Monitoring, RED Metrics & Prometheus Alerts](#6-monitoring-red-metrics--prometheus-alerts)
7. [Troubleshooting Common Incidents](#7-troubleshooting-common-incidents)

---

## 1. Cluster Bootstrap & Initialization

### Generating a Server Bootstrap Token

When Argus starts in an uninitialized environment, it generates a high-entropy bootstrap token with the `abt_` prefix:

```bash
# Locate token in container startup logs
docker logs autorix-argus | grep "Bootstrap token generated"
```

Output:
```text
level=INFO msg="Bootstrap token generated" token="abt_01917f8a7b6c5d4e3f2a1b0c9d8e7f6a"
```

### Initializing Root Owner via CLI or UI

* **Via UI**: Navigate to `http://localhost:3000/setup`, input the token, and complete the form.
* **Via cURL**:
```bash
curl -X POST http://localhost:4400/v1/auth/bootstrap \
  -H "Content-Type: application/json" \
  -d '{
    "bootstrap_token": "abt_01917f8a7b6c5d4e3f2a1b0c9d8e7f6a",
    "name": "Cluster Administrator",
    "email": "admin@autorix.local",
    "password": "SecretMasterKey#2026"
  }'
```

Once consumed, the bootstrap token is permanently hashed and invalidated.

---

## 2. Operator Account Management & Lockout Recovery

### Lockout Mechanism

To protect against brute-force attacks, Argus enforces:
* **Threshold**: 5 consecutive failed login attempts.
* **Lockout Duration**: 15 minutes (`locked_until`).
* **HTTP Response**: `429 Too Many Requests` — `"account is temporarily locked due to repeated failed attempts"`.

### Emergency Account Unlock (Break-Glass Runbook)

If an operator is locked out and needs immediate access:

```bash
# 1. Connect to PostgreSQL
docker exec -it autorix-postgres psql -U autorix -d autorix_argus

# 2. Inspect locked operator
SELECT id, email, failed_attempts, locked_until 
FROM operators 
JOIN operator_credentials ON operators.id = operator_credentials.operator_id 
WHERE email = 'admin@autorix.local';

# 3. Reset failed attempts and clear lockout timestamp
UPDATE operator_credentials 
SET failed_attempts = 0, locked_until = NULL 
WHERE operator_id = (SELECT id FROM operators WHERE email = 'admin@autorix.local');
```

---

## 3. Engine Provisioning & Token Minting

### Issuing an Enrollment Token (`aet_...`)

```bash
# Login as admin operator
SESSION_TOKEN=$(curl -s -X POST http://localhost:4400/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@autorix.local","password":"SecretMasterKey#2026"}' | jq -r .session.token)

# Mint token for a new Themis engine instance
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

## 4. Cryptographic Audit Trail Integrity & Forensic Verification

### Scheduled Integrity Check

Set up a cron job or monitoring check to query `/v1/audit/verify` every hour:

```bash
STATUS=$(curl -s http://localhost:4400/v1/audit/verify | jq -r .verified)

if [ "$STATUS" != "true" ]; then
  echo "CRITICAL: Audit trail tamper detected!" | mail -s "SECURITY ALERT: Autorix Audit Broken" secops@company.internal
fi
```

### Forensic Export

For external SOC 2 Type II or ISO 27001 auditors:

```bash
# Generate signed CSV export
curl -s http://localhost:4400/v1/audit/export?format=csv -o audit-trail-$(date +%Y%m%d).csv
```

---

## 5. Database Backup & Disaster Recovery

Autorix organizes its data into isolated PostgreSQL databases:
* `autorix_argus`: Fleet instances, operators, audit records, compliance
* `autorix_nexus`: Zanzibar relation tuples and namespaces
* `autorix_ego`: Identity profiles, Argon2id credentials, TOTP secrets
* `autorix_janus`: OAuth2 clients, authorization codes, refresh tokens
* `autorix_vulcan`: API keys, Macaroon secrets, revocation lists
* `autorix_hermes`: SAML IdP metadata, SCIM directory mappings
* `autorix_themis`: ABAC CEL policies
* `autorix_aegis`: Proxy routing rules and upstream configurations

### Automated Backup Script

```bash
#!/usr/bin/env bash
set -euo pipefail

BACKUP_DIR="/var/backups/autorix/$(date +%Y-%m-%d_%H%M%S)"
mkdir -p "$BACKUP_DIR"

DATABASES=("autorix_argus" "autorix_nexus" "autorix_ego" "autorix_janus" "autorix_vulcan" "autorix_hermes" "autorix_themis" "autorix_aegis")

for DB in "${DATABASES[@]}"; do
  echo "Backing up $DB..."
  docker exec autorix-postgres pg_dump -U autorix -Fc "$DB" > "$BACKUP_DIR/$DB.dump"
done

echo "Backup complete: $BACKUP_DIR"
```

### Disaster Restore Procedure

```bash
# Restore specific database
docker exec -i autorix-postgres pg_restore -U autorix -d autorix_argus --clean --if-exists < /var/backups/autorix/.../autorix_argus.dump
```

---

## 6. Monitoring, RED Metrics & Prometheus Alerts

Autorix exports standard Prometheus metrics on port `9090` and on each engine's `/metrics` endpoint:

### Key Metrics

| Metric | Type | Description |
| :--- | :--- | :--- |
| `http_requests_total{engine, method, path, status}` | Counter | Request throughput and HTTP response status distribution. |
| `http_request_duration_seconds{engine, method, path}` | Histogram | Latency percentiles (p50, p95, p99). |
| `grpc_requests_total{engine, method, code}` | Counter | gRPC RPC volume and status codes. |
| `grpc_request_duration_seconds{engine, method}` | Histogram | gRPC latency distribution. |
| `postgres_pool_total_connections{engine}` | Gauge | PostgreSQL pool connection utilization. |
| `postgres_pool_acquired_connections{engine}` | Gauge | Currently active database queries. |

---

## 7. Troubleshooting Common Incidents

### Incident 1: Aegis PEP Returning 502 Bad Gateway
1. Verify upstream service is healthy:
   ```bash
   curl -I http://localhost:8080/health/ready # For Nexus
   ```
2. Verify rule routing configuration:
   ```bash
   curl -s http://localhost:4456/v1/rules
   ```

### Incident 2: Zanzibar ReBAC Graph Traversal Latency Spikes
1. Inspect Nexus relation tuple volume:
   ```bash
   docker exec autorix-postgres psql -U autorix -d autorix_nexus -c "SELECT count(*) FROM relation_tuples;"
   ```
2. Check database indexes and connection pool metrics in Prometheus (`http://localhost:9090`).
