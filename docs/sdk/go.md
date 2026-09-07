# Go SDK

Module: `github.com/autorix-cl/autorix/sdk/go`

The Go SDK is a synchronous `net/http` client. Public operations accept `context.Context`.

## Create a client

```go
client := autorix.NewClient(autorix.Config{
    NexusURL: "http://localhost:8080", ThemisURL: "http://localhost:4488",
    EgoURL: "http://localhost:4433", JanusURL: "http://localhost:4444",
    VulcanURL: "http://localhost:4466", ArgusURL: "http://localhost:4400",
    HTTPClient: &http.Client{Timeout: 10 * time.Second},
})
```

`NewClient` supplies localhost defaults and a 10-second HTTP timeout. `WithBaseURL`, `WithAPIKey`, `WithRetryConfig`, and `WithCache` are available.

## Implemented clients

- **Nexus:** `Check`, `CheckBatch`, `Expand`, and `LookupResources`. `Client.Check` remains as a compatibility wrapper. `Check` returns `false` and an error on transport or service failure.
- **Themis:** `Evaluate` for active policies only.
- **Vulcan:** `Verify` and `Attenuate` only.
- **Ego:** `WhoAmI` only.
- **Argus:** `VerifyAuditTrail` only.

```go
allowed, err := client.Nexus.Check(ctx, autorix.CheckRequest{
    Namespace: "documents", Object: "document-42", Relation: "viewer", SubjectID: "user-7",
})
result, err := client.Themis.Evaluate(ctx, autorix.EvaluatePolicyRequest{Context: map[string]interface{}{"request": map[string]interface{}{}}})
key, err := client.Vulcan.Verify(ctx, "presented-token", nil)
```

## Janus OAuth/OIDC

```go
url, err := client.Janus.AuthorizationURL("web-client", "https://app.example/callback", state, "openid offline_access", codeChallenge)
metadata, err := client.Janus.Discovery(ctx)
tokens, err := client.Janus.ExchangeToken(ctx, autorix.TokenRequest{
    GrantType: "authorization_code", ClientID: "web-client", Code: code,
    CodeVerifier: verifier, RedirectURI: "https://app.example/callback",
})
err = client.Janus.Revoke(ctx, clientID, clientSecret, tokens.RefreshToken, "refresh_token")
```

`AuthorizationURL` applies `S256` when a code challenge is supplied. The SDK does not generate or store PKCE verifiers or tokens. `ExchangeToken` deliberately does not retry because authorization codes and refresh tokens can be single-use. Pass `ClientSecret` only from a confidential backend. `Introspect` and `GetJWKS` remain available; successful JWKS responses are cached for five minutes.

## `net/http` middleware

`Middleware` reads `X-User-ID`, `X-User-Email`, and `X-User-Roles` from a trusted proxy. `RequireAuth` requires those headers; `RequirePermission` runs a Nexus check.

```go
handler := client.RequirePermission("documents", "editor", func(r *http.Request) string {
    return r.PathValue("id")
}, next)
```

Only use these headers behind a proxy that removes client-supplied copies. The middleware does not validate an Ego session.

## Retry and cache notes

The retry helper is used for Nexus `Check`, Themis `Evaluate`, Ego `WhoAmI`, Janus discovery/introspection, and Vulcan methods. `Expand`, `LookupResources`, `GetJWKS`, and Argus verification use the configured HTTP client directly. Nexus cache entries are local to the process.
