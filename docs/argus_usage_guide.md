# Managing Fleet Control and Compliance with Autorix Argus

Autorix Argus is the central governance control plane for the Autorix Zero-Trust suite. It manages engine registration, cryptographic enrollment (`aet_` tokens), administrative sessions, and maintains an immutable, tamper-evident SHA-256 chained audit log for continuous SOC 2 and ISO 27001 compliance.

## Quick Path: Initial Cluster Claim

To bootstrap your environment, retrieve your one-time bootstrap token and claim the root administrator account.

1.  **Find the token** in Argus logs:
    ```bash
    docker logs autorix-argus | grep "Bootstrap token generated"
    ```
2.  **Claim the cluster**:
    ```bash
    curl -X POST http://localhost:4400/v1/auth/bootstrap \
      -H "Content-Type: application/json" \
      -d '{
        "bootstrap_token": "abt_0123456789abcdef...",
        "name": "Cluster Administrator",
        "email": "admin@autorix.local",
        "password": "SecretMasterKey#2026"
      }'
    ```

---

## Details

### 1. Fleet Orchestration & Enrollment

Argus uses gRPC streaming (`:50053`) to track engine heartbeats, build SHAs, and dependency topologies. Engines securely join the fleet using high-entropy Enrollment Tokens (`aet_`).

To mint an enrollment token for a new Nexus engine (`POST /v1/enrollment-tokens`):
```bash
curl -X POST http://localhost:4400/v1/enrollment-tokens \
  -H "Authorization: Bearer <session_token>" \
  -H "Content-Type: application/json" \
  -d '{"engine_type": "nexus", "environment": "production", "uses_allowed": 1, "expires_in_seconds": 86400}'
```

### 2. Immutable Cryptographic Audit Trail

Every mutating action across the Autorix ecosystem is appended to the Argus audit chain. The integrity of the chain is cryptographically guaranteed because every record includes the hash of the preceding record:

`Record Hash = SHA-256(prev_hash + id + action + resource_type + resource_id + outcome + created_at)`

You can verify chain integrity mathematically via `GET /v1/audit/verify`.

### 3. API Reference

Argus provides REST (`4400`) and gRPC (`50053`) interfaces for fleet control.

*   **Fleet Management**: `GET /v1/instances` | `POST /v1/instances/{id}/force-remove` | `GET /v1/topology` | `GET /v1/stream` (SSE state stream)
*   **Enrollment Tokens**: `POST /v1/enrollment-tokens` | `GET /v1/enrollment-tokens` | `DELETE /v1/enrollment-tokens/{id}`
*   **Operator Sessions**: `POST /v1/auth/login` | `GET /v1/auth/session` | `DELETE /v1/auth/session`
*   **Audit & Compliance**: `GET /v1/audit` | `POST /v1/audit` | `GET /v1/audit/verify` | `GET /v1/audit/export` | `GET /v1/compliance/evidence` (Generates SOC 2 reports)

### 4. gRPC Control Interface (`:50053`)

Implements `argus.v1.ArgusControlService`:
*   `Register(RegistrationRequest)`
*   `Heartbeat(HeartbeatRequest)`
*   `ListInstances(ListInstancesRequest)`
*   `GetInstance(GetInstanceRequest)`
*   `Evict(EvictRequest)`

---

## Checklist: Argus Deployment Verification

- [ ] Claim the root cluster administrator account using the `abt_` token.
- [ ] Verify the Argus REST API (`4400`) and gRPC (`50053`) are only accessible internally.
- [ ] Mint your first `aet_` enrollment token and deploy a worker engine (e.g., Nexus or Ego).
- [ ] Make a configuration change and execute `GET /v1/audit/verify` to ensure the SHA-256 chain is intact.
- [ ] Export your first SOC 2 compliance evidence report (`GET /v1/compliance/evidence`).

---

## Next Step

With your fleet management running, set up the [Console Usage Guide](./console_usage_guide.md) for a UI-based experience, or start defining authorization graphs in the [Nexus Usage Guide](./nexus_usage_guide.md).
