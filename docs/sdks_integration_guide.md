# Autorix Official Client SDKs Integration Guide

The complete SDK documentation has been organized into dedicated reference manuals for each supported programming language and stack:

* 📦 [**SDKs Overview & Shared Architecture Standards**](/sdk/)
* 🐹 [**Go SDK Reference Manual**](/sdk/go) (`github.com/autorix-cl/autorix/sdk/go`)
* ⚛️ [**TypeScript / React SDK Reference Manual**](/sdk/typescript) (`@autorix/sdk-js`)
* 🐍 [**Python / FastAPI SDK Reference Manual**](/sdk/python) (`autorix`)
* 💻 [**CLI & Direct REST/gRPC Integration**](/sdk/cli) (`autorixctl`)

---

## Shared Capabilities Across All Official SDKs

* **Fail-Closed Security Posture**: Evaluates to `allowed: false` on unrecoverable network failure.
* **Exponential Backoff & Full Jitter**: Protects against thundering herd during cluster failover.
* **In-Memory Decision Caching**: Local sub-millisecond evaluation cache with configurable TTL.
* **Vectorized Batch Checks**: Parallelized permission evaluation over worker pools.
