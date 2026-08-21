# Autorix Console: Manual de Administración y Studios

**Autorix Console** es el panel de control web unificado para la suite Autorix. Construido sobre **Next.js 15 App Router**, ofrece interfaces de usuario dedicadas (*Studios*) para administrar cada uno de los motores Zero-Trust con protección CSRF de doble envío y verificación criptográfica en vivo.

---

## 🖥️ 1. Recorrido por los 9 Studios de Gestión

| Studio | Ruta | Motor Asociado | Propósito |
| :--- | :--- | :--- | :--- |
| **Fleet Studio** | `/fleet` | Autorix Argus | Visualización de topología de nodos, estado de salud en vivo y generación de tokens `aet_`. |
| **Identities Studio** | `/identities` | Autorix Ego | Administración de cuentas de usuario, traits dinámicos, sesiones activas y reseteo de MFA. |
| **API Keys Studio** | `/api-keys` | Autorix Vulcan | Emisión de claves `av_live_...`, atenuación interactiva de Macaroons y revocación instantánea. |
| **Permissions Studio**| `/permissions`| Autorix Nexus | Explorador de grafos Zanzibar, simulador de checks y editor de tuplas. |
| **Proxy Rules Studio**| `/proxy-rules`| Autorix Aegis | Editor de reglas perimetrales del proxy PEP y simulador dry-run de peticiones. |
| **Policies Studio** | `/policies` | Autorix Themis | Editor de políticas ABAC en Google CEL, simulación en vivo y rollback de versiones. |
| **Enterprise Studio** | `/enterprise` | Autorix Hermes | Configuración de proveedores SAML 2.0 y monitoreo de sincronización SCIM 2.0. |
| **Audit Studio** | `/audit` | Autorix Argus | Visor de eventos inmutables y botón de verificación de integridad criptográfica SHA-256. |
| **Compliance Studio** | `/compliance` | Autorix Argus | Panel de evidencia continua para controles de auditoría SOC 2 e ISO 27001. |
