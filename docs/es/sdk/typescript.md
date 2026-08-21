# SDK Oficial de TypeScript / JavaScript para Autorix

El SDK oficial de TypeScript (`@autorix/sdk-js`) proporciona integración nativa con Node.js, Next.js, React y navegadores modernos.

---

## 📦 Instalación

```bash
npm install @autorix/sdk-js
# o con pnpm / yarn
pnpm add @autorix/sdk-js
```

---

## 🚀 Uso Rápido

```typescript
import { AutorixClient } from '@autorix/sdk-js'

const autorix = new AutorixClient({
  nexusUrl: 'http://localhost:8080',
  themisUrl: 'http://localhost:4488',
  egoUrl: 'http://localhost:4433'
})

// Verificar permisos en Nexus
const { allowed } = await autorix.nexus.check({
  namespace: 'documents',
  object: 'roadmap_2026',
  relation: 'viewer',
  subjectNamespace: 'user',
  subjectId: 'alice'
})
```
