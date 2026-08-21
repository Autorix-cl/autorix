# SDKs Oficiales de Autorix: Visión General y Arquitectura

Los SDKs oficiales de Autorix proporcionan a los desarrolladores clientes fuertemente tipados, de alto rendimiento y listos para producción para integrar la suite de autenticación y autorización en microservicios, SPAs y APIs.

---

## 🛠️ Lenguajes Soportados

* 🐹 [**Go SDK**](/es/sdk/go) (`github.com/autorix-cl/autorix/sdk/go`)
* ⚛️ [**TypeScript / React SDK**](/es/sdk/typescript) (`@autorix/sdk-js`)
* 🐍 [**Python / FastAPI SDK**](/es/sdk/python) (`autorix`)
* 💻 [**CLI y REST/gRPC Directo**](/es/sdk/cli) (`autorixctl`)

---

## 📐 Principios de Ingeniería Compartidos

### 1. Seguridad por Defecto (*Fail-Closed*)
Si el SDK no puede comunicarse con los motores debido a una partición de red o caída de base de datos, el cliente **siempre evalúa a denegación estricta** (`allowed: false`).

### 2. Resiliencia: Exponential Backoff & Full Jitter
Para proteger contra avalanchas de tráfico (*Thundering Herd*), todos los SDKs aplican reintentos con dispersión aleatoria:
```text
Sleep Delay = UniformRandom(0, min(InitialDelay * Factor^Attempt, MaxDelay))
```

### 3. Verificación Vectorizada en Lote (`BatchCheck`)
Permite comprobar múltiples tuplas de permisos concurrentemente en un único ciclo de reloj.
