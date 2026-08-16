# Guía de Uso para Desarrolladores: Autorix Ego

**Autorix Ego** es el motor de gestión de identidades y ciclo de vida de usuarios del ecosistema Autorix (equivalente a ORY Kratos).

---

## 1. Características Principales

* **Seguridad Criptográfica**: Hashing de contraseñas con **Argon2id** (resistente a GPU cracking y fuerza bruta).
* **Perfiles Dinámicos**: Almacenamiento de `traits` en JSONB validados contra `default.identity.schema.json`.
* **Sesiones Seguras**: Generación de tokens de 256 bits de entropía con hashing SHA-256 en base de datos.
* **REST API Headless**: Endpoints listos para ser consumidos por cualquier frontend (Next.js, Vite, Mobile).

---

## 2. Endpoints de la API (`http://localhost:4433`)

### 1. Registro de Usuario (`POST /self-service/registration`)
Crea una nueva identidad con contraseña y devuelve la sesión inmediatamente.

**Request:**
```bash
curl -X POST http://localhost:4433/self-service/registration \
  -H "Content-Type: application/json" \
  -d '{
    "traits": {
      "email": "developer@autorix.io",
      "name": {
        "first": "Alan",
        "last": "Turing"
      }
    },
    "password": "SuperSecretPassword123!"
  }'
```

**Response (`201 Created`):**
```json
{
  "identity": {
    "id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
    "schema_id": "default",
    "traits": {
      "email": "developer@autorix.io",
      "name": { "first": "Alan", "last": "Turing" }
    },
    "state": "active"
  },
  "session": {
    "id": "e3b0c442-98fc-1c14-9afb-4c7fa43f29b4",
    "token": "4a7b9c...", 
    "expires_at": "2026-09-15T13:45:00Z"
  }
}
```
*(Nota: El endpoint también inyecta la cookie HTTP-Only `autorix_session_token`).*

---

### 2. Inicio de Sesión (`POST /self-service/login`)

**Request:**
```bash
curl -X POST http://localhost:4433/self-service/login \
  -H "Content-Type: application/json" \
  -d '{
    "identifier": "developer@autorix.io",
    "password": "SuperSecretPassword123!"
  }'
```

**Response (`200 OK`):**
Devuelve la sesión activa y renueva la cookie `autorix_session_token`.

---

### 3. Verificar Sesión Activa (`GET /sessions/whoami`)

Podés autenticarte pasando el header `Authorization: Bearer <token>` o enviando la Cookie:

```bash
curl http://localhost:4433/sessions/whoami \
  -H "Authorization: Bearer <tu_session_token>"
```

**Response (`200 OK`):**
```json
{
  "id": "e3b0c442-98fc-1c14-9afb-4c7fa43f29b4",
  "identity_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
  "identity": {
    "id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
    "traits": {
      "email": "developer@autorix.io",
      "name": { "first": "Alan", "last": "Turing" }
    }
  },
  "expires_at": "2026-09-15T13:45:00Z"
}
```

---

### 4. Cerrar Sesión (`POST /self-service/logout`)
Invalida el token en la base de datos y borra la cookie del navegador.

```bash
curl -X POST http://localhost:4433/self-service/logout \
  -H "Authorization: Bearer <tu_session_token>"
```

---

## 3. Puesta en Marcha

### Con Docker Compose
```bash
docker compose up --build
```
Levanta tanto **Nexus** (puerto `50051`) como **Ego** (puerto `4433`) con PostgreSQL.
