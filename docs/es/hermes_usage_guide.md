# Autorix Hermes: Manual de SAML 2.0 y SCIM 2.0

**Autorix Hermes** es el motor de integración de identidad empresarial de Autorix. Proporciona federación Web SSO mediante **SAML 2.0** (proveedores como Okta, Azure Active Directory, Google Workspace) y sincronización continua de cuentas mediante el estándar **SCIM 2.0 (RFC 7643 / RFC 7644)**.

---

## 📡 1. Referencia de la API REST

Hermes se ejecuta en el puerto HTTP `4477`.

### 1.1 Metadatos SAML 2.0 SP (`GET /saml/metadata`)

Descarga el XML de metadatos del Service Provider (SP) para importar en Okta / Azure AD.

---

### 1.2 Aprovisionamiento SCIM 2.0 (`/scim/v2/Users`)

Hermes implementa la especificación completa RFC 7643/7644:
- `GET /scim/v2/Users`: Lista usuarios con filtros y paginación.
- `POST /scim/v2/Users`: Crea un nuevo usuario aprovisionado por el IdP empresarial.
- `PATCH /scim/v2/Users/{id}`: Actualiza o desactiva usuarios en tiempo real.
- `GET /scim/v2/Groups`: Lista grupos empresariales sincronizados.
