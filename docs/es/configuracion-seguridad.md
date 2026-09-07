# Configuración de seguridad

Estos controles son una base de integración, no una certificación ni un modelo
de amenazas completo.

## Confianza JWT en Aegis

Configurá exactamente una fuente de claves confiable: `JWT_JWKS_URL` o
`JWT_PUBLIC_KEY_FILE`. Definí `JWT_ISSUER` y `JWT_AUDIENCE` con los valores
exactos que acepta tu API. En producción, JWKS debe usar HTTPS;
`JWT_ALLOW_INSECURE_JWKS=true` es solo para Compose local.

Aegis acepta únicamente access tokens. El operador controla issuer, audience,
recurso y JWKS; una petición o regla no puede elegir la URL de JWKS.

## Administración y Console

- OAuth/OIDC público de Janus permanece en `:4444`; `/admin/*` usa el listener
  privado.
- No publiques `:4456` de Aegis ni habilites CORS amplio.
- `JANUS_ADMIN_INTERNAL_URL` es solo del BFF del servidor; nunca lo expongas
  como `NEXT_PUBLIC_*`.
- Registrá URIs de redirección, grants, scopes y audiencias exactas por cliente;
  exigí PKCE S256 para authorization code.

Usá un gestor de secretos, identidad de workload, TLS a PostgreSQL e imágenes
productivas fijadas por digest OCI verificado. Consultá la
[English security-boundary migration guide](/security_boundary_migration).
