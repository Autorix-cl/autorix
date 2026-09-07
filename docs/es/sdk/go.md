# SDK de Go

Módulo: `github.com/autorix-cl/autorix/sdk/go`
Versión de Go declarada: `1.25.6`

El SDK de Go es el de mayor cobertura actual. Es un cliente síncrono basado en `net/http` y sus métodos públicos reciben `context.Context`.

## Crear un cliente

```go
import (
    "net/http"
    "time"

    autorix "github.com/autorix-cl/autorix/sdk/go"
)

client := autorix.NewClient(autorix.Config{
    NexusURL:  "http://localhost:8080",
    ThemisURL: "http://localhost:4488",
    EgoURL:    "http://localhost:4433",
    JanusURL:  "http://localhost:4444",
    VulcanURL: "http://localhost:4466",
    ArgusURL:  "http://localhost:4400",
    HTTPClient: &http.Client{Timeout: 10 * time.Second},
    EnableCache: true,
    CacheTTL: 10 * time.Second,
})
```

`NewClient` usa direcciones localhost para motores no configurados y un tiempo de espera HTTP de 10 segundos cuando no se proporciona cliente. También están disponibles `WithBaseURL`, `WithAPIKey`, `WithRetryConfig` y `WithCache`.

## Clientes implementados

### Nexus

`Check` valida namespace, objeto, relación y sujeto, y devuelve `(bool, error)`. Ante un error de solicitud o respuesta devuelve `false` y un error.

```go
allowed, err := client.Nexus.Check(ctx, autorix.CheckRequest{
    Namespace: "documents",
    Object: "document-42",
    Relation: "viewer",
    SubjectID: "user-7",
})
```

También están disponibles:

- `CheckBatch(ctx, []CheckRequest) ([]bool, error)`, que ejecuta comprobaciones individuales de forma concurrente.
- `Expand(ctx, namespace, object, relation) (map[string]interface{}, error)`.
- `LookupResources(ctx, namespace, relation, subjectID, subjectNamespace) ([]string, error)`.
- `Client.Check(...)`, ayuda heredada que delega en `Nexus.Check`.

### Themis y Vulcan

```go
result, err := client.Themis.Evaluate(ctx, autorix.EvaluatePolicyRequest{
    TenantID: "default",
    Context: map[string]interface{}{"request": map[string]interface{}{}},
})

key, err := client.Vulcan.Verify(ctx, "presented-token", nil)
attenuated, err := client.Vulcan.Attenuate(ctx, "macaroon-token", []string{"scope = reports:read"})
```

`Themis.Evaluate` evalúa políticas activas. `Vulcan.Verify` verifica una API key o macaroon y `Vulcan.Attenuate` solicita la atenuación a Vulcan. Ninguno proporciona CRUD de claves o políticas.

### Ego, Janus y Argus

```go
session, err := client.Ego.WhoAmI(ctx, "session-token")
introspection, err := client.Janus.Introspect(ctx, "access-token")
jwks, err := client.Janus.GetJWKS(ctx)
audit, err := client.Argus.VerifyAuditTrail(ctx)
```

`Janus.GetJWKS` guarda el resultado durante cinco minutos. `Discovery`, `AuthorizationURL`, `ExchangeToken` y `Revoke` cubren discovery, autorización con PKCE S256, intercambio y revocación. El SDK no guarda verificadores PKCE ni tokens; la aplicación es responsable de ambos. Los intercambios y la revocación no se reintentan.

## Middleware de `net/http`

`Middleware` lee cabeceras de identidad inyectadas por un proxy de confianza: `X-User-ID`, `X-User-Email` y `X-User-Roles`. `RequireAuth` exige una identidad de esas cabeceras; `RequirePermission` ejecuta después una comprobación Nexus.

```go
handler := client.RequirePermission(
    "documents",
    "editor",
    func(r *http.Request) string { return r.PathValue("id") },
    http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        user, _ := autorix.UserFromContext(r.Context())
        fmt.Fprintln(w, user.ID)
    }),
)
```

Solo confíe en estas cabeceras si un proxy autenticado elimina cualquier copia enviada por el cliente antes de reenviar la solicitud. Este middleware no valida una sesión de Ego por sí mismo.

## Notas sobre reintentos y caché

La ayuda compartida de reintentos se usa en `Nexus.Check`, `Themis.Evaluate`, `Janus.Introspect`, `Ego.WhoAmI` y métodos de Vulcan. No todas las solicitudes Go se reintentan: `Expand`, `LookupResources`, `GetJWKS` y la verificación de Argus llaman directamente al cliente HTTP configurado. La caché Nexus es local al proceso y no se invalida al cambiar relaciones.
