# Official SDKs & Integration Guide: Autorix (Go, TypeScript/React, Python)

**Autorix SDKs** provide enterprise-grade client libraries and middleware for **Go**, **TypeScript/React**, and **Python**. Designed to Google/AWS reliability standards, they feature **exponential backoff with full jitter**, **vectorized batch evaluation**, **in-memory decision caching**, and native integration across all 8 Autorix engines.

---

## 1. Go SDK (`github.com/autorix-cl/autorix/sdk/go`)

### 1.1 Installation
```bash
go get github.com/autorix-cl/autorix/sdk/go
```

### 1.2 Enterprise Client Initialization & Resilience Configuration

```go
package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/autorix-cl/autorix/sdk/go"
)

func main() {
	// Initialize client with custom exponential backoff & jitter
	client := autorix.NewClient(
		autorix.Config{
			NexusURL:  "http://localhost:8080",
			ThemisURL: "http://localhost:4488",
			EgoURL:    "http://localhost:4433",
			VulcanURL: "http://localhost:4466",
			APIKey:    "av_live_01918a7b6c5d4e3f2a1b0c9d8e7f6a",
			EnableCache: true,
			CacheTTL:    30 * time.Second,
			RetryConfig: autorix.RetryConfig{
				MaxRetries:    3,
				InitialDelay:  50 * time.Millisecond,
				MaxDelay:      2 * time.Second,
				BackoffFactor: 2.0,
			},
		},
	)

	ctx := context.Background()

	// 1. Single ReBAC Check
	allowed, err := client.Nexus.Check(ctx, autorix.CheckRequest{
		Namespace: "documents",
		Object:    "roadmap_2026",
		Relation:  "editor",
		SubjectID: "usr_alice",
	})
	fmt.Printf("Single Check: %v (err: %v)\n", allowed, err)

	// 2. Vectorized Batch Check (Evaluates in parallel)
	batchResults, err := client.Nexus.CheckBatch(ctx, []autorix.CheckRequest{
		{Namespace: "documents", Object: "doc_1", Relation: "viewer", SubjectID: "usr_alice"},
		{Namespace: "documents", Object: "doc_2", Relation: "editor", SubjectID: "usr_alice"},
		{Namespace: "documents", Object: "doc_3", Relation: "admin", SubjectID: "usr_alice"},
	})
	fmt.Printf("Batch Results: %v\n", batchResults)

	// 3. Themis ABAC CEL Policy Evaluation
	policyRes, _ := client.Themis.Evaluate(ctx, autorix.EvaluatePolicyRequest{
		TenantID: "default",
		Context: map[string]interface{}{
			"auth": map[string]interface{}{
				"claims": map[string]interface{}{"department": "finance"},
				"mfa":    true,
			},
		},
	})
	fmt.Printf("Themis Evaluation: All Passed = %v\n", policyRes.AllPassed)

	// 4. Vulcan Macaroon Attenuation & Verification
	attenuatedKey, _ := client.Vulcan.Attenuate(ctx, "av_live_01918a...", []string{
		"ip = 10.0.4.15",
		"time < 2026-08-21T00:00:00Z",
	})
	fmt.Printf("Attenuated Key: %s\n", attenuatedKey)

	// 5. HTTP Middleware Protection
	http.Handle("/api/v1/documents/{id}", client.RequirePermission(
		"documents",
		"editor",
		func(r *http.Request) string { return r.PathValue("id") },
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, _ := autorix.UserFromContext(r.Context())
			fmt.Fprintf(w, "Access granted to %s", user.Email)
		}),
	))

	http.ListenAndServe(":8080", nil)
}
```

---

## 2. TypeScript / React SDK (`@autorix/sdk-js`)

### 2.1 Installation
```bash
npm install @autorix/sdk-js
```

### 2.2 Client Usage in Node.js / Next.js API Routes

```typescript
import { AutorixClient } from "@autorix/sdk-js";

const client = new AutorixClient({
  nexusUrl: process.env.AUTORIX_NEXUS_URL || "http://localhost:8080",
  themisUrl: process.env.AUTORIX_THEMIS_URL || "http://localhost:4488",
  apiKey: process.env.AUTORIX_API_KEY,
  enableCache: true,
  cacheTtlMs: 15_000,
  retryConfig: { maxRetries: 3, initialDelayMs: 50 },
});

// Single Check
const { allowed } = await client.check({
  namespace: "dashboards",
  object: "analytics_prod",
  relation: "viewer",
  subject: "usr_9988",
});

// Batch Check
const results = await client.checkBatch([
  { namespace: "reports", object: "rep_1", relation: "read", subject: "usr_9988" },
  { namespace: "reports", object: "rep_2", relation: "write", subject: "usr_9988" },
]);
```

### 2.3 React Provider & Hooks

Wrap your application root with `<AutorixProvider>`:

```tsx
import { AutorixProvider } from "@autorix/sdk-js";

export function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <AutorixProvider config={{ baseUrl: "http://localhost:4455" }}>
      {children}
    </AutorixProvider>
  );
}
```

#### Parallel Batch Checking in React Components

```tsx
import { useSession, useBatchPermissions, usePolicy } from "@autorix/sdk-js";

export function ProjectTable({ documents }: { documents: Array<{ id: string; title: string }> }) {
  const { user, isAuthenticated } = useSession();

  // Evaluates permissions for all documents in parallel
  const { results, checking } = useBatchPermissions(
    documents.map((doc) => ({
      namespace: "documents",
      object: doc.id,
      relation: "editor",
    }))
  );

  // Evaluates enterprise MFA ABAC policy
  const { passed: mfaOk } = usePolicy({ user });

  if (checking) return <div>Evaluating permissions...</div>;
  if (!isAuthenticated) return <div>Please authenticate.</div>;

  return (
    <table>
      <tbody>
        {documents.map((doc, idx) => (
          <tr key={doc.id}>
            <td>{doc.title}</td>
            <td>
              {results[idx]?.allowed && mfaOk ? (
                <button>Edit</button>
              ) : (
                <span>Read-Only</span>
              )}
            </td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}
```

---

## 3. Python SDK (`autorix-python`)

### 3.1 Installation
```bash
pip install autorix
```

### 3.2 FastAPI Integration with Async Support

```python
from fastapi import FastAPI, Depends, Header, HTTPException
from autorix.fastapi import AutorixSecurity
from autorix import AutorixClient, RetryConfig

app = FastAPI()
auth = AutorixSecurity()

client = AutorixClient(
    nexus_url="http://nexus:8080",
    themis_url="http://themis:4488",
    vulcan_url="http://vulcan:4466",
    retry_config=RetryConfig(max_retries=3, initial_delay=0.05)
)

@app.get("/api/v1/projects/{project_id}")
async def get_project(
    project_id: str,
    x_user_id: str = Header(None),
    x_user_email: str = Header(None)
):
    user = auth.get_current_user(x_user_id=x_user_id, x_user_email=x_user_email)
    if not user:
        raise HTTPException(status_code=401, detail="Unauthorized")

    # Async ReBAC Check with automatic retries
    res = await client.acheck(
        namespace="projects",
        object_id=project_id,
        relation="viewer",
        subject=user.id
    )

    if not res.allowed:
        raise HTTPException(status_code=403, detail="Access denied by Nexus")

    # Themis ABAC CEL Policy Check
    policy_res = client.evaluate_policy(
        policy_context={"user": {"id": user.id, "email": user.email}}
    )

    return {
        "project_id": project_id,
        "owner": user.email,
        "policy_verified": policy_res.all_passed
    }
```
