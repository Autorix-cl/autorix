# Guía de Uso para Desarrolladores: Autorix Nexus

**Autorix Nexus** es un motor de autorización de ultra-baja latencia que combina **ReBAC** (Control de Acceso Basado en Relaciones, inspirado en Google Zanzibar) con **ABAC** (Control de Acceso Basado en Atributos mediante Google CEL) en un único servidor gRPC.

---

## 1. Conceptos Fundamentales

### ¿Cómo piensa Nexus?
Nexus no almacena listas de permisos estáticas (`alice -> can_read`). Almacena **Relaciones**:

$$\text{namespace:object\#relation@subject}$$

| Elemento | Ejemplo | Descripción |
| :--- | :--- | :--- |
| **Namespace** | `document` | Tipo de entidad o recurso. |
| **Object** | `financial_report_2026` | Identificador único del recurso. |
| **Relation** | `viewer`, `editor`, `owner` | Qué relación existe. |
| **Subject** | `user:alice` o `group:finance#member` | Quién tiene la relación (usuario o grupo). |
| **Caveat (ABAC)** | `is_work_hours` | Condición lógica que debe cumplirse en tiempo real. |

---

## 2. Puesta en Marcha Local

### Con Docker Compose (Recomendado)
Levanta PostgreSQL con las migraciones aplicadas y el servidor Nexus:

```bash
docker compose up --build -d
```

Verificá que el servidor gRPC esté escuchando en el puerto `50051`:
```bash
docker compose logs -f nexus
```

### Sin Docker (Desarrollo Local)
1. Asegurate de tener PostgreSQL corriendo y creá la base de datos `autorix_nexus`.
2. Aplicá los scripts en `nexus/migrations/`.
3. Ejecutá:
```bash
cd nexus
export DATABASE_URL="postgres://autorix:autorix_password@localhost:5432/autorix_nexus?sslmode=disable"
make run
```

---

## 3. Pruebas Rápidas con `grpcurl`

Podés interactuar con Nexus directamente desde la terminal usando `grpcurl` (gracias a que gRPC Reflection viene habilitado).

### Listar los métodos disponibles
```bash
grpcurl -plaintext localhost:50051 list autorix.nexus.v1.NexusService
```

### Evaluar un Permiso (`Check`)

#### Caso 1: Chequeo de acceso directo simple
```bash
grpcurl -plaintext -d '{
  "namespace": "document",
  "object": "doc_100",
  "relation": "viewer",
  "subject_id": "alice",
  "subject_namespace": "user"
}' localhost:50051 autorix.nexus.v1.NexusService/Check
```

**Respuesta:**
```json
{
  "allowed": true,
  "reason": "direct match"
}
```

#### Caso 2: Chequeo con Contexto Dinámico (ABAC / Caveats)
Si el recurso requiere validar la IP del usuario o la hora del request, pasá el `request_context`:

```bash
grpcurl -plaintext -d '{
  "namespace": "document",
  "object": "payroll_q1",
  "relation": "viewer",
  "subject_id": "bob",
  "subject_namespace": "user",
  "request_context": {
    "ip": "192.168.1.100",
    "mfa_authenticated": true
  }
}' localhost:50051 autorix.nexus.v1.NexusService/Check
```

---

## 4. Integración en Clientes

### En Go (Golang)

```go
package main

import (
	"context"
	"log"

	pb "github.com/autorix/nexus/api/autorix/nexus/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/structpb"
)

func main() {
	conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewNexusServiceClient(conn)

	// Contexto dinámico para el Caveat (ABAC)
	reqCtx, _ := structpb.NewStruct(map[string]interface{}{
		"ip": "192.168.1.50",
	})

	resp, err := client.Check(context.Background(), &pb.CheckRequest{
		Namespace:        "project",
		Object:           "proj_alpha",
		Relation:         "maintainer",
		SubjectId:        "user_123",
		SubjectNamespace: "user",
		RequestContext:   reqCtx,
	})
	if err != nil {
		log.Fatalf("error checking permission: %v", err)
	}

	if resp.Allowed {
		log.Println("Acceso concedido!")
	} else {
		log.Printf("Acceso denegado: %s\n", resp.Reason)
	}
}
```

### En Node.js / TypeScript

```typescript
import * as grpc from '@grpc/grpc-js';
import * as protoLoader from '@grpc/proto-loader';

const packageDefinition = protoLoader.loadSync('nexus.proto');
const proto = grpc.loadPackageDefinition(packageDefinition) as any;

const client = new proto.autorix.nexus.v1.NexusService(
  'localhost:50051',
  grpc.credentials.createInsecure()
);

client.Check(
  {
    namespace: 'document',
    object: 'doc_100',
    relation: 'editor',
    subject_id: 'alice',
    subject_namespace: 'user',
    request_context: {
      fields: {
        ip: { stringValue: '192.168.1.100' }
      }
    }
  },
  (err: any, response: any) => {
    if (err) console.error(err);
    console.log('Permitido:', response.allowed);
  }
);
```

---

## 5. Modelado de Jerarquías y Grupos (Usersets)

Para modelar que **"Todos los miembros del grupo Ingeniería pueden ver el Documento 1"**:

1. Insertá la tupla de grupo:
   - `namespace`: `group`, `object`: `engineering`, `relation`: `member`, `subject`: `user:alice`
2. Insertá la tupla de permiso apuntando al grupo:
   - `namespace`: `document`, `object`: `doc_1`, `relation`: `viewer`, `subject`: `group:engineering#member`

Cuando consultes si `alice` puede ver `doc_1`, Nexus resolverá la cadena concurrentemente en memoria y devolverá `allowed: true` con `reason: "indirect match"`.

---

## 6. Buenas Prácticas para Producción

1. **Uso de Connection Pools gRPC**: Reutilizá un único `*grpc.ClientConn` compartido en tus servicios clientes (no abras una conexión por petición HTTP).
2. **Timeouts en Context**: Pasá siempre un `context.WithTimeout(ctx, 50*time.Millisecond)` en las llamadas gRPC a Nexus. La evaluación debe resolverse en microsegundos.
3. **Validación Temprana**: Inyectá `Nexus` en tu API Gateway (Autorix Aegis) para validar permisos antes de que el tráfico toque los microservicios de negocio.
