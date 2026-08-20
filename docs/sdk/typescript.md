# TypeScript & React SDK Reference Manual (`@autorix/sdk-js`)

The official **Autorix TypeScript / JavaScript SDK** provides universal client libraries for Node.js, Next.js App Router/Pages Router, Express, and React 19 web applications.

---

## 1. Installation

```bash
npm install @autorix/sdk-js
```

---

## 2. Universal Client Initialization (Node.js / Next.js)

```typescript
import { AutorixClient } from "@autorix/sdk-js";

export const autorix = new AutorixClient({
  baseUrl: process.env.AUTORIX_BASE_URL || "http://localhost:4455",
  nexusUrl: process.env.AUTORIX_NEXUS_URL || "http://localhost:8080",
  themisUrl: process.env.AUTORIX_THEMIS_URL || "http://localhost:4488",
  egoUrl: process.env.AUTORIX_EGO_URL || "http://localhost:4433",
  janusUrl: process.env.AUTORIX_JANUS_URL || "http://localhost:4444",
  vulcanUrl: process.env.AUTORIX_VULCAN_URL || "http://localhost:4466",
  apiKey: process.env.AUTORIX_API_KEY,
  
  // Performance & Caching
  enableCache: true,
  cacheTtlMs: 15_000,
  
  // Resilience: Retry with Exponential Backoff & Jitter
  retryConfig: {
    maxRetries: 3,
    initialDelayMs: 50,
    maxDelayMs: 2000,
    backoffFactor: 2.0,
  },
});
```

---

## 3. Server-Side Operations Reference

### 3.1 Evaluating ReBAC Permissions (`check` & `checkBatch`)

```typescript
// Single ReBAC check
const decision = await autorix.check({
  namespace: "dashboards",
  object: "analytics_prod",
  relation: "viewer",
  subject: "usr_9988",
  context: { ip: "192.168.1.50" },
  explain: true,
});

if (decision.allowed) {
  console.log("Access granted!");
}

// Vectorized Batch check (runs concurrently)
const batchResults = await autorix.checkBatch([
  { namespace: "reports", object: "q1", relation: "read", subject: "usr_9988" },
  { namespace: "reports", object: "q2", relation: "write", subject: "usr_9988" },
  { namespace: "reports", object: "q3", relation: "delete", subject: "usr_9988" },
]);
```

### 3.2 Evaluating Themis ABAC CEL Policies (`evaluatePolicy`)

```typescript
const policyResult = await autorix.evaluatePolicy({
  tenantId: "default",
  context: {
    request: {
      auth: {
        claims: { department: "finance" },
        mfa: true,
      },
    },
    resource: {
      amount: 25000,
    },
  },
});

if (policyResult.allPassed) {
  // Proceed with transaction
}
```

### 3.3 Validating API Keys & Macaroons (`verifyApiKey`)

```typescript
const verification = await autorix.verifyApiKey("av_live_9f8e7d6c5b4a3f2e...", {
  ip: "10.0.4.15",
  method: "POST",
  required_scope: "invoices:write",
});

if (verification.valid) {
  console.log(`Key Name: ${verification.name}, Scopes: ${verification.scopes}`);
}
```

---

## ⚛️ 4. React 19 Hooks & Provider

### 4.1 Provider Setup in Next.js Root Layout

```tsx
// app/layout.tsx
import { AutorixProvider } from "@autorix/sdk-js";

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body>
        <AutorixProvider config={{ baseUrl: "http://localhost:4455" }}>
          {children}
        </AutorixProvider>
      </body>
    </html>
  );
}
```

---

### 4.2 `useSession()` Hook
Retrieves current user identity, traits, and authentication status:

```tsx
import { useSession } from "@autorix/sdk-js";

export function UserBadge() {
  const { user, isAuthenticated, loading, refreshSession } = useSession();

  if (loading) return <span>Loading profile...</span>;
  if (!isAuthenticated) return <a href="/login">Sign In</a>;

  return (
    <div>
      <p>Signed in as: {user?.traits.email}</p>
      <button onClick={() => refreshSession()}>Refresh Session</button>
    </div>
  );
}
```

---

### 4.3 `usePermission()` Hook
ReBAC check for an individual resource:

```tsx
import { usePermission } from "@autorix/sdk-js";

export function DocumentEditorButton({ documentId }: { documentId: string }) {
  const { allowed, checking } = usePermission("documents", documentId, "editor");

  if (checking) return <button disabled>Checking...</button>;
  if (!allowed) return null; // Hidden if unauthorized

  return <button onClick={() => openEditor(documentId)}>Edit Document</button>;
}
```

---

### 4.4 `useBatchPermissions()` Hook
Vectorized evaluation for tables, cards, or lists of resources:

```tsx
import { useBatchPermissions } from "@autorix/sdk-js";

export function FileExplorer({ files }: { files: Array<{ id: string; name: string }> }) {
  const { results, checking } = useBatchPermissions(
    files.map((file) => ({
      namespace: "files",
      object: file.id,
      relation: "deleter",
    }))
  );

  return (
    <ul>
      {files.map((file, idx) => (
        <li key={file.id}>
          <span>{file.name}</span>
          {results[idx]?.allowed && (
            <button onClick={() => deleteFile(file.id)}>Delete</button>
          )}
        </li>
      ))}
    </ul>
  );
}
```

---

### 4.5 `usePolicy()` Hook
Real-time evaluation of Google CEL ABAC rules:

```tsx
import { usePolicy, useSession } from "@autorix/sdk-js";

export function HighValueWireTransfer({ amount }: { amount: number }) {
  const { user } = useSession();
  const { passed, evaluating } = usePolicy({
    user,
    transfer_amount: amount,
    time_of_day: new Date().getHours(),
  });

  return (
    <div>
      <button disabled={evaluating || !passed}>
        {evaluating ? "Evaluating Compliance..." : "Execute Transfer"}
      </button>
      {!passed && !evaluating && <p className="text-red-500">Action blocked by risk policy</p>}
    </div>
  );
}
```
