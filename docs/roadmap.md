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
| **Autorix Themis** | Core Engine | `4488` REST + `50052` gRPC | Motor de políticas ABAC con expresiones Google CEL. |
| **Autorix Console** | Admin UI | `3000` HTTP | Consola web moderna en Next.js 15 App Router para gestión de toda la suite. |
| **Autorix SDKs** | Librerías Cliente | `sdk/` | SDKs oficiales para Go (`autorix-go`), TypeScript/React (`@autorix/sdk-js`) y Python (`autorix-python`). |
| **Helm Charts & K8s** | Infraestructura | `deploy/` | Despliegue declarativo en Kubernetes con HPA, Ingress TLS y alertas Prometheus. |

---

## 2. Estado real del proyecto (auditado, no autodeclarado)

> Esta sección reemplaza una anterior que se autodeclaraba "100% COMPLETADO" — incluyendo liveness probes y observabilidad que en los hechos no existían. Ese estado nunca fue verificado contra el código real; esta versión sí lo está. El plan de trabajo vivo, con fases, specs y tareas concretas para cerrar cada gap real que sigue abierto, vive en el **Autorix Control Plane Roadmap** (control plane / Argus, identidad de la console, design system, profundidad de cada engine, observabilidad real, gobernanza y ecosistema) — pedile el link al equipo si no lo tenés a mano.

- [x] **7 motores Go** (no 6 — falta Themis en el diagrama arriba, ya existe en el código): Nexus, Ego, Janus, Aegis, Vulcan, Hermes, Themis. Cada uno con su propio módulo, base de datos y tests unitarios propios.
- [x] **Contrato de salud uniforme** (`/health/alive`, `/health/ready`, `/info`, más `grpc.health.v1.Health` en Nexus/Themis) implementado y verificado en los 7 engines — esto es nuevo: antes tres engines mentían con `"status":"healthy"` sin chequear nada, y cuatro no tenían health check alguno. El Helm chart de Ego probaba un endpoint que no existía (`/health/alive`) y entraba en CrashLoopBackOff en Kubernetes; ya está corregido y verificado con un guard de CI que compara los probes del chart contra las rutas reales del código Go.
- [x] **Librería compartida `platform`** (config, logging estructurado, middleware HTTP, lifecycle/graceful shutdown, bootstrap de Postgres con reintento, paginación) adoptada por los 7 engines.
- [x] **Console Web UI**: Next.js App Router en `console/` (`:3000`). Las 17 rutas BFF ya propagan errores reales del engine upstream (antes silenciaban cualquier fallo devolviendo `200 []`, indistinguible de una colección vacía), validan la respuesta contra un schema Zod en el límite, y las 8 páginas usan TanStack Query con estados de carga/vacío/error/"engine no disponible" diferenciados y feedback de mutaciones vía toasts.
- [x] **Autenticación y RBAC de la console**: implementado en Next.js App Router con `middleware.ts`, sesiones con cookies `HttpOnly`, tokens HMAC criptográficos, roles (Admin, Operator, Auditor, ReadOnly), flujo de setup inicial y login autenticado contra Ego.
- [x] **Profundidad de autorización real en Nexus**: los métodos gRPC (`WriteTuples`, `DeleteTuples`, `WriteCaveats`, `Expand`) están conectados y probados contra el repositorio y motor de resolución de grafo ReBAC/Zanzibar con soporte de rewrites, expand inverso y decision traces.
- [x] **SDKs oficiales y CLI**: Go, TypeScript/React y Python en `sdk/` verificados con llamadas reales a `/check` en Nexus, semántica fail-closed ante errores de red, suite de conformidad cruzada (`sdk/conformance_test.sh`), CLI unificado `autorixctl` en `cmd/autorixctl/` y Terraform Provider en `terraform-provider-autorix/`.
- [~] **Kubernetes / Helm**: chart umbrella en `deploy/helm/autorix`, con los probes de liveness/readiness ahora apuntando a endpoints que existen de verdad (antes no era el caso para varios engines) y el chart de Themis, que faltaba por completo, ya está agregado.
- [x] **Observabilidad (Prometheus/Grafana)**: módulo compartido `platform/metrics` y `platform/tracing` adoptado por los 7 motores y Argus. Todos los engines exponen `/metrics`, emiten métricas de dominio (`autorix_nexus_check_duration_seconds_bucket`, `autorix_aegis_requests_total`, `autorix_ego_active_sessions`, `autorix_themis_evaluation_duration_seconds_bucket`, `autorix_janus_tokens_issued_total`, `autorix_vulcan_keys_verified_total`, `autorix_hermes_scim_sync_total`), propagan Request/Correlation ID, y cuentan con Prometheus scraper en Docker Compose y ServiceMonitors en Helm charts reconciliados con las alertas y dashboards de Grafana.
- [x] **CI/CD**: pipeline en GitHub Actions (`.github/workflows/ci.yml`) — corregido para incluir los 7 engines (antes faltaban Themis y el módulo `platform`) y la versión de Go correcta, más un job dedicado que falla si un probe de Helm queda desincronizado del código real.
