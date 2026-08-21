# Autorix Ego: Manual de Identidad, Traits y Sesiones

**Autorix Ego** es el motor central de gestión de identidades y usuarios de Autorix, inspirado en **Ory Kratos**. Administra el ciclo de vida completo del usuario, hashing de contraseñas de alta seguridad con **Argon2id**, autenticación de dos factores **TOTP RFC 6238**, biometría con **WebAuthn / Passkeys**, y un motor desacoplado de formularios vía **Flow State-Machine**.

---

## 🏛️ 1. Parámetros Criptográficos y Seguridad

* **Argon2id**:
  - Memoria: `64 MB` ($m=65536$)
  - Iteraciones en el tiempo: `3` ($t=3$)
  - Hilos paralelos: `4` ($p=4$)
  - Sal: 16 bytes generados con `crypto/rand`
* **MFA TOTP**: Cumplimiento estricto con **RFC 6238** (SHA-1, 6 dígitos, paso de 30 segundos, tolerancia de deriva de reloj de $\pm 1$ ventana).
* **WebAuthn / FIDO2**: Registro y desafío biométrico de hardware mediante claves públicas de seguridad.

---

## 📡 2. Referencia de la API REST

Ego se ejecuta en el puerto HTTP `4433`.

### 2.1 State-Machine Flow API (Paridad Ory Kratos)

#### Iniciar Flujo de Registro (`GET /self-service/registration/browser`)
Inicializa un nuevo flujo de registro generando un `flow_id`, token anti-CSRF y los nodos dinámicos para renderizar en el cliente:

```bash
curl -X GET http://localhost:4433/self-service/registration/browser
```

##### Respuesta (`200 OK`)
```json
{
  "id": "f8a1b2c3-d4e5-6789-0123-456789abcdef",
  "type": "registration",
  "state": "choose_method",
  "csrf_token": "csrf_9f8e7d6c...",
  "expires_at": "2026-08-20T22:15:00Z",
  "ui_nodes": [
    { "type": "input", "group": "password", "attributes": { "name": "traits.email", "type": "email" } },
    { "type": "input", "group": "password", "attributes": { "name": "password", "type": "password" } },
    { "type": "input", "group": "webauthn", "attributes": { "name": "webauthn_register", "type": "button" } }
  ]
}
```

---

### 2.2 Registro de Usuario (`POST /self-service/registration`)

* **Método**: `POST`
* **Ruta**: `/self-service/registration?flow={flow_id}`

#### Cuerpo de la Petición
```json
{
  "method": "password",
  "csrf_token": "csrf_9f8e7d6c...",
  "traits": {
    "email": "alice@enterprise.corp",
    "name": "Alice Smith",
    "department": "Engineering"
  },
  "password": "SuperSecurePassword#2026"
}
```

#### Respuesta (`201 Created`)
```json
{
  "identity": {
    "id": "c1a2b3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d",
    "schema_id": "default",
    "state": "active",
    "traits": {
      "email": "alice@enterprise.corp",
      "name": "Alice Smith"
    },
    "created_at": "2026-08-20T08:00:00Z"
  },
  "session": {
    "id": "s1a2b3c4-d5e6-7f8a-9b0c-1d2e3f4a5b6c",
    "token": "ast_9f8e7d6c5b4a...",
    "expires_at": "2026-08-27T08:00:00Z"
  }
}
```

---

### 2.3 Inicio de Sesión (`POST /self-service/login`)

* **Método**: `POST`
* **Ruta**: `/self-service/login`

```json
{
  "identifier": "alice@enterprise.corp",
  "password": "SuperSecurePassword#2026"
}
```

---

### 2.4 Validar Sesión Activa (`GET /sessions/whoami`)

* **Método**: `GET`
* **Ruta**: `/sessions/whoami`
* **Cabecera**: `Authorization: Bearer ast_...` o Cookie `autorix_session_token`.
