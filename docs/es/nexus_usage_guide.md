# Autorix Nexus: Manual de Uso de ReBAC Zanzibar y APL

**Autorix Nexus** es un motor de control de acceso basado en relaciones (ReBAC) de alto rendimiento inspirado en **Google Zanzibar** y **Ory Keto**. Proporciona verificación de permisos (`Check`) en submilisegundos, recorrido de grafos con detección de ciclos, resolución inversa de permisos y caveats dinámicos evaluados mediante **Google Common Expression Language (CEL)**.

---

## 🏛️ 1. Arquitectura del Modelo de Datos Zanzibar

A diferencia de los sistemas RBAC tradicionales que asignan roles planos a usuarios (`alice -> role:admin`), Nexus representa los permisos como un **grafo dirigido de relaciones**:

```text
namespace:object#relation@subject_namespace:subject_id[#subject_relation] [with caveat]
```

### 1.1 Estructura de Tuplas Zanzibar

Una tupla de relación en Nexus consta de:

| Campo | Tipo | Ejemplo | Descripción |
| :--- | :--- | :--- | :--- |
| `namespace` | `string` | `documents` | El tipo de recurso o dominio del objeto. |
| `object` | `string` | `finances_2026` | El identificador único de la instancia del recurso. |
| `relation` | `string` | `editor` | La relación directa que se otorga. |
| `subject_namespace` | `string` | `user` o `group` | El espacio de nombres del sujeto. |
| `subject_id` | `string` | `alice` o `eng_team` | El identificador único del sujeto. |
| `subject_relation` | `string` (opcional) | `member` | Permite referenciar usuarios pertenecientes a un conjunto (*userset rewrite*). |
| `caveat_name` | `string` (opcional) | `ip_allowlist` | Nombre de la regla CEL registrada que debe cumplirse en tiempo de ejecución. |
| `caveat_context` | `jsonb` (opcional) | `{"allowed_ips": ["10.0.0.0/24"]}` | Parámetros estáticos pasados a la regla CEL. |

---

## ⚡ 2. Evaluación de Permisos y Recorrido de Grafos

### 2.1 Algoritmo de Verificación (`POST /check`)

Cuando se solicita `POST /check`:
1. **Búsqueda Directa en Caché/BD**: Nexus busca tuplas directas que coincidan con `(namespace, object, relation, subject)`.
2. **Evaluación de Reescribir Userset (Userset Rewrites)**: Si el esquema de namespace define relaciones indirectas (por ejemplo: `viewer = editor || owner`), Nexus expande el grafo.
3. **Tuplas a Userset (Tuple to Userset)**: Si un objeto apunta a un padre (`document:123#parent@folder:456`), Nexus evalúa los permisos heredados en el padre.
4. **Detección de Ciclos y Límite de Profundidad**: Para evitar bucles infinitos en grafos circulares, Nexus mantiene un registro de nodos visitados y un límite estricto de recursión (`max_depth = 32`).

---

## 📜 3. Autorix Permission Language (APL)

Nexus incluye un compilador nativo de **Autorix Permission Language (APL)**, otorgando paridad con Ory Keto OPL. Permite modelar relaciones y permisos mediante clases estilo TypeScript en lugar de escribir esquemas JSON manualmente:

```typescript
import { Namespace, Context } from "@autorix/nexus-types"

class Document implements Namespace {
  related: {
    owner: User[]
    editor: User[]
    viewer: User[]
    parent: Folder[]
  }

  permits = {
    view: (ctx: Context): boolean =>
      this.related.viewer.includes(ctx.subject) ||
      this.related.editor.includes(ctx.subject) ||
      this.related.owner.includes(ctx.subject) ||
      this.related.parent.traverse((p) => p.related.view),

    edit: (ctx: Context): boolean =>
      this.related.editor.includes(ctx.subject) ||
      this.related.owner.includes(ctx.subject),

    owner: (ctx: Context): boolean =>
      this.related.owner.includes(ctx.subject)
  }
}
```

---

## 📡 4. Referencia Completa de la API REST

Nexus se ejecuta en el puerto HTTP `8080` y gRPC `50051`.

### 4.1 Verificación de Permisos (`POST /check`)

* **Método**: `POST`
* **Ruta**: `/check`
* **Cabeceras**: `Content-Type: application/json`

#### Cuerpo de la Petición
```json
{
  "namespace": "documents",
  "object": "report_q3",
  "relation": "viewer",
  "subject_namespace": "user",
  "subject_id": "alice",
  "context": {
    "ip": "10.0.4.15",
    "time": "2026-08-20T14:30:00Z"
  }
}
```

#### Respuesta (`200 OK`)
```json
{
  "allowed": true
}
```

---

### 4.2 Crear Tuplas de Relación (`POST /tuples`)

* **Método**: `POST`
* **Ruta**: `/tuples`

#### Cuerpo de la Petición
```json
{
  "tuples": [
    {
      "namespace": "documents",
      "object": "roadmap_2026",
      "relation": "editor",
      "subject_namespace": "user",
      "subject_id": "bob"
    }
  ]
}
```

#### Respuesta (`201 Created`)
```json
[
  {
    "namespace": "documents",
    "object": "roadmap_2026",
    "relation": "editor",
    "subject_namespace": "user",
    "subject_id": "bob"
  }
]
```

---

### 4.3 Expansión de Árbol de Permisos (`POST /expand`)

* **Método**: `POST`
* **Ruta**: `/expand`

#### Cuerpo de la Petición
```json
{
  "namespace": "documents",
  "object": "roadmap_2026",
  "relation": "viewer"
}
```

---

### 4.4 Búsqueda Inversa de Recursos (`POST /lookup/resources`)

Encuentra todos los recursos a los que un sujeto tiene acceso:

* **Método**: `POST`
* **Ruta**: `/lookup/resources`

```json
{
  "namespace": "documents",
  "relation": "viewer",
  "subject_namespace": "user",
  "subject_id": "alice"
}
```

#### Respuesta (`200 OK`)
```json
{
  "resource_ids": ["doc_1", "doc_8", "roadmap_2026"]
}
```

---

### 4.5 Interfaz gRPC (`:50051`)

Nexus implementa el servicio protobuf `nexus.v1.NexusService`:
- `Check(CheckRequest) returns (CheckResponse)`
- `Expand(ExpandRequest) returns (ExpandResponse)`
- `LookupSubjects(LookupSubjectsRequest) returns (LookupSubjectsResponse)`
- `LookupResources(LookupResourcesRequest) returns (LookupResourcesResponse)`
