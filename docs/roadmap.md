# Autorix: Arquitectura, Patrones y Roadmap a Producción

Autorix es un ecosistema moderno de **Identity and Access Management (IAM)**. Su filosofía radica en la **descentralización extrema, Zero Trust y el patrón de API Headless**. Cada microservicio es un binario estático e independiente en Go, con su propio almacén de datos, evitando el acoplamiento y los cuellos de botella de bases de datos compartidas.

---

## 1. Topología de Microservicios

La suite consta de 6 motores principales, inspirados en la arquitectura de ORY pero mejorados con capacidades híbridas y despliegues modernos:

### 👤 Autorix Ego (Identidad)
* **Rol:** Identity Management System.
* **Responsabilidad:** Registro, login, verificación de emails, MFA, recuperación de contraseñas, y persistencia de esquemas de identidad personalizados (JSON Schema).
* **Restricción:** No emite tokens de acceso (JWT) para APIs, solo maneja el ciclo de vida del usuario (Cookies/Sesiones).

### 🧠 Autorix Nexus (Autorización Híbrida)
* **Rol:** Global Authorization Server (Zanzibar + ABAC).
* **Responsabilidad:** Almacena tuplas de relaciones (grafos) y compila expresiones de acceso dinámicas (Google CEL). Responde preguntas de alta velocidad como: *¿Puede el usuario X leer el archivo Y asumiendo que su IP es Z?*

### 🔑 Autorix Janus (Delegación OIDC/OAuth2)
* **Rol:** OAuth 2.0 y OpenID Connect Provider.
* **Responsabilidad:** Actúa como el motor de "delegación". Delega la pantalla de login a Ego, y la pantalla de consentimiento al panel de control. Emite tokens (JWT, Opacos), maneja revocaciones y firmas (JWKS).

### 🛡️ Autorix Aegis (Zero Trust Proxy)
* **Rol:** Identity & Access Proxy (IAP) / Edge Gateway.
* **Responsabilidad:** Se sienta frente a tus APIs de backend. Intercepta toda petición HTTP, valida los JWT contra Janus y pide permisos a Nexus. Si todo está en orden, inyecta headers de identidad (`X-User-ID`) y redirige el tráfico.

### 🔌 Autorix Hermes (Sincronización Empresarial)
* **Rol:** Enterprise SAML/SCIM Bridge.
* **Responsabilidad:** Sincroniza identidades de Active Directory, Okta o Google Workspace. Transforma aserciones SAML a OIDC para que el ecosistema interno funcione puramente con protocolos modernos.

### ⚡ Autorix Vulcan (API Keys & Máquinas)
* **Rol:** Machine-to-Machine & API Keys Manager.
* **Responsabilidad:** Gestiona tokens estáticos largos, rotación automática y emite tokens de capacidad criptográfica (Macaroons) que incluyen los permisos embebidos, ideal para integraciones B2B.

---

## 2. Stack Tecnológico y Patrones

### Tecnologías Core
- **Lenguaje:** Go (Golang). Excepcional concurrencia (goroutines) y bajo footprint de memoria.
- **Base de Datos:** PostgreSQL (aislada por servicio). Uso extensivo de índices GIN, B-Tree y campos `JSONB`.
- **Caché & PubSub:** Redis (para invalidación de tokens y caché de grafos en memoria).
- **Reglas Dinámicas:** Google CEL (Common Expression Language) para ABAC a latencias de microsegundos.
- **Transporte:** gRPC (interno) y REST/JSON (externo).

### Patrones Arquitectónicos
1. **Arquitectura Hexagonal (Ports & Adapters):** El dominio de cada servicio (ej. la expansión de árboles en Nexus) no conoce si los datos vienen de Postgres o Memoria, ni si el request llegó por gRPC o HTTP.
2. **Zero Trust Network Architecture (ZTNA):** Los microservicios internos no confían entre sí ciegamente. Todo tráfico externo primero es depurado y autenticado por Aegis.
3. **Database-per-Service:** Ego tiene su base de datos de usuarios; Janus tiene su base de tokens; Nexus tiene su base de tuplas. La sincronización se da por eventos.
4. **Outbox Pattern:** Para asegurar la consistencia eventual al crear un usuario en Ego y replicarlo a Nexus sin perder eventos si la red se cae.

---

## 3. Matriz de Comunicación Inter-Servicios

La comunicación es crítica para mantener la latencia del usuario final por debajo de los 50ms.

> [!IMPORTANT]
> **Regla de Oro:** La ruta crítica (evaluación de tokens y permisos) se hace por **gRPC**. La gestión administrativa se hace por **REST**.

### El Flujo de una Petición a Producción (Runtime)
1. **El Usuario hace una petición** a `api.empresa.com/facturas`.
2. **Aegis intercepta** la petición.
3. **Aegis llama a Janus (gRPC)** mediante introspección: *"¿Este token es válido y de quién es?"* -> Janus responde *"Es válido, es de Bob"*.
4. **Aegis llama a Nexus (gRPC)** usando `Check()`: *"¿Bob tiene relación de `viewer` sobre `facturas`? (Pasando IP de Bob como Contexto CEL)"*. -> Nexus evalúa el grafo y el ABAC en memoria/Postgres y responde `true`.
5. **Aegis enruta** el tráfico al microservicio de Facturas, inyectando el header `X-User-ID: bob`.

### Comunicación Asíncrona (Eventos)
Cuando Ego crea un usuario nuevo, escribe el evento en una tabla Outbox. Un worker lee la tabla y publica en un Message Broker (RabbitMQ/Kafka) o directamente a Nexus para crear una tupla por defecto (`namespace:user#self@bob`).

---

## 4. Roadmap a Producción (Fases de Ejecución)

### Fase 1: El Motor de Autorización (Fundamentos) 🚀
*Objetivo: Tener a Nexus evaluando permisos a gran velocidad.*
- [ ] Implementar parser y evaluador ABAC usando `cel-go` en Nexus.
- [ ] Construir motor de expansión de grafos (Zanzibar) en PostgreSQL y memoria.
- [ ] Configurar servidores gRPC y compilar Protobufs.
- [ ] Pruebas de estrés y benchmarking (< 10ms P99).

### Fase 2: Identidad y Delegación (Control de Acceso) 🔐
*Objetivo: Poder registrar usuarios y emitir tokens válidos.*
- [ ] Construir Autorix Ego (Login, Registro, Recuperación) usando perfiles JSON Schema.
- [ ] Construir Autorix Janus (OAuth2 Provider) soportando flujos Authorization Code y Client Credentials.
- [ ] Integrar el panel de UI en React (App administrativa).

### Fase 3: Intercepción Zero Trust (El Borde) 🛡️
*Objetivo: Proteger APIs detrás del muro de Autorix.*
- [ ] Construir Autorix Aegis como un Reverse Proxy optimizado.
- [ ] Desarrollar pipelines de middlewares en Aegis (Autenticador -> Autorizador -> Mutador).
- [ ] Configurar inyección de headers para backends confiables.

### Fase 4: Escalabilidad B2B y Enterprise 🏢
*Objetivo: Integrar a clientes corporativos pesados.*
- [ ] Desarrollar Autorix Vulcan (Emisión de Macaroons y validación criptográfica descentralizada).
- [ ] Desarrollar Autorix Hermes para sincronizar directorios externos vía SCIM.

### Fase 5: Operaciones y Producción 🌐
*Objetivo: Preparar la suite para Kubernetes y alta disponibilidad.*
- [ ] Implementar métricas de Prometheus y traces de OpenTelemetry en todos los binarios.
- [ ] Crear Helm Charts con auto-escalado horizontal.
- [ ] Implementar invalidación global de caché usando Redis Pub/Sub en Nexus.
- [ ] Auditoría de seguridad y pentesting final.
