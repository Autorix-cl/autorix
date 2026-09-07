# Assurance evidence matrix

This matrix helps operators collect evidence for their own ISO/IEC 27001:2022,
SOC 2, and NIST CSF 2.0 assessments. It is not a statement of compliance,
certification, attestation, audit completion, or control operating
effectiveness. A qualified assessor and the deploying organization determine
scope, applicability, evidence sufficiency, and risk acceptance.

## How to use this matrix

1. Select controls applicable to the organization and deployment.
2. Capture repository evidence and environment-specific evidence for the same
   release period.
3. Test operating effectiveness, record exceptions, and have the accountable
   control owner and assessor evaluate the result.

`Repository evidence` shows an implementation starting point only. `Operational
evidence required` cannot be produced by this source repository and remains the
operator's responsibility.

| Assurance objective | ISO/IEC 27001:2022 reference | SOC 2 reference | NIST CSF 2.0 function | Repository evidence candidate | Operational evidence required | Status |
|---|---|---|---|---|---|---|
| Access control and least privilege | Annex A 5.15, 5.18, 8.2, 8.3 | CC6.1, CC6.2, CC6.3 | Protect (PR.AA) | OAuth/OIDC, RBAC, ReBAC/ABAC engines; private admin boundaries; tests and configuration guards | Identity lifecycle, MFA enforcement, access-review records, privileged-access approvals | Partial |
| Authentication and credential protection | Annex A 5.17, 8.5 | CC6.1, CC6.6 | Protect (PR.AA, PR.DS) | Ego authentication mechanisms; Janus token flows; Aegis JWT validation | Password/MFA policy evidence, secrets inventory, key custody, user-training and exception records | Partial |
| Secure configuration and hardening | Annex A 8.9, 8.20, 8.22 | CC6.6, CC7.1 | Protect (PR.PS, PR.IR) | Helm security contexts, NetworkPolicies, private listeners, deployment security-boundary checks | Cluster admission settings, approved baselines, configuration reviews, egress proof, exceptions | Partial |
| Cryptographic key management | Annex A 8.24 | CC6.1, CC6.7 | Protect (PR.DS) | Janus persisted signing keys, JWKS, rotation and validation code | HSM/KMS design, separation of duties, key ceremonies, backup/escrow, revocation drill records | Partial |
| Logging, monitoring, and auditability | Annex A 8.15, 8.16, 8.17 | CC7.1, CC7.2 | Detect (DE.CM), Govern (GV.OC) | Argus audit capabilities; metrics endpoints; runbook verification commands | Central log integrity, SIEM alerts, retention proof, time-sync evidence, alert test results | Partial |
| Vulnerability and change management | Annex A 8.8, 8.32 | CC7.1, CC8.1 | Identify (ID.RA), Govern (GV.RM) | CI workflow and repository history; `SECURITY.md`; release checklist | SBOM provenance, scan results, remediation SLAs, CAB/change records, independent review | Operator-owned |
| Incident response | Annex A 5.24–5.28 | CC7.3, CC7.4 | Respond (RS.MA, RS.CO) | Security disclosure policy; operations runbook | Incident plan, on-call roster, tabletop outcomes, evidence preservation, external notifications | Operator-owned |
| Backup and recovery | Annex A 8.13, 5.30 | A1.2, CC7.5 | Recover (RC.RP) | Operations runbook backup/restore examples | Approved RTO/RPO, encrypted backup reports, restore/failover drill evidence, business continuity plan | Operator-owned |
| Third-party and supplier risk | Annex A 5.19–5.23 | CC9.2 | Govern (GV.SC) | Dependency manifests and deployment references | Vendor inventory, risk assessments, contractual terms, continuous monitoring | Operator-owned |
| Privacy and regulatory obligations | Context-dependent | Context-dependent | Govern (GV.OC, GV.RM) | Product data-flow documentation where available | Lawful basis, DPIA, retention/deletion proof, residency, notices, regulator-specific assessments | Operator-owned |

## Evidence quality rules

Evidence should be attributable to a release and control owner, timestamped,
protected from alteration, retained per policy, and independently reviewable.
Screenshots alone are weak evidence; pair them with immutable logs, signed
artifacts, configuration exports, test output, tickets, and approval records.

A generated audit export or dashboard is evidence material, not proof that a
control operated effectively for a period. The organization must verify
completeness, retention, access control, and the process that produced it.

## Non-claims

Autorix does not claim ISO 27001 certification, SOC 2 attestation, NIST CSF
conformance, PCI DSS compliance, or banking-production approval. Deployments
may have different code, configuration, cloud controls, personnel processes,
and regulatory scope; those conditions cannot be certified by a repository.
