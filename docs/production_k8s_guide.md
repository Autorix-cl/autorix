# Production Kubernetes Deployment Guide: Autorix IAM Suite

This guide details the end-to-end deployment of the **Autorix Zero-Trust IAM Suite** on **Kubernetes** (EKS, GKE, AKS, Talos, K3s) using the official Helm Chart and production-grade ingress, autoscaling, and secret management patterns.

---

## 1. Kubernetes Architecture Topology

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

---

## 2. Helm Installation & Configuration

### Step 1: Create Database & Master Secrets

```bash
kubectl create namespace autorix

kubectl create secret generic autorix-secrets \
  --namespace autorix \
  --from-literal=postgres-password="ProductionPostgresPassword#2026" \
  --from-literal=master-encryption-key="32-byte-hex-encoded-master-key-here" \
  --from-literal=jwt-signing-key="-----BEGIN RSA PRIVATE KEY-----\n..."
```

### Step 2: Deploy Helm Chart

```bash
helm upgrade --install autorix deploy/helm/autorix \
  --namespace autorix \
  --values deploy/helm/autorix/values-production.yaml \
  --set global.domain="autorix.enterprise.io" \
  --set global.postgresql.host="postgres-ha.database.svc.cluster.local"
```

---

## 3. Pod Verification & Autoscaling (HPA)

```bash
kubectl get pods -n autorix
```

**Expected Pod Output:**
```text
NAME                                  READY   STATUS    RESTARTS   AGE
autorix-argus-7f8a9b-1                1/1     Running   0          5m
autorix-argus-7f8a9b-2                1/1     Running   0          5m
autorix-themis-8e9f0a-1               1/1     Running   0          5m
autorix-themis-8e9f0a-2               1/1     Running   0          5m
autorix-nexus-7c4a1e-1                1/1     Running   0          5m
autorix-nexus-7c4a1e-2                1/1     Running   0          5m
autorix-ego-8d5c2f-1                  1/1     Running   0          5m
autorix-janus-9e6b3a-1                1/1     Running   0          5m
autorix-aegis-6f8d9b-1                1/1     Running   0          5m
autorix-aegis-6f8d9b-2                1/1     Running   0          5m
autorix-vulcan-5f2a1b-1               1/1     Running   0          5m
autorix-hermes-4c3d2e-1               1/1     Running   0          5m
autorix-console-1a2b3c-1              1/1     Running   0          5m
autorix-console-1a2b3c-2              1/1     Running   0          5m
```

### Horizontal Pod Autoscaler (HPA) Configuration

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: autorix-nexus-hpa
  namespace: autorix
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: autorix-nexus
  minReplicas: 3
  maxReplicas: 20
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
```

---

## 4. Liveness & Readiness Probes Matrix

All Go engines implement non-blocking health probes using `/health/live` and `/health/ready`:

| Service | Probe Type | HTTP Endpoint | Initial Delay | Timeout |
| :--- | :--- | :--- | :--- | :--- |
| **Argus** | Liveness / Readiness | `:4400/health/ready` | 3s | 1s |
| **Themis** | Liveness / Readiness | `:4488/health/ready` | 3s | 1s |
| **Nexus** | Liveness / Readiness | `:8080/health/ready` | 3s | 1s |
| **Ego** | Liveness / Readiness | `:4433/health/ready` | 3s | 1s |
| **Janus** | Liveness / Readiness | `:4444/health/ready` | 3s | 1s |
| **Aegis** | Liveness / Readiness | `:4456/health/ready` | 3s | 1s |
| **Vulcan** | Liveness / Readiness | `:4466/health/ready` | 3s | 1s |
| **Hermes** | Liveness / Readiness | `:4477/health/ready` | 3s | 1s |
| **Console** | Liveness / Readiness | `:3000/api/health` | 5s | 2s |

---

## 5. Multi-Environment & Tenant Isolation

In multi-tenant deployments, isolate environments using Kubernetes namespaces or Argus environment partitioning:
* `production`: Production cluster with strict P8 cryptographic audit retention.
* `staging`: Pre-production validation environment.
* `qa` / `development`: Ephemeral test environments.
