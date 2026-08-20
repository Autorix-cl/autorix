# Autorix Official Client SDKs

**Autorix SDKs** provide idiomatic, type-safe, and enterprise-grade client libraries for **Go**, **TypeScript/React**, and **Python**. Engineered to Google and AWS reliability standards, each SDK features built-in **exponential backoff with full jitter**, **vectorized batch evaluation**, **in-memory decision caching**, and distributed tracing propagation.

---

## 📦 Supported Languages & Frameworks

| Language / Stack | Package Name | Installation | Key Features |
| :--- | :--- | :--- | :--- |
| **Go** | `github.com/autorix-cl/autorix/sdk/go` | `go get github.com/autorix-cl/autorix/sdk/go` | Sub-clients for all 8 engines, `CheckBatch`, Full Jitter Retries, HTTP Middleware (`RequirePermission`). |
| **TypeScript / React** | `@autorix/sdk-js` | `npm install @autorix/sdk-js` | Universal Node.js + React 19 hooks (`useSession`, `usePermission`, `useBatchPermissions`, `usePolicy`), in-memory cache. |
| **Python** | `autorix` | `pip install autorix` | Sync & Async (`httpx`) clients, FastAPI `Depends()` security integration, `check_batch`, Pydantic models. |
| **CLI & Direct** | `autorixctl` / HTTP / gRPC | Precompiled binary | Raw REST endpoints, gRPC Protobuf reflection for Rust, C#, PHP, Java. |

---

## 🏛️ Shared Architectural Standards Across All SDKs

All Autorix client libraries adhere to the following reliability principles:

```text
       [ Application Code ]
                │
                ▼
  ┌───────────────────────────┐
  │       Autorix Client      │
  │                           │
  │  ┌─────────────────────┐  │ (Checks local cache first: TTL 10-30s)
  │  │ Local Memory Cache  │  │
  │  └──────────┬──────────┘  │
  │             │ (Cache Miss)│
  │  ┌──────────▼──────────┐  │ (Exponential Backoff + Full Jitter)
  │  │ Retry Engine        │  │ (Retries 429, 502, 503, 504, Timeouts)
  │  └──────────┬──────────┘  │
  │             │             │
  │  ┌──────────▼──────────┐  │ (Injects X-Request-ID, W3C traceparent)
  │  │ Telemetry Injector  │  │
  │  └──────────┬──────────┘  │
  └─────────────┼─────────────┘
                │
                ▼ (HTTP/1.1 REST or gRPC HTTP/2)
       [ Autorix IAM Cluster ]
```

### 1. Fail-Closed Security Posture
If a network partition occurs and retries are exhausted, the SDK **always returns `allowed = false`** and logs the error, ensuring no unauthorized action is inadvertently permitted during cluster degradation.

### 2. Resilience: Exponential Backoff & Full Jitter
To prevent *Thundering Herd* spikes against the database during cluster failovers, all SDKs implement randomized jitter backoff:
$$\text{Sleep Delay} = \text{UniformRandom}(0, \min(\text{InitialDelay} \times \text{Factor}^{\text{Attempt}}, \text{MaxDelay}))$$

### 3. Vectorized Parallel Execution (`BatchCheck`)
Instead of executing sequential HTTP round-trips for each resource on a page, batch checks evaluate all permissions concurrently across worker pools.

---

## 🚀 Choose Your SDK

* 🐹 [**Go SDK Reference Manual**](/sdk/go)
* ⚛️ [**TypeScript & React SDK Reference Manual**](/sdk/typescript)
* 🐍 [**Python & FastAPI SDK Reference Manual**](/sdk/python)
* 💻 [**CLI & Universal REST/gRPC Integration**](/sdk/cli)
