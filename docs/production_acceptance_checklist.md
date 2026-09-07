# Production acceptance checklist

A deployment is ready to enter a production change window only when every
applicable **required** item has an accountable owner, dated evidence, and
recorded risk acceptance. This checklist is an engineering gate, not a
certification or a statement that Autorix is suitable for banking use.

## Quick path

1. Create a release record with the exact source commit, images, Helm values,
   environment, approver, and rollback owner.
2. Attach evidence for every required control below; mark unmet items as a
   blocked release or obtain documented risk acceptance from the accountable
   organization.
3. Execute the deployment and rollback drills in a production-like environment,
   then record the results in the release record.

## Release identity and supply chain

- [ ] **Required:** Source commit, build timestamp, image digest, SBOM, and
  dependency-vulnerability scan are attached to the release record.
- [ ] **Required:** Images are signed and the cluster admission policy verifies
  the signer and digest before deployment.
- [ ] **Required:** Critical and high findings have been remediated or have a
  time-bounded, approved exception with compensating controls.
- [ ] **Required:** CI results cover unit, integration, race, configuration,
  and Helm-render checks relevant to the release.
- [ ] **Required:** An independent reviewer approves the production diff and
  change record.

## Identity, authorization, and cryptography

- [ ] **Required:** Production issuer, redirect URIs, CORS origins, resource
  audiences, and Aegis upstreams are explicitly allowlisted and reviewed.
- [ ] **Required:** Administrative listeners are private; access is protected
  by identity, authorization, network controls, and audit logging.
- [ ] **Required:** Signing keys, database credentials, certificates, and
  encryption material originate from an approved secrets system. No default or
  development secret is deployed.
- [ ] **Required:** Key rotation, emergency revocation, and JWKS cache-expiry
  procedures have been rehearsed without losing token-verification continuity.
- [ ] **Required:** Privileged access follows least privilege, MFA, periodic
  access review, and a tested break-glass procedure.

## Platform and data protection

- [ ] **Required:** Database transport uses certificate validation, least-
  privilege service accounts, encrypted storage, tested backups, and a defined
  retention period.
- [ ] **Required:** Kubernetes admission, namespaces, service accounts,
  NetworkPolicies, pod-security settings, resource quotas, and egress controls
  have been validated in the target cluster.
- [ ] **Required:** Ingress TLS, HSTS policy, rate limits, WAF/DDoS controls,
  and certificate renewal ownership are documented and tested.
- [ ] **Required:** Tenant boundaries, data residency, retention, deletion,
  and encryption requirements have been approved by the data owner.
- [ ] **Required:** No production workload relies on local mutable state unless
  a documented persistent volume, backup, recovery procedure, and ownership
  model exist.

## Resilience, monitoring, and incident response

- [ ] **Required:** Availability objectives, RTO, RPO, alert routes, on-call
  ownership, and dependency escalation paths are approved.
- [ ] **Required:** Restore from an encrypted backup was completed in a
  production-like environment and met the approved RTO/RPO.
- [ ] **Required:** Dashboards and alerts cover authentication failures,
  authorization denials, token/key lifecycle failures, database health,
  saturation, and audit-log integrity.
- [ ] **Required:** Logs are centralized, protected from unauthorized changes,
  time-synchronized, retained per policy, and tested for incident retrieval.
- [ ] **Required:** An incident tabletop exercises credential compromise,
  service outage, data exposure, and rollback; actions are tracked to closure.

## Security assurance and change control

- [ ] **Required:** A current threat model covers public endpoints, private
  administration, trust boundaries, data flows, abuse cases, and mitigations.
- [ ] **Required:** A qualified independent penetration test covers the target
  architecture and configuration. Findings are remediated or formally accepted.
- [ ] **Required:** Legal, privacy, regulatory, and third-party risk reviews
  required by the deploying organization are complete.
- [ ] **Required:** The release has an approved rollback plan, tested rollback
  artifact, communications plan, and post-deployment verification plan.

## Evidence record

Use one release record per production change. At minimum retain:

| Field | Required evidence |
|---|---|
| Release identity | Commit SHA, image digests, SBOM, environment, change ticket |
| Approvals | Engineering, security, operations, and business/risk approvers |
| Test results | CI URLs/logs, integration environment, test data scope |
| Risk decisions | Exceptions, expiry dates, compensating controls, accountable owner |
| Operational proof | Restore, failover, key rotation, rollback, and incident-drill results |
| Post-deployment | Health checks, metrics, alerts, audit verification, sign-off |

## Framework mapping

See the [assurance evidence matrix](./assurance_evidence_matrix.md) for
candidate evidence mapped to ISO/IEC 27001:2022, SOC 2 Trust Services Criteria,
and NIST CSF 2.0. The matrix identifies evidence sources; it does not establish
control design, operating effectiveness, or certification.
