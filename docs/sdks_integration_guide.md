# Official SDKs & Integration Guide: Autorix (Go, TypeScript/React, Python)

**Autorix SDKs** provide native client libraries and middleware for Go, TypeScript/React, and Python, making it trivial to enforce Zero-Trust ReBAC/ABAC authorization, manage identities, and handle attenuated API keys.

---

## 1. Go SDK (`github.com/autorix-cl/autorix/sdk/go`)

### Installation
```bash
go get github.com/autorix-cl/autorix/sdk/go
```

### Usage in Microservices behind Autorix Aegis
```go
package main

import (
	"fmt"
	"net/http"

	"github.com/autorix-cl/autorix/sdk/go"
)

func main() {
	// 1. Initialize client connected to Nexus (ReBAC) and Themis (ABAC)
	client := autorix.NewClient(autorix.Config{
		NexusAddr:   "nexus:50051",
		ThemisAddr:  "themis:50052",
		EnableCache: true,
	})

	// 2. Protect HTTP handler with Autorix middleware
	http.Handle("/api/v1/documents/", client.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, _ := autorix.UserFromContext(r.Context())

		// 3. Evaluate ReBAC relation in Nexus (< 2ms)
		allowed, _ := client.Check(r.Context(), "document", "payroll_2026", "viewer", user.ID, nil)
		if !allowed {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		fmt.Fprintf(w, "Access granted to %s", user.Email)
	})))

	http.ListenAndServe(":8080", nil)
}
```

---

## 2. TypeScript / React SDK (`@autorix/sdk-js`)

### Installation
```bash
npm install @autorix/sdk-js
```

### Setup Provider in Next.js / React
```tsx
import { AutorixProvider } from "@autorix/sdk-js";

export function App({ children }: { children: React.ReactNode }) {
  return (
    <AutorixProvider config={{ baseUrl: "http://localhost:4455" }}>
      {children}
    </AutorixProvider>
  );
}
```

### Using Hooks (`useSession`, `usePermission`, `usePolicy`)
```tsx
import { useSession, usePermission, usePolicy } from "@autorix/sdk-js";

export function DocumentViewer({ docId }: { docId: string }) {
  const { user, isAuthenticated, loading } = useSession();
  const { allowed, checking } = usePermission("document", docId, "editor");
  const { passed: mfaPassed } = usePolicy("pol_enforce_mfa", { user });

  if (loading || checking) return <div>Loading permissions...</div>;
  if (!isAuthenticated) return <div>Please sign in.</div>;

  return (
    <div>
      <h2>Welcome, {user?.traits.email}</h2>
      {allowed && mfaPassed ? (
        <button>Edit Document</button>
      ) : (
        <p>Read-only view</p>
      )}
    </div>
  );
}
```

---

## 3. Python SDK (`autorix-python`)

### Installation
```bash
pip install autorix
```

### FastAPI Integration Example
```python
from fastapi import FastAPI, Depends, Header, HTTPException
from autorix.fastapi import AutorixSecurity
from autorix import AutorixClient

app = FastAPI()
auth = AutorixSecurity()
client = AutorixClient(nexus_url="http://nexus:50051", themis_url="http://themis:50052")

@app.get("/api/reports/{report_id}")
async def get_report(
    report_id: str,
    x_user_id: str = Header(None),
    x_user_email: str = Header(None)
):
    user = auth.get_current_user(x_user_id=x_user_id, x_user_email=x_user_email)
    if not user:
        raise HTTPException(status_code=401, detail="Unauthorized")

    # Check ReBAC permissions
    result = client.check("report", report_id, "viewer", user.id)
    if not result.allowed:
        raise HTTPException(status_code=403, detail="Forbidden")

    return {"report_id": report_id, "owner": user.email}
```
