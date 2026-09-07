# PostgreSQL Disaster Recovery Runbook

**Target:** prove a maximum **RPO of 15 minutes** and **RTO of 4 hours** for
Autorix PostgreSQL. These are acceptance targets, not guarantees. Only a
timestamped restore drill with retained evidence proves them.

## Preconditions

- CNPG is healthy with WAL archiving enabled and a successful base backup.
- Object storage is encrypted, versioned/immutable according to retention
  policy, and reachable from the recovery cluster.
- ESO has synchronized `cnpg-object-store-credentials` into the isolated
  `autorix-recovery` namespace using least privilege.
- The incident commander has selected a recovery timestamp and approved the
  new, isolated recovery namespace.

## Quarterly restore drill

1. Record the drill start in UTC and identify the latest committed test row
   (or an approved non-sensitive checksum) in production.
2. Create the recovery namespace and apply a reviewed copy of
   `../cloudnative-pg/restore.example.yaml`, replacing only the namespace,
   storage class, object-store location, and recovery point if required.
3. Wait for CNPG to report the recovery cluster ready. Record pod readiness,
   restore logs, base-backup timestamp, and the last recovered WAL timestamp.
4. Connect through a temporary, access-controlled client and verify schema
   migrations plus the selected test row/checksum. Do not attach production
   applications to the restored cluster.
5. Calculate RPO as `recovery timestamp - last committed test timestamp`.
   It must be **<= 15 minutes**.
6. Calculate RTO as `ready-and-validated UTC - drill start UTC`. It must be
   **<= 4 hours**.
7. Delete recovery access and namespace after evidence is retained; never
   copy recovery credentials into tickets or chat.

## Incident recovery

1. Declare the incident and freeze application writes before choosing the
   recovery target. Preserve the failed primary for forensics.
2. Follow the restore-drill steps in a new namespace/cluster. Do **not**
   overwrite the original volume or reuse its Cluster name.
3. Obtain application-owner approval after data validation. Rotate database
   credentials, verify ESO synchronization, and update application endpoints
   through the normal change process.
4. Re-enable writes only after health checks, Aegis migration Job status, and
   application smoke tests succeed.

## Evidence and failure criteria

Keep the manifest revision, CNPG status, backup/WAL timestamps, validation
queries or checksums, timings, and approver identities with the drill record.
Any unavailable backup, WAL gap, failed validation, RPO breach, or RTO breach
is a failed drill: open a remediation item and repeat the drill after fixing
the cause. A green deployment pipeline is not DR evidence.
