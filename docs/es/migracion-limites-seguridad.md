# Migrar la confianza JWT y la administración privada

Este endurecimiento elimina la aceptación de JWT sin firma y separa la administración de Janus de su listener público. **No** es una certificación bancaria ni una evaluación integral de seguridad. Actualice en conjunto los motores, Console y el despliegue: los clientes que usen rutas públicas antiguas `/admin/*` dejarán de funcionar.

## Ruta rápida

1. Mantenga OAuth/OIDC público de Janus en `:4444`. Envíe las solicitudes de administración y de aceptación de login/consentimiento al listener privado `:4445`. Janus independiente vincula la administración a `127.0.0.1` por defecto; los contenedores usan explícitamente `ADMIN_HOST=0.0.0.0`.
2. Configure emisor, audiencia y JWKS confiables de Aegis con las variables siguientes. Un `jwks_url` provisto por una regla no establece confianza.
3. Configure `JANUS_ADMIN_INTERNAL_URL` de Console por separado de `JANUS_INTERNAL_URL`. Solo código del servidor puede usar la dirección administrativa.
4. Ejecute `python3 scripts/ci/check_security_boundaries.py` y las pruebas de motores/Console. Desde fuera del despliegue, confirme que `/admin/clients` devuelve `404` en el listener público de Janus.

## Configuración de confianza

| Variable | Propósito |
| --- | --- |
| `JWT_ISSUER` | `iss` exacto esperado; debe coincidir con `ISSUER_URL` de Janus. |
| `JWT_AUDIENCE` | `aud` exacto esperado; los despliegues provistos lo alinean con el emisor para tokens M2M actuales de Janus. |
| `JWT_JWKS_URL` | Fuente JWKS controlada por el operador; nunca se obtiene desde el token o la solicitud. |
| `JWT_ALLOW_INSECURE_JWKS` | Su valor predeterminado es `false`; Compose permite HTTP solo para desarrollo local. |
| `JANUS_INTERNAL_URL` | Backend de Console para JWKS, introspección y revocación públicas. |
| `JANUS_ADMIN_INTERNAL_URL` | Backend de Console para `/admin/*`; nunca debe ser `NEXT_PUBLIC_*`. |

Helm usa el endpoint JWKS HTTPS del emisor. Aegis debe confiar en su certificado y el ingress debe ser alcanzable desde el pod de Aegis. Compose usa JWKS HTTP interno **solo** para desarrollo local; no traslade ese transporte al límite de confianza de producción.

Los tokens de acceso de `client_credentials` usan el emisor como audiencia, mientras que los tokens de acceso de código de autorización y los ID tokens usan el ID del cliente. Este despliegue acepta solamente la audiencia M2M. No acepte tokens de navegador hasta diseñar y probar audiencias de recursos y la separación entre access token e ID token: verificar firma no evita confusión de tipo de token.

## Vinculación del cliente OAuth

- `client_credentials` solo se acepta para clientes confidenciales que registraron explícitamente ese grant. Cada scope solicitado debe estar permitido para el cliente.
- Las autorizaciones exigen `response_type=code`, `redirect_uri` registrado de forma exacta, scopes autorizados y PKCE `S256`.
- El intercambio de código exige el mismo cliente autenticado y el `redirect_uri` exacto. El código se consume solo después de validar esas vinculaciones.

## Administración privada

- Helm expone un Service `ClusterIP` administrativo separado para Janus y nunca lo referencia desde Ingress. Compose no publica Janus `:4445` ni Aegis `:4456`.
- Console valida sesiones con Argus antes de reenviar solicitudes. Janus requiere `oauth2:read`/`oauth2:write`; Aegis requiere `proxy-rules:read`/`proxy-rules:write`. Un operador o auditor no puede mutar estos motores mediante el BFF.
- La aceptación de login/consentimiento pertenece a un backend confiable después de autenticar al usuario. Nunca permita que un `subject` enviado por el navegador establezca identidad.
- `ClusterIP` y un puerto Docker no publicado no autentican cargas. Antes de producción, aplique firewall/NetworkPolicy y transporte interno autenticado.
- Para depuración local use `docker compose exec` o un port-forward temporal autenticado. No restaure puertos administrativos públicos. El smoke test destruye sus volúmenes y es solo para CI descartable.

El chart Helm habilita NetworkPolicies de denegación por defecto para tráfico de entrada. Verifique los selectores de `networkPolicy.ingressController` y `networkPolicy.monitoring` contra el clúster destino: un selector erróneo puede bloquear tráfico legítimo.

Janus deniega CORS de navegador salvo que `CORS_ALLOWED_ORIGINS` esté configurado. El valor Helm permite únicamente el origen Console configurado; agregue cada cliente deliberadamente, nunca `*`.

## Bloqueantes de producción pendientes

Las claves de firma de Janus se persisten en PostgreSQL y la inicialización/rotación se serializa con un bloqueo asesor transaccional. Los reinicios conservan el conjunto de firma y las claves retiradas de verificación. La propagación de rotación multi-instancia y la custodia de claves para recuperación ante desastre requieren verificación operativa antes de declarar alta disponibilidad.

La introspección y revocación exigen cliente confidencial y validan propiedad del token. Conformidad de protocolo, audiencias de recursos, autorización administrativa integral, TLS/credenciales de base de datos, aislamiento de tenants y recuperación operativa aún requieren verificación separada.

## Alcance de verificación

El guard de configuración renderiza Helm y analiza Compose para detectar publicación administrativa o JWT inconsistentes. Las pruebas cubren rutas, sesiones, roles, validación criptográfica y separación pública/administrativa. Son regresiones, no un pentest de despliegue activo.
