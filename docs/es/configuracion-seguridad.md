# Configuración de seguridad

Estos controles son una base de integración, no una certificación ni un modelo
de amenazas completo.

## Confianza JWT en Aegis

Debe configurarse exactamente una fuente de claves confiable: `JWT_JWKS_URL` o
`JWT_PUBLIC_KEY_FILE`. `JWT_ISSUER` y `JWT_AUDIENCE` deben contener los valores
exactos aceptados por la API. En producción, JWKS debe usar HTTPS;
`JWT_ALLOW_INSECURE_JWKS=true` es solo para Compose local.

Aegis acepta únicamente access tokens. El operador controla issuer, audience,
recurso y JWKS; una petición o regla no puede elegir la URL de JWKS.

## Administración y Console

- OAuth/OIDC público de Janus permanece en `:4444`; `/admin/*` usa el listener
  privado.
- El puerto `:4456` de Aegis no debe publicarse ni debe habilitarse CORS amplio.
- `JANUS_ADMIN_INTERNAL_URL` es solo del BFF del servidor; nunca debe exponerse
  como `NEXT_PUBLIC_*`.
- Deben registrarse URIs de redirección, grants, scopes y audiencias exactas por
  cliente; el flujo authorization code debe exigir PKCE S256.

El despliegue debe usar un gestor de secretos, identidad de workload, TLS a
PostgreSQL e imágenes productivas fijadas por digest OCI verificado. Consulte
la [guía de migración de límites de seguridad](./migracion-limites-seguridad.md).
