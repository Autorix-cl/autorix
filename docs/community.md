# Community, support, and release policy

Autorix is an Apache-2.0 licensed, self-hosted IAM project. It is maintained
with best-effort community support; it is not a managed service and has no SLA.

## Get help

| Need | Use |
| --- | --- |
| Setup, configuration, and usage | [GitHub Discussions](https://github.com/Autorix-cl/autorix/discussions) |
| Reproducible defect | [GitHub Issues](https://github.com/Autorix-cl/autorix/issues/new/choose) |
| Security vulnerability | [Private security advisory](https://github.com/Autorix-cl/autorix/security/advisories/new) |

Do not publish secrets, personal data, or vulnerability details in public
channels. Review [SECURITY.md](https://github.com/Autorix-cl/autorix/blob/main/SECURITY.md)
before reporting a security concern.

## Versioning and releases

Autorix uses [Semantic Versioning](https://semver.org/spec/v2.0.0.html):

- **Major** releases may contain breaking changes and provide migration notes.
- **Minor** releases add backward-compatible capabilities.
- **Patch** releases contain backward-compatible fixes.

A signed Git tag and its GitHub Release are the authoritative release record.
Use immutable image digests in production, read release notes before upgrading,
and follow the [upgrade and rollback guide](./upgrade-rollback.md). The
[CHANGELOG](https://github.com/Autorix-cl/autorix/blob/main/CHANGELOG.md)
records notable changes.

## Known limitations

- Alert routing and external notification delivery require operator-managed
  Prometheus and Alertmanager configuration. Autorix does not provide a managed
  monitoring service or on-call coverage.
- Docker Compose is for local development and evaluation, not an Internet-facing
  or production deployment. Use the Kubernetes guide and complete the
  [production acceptance checklist](./production_acceptance_checklist.md) for
  production planning.
- Autorix is not certified to ISO 27001, SOC 2, PCI DSS, or banking standards.
  Each operating organization must perform its own security, compliance, and
  operational approvals.

## Contributing

Read [CONTRIBUTING.md](https://github.com/Autorix-cl/autorix/blob/main/CONTRIBUTING.md)
and the [Code of Conduct](https://github.com/Autorix-cl/autorix/blob/main/CODE_OF_CONDUCT.md)
before opening a contribution. Public product documentation is maintained in
English and neutral technical Spanish.
