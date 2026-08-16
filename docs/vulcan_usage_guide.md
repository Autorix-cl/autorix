# Guía de Uso para Desarrolladores: Autorix Vulcan

**Autorix Vulcan** es el motor especializado en API Keys seguras y **Tokens de Capacidad Macaroons** con atenuación criptográfica descentralizada (equivalente a ORY Talos).

---

## 1. ¿Por qué Macaroons en lugar de API Keys estáticas?

Con una API Key tradicional, si le das una llave a un script externo o microservicio, tiene acceso total y permanente a todos los permisos.

Con **Macaroons de Autorix Vulcan**:
1. Tu backend emite un Macaroon base con permisos amplios.
2. Tu cliente/script puede **atenuar (restringir)** el Macaroon localmente agregando *Caveats* (condiciones):
   - `time_before = 2026-08-17T12:00:00Z` (expira en 1 hora)
   - `ip = 192.168.1.50` (solo usable desde esa IP)
   - `method = GET` (solo lectura)
   - `path_prefix = /api/v1/reports`
3. Cada caveat encadena una nueva firma **HMAC-SHA256**.
4. **Nadie puede remover o relajar un caveat** sin destruir la firma matemática.

---

## 2. Endpoints de la API (`http://localhost:4466`)

### 1. Crear una API Key y Macaroon Base (`POST /keys`)

```bash
curl -X POST http://localhost:4466/keys \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Integration Service Worker",
    "owner_id": "org_98765",
    "scopes": ["invoices:read", "invoices:write"],
    "is_live": true
  }'
```

**Respuesta (`201 Created`):**
```json
{
  "api_key": {
    "id": "c7a807d9-2fb0-438d-b0aa-5c4d29f8f2b7",
    "key_prefix": "av_live",
    "key_hint": "a4f9",
    "name": "Integration Service Worker",
    "owner_id": "org_98765",
    "state": "active"
  },
  "raw_token": "av_live_5d9c28e938bf82384f938472938472938472938472938472938472938472a4f9",
  "macaroon": {
    "location": "https://api.autorix.io",
    "key_id": "c7a807d9-2fb0-438d-b0aa-5c4d29f8f2b7",
    "caveats": [],
    "signature": "8f3b2a..."
  }
}
```

---

### 2. Atenuar un Macaroon Localmente (`POST /keys/attenuate`)

Podés agregar restricciones encadenadas:

```bash
curl -X POST http://localhost:4466/keys/attenuate \
  -H "Content-Type: application/json" \
  -d '{
    "macaroon": {
      "location": "https://api.autorix.io",
      "key_id": "c7a807d9-2fb0-438d-b0aa-5c4d29f8f2b7",
      "caveats": [],
      "signature": "8f3b2a..."
    },
    "caveat": "time_before = 2026-08-17T18:00:00Z"
  }'
```

**Respuesta:**
Devuelve el nuevo Macaroon con el caveat inyectado y la nueva firma HMAC recalculada.

---

### 3. Verificar un Macaroon (`POST /keys/verify`)

El API Gateway o tus microservicios validan el Macaroon contra el contexto del request entrante:

```bash
curl -X POST http://localhost:4466/keys/verify \
  -H "Content-Type: application/json" \
  -d '{
    "macaroon": {
      "location": "https://api.autorix.io",
      "key_id": "c7a807d9-2fb0-438d-b0aa-5c4d29f8f2b7",
      "caveats": [
        {"predicate": "time_before = 2026-08-17T18:00:00Z"},
        {"predicate": "ip = 192.168.1.100"}
      ],
      "signature": "e4f81c..."
    },
    "context": {
      "ip_address": "192.168.1.100",
      "method": "GET",
      "path": "/api/v1/invoices"
    }
  }'
```

**Respuesta (`200 OK`):**
```json
{
  "valid": true,
  "api_key": {
    "id": "c7a807d9-2fb0-438d-b0aa-5c4d29f8f2b7",
    "name": "Integration Service Worker",
    "state": "active"
  }
}
```

---

### 4. Revocar una API Key (`DELETE /keys/{id}`)

```bash
curl -X DELETE http://localhost:4466/keys/c7a807d9-2fb0-438d-b0aa-5c4d29f8f2b7
```
Invalida la llave maestra en la base de datos `autorix_vulcan`, haciendo que todos sus Macaroons derivados fallen la verificación inmediatamente.
