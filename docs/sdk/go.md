# Go SDK Reference Manual (`github.com/autorix-cl/autorix/sdk/go`)

The official **Autorix Go SDK** provides high-throughput, type-safe client interfaces and standard `net/http` middlewares for Go microservices.

---

## 1. Installation

```bash
go get github.com/autorix-cl/autorix/sdk/go
```

Requirements: Go `1.22+`.

---

## 2. Client Initialization & Configuration

The client uses **functional options** and exposes dedicated sub-clients for each engine:

```go
package main

import (
	"time"
	"github.com/autorix-cl/autorix/sdk/go"
)

func main() {
	client := autorix.NewClient(
		autorix.Config{
			NexusURL:  "http://localhost:8080", // Nexus ReBAC Engine
			ThemisURL: "http://localhost:4488", // Themis CEL Policy Engine
			EgoURL:    "http://localhost:4433", // Ego Identity & Session Engine
			JanusURL:  "http://localhost:4444", // Janus OAuth2/OIDC Server
			VulcanURL: "http://localhost:4466", // Vulcan Macaroons & API Keys
			ArgusURL:  "http://localhost:4400", // Argus Control Plane
			APIKey:    "av_live_01918a7b6c5d4e3f2a1b0c9d8e7f6a",
			
			// Resilience: Exponential backoff with full jitter
			RetryConfig: autorix.RetryConfig{
				MaxRetries:    3,
				InitialDelay:  50 * time.Millisecond,
				MaxDelay:      2 * time.Second,
				BackoffFactor: 2.0,
			},
			
			// Performance: In-memory decision caching
			EnableCache: true,
			CacheTTL:    15 * time.Second,
		},
	)
}
```

---

## 3. Sub-Clients & Feature Reference

### 3.1 `client.Nexus` — Zanzibar ReBAC & Graph Evaluation

#### Single Permission Check (`Check`)
```go
allowed, err := client.Nexus.Check(ctx, autorix.CheckRequest{
	Namespace:        "documents",
	Object:           "roadmap_2026_q3",
	Relation:         "editor",
	SubjectNamespace: "user",
	SubjectID:        "usr_alice",
	RequestContext:   map[string]interface{}{"ip": "10.0.4.15"},
	Explain:          true,
})
```

#### Vectorized Batch Check (`CheckBatch`)
Evaluates dozens of relation checks in parallel across internal worker goroutines:
```go
requests := []autorix.CheckRequest{
	{Namespace: "documents", Object: "doc_1", Relation: "viewer", SubjectID: "usr_alice"},
	{Namespace: "documents", Object: "doc_2", Relation: "editor", SubjectID: "usr_alice"},
	{Namespace: "documents", Object: "doc_3", Relation: "owner", SubjectID: "usr_alice"},
}

results, err := client.Nexus.CheckBatch(ctx, requests)
// results = [true, true, false]
```

#### Reverse Lookup: Find Accessible Resources (`LookupResources`)
```go
resources, err := client.Nexus.LookupResources(ctx, "documents", "editor", "usr_alice", "user")
// resources = ["doc_1", "doc_2", "doc_88"]
```

---

### 3.2 `client.Themis` — Google CEL ABAC Policy Evaluation

Evaluates contextual attribute rules dynamically:

```go
res, err := client.Themis.Evaluate(ctx, autorix.EvaluatePolicyRequest{
	TenantID: "default",
	Context: map[string]interface{}{
		"request": map[string]interface{}{
			"auth": map[string]interface{}{
				"claims": map[string]interface{}{"department": "finance"},
				"mfa":    true,
			},
			"time": map[string]interface{}{"hour": 14},
		},
		"resource": map[string]interface{}{
			"amount": 50000,
		},
	},
})

if err == nil && res.AllPassed {
	// Policy passed
}
```

---

### 3.3 `client.Vulcan` — API Keys & Macaroon Attenuation

#### Verify Incoming API Key
```go
res, err := client.Vulcan.Verify(ctx, "av_live_9f8e7d6c5b4a3f2e...", map[string]interface{}{
	"ip":             "10.0.4.15",
	"method":         "POST",
	"required_scope": "ingest:write",
})

if res.Valid {
	fmt.Printf("Authenticated Key: %s (Scopes: %v)\n", res.Name, res.Scopes)
}
```

#### Attenuate Macaroon with Caveats Offline
```go
attenuatedKey, err := client.Vulcan.Attenuate(ctx, "av_live_root_key", []string{
	"time < 2026-08-21T00:00:00Z",
	"ip = 10.0.4.15",
	"scope = ingest:write",
})
```

---

### 3.4 `client.Janus` — OAuth2 & JWKS

#### Token Introspection (RFC 7662)
```go
tokenInfo, err := client.Janus.Introspect(ctx, "eyJhbGciOiJSUzI1Ni...")
if tokenInfo.Active {
	fmt.Printf("Subject: %s, Scopes: %s\n", tokenInfo.Subject, tokenInfo.Scope)
}
```

#### Get Public JWKS Keys with Stale-While-Revalidate
```go
jwks, err := client.Janus.GetJWKS(ctx)
```

---

### 3.5 `client.Ego` — Identity & Sessions

```go
session, err := client.Ego.WhoAmI(ctx, "ast_01918a7b6c5d4e3f...")
if err == nil && session.Active {
	fmt.Printf("User: %s (Department: %v)\n", session.Identity.Email, session.Identity.Traits["department"])
}
```

---

### 3.6 `client.Argus` — Audit Integrity & Governance

```go
auditProof, err := client.Argus.VerifyAuditTrail(ctx)
if auditProof.Verified {
	fmt.Printf("Cryptographic Audit Chain Intact: Length %d, Head %s\n", auditProof.ChainLength, auditProof.HeadHash)
}
```

---

## 🛡️ 4. HTTP Middlewares

### 4.1 `client.RequireAuth`
Extracts identity headers (`X-User-ID`, `X-User-Email`) injected by Aegis and aborts with `401 Unauthorized` if missing:

```go
http.Handle("/api/profile", client.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	user, _ := autorix.UserFromContext(r.Context())
	fmt.Fprintf(w, "Hello, %s", user.Email)
})))
```

### 4.2 `client.RequirePermission`
Enforces a Zanzibar ReBAC check before invoking the handler:

```go
http.Handle("/api/documents/{id}", client.RequirePermission(
	"documents",
	"editor",
	func(r *http.Request) string { return r.PathValue("id") },
	http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, _ := autorix.UserFromContext(r.Context())
		fmt.Fprintf(w, "Editing document for user %s", user.Email)
	}),
))
```
