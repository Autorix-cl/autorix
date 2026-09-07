# Production transition

A Kubernetes/Helm deployment is a separate operating model from Docker
Compose. Complete this transition in a production-like environment first.

1. Build, scan, sign, and record immutable image digests. Production Helm
   rendering intentionally fails until every workload has a real SHA-256 digest.
2. Provision TLS-enabled PostgreSQL, secret-manager-backed per-service database
   URLs, backups, monitoring, and tested restore access.
3. Create the required database Secret and provide a release values file with
   verified image digests, domain, ingress certificates, network selectors, and
   resource limits. Do not put passwords in values files.
4. Deploy with Helm, validate readiness, private-admin boundaries, migrations,
   dashboards, and alert routing.
5. Run an application smoke test, restore drill, and rollback rehearsal before
   accepting traffic.

Use the [Kubernetes production guide](./production_k8s_guide.md),
[production acceptance checklist](./production_acceptance_checklist.md) as engineering inputs. They do not
constitute banking certification, regulatory approval, or an external security
assessment.
