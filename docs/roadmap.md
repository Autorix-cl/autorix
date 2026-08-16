# Autorix: Arquitectura, Patrones y Roadmap a Producción

Autorix es un ecosistema moderno de **Identity and Access Management (IAM)**. Su filosofía radica en la **descentralización extrema, Zero Trust y el patrón de API Headless**. Cada microservicio es un binario estático e independiente en Go, con su propio almacén de datos, evitando el acoplamiento y los cuellos de botella de bases de datos compartidas.

---

## 1. Topología de Microservicios

```text
                                         [ TRÁFICO EXTERNO ]
                                                  │
                                                  ▼
                                       ┌─────────────────────┐
                                       │    Autorix Aegis    │  (Zero Trust Access Proxy :4455)
                                       └──────────┬──────────┘
                                                  │
       ┌──────────────────────┬───────────────────┼───────────────────┬──────────────────────┐
       ▼                      ▼                   ▼                   ▼                      ▼
┌──────────────┐       ┌──────────────┐    ┌──────────────┐    ┌──────────────┐       ┌──────────────┐
│Autorix Janus │       │Autorix Nexus │    │ Autorix Ego  │    │Autorix Vulcan│       │Autorix Hermes│
│(OAuth2/OIDC) │       │(ReBAC / ABAC)│    │(Identity/MFA)│    │ (API Keys)   │       │ (SAML / SCIM)│
│    :4444     │       │    :50051    │    │    :4433     │    │    :4466     │       │    :4477     │
└──────────────┘       └──────────────┘    └──────────────┘    └──────────────┘       └──────────────┘
```

| Componente | Tipo | Puerto / Ruta | Responsabilidad Core |
| :--- | :--- | :--- | :--- |
| **Autorix Nexus** | Core Engine | `50051` gRPC | Motor de Autorización Híbrido ReBAC (Zanzibar) + ABAC (Google CEL). |
| **Autorix Ego** | Core Engine | `4433` REST | Identidad, ciclo de vida de usuarios, hashing Argon2id y sesiones. |
| **Autorix Janus** | Core Engine | `4444` REST/OIDC | Servidor OAuth 2.0 y OpenID Connect con PKCE y claves asimétricas JWKS. |
| **Autorix Aegis** | Core Engine | `4455` HTTP Proxy | Zero Trust Identity & Access Proxy / PEP perimetral. |
| **Autorix Vulcan** | Core Engine | `4466` REST | API Keys seguras con prefijo y tokens de capacidad Macaroons. |
| **Autorix Hermes** | Core Engine | `4477` REST/XML | Puente SAML 2.0 a OIDC y servidor de sincronización de directorios SCIM 2.0. |
| **Autorix Console** | Admin UI | `3000` HTTP | Consola web moderna en Next.js 15 App Router para gestión de toda la suite. |
| **Autorix SDKs** | Librerías Cliente | `sdk/` | SDKs oficiales para Go (`autorix-go`), TypeScript/React (`@autorix/sdk-js`) y Python (`autorix-python`). |
| **Helm Charts & K8s** | Infraestructura | `deploy/` | Despliegue declarativo en Kubernetes con HPA, Ingress TLS y alertas Prometheus. |

---

## 2. Estado del Roadmap (100% COMPLETADO)

- [x] **Fase 1: Los 6 Motores Core en Go**: Nexus, Ego, Janus, Aegis, Vulcan, Hermes (100% tests pasando).
- [x] **Fase 2: Autorix Console Web UI**: Next.js App Router en `console/` (`:3000`).
- [x] **Fase 3: SDKs Oficiales de Integración**: Go, TypeScript/React y Python en `sdk/`.
- [x] **Fase 4: Hardening de Producción, Kubernetes & CI/CD**:
  - Helm Chart Umbrella oficial (`deploy/helm/autorix`).
  - Auto-escalado Horizontal (HPA) y Probes de Liveness/Readiness.
  - Pipeline de CI/CD automatizado en GitHub Actions (`.github/workflows/ci.yml`).
  - Observabilidad con Prometheus Alerts y Grafana Dashboard (`deploy/monitoring/`).
