# TypeScript and React SDK

Package: `@autorix/sdk-js`

The SDK supports Node.js 18+ and browser-like runtimes with `fetch`. React is optional unless importing React exports. Pass `AbortSignal` as the final argument of client operations to cancel a request.

## Create a client

```ts
const autorix = new AutorixClient({
  nexusUrl: "https://nexus.example.internal",
  themisUrl: "https://themis.example.internal",
  janusUrl: "https://login.example.com",
  timeoutMs: 10_000,
});
```

`AutorixConfig` accepts engine URLs, `timeoutMs`, retry/cache configuration, server-only `apiKey`, and an injectable `fetch` implementation. Failures from ordinary operations throw `AutorixApiError` or `AutorixTimeoutError`; permission and capability verification deliberately resolve to denied results.

## Public runtime operations

```ts
await autorix.check({ namespace: "documents", object: "document-42", relation: "viewer", subject: "user-7" });
await autorix.writeTuples([{ namespace: "documents", object: "document-42", relation: "viewer", subjectId: "user-7" }]);
await autorix.listPolicies({ tenantId: "default", limit: 20 });
await autorix.createPolicy(policy);
await autorix.createApiKey({ name: "worker", ownerId: "service-7", scopes: ["reports:read"] });
```

Nexus supports checks, batch checks, tuple list/write/delete, expansion, and subject/resource lookup. Themis supports evaluation, policy CRUD, versions, validation, and dry runs. Vulcan supports key create/list/verify/attenuate/revoke. List operations return `Page<T>` with `data`, `nextCursor`, and `hasMore`.

## Janus OAuth/OIDC

```ts
const authorizationUrl = autorix.createAuthorizationUrl({
  clientId: "spa-client", redirectUri: "https://app.example/callback", state, codeChallenge,
});
const tokens = await autorix.exchangeAuthorizationCode({
  clientId: "spa-client", code, redirectUri: "https://app.example/callback", codeVerifier,
});
const refreshed = await autorix.refreshToken(tokens.refresh_token!, "spa-client");
```

`getOpenIdConfiguration`, `getJwks`, `createAuthorizationUrl`, `exchangeAuthorizationCode`, `refreshToken`, `introspectToken`, and `revokeToken` are available. The client does not generate or persist PKCE values or tokens. Authorization-code and refresh exchanges are not automatically retried.

Use `clientSecret` only in backend code. `introspectToken` and `revokeToken` do not accept a client secret; use a confidential backend integration when Janus requires client authentication.

## React integration and browser boundary

`AutorixProvider` calls Ego `whoami` with cookies enabled. `useAutorix`, `useSession`, `usePermission`, `useBatchPermissions`, and `usePolicy` are available. `logout` is also available on `AutorixClient`; registration and login flows are not.

Never expose `apiKey`, a client secret, or operator credentials in a browser bundle. UI checks are not enforcement: repeat the authorization decision on the backend.

## Retry and cache notes

Only `GET`, `HEAD`, and `OPTIONS` retry by default; non-idempotent writes do not. Nexus decisions are cached in process for 10 seconds by default. The SDK has no OpenTelemetry integration.
