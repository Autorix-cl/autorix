# SDK de TypeScript y React

Paquete: `@autorix/sdk-js`

El SDK TypeScript ofrece un cliente pequeño basado en `fetch` y exportaciones React opcionales. Funciona en Node.js o entornos similares a navegador que proporcionen `fetch`; React solo es necesario para `AutorixProvider` y los hooks.

## Crear un cliente

```ts
import { AutorixClient } from "@autorix/sdk-js";

const autorix = new AutorixClient({
  nexusUrl: "https://nexus.example.internal",
  themisUrl: "https://themis.example.internal",
  egoUrl: "https://ego.example.com",
  enableCache: true,
  cacheTtlMs: 10_000,
});
```

El cliente acepta URLs opcionales para todos los motores, `apiKey`, caché y reintentos. Si se proporciona una `apiKey`, se envía como bearer token. No configure `apiKey` en código de navegador.

## Operaciones implementadas

```ts
const session = await autorix.whoami();

const decision = await autorix.check({
  namespace: "documents",
  object: "document-42",
  relation: "viewer",
  subject: "user-7",
});

const decisions = await autorix.checkBatch([
  { namespace: "documents", object: "document-42", relation: "viewer", subject: "user-7" },
]);

const policies = await autorix.evaluatePolicy({
  tenantId: "default",
  context: { request: {} },
});

const key = await autorix.verifyApiKey("presented-token");
```

`whoami` devuelve `null` ante una respuesta no satisfactoria o fallo de solicitud. `check` y `evaluatePolicy` resuelven resultados de denegación ante un fallo, en lugar de lanzar la respuesta del servicio. `verifyApiKey` también resuelve con `valid: false`. Trate estos resultados como decisiones de denegación y registre `reason` o `error` desde código de servidor.

El paquete cubre OAuth/OIDC público de Janus, expansión y búsquedas Nexus, atenuación Vulcan y CRUD público de Nexus, Themis y Vulcan. Argus y las API administrativas siguen fuera de alcance.

## Integración React

```tsx
import { AutorixProvider, usePermission, useSession } from "@autorix/sdk-js";

function App({ children }: { children: React.ReactNode }) {
  return <AutorixProvider config={{ egoUrl: "https://ego.example.com" }}>{children}</AutorixProvider>;
}

function EditButton({ id }: { id: string }) {
  const { user, isAuthenticated } = useSession();
  const { allowed, checking } = usePermission("documents", id, "editor");
  if (!isAuthenticated || checking || !allowed) return null;
  return <button>Edit</button>;
}
```

Los hooks disponibles son `useAutorix`, `useSession`, `usePermission`, `useBatchPermissions` y `usePolicy`. El proveedor consulta `whoami()` al montarse. Los hooks de permisos y políticas deniegan en la interfaz ante errores.

## Límite del navegador

La integración React usa cookies del navegador mediante `credentials: "include"` para `whoami`. Puede construir flujos OAuth con PKCE, intercambiar códigos, renovar y revocar tokens, pero no genera ni almacena state, verifier o tokens.

Nunca use un hook de navegador como límite de autorización: un usuario puede modificar código y solicitudes del navegador. Compruebe nuevamente la acción solicitada en el servidor. No exponga API keys, secretos de cliente ni credenciales de operador en variables públicas de Next.js o bundles React.

## Notas sobre reintentos y caché

`whoami`, `check`, `evaluatePolicy` y `verifyApiKey` usan la ayuda de reintentos. Reintenta fallos de red y estados `429`, `502`, `503` y `504`; no distingue entre operaciones idempotentes y no idempotentes. Los resultados Nexus se guardan en memoria durante 10 segundos por defecto. No hay ayudas de cancelación, clases de error tipadas, integración OpenTelemetry ni utilidades de paginación.
