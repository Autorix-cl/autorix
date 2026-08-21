# Guía de Integración de SDKs Oficiales de Autorix

La documentación completa de los SDKs oficiales de Autorix se encuentra organizada en manuales dedicados por lenguaje y tecnología:

* 📦 [**Visión General y Principios de los SDKs**](/es/sdk/)
* 🐹 [**Manual del SDK de Go**](/es/sdk/go) (`github.com/autorix-cl/autorix/sdk/go`)
* ⚛️ [**Manual del SDK de TypeScript / React**](/es/sdk/typescript) (`@autorix/sdk-js`)
* 🐍 [**Manual del SDK de Python / FastAPI**](/es/sdk/python) (`autorix`)
* 💻 [**CLI y APIs Universales Directas**](/es/sdk/cli) (`autorixctl`)

---

## Capacidades Compartidas en Todos los SDKs

* **Postura Fail-Closed**: Toda llamada evalúa a `allowed: false` ante errores de red irrecuperables.
* **Exponential Backoff con Full Jitter**: Previene avalanchas de peticiones (*Thundering Herd*) durante conmutaciones por error del clúster.
* **Caché de Decisiones en Memoria**: Caché local de permisos con TTL configurable para latencias sub-milisegundo.
* **Evaluación Vectorizada en Lote**: Verificación paralela de permisos mediante pools de workers.
