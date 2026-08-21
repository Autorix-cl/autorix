# Autorix Janus: Manual de Servidor OAuth 2.0 y OpenID Connect

**Autorix Janus** es un servidor de autorización OpenID Connect (OIDC) y OAuth 2.0 de alto rendimiento inspirado en **Ory Hydra**. Implementa emisión de tokens asimétricos con rotación automática de claves **RS256 JWKS**, validación estricta de **PKCE RFC 7636** y arquitectura de autorización desacoplada mediante **retos de Login y Consentimiento**.

---

## 🏛️ 1. Flujo de Autorización Desacoplado (Paridad Ory Hydra)

Janus está completamente desacoplado de la interfaz de usuario de inicio de sesión:

```text
  [ Navegador ] ────(1. Inicia Auth Code)────► [ Autorix Janus :4444 ]
        ▲                                                │
        │                                                ▼ (2. 302 Redirect con login_challenge)
        ├────────────────────────────────────── [ App de Login / Ego ]
        │                                                │
        │                                                ▼ (3. Valida credenciales y llama)
        │                                        [ PUT /admin/.../login/accept ]
        │                                                │
        ▼ (4. Regresa con login_verifier)               ▼
  [ Autorix Janus ] ────(5. 302 Redirect con consent_challenge)──► [ App de Consentimiento ]
```

---

## 📡 2. Referencia de la API REST

Janus opera en el puerto HTTP `4444`.

### 2.1 Descubrimiento OIDC (`GET /.well-known/openid-configuration`)

Devuelve los metadatos de configuración del servidor OIDC, endpoints públicos y algoritmos de firma soportados.

---

### 2.2 Endpoint de Autorización (`GET /oauth2/auth`)

Inicia el flujo de código de autorización. Redirige al cliente con un `login_challenge`:

```text
HTTP/1.1 302 Found
Location: http://localhost:3000/login?login_challenge=c1a2b3d4-e5f6...
```

#### Aceptar Solicitud de Login (`PUT /admin/oauth2/auth/requests/login/accept`)
Llamado por la interfaz de login tras verificar al usuario:

```json
{
  "challenge": "c1a2b3d4-e5f6...",
  "subject": "usr_alice_smith"
}
```

##### Respuesta (`200 OK`)
```json
{
  "redirect_to": "http://localhost:4444/oauth2/auth?login_challenge=...&login_verifier=..."
}
```

#### Aceptar Solicitud de Consentimiento (`PUT /admin/oauth2/auth/requests/consent/accept`)
Llamado por la interfaz de consentimiento para autorizar scopes:

```json
{
  "challenge": "cc_1a2b3c4d...",
  "granted_scopes": ["openid", "profile", "email"]
}
```

---

### 2.3 Emisión de Tokens (`POST /oauth2/token`)

* **Método**: `POST`
* **Ruta**: `/oauth2/token`
* **Cabeceras**: `Content-Type: application/x-www-form-urlencoded`

#### Flujo Machine-to-Machine (Client Credentials)
```bash
curl -X POST http://localhost:4444/oauth2/token \
  -u "client_backend_billing:SecretKey#2026" \
  -d "grant_type=client_credentials&scope=billing:read billing:write"
```

##### Respuesta (`200 OK`)
```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIsImtpZCI6ImtleV8yMDI2X3EzIi...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "scope": "billing:read billing:write"
}
```

---

### 2.4 Introspección de Tokens RFC 7662 (`POST /oauth2/introspect`)

```bash
curl -X POST http://localhost:4444/oauth2/introspect \
  -d "token=eyJhbGciOiJSUzI1NiIsIn..."
```
