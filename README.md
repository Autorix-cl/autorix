# Autorix: Next-Generation Zero-Trust IAM Ecosystem

**Autorix** es una suite de Gestión de Identidad y Acceso (IAM) moderna, modular y de ultra-bajo rendimiento inspirada en la suite completa de ORY, diseñada bajo una arquitectura **Zero Trust** con microservicios independientes y desacoplados.

---

## 🏛️ Arquitectura del Ecosistema (Los 6 Motores)

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

| Microservicio | Puerto | Protocolo | Equivalente ORY | Descripción |
| :--- | :--- | :--- | :--- | :--- |
| **Autorix Nexus** | `50051` | gRPC | ORY Keto + Zanzibar | Motor de autorización híbrido ReBAC (Zanzibar) + ABAC (Google CEL). |
| **Autorix Ego** | `4433` | REST | ORY Kratos | Identidad, ciclo de vida de usuarios, perfiles dinámicos y hashing Argon2id. |
| **Autorix Janus** | `4444` | REST / OIDC | ORY Hydra | Servidor OAuth 2.0 y OpenID Connect con PKCE y claves asimétricas JWKS. |
| **Autorix Aegis** | `4455` | HTTP Proxy | ORY Oathkeeper | Zero Trust Identity & Access Proxy y PEP perimetral. |
| **Autorix Vulcan** | `4466` | REST | ORY Talos | API Keys con prefijo (`av_live_...`) y Macaroons atenuables localmente. |
| **Autorix Hermes** | `4477` | REST / XML | ORY Polis | Puente SAML 2.0 a OIDC y servidor de sincronización SCIM 2.0. |

---

## 🚀 Inicio Rápido (Docker Compose)

Levanta toda la suite junto con PostgreSQL y sus 5 bases de datos aisladas con un solo comando:

```bash
docker compose up --build
```

---

## 🧪 Ejecución de Pruebas Unitarias y de Integración

Todos los microservicios cuentan con cobertura de tests automatizados (100% Green):

```bash
# Correr tests en cada motor:
(cd nexus && go test -v ./...)
(cd ego && go test -v ./...)
(cd janus && go test -v ./...)
(cd aegis && go test -v ./...)
(cd vulcan && go test -v ./...)
(cd hermes && go test -v ./...)
```

---

## 📚 Documentación Técnica

* 📘 [Manual Maestro de Integración y Referencia de APIs](docs/api_reference_and_integration_guide.md)
* 🧠 [Guía de Uso: Autorix Nexus (ReBAC + ABAC)](docs/nexus_usage_guide.md)
* 👤 [Guía de Uso: Autorix Ego (Identidad y Sesiones)](docs/ego_usage_guide.md)
* 🔑 [Guía de Uso: Autorix Janus (OAuth2 / OIDC)](docs/janus_usage_guide.md)
* 🛡️ [Guía de Uso: Autorix Aegis (Zero Trust Proxy)](docs/aegis_usage_guide.md)
* ⚡ [Guía de Uso: Autorix Vulcan (API Keys & Macaroons)](docs/vulcan_usage_guide.md)
* 🏢 [Guía de Uso: Autorix Hermes (SAML 2.0 & SCIM 2.0)](docs/hermes_usage_guide.md)
* 🗺️ [Roadmap Técnico y Arquitectura General](docs/roadmap.md)
