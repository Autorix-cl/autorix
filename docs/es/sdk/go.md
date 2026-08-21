# SDK Oficial de Go para Autorix

El SDK oficial de Go (`github.com/autorix-cl/autorix/sdk/go`) ofrece clientes de alto rendimiento, seguros para concurrencia (goroutines) y basados en `context.Context` para interactuar con todos los motores de Autorix.

---

## 📦 Instalación

```bash
go get github.com/autorix-cl/autorix/sdk/go
```

---

## 🚀 Inicialización del Cliente

```go
package main

import (
	"context"
	"log"
	"time"

	"github.com/autorix-cl/autorix/sdk/go"
)

func main() {
	client, err := autorix.NewClient(
		autorix.WithNexus("http://localhost:8080"),
		autorix.WithThemis("http://localhost:4488"),
		autorix.WithVulcan("http://localhost:4466"),
		autorix.WithTimeout(5 * time.Second),
	)
	if err != nil {
		log.Fatalf("Error al inicializar cliente: %v", err)
	}

	// 1. Verificación de permisos con Nexus (ReBAC)
	allowed, err := client.Nexus.Check(context.Background(), autorix.CheckRequest{
		Namespace: "documents",
		Object:    "roadmap_2026",
		Relation:  "viewer",
		Subject:   "user:alice",
	})
	if err != nil || !allowed {
		log.Println("Acceso denegado")
	}
}
```
