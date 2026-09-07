# Deploy Autorix IAM Suite on Kubernetes

This guide details the end-to-end deployment of the **Autorix Zero-Trust IAM Suite** on **Kubernetes** (EKS, GKE, AKS, Talos, K3s) using the official Helm Chart and production-grade ingress, autoscaling, and secret management patterns.

## Quick path

Deploy the suite using the official Helm chart and pre-configured production values:

```bash
kubectl create namespace autorix

kubectl create secret generic autorix-db-credentials \
  --namespace autorix \
  --from-literal=argus-database-url='postgres://.../autorix_argus?sslmode=verify-full' \
  --from-literal=ego-database-url='postgres://.../autorix_ego?sslmode=verify-full' \
  --from-literal=hermes-database-url='postgres://.../autorix_hermes?sslmode=verify-full' \
  --from-literal=janus-database-url='postgres://.../autorix_janus?sslmode=verify-full' \
  --from-literal=nexus-database-url='postgres://.../autorix_nexus?sslmode=verify-full' \
  --from-literal=themis-database-url='postgres://.../autorix_themis?sslmode=verify-full' \
  --from-literal=vulcan-database-url='postgres://.../autorix_vulcan?sslmode=verify-full'

helm upgrade --install autorix deploy/helm/autorix \
  --namespace autorix \
  --values deploy/helm/autorix/values-production.yaml \
  --set global.domain="autorix.enterprise.io" \
  --set global.postgresql.host="postgres-ha.database.svc.cluster.local"
```

The secret named by `global.postgresql.existingSecret` must contain all seven
keys above. Each value is the complete service-specific `DATABASE_URL`; choose
the TLS mode and credentials required by the managed PostgreSQL deployment.

## Details

### Architecture Topology

```text
                           [ INCOMING TRAFFIC ]
                                    │
                                    ▼ (TLS 1.3 Termination)
                        ┌──────────────────────┐
                        │   Kubernetes Ingress │ (NGINX / Envoy / Traefik)
                        └──────────┬───────────┘
                                   │
         ┌─────────────────────────┼─────────────────────────┐
         │                         │                         │
         ▼                         ▼                         ▼
  auth.domain.com           api.domain.com            console.domain.com
┌─────────────────┐       ┌─────────────────┐       ┌──────────────────┐
│  Janus (OIDC)   │       │  Aegis (Proxy)  │       │ Autorix Console  │
│  (Replicas: 2+) │       │  (Replicas: 3+) │       │  (Replicas: 2+)  │
└────────┬────────┘       └────────┬────────┘       └────────┬─────────┘
         │                         │ (Internal gRPC)         │
         │                         ▼                         │
         │                ┌─────────────────┐                │
         │                │  Nexus / Themis │                │
         │                │  (Replicas: 3+) │                │
         │                └────────┬────────┘                │
         │                         │                         │
         └─────────────────────────┼─────────────────────────┘
                                   │
                                   ▼ (Fleet Management & Audit)
                          ┌─────────────────┐
                          │  Argus Control  │
                          │  (Replicas: 2+) │
                          └────────┬────────┘
                                   │
                                   ▼
                          ┌─────────────────┐
                          │ PostgreSQL HA   │ (Cloud SQL / RDS / Stolon)
                          └─────────────────┘
```

### High Availability and Autoscaling

Services like Nexus and Aegis handle high-throughput workloads. Deploy Horizontal Pod Autoscalers (HPA) targeting CPU/Memory (e.g., target 70% CPU utilization) to scale these components up to 20+ replicas dynamically.

### Health Probes Matrix

All Go engines implement non-blocking health probes using `/health/live` and `/health/ready`.
- **Argus:** `:4400/health/ready`
- **Themis:** `:4488/health/ready`
- **Nexus:** `:8080/health/ready`
- **Ego:** `:4433/health/ready`
- **Janus:** `:4444/health/ready`
- **Aegis:** `:4456/health/ready`
- **Vulcan:** `:4466/health/ready`
- **Hermes:** `:4477/health/ready`
- **Console:** `:3000/api/health`

### Multi-Environment & Tenant Isolation

Isolate multi-tenant deployments using Kubernetes namespaces or Argus environment partitioning:
- **production:** Production cluster with strict P8 cryptographic audit retention.
- **staging:** Pre-production validation environment.
- **qa / development:** Ephemeral test environments.

## Checklist

- [ ] Provision a highly available PostgreSQL cluster (e.g., Cloud SQL, RDS).
- [ ] Create the `autorix-secrets` Kubernetes secret with database passwords and cryptographic keys.
- [ ] Deploy the Helm chart pointing to your production database host and base domain.
- [ ] Configure Horizontal Pod Autoscaler (HPA) for highly utilized components (Nexus, Aegis).
- [ ] Verify pod readiness and liveness via `kubectl get pods -n autorix`.

## Next step

Configure OAuth2 client applications and issue access tokens using the [Autorix Janus Usage Guide](./janus_usage_guide.md).
