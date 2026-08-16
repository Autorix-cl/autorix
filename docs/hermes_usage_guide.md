# Guía de Uso para Desarrolladores: Autorix Hermes

**Autorix Hermes** es el puente de autenticación corporativa **SAML 2.0 a OpenID Connect** y servidor de sincronización de directorios **SCIM 2.0 (RFC 7643 / RFC 7644)** del ecosistema Autorix (equivalente a ORY Polis).

---

## 1. Módulos Principales

1. **SAML 2.0 Service Provider (SP)**:
   - Permite que tu empresa integre Single Sign-On (SSO) con **Okta, Microsoft Entra ID (Azure AD), Google Workspace y PingIdentity**.
   - Consume aserciones XML, valida firmas X.509 y traduce atributos corporativos a identidades modernas de Autorix.
2. **SCIM 2.0 Directory Sync Server**:
   - Sincroniza usuarios y grupos automáticamente en tiempo real desde el directorio corporativo.

---

## 2. Endpoints de la API (`http://localhost:4477`)

### Módulo SAML 2.0
* `GET /saml/metadata`: Descarga el XML de metadata del SP para subir a Okta o Azure AD.
* `GET /saml/login?provider={provider_id}`: Inicia el flujo de autenticación SAML redirigiendo al IdP.
* `POST /saml/acs`: Endpoint Assertion Consumer Service que recibe el callback POST con la aserción firmada.
* `POST /admin/saml/providers`: Registra un nuevo Identity Provider corporativo.

### Módulo SCIM 2.0
* `GET /scim/v2/ServiceProviderConfig`: Metadatos y características soportadas por el servidor SCIM.
* `GET /scim/v2/Users`: Listado paginado de usuarios sincronizados.
* `POST /scim/v2/Users`: Creación y aprovisionamiento de un nuevo usuario.

---

## 3. Guía Paso a Paso: Configurar un IdP Corporativo

### 1. Registrar el Proveedor SAML en Hermes

```bash
curl -X POST http://localhost:4477/admin/saml/providers \
  -H "Content-Type: application/json" \
  -d '{
    "id": "okta-corporate",
    "display_name": "Okta Enterprise SSO",
    "idp_entity_id": "http://www.okta.com/exk123456789",
    "idp_sso_url": "https://company.okta.com/app/sso/saml",
    "idp_certificate_pem": "-----BEGIN CERTIFICATE-----\nMIIDpDCCAoygAwIBAgIGAX...\n-----END CERTIFICATE-----",
    "attribute_mapping": {
      "email": "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress",
      "first_name": "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/givenname",
      "last_name": "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/surname"
    }
  }'
```

---

### 2. Obtener la Metadata del Service Provider (para Okta/Azure AD)

```bash
curl http://localhost:4477/saml/metadata
```

Subí este XML en la consola de administración de tu proveedor SAML.

---

### 3. Probar el Aprovisionamiento de Usuarios vía SCIM 2.0

```bash
curl -X POST http://localhost:4477/scim/v2/Users \
  -H "Content-Type: application/scim+json" \
  -d '{
    "schemas": ["urn:ietf:params:scim:schemas:core:2.0:User"],
    "externalId": "okta_user_998877",
    "userName": "margaret.hamilton@nasa.gov",
    "displayName": "Margaret Hamilton",
    "emails": [
      {
        "value": "margaret.hamilton@nasa.gov",
        "type": "work",
        "primary": true
      }
    ],
    "active": true
  }'
```

**Respuesta (`201 Created`):**
```json
{
  "schemas": ["urn:ietf:params:scim:schemas:core:2.0:User"],
  "id": "e47ac10b-58cc-4372-a567-0e02b2c3d479",
  "externalId": "okta_user_998877",
  "userName": "margaret.hamilton@nasa.gov",
  "displayName": "Margaret Hamilton",
  "emails": [
    {
      "value": "margaret.hamilton@nasa.gov",
      "type": "work",
      "primary": true
    }
  ],
  "active": true,
  "meta": {
    "resourceType": "User",
    "created": "2026-08-16T15:00:00Z",
    "location": "http://localhost:4477/scim/v2/Users/e47ac10b-58cc-4372-a567-0e02b2c3d479"
  }
}
```
