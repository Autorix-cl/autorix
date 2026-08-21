# Autorix Aegis: Manual de Proxy PEP Zero-Trust

**Autorix Aegis** es un proxy inverso perimetral de aplicación de políticas de acceso (PEP) de alto rendimiento inspirado en **Ory Oathkeeper**. Se sitúa en la frontera de ingreso y evalúa cada petición entrante a través de un **pipeline determinista de 4 etapas** antes de reenviarla a los servicios internos.

---

## 🏛️ 1. Pipeline de Ejecución en 4 Etapas

```text
  [ Petición Inbound HTTP ]
              │
              ▼
   ┌─────────────────────────────────────────────────────────────┐
   │                        Autorix Aegis                        │
   │                                                             │
   │  ┌───────────────────────────────────────────────────────┐  │
   │  │ 1. Matcher: Emparejamiento de URL y Métodos HTTP      │  │
   │  └───────────────────────────┬───────────────────────────┘  │
   │                              │                              │
   │  ┌───────────────────────────▼───────────────────────────┐  │
   │  │ 2. Authenticator: Validación de JWT / Macaroon / Sesión│  │
   │  └───────────────────────────┬───────────────────────────┘  │
   │                              │                              │
   │  ┌───────────────────────────▼───────────────────────────┐  │
   │  │ 3. Authorizer: Verificación ReBAC (Nexus) / ABAC (Themis)│  │
   │  └───────────────────────────┬───────────────────────────┘  │
   │                              │                              │
   │  ┌───────────────────────────▼───────────────────────────┐  │
   │  │ 4. Mutator: Inyección de Cabeceras Sanitizadas        │  │
   │  └───────────────────────────┬───────────────────────────┘  │
   └──────────────────────────────┼──────────────────────────────┘
                                  │
                                  ▼
                     [ Servicio Upstream / Backend ]
```

---

## 📡 2. Referencia de la API de Administración

Aegis expone el puerto del Proxy PEP en `:4455` y la API de Administración en `:4456`.

### 2.1 Crear Regla de Acceso (`POST /rules`)

* **Método**: `POST`
* **Ruta**: `/rules` (en puerto `:4456`)

#### Cuerpo de la Petición
```json
{
  "id": "rule_billing_api_v1",
  "description": "Protege endpoints de facturación requiriendo scope billing:read",
  "match": {
    "url": "http://api.corp.local/api/v1/billing/<.*>",
    "methods": ["GET", "POST"]
  },
  "authenticators": [
    {
      "handler": "jwt",
      "config": {
        "jwks_url": "http://janus:4444/.well-known/jwks.json",
        "required_scope": ["billing:read"]
      }
    }
  ],
  "authorizer": {
    "handler": "remote_json",
    "config": {
      "remote": "http://nexus:8080/check"
    }
  },
  "mutators": [
    {
      "handler": "header",
      "config": {
        "headers": {
          "X-User-ID": "{{ .Subject }}"
        }
      }
    }
  ],
  "upstream": {
    "url": "http://billing-service:8080"
  }
}
```

---

### 2.2 Simulador Dry-Run de Reglas (`POST /rules/test-match`)

Prueba qué regla coincidirá con una petición sin enviar tráfico real:

```json
{
  "url": "http://api.corp.local/api/v1/billing/invoices/99",
  "method": "GET"
}
```
