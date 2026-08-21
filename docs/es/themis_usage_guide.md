# Autorix Themis: Manual de Políticas ABAC y Google CEL

**Autorix Themis** es un motor de políticas de control de acceso basado en atributos (ABAC) de alta velocidad impulsado por **Google Common Expression Language (CEL)**. Proporciona evaluación estricta en memoria, anillos de prioridad configurables, versionado inmutable con rollback y suite automatizada de pruebas de políticas.

---

## 🏛️ 1. Arquitectura y Modelo de Evaluación

```text
  [ Petición de Evaluación ]
              │
              ▼
   ┌─────────────────────────────────────────────────────────────┐
   │                       Autorix Themis                        │
   │                                                             │
   │  ┌───────────────────────────────────────────────────────┐  │
   │  │ Anillo 1: Políticas Críticas de Denegación (Priority 0) │  │
   │  └───────────────────────────┬───────────────────────────┘  │
   │                              │                              │
   │  ┌───────────────────────────▼───────────────────────────┐  │
   │  │ Anillo 2: Políticas de Autorización Estándar (10-100) │  │
   │  └───────────────────────────┬───────────────────────────┘  │
   │                              │                              │
   │  ┌───────────────────────────▼───────────────────────────┐  │
   │  │ Anillo 3: Políticas de Fallback por Defecto (1000+)   │  │
   │  └───────────────────────────────────────────────────────┘  │
   └─────────────────────────────────────────────────────────────┘
```

### 1.1 Variables Disponibles en el Contexto CEL

Cada expresión evaluada en Themis tiene acceso a las siguientes estructuras fuertemente tipadas:

| Variable | Tipo | Descripción |
| :--- | :--- | :--- |
| `request.auth` | `map` | Reclamaciones del token (`claims`), roles, `identity_id`, `mfa_authenticated`. |
| `request.ip` | `string` | Dirección IP de origen del cliente. |
| `request.time` | `timestamp` | Marca de tiempo ISO-8601 del momento de la evaluación. |
| `resource` | `map` | Atributos del recurso objetivo (`owner_id`, `classification`, `amount`). |
| `context` | `map` | Variables del entorno (`is_break_glass`, `geo_country`). |

---

## 📡 2. Referencia de la API REST

Themis se ejecuta en el puerto HTTP `4488` y gRPC `50052`.

### 2.1 Evaluar Políticas (`POST /evaluate`)

* **Método**: `POST`
* **Ruta**: `/evaluate`

#### Cuerpo de la Petición
```json
{
  "resource_type": "bank_account",
  "action": "transfer",
  "request": {
    "auth": {
      "claims": { "role": "financial_manager" },
      "mfa": true
    },
    "ip": "10.240.0.12",
    "time": "2026-08-20T14:30:00Z"
  },
  "resource": {
    "amount": 25000.0,
    "currency": "USD"
  },
  "context": {
    "geo_country": "US"
  }
}
```

#### Respuesta (`200 OK`)
```json
{
  "allowed": true,
  "matched_policy_id": "pol_high_value_transfers",
  "evaluation_time_us": 142
}
```

---

### 2.2 Gestión de Políticas (`POST /policies`)

Crea o actualiza una política CEL con orden de prioridad:

```json
{
  "name": "high_value_wire_transfers",
  "description": "Requiere rol de finanzas y MFA para transferencias superiores a $10,000",
  "priority": 10,
  "effect": "allow",
  "expression": "request.auth.claims.role == 'financial_manager' && request.auth.mfa == true && resource.amount <= 1000000.0"
}
```

---

### 2.3 Interfaz gRPC (`:50052`)

Themis expone `themis.v1.ThemisService`:
- `Evaluate(EvaluateRequest) returns (EvaluateResponse)`
- `ValidatePolicy(ValidatePolicyRequest) returns (ValidatePolicyResponse)`
- `GetPolicy(GetPolicyRequest) returns (GetPolicyResponse)`
- `CreatePolicy(CreatePolicyRequest) returns (CreatePolicyResponse)`
