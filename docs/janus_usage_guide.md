# Guía de Uso para Desarrolladores: Autorix Janus

**Autorix Janus** es el servidor de autorización OAuth 2.0 y proveedor OpenID Connect (OIDC) Headless del ecosistema Autorix (equivalente a ORY Hydra).

---

## 1. Conceptos y Arquitectura

Janus resuelve el desafío de seguridad y delegación para APIs:
* **Firma Asimétrica (RS256)**: Firma `access_token` e `id_token` JWT usando claves RSA que rota y expone públicamente en `/.well-known/jwks.json`.
* **Zero Latency Verification**: Tu API Gateway (Autorix Aegis) o tus microservicios solo descargan el JWKS una vez y validan criptográficamente millones de JWTs por segundo en memoria local.
* **PKCE Obligatorio (RFC 7636)**: Máxima seguridad para aplicaciones frontend (SPA / React) y aplicaciones móviles nativas.

---

## 2. Endpoints Públicos (`http://localhost:4444`)

| Endpoint | Protocolo | Descripción |
| :--- | :--- | :--- |
| `GET /.well-known/openid-configuration` | OIDC Discovery | Metadata de configuración de endpoints y algoritmos soportados |
| `GET /.well-known/jwks.json` | RFC 7517 | Conjunto de claves públicas para validación de JWTs |
| `POST /oauth2/token` | RFC 6749 | Intercambio de credenciales o códigos por tokens de acceso |
| `POST /admin/clients` | Admin API | Registro de nuevas aplicaciones cliente |

---

## 3. Guía Paso a Paso

### 1. Registrar una Aplicación Cliente (Machine-to-Machine)

```bash
curl -X POST http://localhost:4444/admin/clients \
  -H "Content-Type: application/json" \
  -d '{
    "client_id": "billing-service",
    "client_name": "Servicio de Facturación",
    "client_secret": "SuperSecretClientSecret123!",
    "grant_types": ["client_credentials"],
    "scopes": ["invoices:read", "invoices:write"],
    "is_public": false
  }'
```

---

### 2. Obtener un Access Token (Flujo `client_credentials`)

```bash
curl -X POST http://localhost:4444/oauth2/token \
  -u "billing-service:SuperSecretClientSecret123!" \
  -d "grant_type=client_credentials&scope=invoices:read"
```

**Respuesta (`200 OK`):**
```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIsImtpZCI6IjY3ODkw...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "scope": "invoices:read"
}
```

---

### 3. Verificar el Token con JWKS (`/.well-known/jwks.json`)

Podés consultar las claves públicas en cualquier momento:

```bash
curl http://localhost:4444/.well-known/jwks.json
```

**Respuesta:**
```json
{
  "keys": [
    {
      "kty": "RSA",
      "use": "sig",
      "alg": "RS256",
      "kid": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
      "n": "u1P5tQ...",
      "e": "AQAB"
    }
  ]
}
```

---

## 4. Validación de Tokens en Go (en tus Microservicios)

```go
package main

import (
	"fmt"
	"log"

	"github.com/golang-jwt/jwt/v5"
)

func ValidateTokenWithPublicKey(tokenString string, publicKeyPEM []byte) (*jwt.MapClaims, error) {
	pubKey, err := jwt.ParseRSAPublicKeyFromPEM(publicKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return pubKey, nil
	})
	if err != nil || !token.Valid {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	claims := token.Claims.(jwt.MapClaims)
	return &claims, nil
}
```
