# SDKs de Autorix

Autorix ofrece SDKs para Go, TypeScript/React y Python para las API públicas de ejecución. Las API `/admin` y los servicios Argus, Aegis y Hermes no se incluyen en esta versión de los SDKs.

## Matriz de capacidades

| Área | Go | TypeScript / React | Python |
| --- | --- | --- | --- |
| Nexus: comprobaciones, expansión y búsquedas | Sí | Sí | Sí |
| Nexus: gestión pública de tuplas | No | Sí | Sí |
| Themis: evaluación y políticas públicas | Evaluación | Sí | Sí |
| Vulcan: verificación y atenuación | Sí | Sí | Sí |
| Vulcan: ciclo público de claves | No | Sí | Sí |
| Ego: sesión y flujo de identidad | `whoami` | sesión/cierre | registro, sesión y cierre |
| Janus: OAuth/OIDC | discovery, PKCE, token, revocación | Sí | Sí |
| Integraciones | `net/http` | React | FastAPI, Flask, Django |

## Límite navegador y backend

TypeScript puede usar PKCE en un navegador, pero la aplicación debe crear y conservar state, verifier y tokens. No incluya secretos de cliente, API keys ni credenciales privilegiadas en código de navegador. La gestión de claves, tuplas y políticas debe realizarse desde un backend controlado, y toda decisión mostrada en la interfaz debe volver a aplicarse en el backend.

## Fiabilidad

Los SDKs exponen errores tipados y paginación donde la API lo permite. Solo las lecturas idempotentes se reintentan; los intercambios OAuth y las operaciones de escritura no se reintentan automáticamente. Las comprobaciones de permisos fallan cerradas.

- [SDK de Go](/es/sdk/go)
- [SDK de TypeScript y React](/es/sdk/typescript)
- [SDK de Python](/es/sdk/python)
- [CLI y API directa](/es/sdk/cli)
