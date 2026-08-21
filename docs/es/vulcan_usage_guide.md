# Autorix Vulcan: Manual de API Keys y Atenuación de Macaroons

**Autorix Vulcan** es un motor empresarial de emisión de claves de API y tokens de capacidad descentralizados inspirado en **Google Macaroons** y **Ory Talos**. Emite API keys con prefijos reconocibles para análisis de seguridad y permite **atenuación offline de caveats** mediante firmas criptográficas encadenadas con HMAC-SHA256.

---

## 🏛️ 1. Criptografía de Macaroons y Atenuación Descentralizada

* **Atenuación sin Base de Datos**: Un servicio que recibe una API key raíz puede restringirla agregando restricciones (*caveats* como `time_before = 2026-08-21` o `method = GET`) **sin realizar llamadas de red a Vulcan**.
* **Protección contra Manipulaciones**: Cada caveat actualiza la firma HMAC de forma irreversible:
  ```text
  Sig_0   = HMAC-SHA256(K_root, KeyID)
  Sig_i+1 = HMAC-SHA256(Sig_i, Caveat_i+1)
  ```

---

## 🔑 2. Prefijos de Entorno

| Prefijo | Entorno | Propósito |
| :--- | :--- | :--- |
| `av_live_` | Producción | Acceso a datos reales de clientes. |
| `av_test_` | Staging / Pruebas | Datos simulados para desarrollo seguro. |

---

## 📡 3. Referencia de la API REST

Vulcan se ejecuta en el puerto `4466`.

### 3.1 Crear Clave de API (`POST /keys`)

* **Método**: `POST`
* **Ruta**: `/keys`

#### Cuerpo de la Petición
```json
{
  "name": "Pipeline de Ingesta ETL",
  "description": "Agregador diario de métricas",
  "owner_id": "svc_etl_runner",
  "scopes": ["ingest:write", "metrics:read"],
  "is_live": true,
  "expires_at": "2026-12-31T23:59:59Z"
}
```

#### Respuesta (`201 Created`)
```json
{
  "api_key": {
    "id": "c1a2b3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d",
    "key_prefix": "av_live",
    "key_hint": "9f8e",
    "name": "Pipeline de Ingesta ETL",
    "scopes": ["ingest:write", "metrics:read"],
    "state": "active"
  },
  "raw_token": "av_live_9f8e7d6c5b4a3f2e1d0c9b8a7f6e5d4c3b2a10e9f8a7b6c5d4e3f2a1b0c9"
}
```

---

### 3.2 Verificar Macaroon (`POST /keys/verify`)

Evalúa la cadena de firmas y los caveats frente al contexto en tiempo de ejecución:

```json
{
  "macaroon": {
    "key_id": "c1a2b3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d",
    "caveats": [{ "clause": "time_before = 2026-08-21T00:00:00Z" }],
    "signature": "b5e9f02d8c4e3f1a..."
  },
  "context": {
    "now": "2026-08-20T14:30:00Z",
    "ip_address": "10.0.4.15"
  }
}
```
