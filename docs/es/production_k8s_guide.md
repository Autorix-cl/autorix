# Guía de Despliegue en Producción con Kubernetes

Esta guía detalla el despliegue de **Autorix** en entornos Kubernetes empresariales con alta disponibilidad, sondas de salud (Health Probes), ServiceMonitors de Prometheus y escalabilidad horizontal.

---

## 🏛️ 1. Arquitectura en Kubernetes

Cada motor se despliega como un `Deployment` independiente sin estado conectado a un clúster gestionado de PostgreSQL (ej. AWS RDS, Google Cloud SQL o Crunchy Data PGO).

### Sondas de Salud Estándar

| Motor | Tipo de Sonda | Endpoint |
| :--- | :--- | :--- |
| **Nexus** | Liveness / Readiness | `:8080/health/ready` |
| **Themis** | Liveness / Readiness | `:4488/health/ready` |
| **Ego** | Liveness / Readiness | `:4433/health/ready` |
| **Janus** | Liveness / Readiness | `:4444/health/ready` |
| **Aegis** | Liveness / Readiness | `:4456/health/ready` |
| **Vulcan** | Liveness / Readiness | `:4466/health/ready` |
| **Hermes** | Liveness / Readiness | `:4477/health/ready` |
| **Argus** | Liveness / Readiness | `:4400/health/ready` |
