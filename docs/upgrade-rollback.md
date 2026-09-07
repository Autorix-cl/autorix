# Upgrade and rollback

Treat upgrades as controlled changes. Capture the source commit, Helm values,
image digests, database migration version, approver, and rollback owner before
starting.

## Upgrade

1. Review release notes and rendered Helm diff with the production digest
   override.
2. Back up every service database and verify a restore in a separate
   environment.
3. Apply migrations through the chart lifecycle hook, then deploy the release.
4. Verify health, OAuth flows, Aegis access rules, Console administration,
   metrics, and alert delivery.

## Rollback

1. Stop or drain the rollout if safety allows.
2. Roll back the workload release to the previously approved image digest and
   values; do not switch to a mutable tag.
3. Do **not** blindly roll back a database schema. Restore from the tested
   backup only when the migration plan explicitly supports it and the incident
   commander approves the data-loss risk.
4. Re-run smoke tests and record the incident, timestamps, evidence, and
   follow-up actions.

A Helm rollback changes Kubernetes manifests; it does not undo data mutations,
key rotation, or third-party side effects. Maintain backward-compatible schema
changes until the rollback window closes.
