# SDK de Python

Paquete: `autorix` · Python 3.9+

El SDK de Python cubre las API públicas de ejecución de Ego, Janus, Nexus, Themis y Vulcan. Incluye resultados tipados, `AutorixHTTPError`, páginas con cursor e integraciones para FastAPI, Flask y Django.

## Instalación

```bash
pip install autorix
pip install 'autorix[fastapi]' 'autorix[flask]' 'autorix[django]'
```

## Inicio rápido

```python
from autorix import AutorixClient

with AutorixClient(nexus_url="http://localhost:8080") as client:
    decision = client.check("document", "42", "viewer", "user-7")
    if not decision.allowed:
        raise PermissionError(decision.reason)
```

## Cobertura pública de ejecución

- **Nexus:** comprobaciones, lotes, lectura/escritura/eliminación de tuplas, expansión y búsquedas.
- **Themis:** evaluación, CRUD de políticas, validación, dry runs, versiones, fixtures y suites de prueba.
- **Vulcan:** creación, listado y revocación de claves; verificación y atenuación de macaroons. La verificación espera la estructura de macaroon del servidor; las cadenas no compatibles se deniegan localmente.
- **Ego:** registro, inicio de sesión, `whoami` y cierre de sesión.
- **Janus:** discovery/JWKS OIDC, URL de autorización, intercambio de código, refresh, client credentials, introspección y revocación.

`Page` contiene `data`, `next_cursor` y `has_more`. Las API administrativas y Argus, Aegis y Hermes no forman parte de este SDK.

## Seguridad OAuth

`authorization_url` requiere state y challenge PKCE proporcionados por la aplicación. El SDK no genera ni persiste tokens. `exchange_code` y `refresh_token` no se reintentan porque sus credenciales pueden ser de un solo uso. Un secreto de cliente solo debe utilizarse en un backend controlado, nunca en el navegador.

## Integraciones de framework

`AutorixSecurity` ofrece dependencias para FastAPI. `autorix.flask.require_permission` y `autorix.django.require_permission` ofrecen decoradores de rutas. Consumen cabeceras de identidad confiables; úselos únicamente detrás de un proxy que elimine cabeceras `X-User-*` enviadas por clientes. El backend sigue siendo el punto de aplicación de autorización.

## Fiabilidad

Las solicitudes respetan el timeout configurado y devuelven errores HTTP tipados. Solo se reintentan lecturas idempotentes. Las comprobaciones de permisos fallan de forma cerrada: ante un error devuelven `allowed=False`.
