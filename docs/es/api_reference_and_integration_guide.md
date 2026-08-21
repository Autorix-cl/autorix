# Guía Maestra de Arquitectura e Integración de Flujos

Esta guía detalla la arquitectura de integración entre todos los motores de **Autorix Zero-Trust IAM**, su matriz de puertos y el estándar unificado de credenciales.

---

## 🌐 1. Matriz de Puertos y Protocolos

| Motor | Puerto REST | Puerto gRPC | Propósito Principal |
| :--- | :--- | :--- | :--- |
| **Autorix Argus** | `4400` | `50053` | Plano de control, registro de flota, auditoría SHA-256. |
| **Autorix Nexus** | `8080` | `50051` | Control de acceso ReBAC (Google Zanzibar) y APL. |
| **Autorix Themis** | `4488` | `50052` | Motor de políticas ABAC en Google CEL. |
| **Autorix Ego** | `4433` | - | Identidad, hashing Argon2id, MFA TOTP y WebAuthn. |
| **Autorix Janus** | `4444` | - | Servidor OAuth 2.0 / OpenID Connect con retos desacoplados. |
| **Autorix Aegis** | `4455` (PEP) / `4456` (Admin) | - | Proxy inverso PEP Zero-Trust perimetral. |
| **Autorix Vulcan** | `4466` | - | API Keys con prefijos y Macaroons atenuables offline. |
| **Autorix Hermes** | `4477` | - | Federación SAML 2.0 y sincronización SCIM 2.0. |
| **Autorix Console** | `3000` | - | Panel de administración web (Next.js 15 App Router). |

---

## 🔑 2. Estándar de Credenciales

| Prefijo | Tipo | Formato | Propósito |
| :--- | :--- | :--- | :--- |
| `abt_` | **Bootstrap Token** | Hex de alta entropía | Reclamo inicial del Master Owner en Argus. |
| `aet_` | **Enrollment Token** | Hex de alta entropía | Token de un solo uso para enrolar un motor a la flota. |
| `ast_` | **Session Token** | Token seguro con sal | Sesiones de operadores en Console y usuarios en Ego. |
| `av_live_` | **Live API Key** | Macaroon HMAC-SHA256 | Clave de API de producción en Vulcan. |
| `av_test_` | **Test API Key** | Macaroon HMAC-SHA256 | Clave de API para staging y pruebas. |
| `eyJ...` | **JWT Token** | RS256 Asimétrico | Token de acceso OIDC emitido por Janus. |
