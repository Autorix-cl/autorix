# Guía Oficial de SDKs de Integración: Autorix (Go, TypeScript/React, Python)

Los SDKs de **Autorix** permiten a los desarrolladores integrar la gestión de identidad, tokens y evaluación de permisos ReBAC/ABAC en cualquier stack en cuestión de minutos.

---

## 1. Go SDK (`github.com/autorix-cl/autorix/sdk/go`)

### Instalación
```bash
go get github.com/autorix-cl/autorix/sdk/go
```

### Uso en Microservicios detrás de Autorix Aegis
```go
package main

import (
	"fmt"
	"net/http"

	"github.com/autorix-cl/autorix/sdk/go"
)

func main() {
	// 1. Inicializar cliente con caché local en memoria
	client := autorix.NewClient(autorix.Config{
		NexusAddr:   "nexus:50051",
		EnableCache: true,
	})

	// 2. Proteger endpoint con Middleware de Autorix
	http.Handle("/api/v1/documents/", client.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, _ := autorix.UserFromContext(r.Context())

		// 3. Evaluar permiso en Nexus (< 2ms)
		allowed, _ := client.Check(r.Context(), "document", "payroll_2026", "viewer", user.ID, nil)
		if !allowed {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		fmt.Fprintf(w, "Acceso concedido a %s", user.Email)
	})))

	http.ListenAndServe(":8080", nil)
}
```

---

## 2. TypeScript / React SDK (`@autorix/sdk-js`)

### Instalación
```bash
npm install @autorix/sdk-js
```

### Configuración del Provider en React / Next.js
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

### Uso de Hooks (`useSession`, `usePermission`)
```tsx
import { useSession, usePermission } from "@autorix/sdk-js";

export function DocumentViewer({ docId }: { docId: string }) {
  const { user, isAuthenticated, loading } = useSession();
  const { allowed, checking } = usePermission("document", docId, "editor");

  if (loading || checking) return <div>Cargando permisos...</div>;
  if (!isAuthenticated) return <div>Por favor iniciá sesión.</div>;

  return (
    <div>
      <h2>Bienvenido, {user?.traits.email}</h2>
      {allowed ? (
        <button>Editar Documento</button>
      ) : (
        <p>Solo lectura</p>
      )}
    </div>
  );
}
```

---

## 3. Python SDK (`autorix-python`)

### Instalación
```bash
pip install autorix
```

### Integración en FastAPI
```python
from fastapi import FastAPI, Depends, Header
from autorix.fastapi import AutorixSecurity
from autorix import AutorixClient

app = FastAPI()
auth = AutorixSecurity()
client = AutorixClient(nexus_url="http://nexus:50051")

@app.get("/api/reports/{report_id}")
async def get_report(
    report_id: str,
    x_user_id: str = Header(None),
    x_user_email: str = Header(None)
):
    user = auth.get_current_user(x_user_id=x_user_id, x_user_email=x_user_email)
    if not user:
        return {"error": "Unauthorized"}

    result = client.check("report", report_id, "viewer", user.id)
    if not result.allowed:
        return {"error": "Forbidden"}

    return {"report_id": report_id, "owner": user.email}
```
