# Integrate Autorix Client SDKs

Integrate Autorix into your application stack using our official client SDKs, which provide fail-closed security, exponential backoff, in-memory caching, and vectorized batch checks.

## Quick path

Select your language SDK and install the package to get started immediately:

* 🐹 **Go:** [SDK Reference](/sdk/go) (`github.com/autorix-cl/autorix/sdk/go`)
* ⚛️ **TypeScript / React:** [SDK Reference](/sdk/typescript) (`@autorix/sdk-js`)
* 🐍 **Python / FastAPI:** [SDK Reference](/sdk/python) (`autorix`)
* 💻 **CLI / REST / gRPC:** [Reference](/sdk/cli) (`autorixctl`)

## Details

### Shared SDK Capabilities
All official SDKs share these core capabilities out of the box:
* **Fail-Closed Security Posture:** Safely evaluates to `allowed: false` on unrecoverable network failure.
* **Exponential Backoff & Full Jitter:** Protects against thundering herd during cluster failover.
* **In-Memory Decision Caching:** Local sub-millisecond evaluation cache with configurable TTL.
* **Vectorized Batch Checks:** Parallelized permission evaluation over worker pools.

For shared architectural standards, see the [SDKs Overview](/sdk/).

## Checklist

* [ ] Choose the correct SDK for your application stack.
* [ ] Verify your network allows the SDK to connect to Autorix services.
* [ ] Configure your SDK with the appropriate caching and backoff settings.

## Next step

Go directly to your language's SDK reference manual (linked in the Quick path) and complete the initial setup and authentication steps.
