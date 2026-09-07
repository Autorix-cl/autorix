# CLI e integración HTTP/gRPC directa

`autorixctl` es un CLI pequeño de plano de control ubicado en `cmd/autorixctl`. Para lenguajes sin SDK compatible, use los contratos REST/gRPC del servicio directamente y gestione autenticación, tiempos de espera, reintentos y errores en su propio cliente.

## Compilar desde fuente

```bash
go build -o autorixctl ./cmd/autorixctl
```

## Comandos CLI implementados

```bash
# Evaluar una comprobación de relación Nexus.
autorixctl check --ns documents --obj document-42 --rel viewer --subj user-7

# Crear un token de inscripción Argus.
autorixctl token mint --engine nexus --env production --description "nexus worker"

# Consultar o verificar la auditoría Argus.
autorixctl audit list
autorixctl audit verify
```

Los nombres de comando y flags anteriores conforman la interfaz actual. `tokens mint` (plural), `--namespace`, `--object`, `--relation`, `--subject` y `audit export` no están implementados.

## REST directo

Una comprobación Nexus directa envía JSON a `POST /check` en la URL Nexus:

```bash
curl --fail-with-body \
  --request POST http://localhost:8080/check \
  --header 'Content-Type: application/json' \
  --data '{
    "namespace": "documents",
    "object": "document-42",
    "relation": "viewer",
    "subject_id": "user-7",
    "subject_namespace": "user"
  }'
```

No interprete un fallo de red o HTTP como autorización. Deniegue la solicitud protegida y registre el fallo.

## gRPC directo

Las definiciones Protobuf están disponibles en `api/autorix/<service>/v1/`. Genere clientes con su propia cadena compatible de `protoc` o Buf después de elegir el contrato de servicio necesario. Autorix no distribuye ni mantiene actualmente SDKs generados para Rust, .NET, Java, PHP, Ruby u otros lenguajes.
