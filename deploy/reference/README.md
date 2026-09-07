# Production Operations References

These are **reference manifests**, not Helm defaults. They give platform
operators an OSS baseline for External Secrets Operator (ESO), CloudNativePG
(CNPG), and a tested recovery process without storing credentials in Git.

## Apply in this order

1. Install and pin ESO and CNPG versions; validate their CRDs in a staging
   cluster.
2. Create the platform-owned `ClusterSecretStore` named `platform-secrets`,
   scoped so the `autorix` namespace can read only its own remote paths.
3. Apply the ESO manifests, then verify both generated Secrets exist and have
   never been rendered into Helm output or logs.
4. Customize and apply the CNPG Cluster and ScheduledBackup references.
5. Execute the restore drill in
   [`runbooks/postgresql-disaster-recovery.md`](runbooks/postgresql-disaster-recovery.md)
   before treating the deployment as production-ready.

## Boundaries

| Artifact | Owns | Does not own |
|---|---|---|
| `external-secrets/` | Synchronizing references from a secret manager | Provider credentials, provider setup, or secret values |
| `cloudnative-pg/cluster.example.yaml` | HA PostgreSQL and WAL/base-backup configuration | A real bucket, endpoint, storage class, or credential |
| `cloudnative-pg/restore.example.yaml` | Isolated recovery shape | In-place production restore |

Every `REPLACE_*` value must be changed through the platform change-control
process. Do not apply examples with placeholder object-store values.
