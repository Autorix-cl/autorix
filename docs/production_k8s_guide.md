# Guía de Despliegue en Producción y Kubernetes: Autorix (Fase 4)

Esta guía detalla el procedimiento para desplegar la suite completa **Autorix** en clústeres **Kubernetes** (EKS, GKE, AKS, K3s) utilizando el **Helm Chart oficial**.

---

## 1. Arquitectura de Despliegue en Kubernetes

```text
                           [ CLIENTES & SPAs ]
                                   │
                                   ▼ (HTTPS / TLS)
                       ┌──────────────────────┐
                       │   Kubernetes Ingress │ (NGINX / Traefik)
                       └──────────┬───────────┘
                                  │
         ┌────────────────────────┼────────────────────────┐
         │                        │                        │
         ▼                        ▼                        ▼
  auth.domain.com          api.domain.com          console.domain.com
┌─────────────────┐      ┌─────────────────┐      ┌──────────────────┐
│  Janus (OIDC)   │      │  Aegis (Proxy)  │      │ Autorix Console  │
│    Replicas: 2  │      │   Replicas: 3   │      │   Replicas: 2    │
└─────────────────┘      └────────┬────────┘      └──────────────────┘
                                  │ (gRPC interno)
                                  ▼
                         ┌─────────────────┐
                         │  Nexus (ReBAC)  │
                         │   Replicas: 3   │
                         └────────┬────────┘
                                  │
                                  ▼
                         ┌─────────────────┐
                         │   PostgreSQL    │ (HA Database Cluster)
                         └─────────────────┘
```

---

## 2. Instalación con Helm

### 1. Añadir Secretos de Base de Datos
```bash
kubectl create secret generic autorix-db-credentials \
  --from-literal=password="YourStrongPostgresPassword2026!"
```

---

### 2. Desplegar la Suite con Helm
```bash
helm upgrade --install autorix deploy/helm/autorix \
  --namespace autorix \
  --create-namespace \
  --set global.domain="yourcompany.com" \
  --set global.postgresql.host="postgres-cluster.database.svc.cluster.local"
```

---

### 3. Verificar el Estado de los Pods y el Auto-Escalado (HPA)
```bash
kubectl get pods -n autorix
kubectl get hpa -n autorix
```

Salida esperada:
```text
NAME                                  READY   STATUS    RESTARTS   AGE
autorix-aegis-6f8d9b-1                1/1     Running   0          2m
autorix-aegis-6f8d9b-2                1/1     Running   0          2m
autorix-aegis-6f8d9b-3                1/1     Running   0          2m
autorix-nexus-7c4a1e-1                1/1     Running   0          2m
autorix-nexus-7c4a1e-2                1/1     Running   0          2m
autorix-nexus-7c4a1e-3                1/1     Running   0          2m
autorix-ego-8d5c2f-1                  1/1     Running   0          2m
autorix-janus-9e6b3a-1                1/1     Running   0          2m
autorix-vulcan-5f2a1b-1               1/1     Running   0          2m
autorix-hermes-4c3d2e-1               1/1     Running   0          2m
autorix-console-1a2b3c-1              1/1     Running   0          2m
```

---

## 3. Observabilidad y Monitoreo

1. **Importar Reglas de Alerta en Prometheus**:
   ```bash
   kubectl apply -f deploy/monitoring/prometheus-alerts.yaml -n monitoring
   ```
2. **Cargar Dashboard en Grafana**:
   - Abrí tu consola de Grafana $\rightarrow$ **Dashboards** $\rightarrow$ **Import** $\rightarrow$ subí el archivo `deploy/monitoring/grafana-dashboard.json`.
