# Security policy and support

Autorix is not certified to ISO 27001, SOC 2, PCI DSS, or any banking or
financial-services standard. Features, tests, deployment examples, and evidence
exports can support a customer's assurance program, but they are not a
certification, attestation, audit result, or production approval.

## Report a vulnerability

Do **not** report suspected vulnerabilities in public issues, discussions,
logs, screenshots, or pull requests.

Use the repository's private GitHub security advisory form:

<https://github.com/Autorix-cl/autorix/security/advisories/new>

If the form is unavailable, open a minimal public issue asking maintainers for a
private reporting channel; do not include vulnerability details.

Include:

- affected release, commit, deployment mode, and component;
- impact and prerequisites;
- reproducible proof of concept or clear reproduction steps;
- suggested mitigation, if known; and
- a secure contact method and disclosure deadline, if applicable.

Please avoid accessing, modifying, or exfiltrating data that you do not own.
Testing must stay within accounts and environments you are authorized to use.

## Triage and disclosure

Maintainers will acknowledge a private report, assess severity and affected
versions, coordinate a fix, and publish release notes or an advisory when a fix
is available. Timing depends on reproducibility, impact, maintainers, and a
safe remediation path; this repository makes no guaranteed response or fix-time
commitment.

Reporters should keep details private until maintainers confirm coordinated
disclosure or the agreed disclosure deadline expires. Credit is offered only
with the reporter's permission.

## Support boundary

This repository currently provides best-effort, community support only. It does
not provide a support SLA, 24/7 incident response, managed operations, a
warranty, or regulatory compliance consulting. Organizations with regulated or
critical workloads must establish their own incident-response, vendor-risk,
change-management, backup/recovery, monitoring, and support arrangements.

## Production assurance

Before approving a deployment, the accountable organization must complete the
[production acceptance checklist](docs/production_acceptance_checklist.md),
retain the requested evidence, and obtain its own risk and change approvals.
The [assurance evidence matrix](docs/assurance_evidence_matrix.md) maps useful
repository evidence to common frameworks and explicitly identifies the
operational evidence that code cannot provide.
